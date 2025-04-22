package blobcache

import (
	"context"
	"fmt"
	"path"
	"strings"
	"syscall"
	"time"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
)

type BlobFsNode struct {
	Path     string
	ID       string
	PID      string
	Name     string
	Target   string
	Hash     string
	Attr     fuse.Attr
	Prefetch *bool
}
type FSNode struct {
	fs.Inode
	filesystem *BlobFs
	bfsNode    *BlobFsNode
	attr       fuse.Attr
}

func (n *FSNode) log(format string, v ...interface{}) {
	if n.filesystem.verbose {
		Logger.Infof(fmt.Sprintf("(%s) %s", n.bfsNode.Path, format), v...)
	}
}

func (n *FSNode) OnAdd(ctx context.Context) {
	n.log("OnAdd called")
}

func (n *FSNode) Getattr(ctx context.Context, fh fs.FileHandle, out *fuse.AttrOut) syscall.Errno {
	n.log("Getattr called")

	node := n.bfsNode

	// Fill in the AttrOut struct
	out.Ino = node.Attr.Ino
	out.Size = node.Attr.Size
	out.Blocks = node.Attr.Blocks
	out.Atime = node.Attr.Atime
	out.Mtime = node.Attr.Mtime
	out.Ctime = node.Attr.Ctime
	out.Mode = node.Attr.Mode
	out.Nlink = node.Attr.Nlink
	out.Owner = node.Attr.Owner
	out.Atimensec = node.Attr.Atimensec
	out.Mtimensec = node.Attr.Mtimensec
	out.Ctimensec = node.Attr.Ctimensec

	return fs.OK
}

func metaToAttr(metadata *BlobFsMetadata) fuse.Attr {
	return fuse.Attr{
		Ino:       metadata.Ino,
		Size:      metadata.Size,
		Blocks:    metadata.Blocks,
		Atime:     metadata.Atime,
		Mtime:     metadata.Mtime,
		Ctime:     metadata.Ctime,
		Atimensec: metadata.Atimensec,
		Mtimensec: metadata.Mtimensec,
		Ctimensec: metadata.Ctimensec,
		Mode:      metadata.Mode,
		Nlink:     metadata.Nlink,
		Owner: fuse.Owner{
			Uid: metadata.Uid,
			Gid: metadata.Gid,
		},
		Rdev:    metadata.Rdev,
		Blksize: metadata.Blksize,
		Padding: metadata.Padding,
	}
}

func (n *FSNode) inodeFromFsId(ctx context.Context, fsId string) (*fs.Inode, *fuse.Attr, error) {
	metadata, err := n.filesystem.CoordinatorClient.GetFsNode(ctx, fsId)
	if err != nil {
		return nil, nil, syscall.ENOENT
	}

	// Fill out the child node's attributes
	attr := metaToAttr(metadata)

	// Create a new Inode on lookup
	node := n.NewInode(ctx,
		&FSNode{filesystem: n.filesystem, bfsNode: &BlobFsNode{
			Path:     metadata.Path,
			ID:       metadata.ID,
			PID:      metadata.PID,
			Name:     metadata.Name,
			Hash:     metadata.Hash,
			Attr:     attr,
			Target:   "",
			Prefetch: nil,
		}, attr: attr},
		fs.StableAttr{Mode: metadata.Mode, Ino: metadata.Ino, Gen: metadata.Gen},
	)

	return node, &attr, nil
}

func (n *FSNode) Lookup(ctx context.Context, name string, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	fullPath := path.Join(n.bfsNode.Path, name) // Construct the full of this file path from root
	n.log("Lookup called with path: %s", fullPath)

	// Force caching of a specific full path if the path contains a special illegal character '%'
	// This is a hack to trigger caching from external callers without going through the GRPC service directly
	if strings.Contains(fullPath, "%") {
		sourcePath := strings.ReplaceAll(fullPath, "%", "/")

		n.log("Storing content from source with path: %s", sourcePath)

		cacheSource := CacheSource{
			Path:        sourcePath,
			BucketName:  "",
			Region:      "",
			EndpointURL: "",
			AccessKey:   "",
			SecretKey:   "",
		}

		_, err := n.filesystem.Client.StoreContentFromSource(cacheSource)
		if err != nil {
			return nil, syscall.ENOENT
		}

		node, attr, err := n.inodeFromFsId(ctx, GenerateFsID(sourcePath))
		if err != nil {
			return nil, syscall.ENOENT
		}

		out.Attr = *attr
		return node, fs.OK
	}

	node, attr, err := n.inodeFromFsId(ctx, GenerateFsID(fullPath))
	if err != nil {
		return nil, syscall.ENOENT
	}

	out.Attr = *attr
	return node, fs.OK
}

func (n *FSNode) Opendir(ctx context.Context) syscall.Errno {
	n.log("Opendir called")
	return 0
}

func (n *FSNode) Open(ctx context.Context, flags uint32) (fh fs.FileHandle, fuseFlags uint32, errno syscall.Errno) {
	n.log("Open called with flags: %v", flags)

	// Enable DirectIO if specified
	if n.filesystem.Config.BlobFs.DirectIO {
		fuseFlags |= fuse.FOPEN_DIRECT_IO
		fuseFlags &= ^uint32(fuse.FOPEN_KEEP_CACHE)
		return nil, fuseFlags, fs.OK
	}

	return nil, 0, fs.OK
}

func (n *FSNode) Read(ctx context.Context, f fs.FileHandle, dest []byte, off int64) (fuse.ReadResult, syscall.Errno) {
	n.log("Read called with offset: %v", off)

	// Don't try to read 0 byte files
	if n.bfsNode.Attr.Size == 0 {
		return fuse.ReadResultData(dest[:0]), fs.OK
	}

	buffer, err := n.filesystem.Client.GetContent(n.bfsNode.Hash, off, int64(len(dest)))
	if err != nil {
		if err == ErrContentNotFound {

			sourcePath := n.bfsNode.Path
			cacheSource := CacheSource{
				Path:        sourcePath,
				BucketName:  "",
				Region:      "",
				EndpointURL: "",
				AccessKey:   "",
				SecretKey:   "",
			}
			_, err = n.filesystem.Client.StoreContentFromSourceWithLock(cacheSource)
			// If multiple clients try to store the same file, some may get ErrUnableToAcquireLock
			// In this case, we should tell the client to retry the Read instead of returning an error
			if err != nil && err == ErrUnableToAcquireLock {
				return nil, syscall.EAGAIN
			} else if err != nil {
				return nil, syscall.EIO
			}

			buffer, err = n.filesystem.Client.GetContent(n.bfsNode.Hash, off, int64(len(dest)))
			if err != nil {
				return nil, syscall.EIO
			}
		}

		return nil, syscall.EIO
	}

	return fuse.ReadResultData(buffer), fs.OK
}

func (n *FSNode) Readlink(ctx context.Context) ([]byte, syscall.Errno) {
	n.log("Readlink called")

	if n.bfsNode.Target == "" {
		return nil, syscall.EINVAL
	}

	// In this case, we don't need to read the file
	return []byte(n.bfsNode.Target), fs.OK
}

func (n *FSNode) Readdir(ctx context.Context) (fs.DirStream, syscall.Errno) {
	n.log("Readdir called")

	children, err := n.filesystem.CoordinatorClient.GetFsNodeChildren(ctx, GenerateFsID(n.bfsNode.Path))
	if err != nil {
		return nil, fs.ENOATTR
	}

	dirEntries := []fuse.DirEntry{}
	for _, child := range children {
		dirEntries = append(dirEntries, fuse.DirEntry{
			Mode: child.Mode,
			Name: child.Name,
			Ino:  child.Ino,
		})
	}

	return fs.NewListDirStream(dirEntries), fs.OK
}

func (n *FSNode) Create(ctx context.Context, name string, flags uint32, mode uint32, out *fuse.EntryOut) (inode *fs.Inode, fh fs.FileHandle, fuseFlags uint32, errno syscall.Errno) {
	n.log("Create called with name: %s, flags: %v, mode: %v", name, flags, mode)

	// Construct the full path for the new file
	fullPath := path.Join(n.bfsNode.Path, name)

	// Generate a new FsID for the file
	newFsId := GenerateFsID(fullPath)

	// Initialize default metadata for the new file
	now := time.Now()
	nowSec := uint64(now.Unix())
	nowNsec := uint32(now.Nanosecond())
	metadata := &BlobFsMetadata{
		PID:       n.bfsNode.ID,
		ID:        newFsId,
		Name:      name,
		Path:      fullPath,
		Ino:       0, // This will be set later
		Mode:      mode,
		Atime:     nowSec,
		Mtime:     nowSec,
		Ctime:     nowSec,
		Atimensec: nowNsec,
		Mtimensec: nowNsec,
		Ctimensec: nowNsec,
		Size:      0,
		Hash:      "", // No hash for a new file
	}

	// Set metadata in the coordinator
	err := n.filesystem.CoordinatorClient.SetFsNode(ctx, newFsId, metadata)
	if err != nil {
		return nil, nil, 0, syscall.EIO
	}

	// Add the new file as a child of the current node
	err = n.filesystem.CoordinatorClient.AddFsNodeChild(ctx, n.bfsNode.ID, newFsId)
	if err != nil {
		return nil, nil, 0, syscall.EIO
	}

	// Create a new inode for the file
	inode = n.NewInode(ctx, &FSNode{filesystem: n.filesystem, bfsNode: &BlobFsNode{
		Path: fullPath,
		ID:   newFsId,
		Name: name,
		Attr: fuse.Attr{
			Mode: mode,
		},
	}}, fs.StableAttr{Mode: mode, Ino: metadata.Ino})

	// Fill in the EntryOut struct
	out.Attr = fuse.Attr{
		Mode: mode,
		Ino:  metadata.Ino,
	}

	return inode, nil, 0, fs.OK
}

func (n *FSNode) Mkdir(ctx context.Context, name string, mode uint32, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	n.log("Mkdir called with name: %s, mode: %v", name, mode)

	// Construct the full path for the new directory
	fullPath := path.Join(n.bfsNode.Path, name)

	// Generate a new FsID for the directory
	newFsId := GenerateFsID(fullPath)

	// Initialize default metadata for the new directory
	now := time.Now()
	nowSec := uint64(now.Unix())
	nowNsec := uint32(now.Nanosecond())
	metadata := &BlobFsMetadata{
		PID:       n.bfsNode.ID,
		ID:        newFsId,
		Name:      name,
		Path:      fullPath,
		Ino:       0, // This will be set later
		Mode:      fuse.S_IFDIR | mode,
		Atime:     nowSec,
		Mtime:     nowSec,
		Ctime:     nowSec,
		Atimensec: nowNsec,
		Mtimensec: nowNsec,
		Ctimensec: nowNsec,
		Size:      0,
		Hash:      "", // No hash for a new directory
	}

	// Set metadata in the coordinator
	err := n.filesystem.CoordinatorClient.SetFsNode(ctx, newFsId, metadata)
	if err != nil {
		return nil, syscall.EIO
	}

	// Add the new directory as a child of the current node
	err = n.filesystem.CoordinatorClient.AddFsNodeChild(ctx, n.bfsNode.ID, newFsId)
	if err != nil {
		return nil, syscall.EIO
	}

	// Create a new inode for the directory
	inode := n.NewInode(ctx, &FSNode{filesystem: n.filesystem, bfsNode: &BlobFsNode{
		Path: fullPath,
		ID:   newFsId,
		Name: name,
		Attr: fuse.Attr{
			Mode: fuse.S_IFDIR | mode,
		},
	}}, fs.StableAttr{Mode: fuse.S_IFDIR | mode, Ino: metadata.Ino})

	// Fill in the EntryOut struct
	out.Attr = fuse.Attr{
		Mode: fuse.S_IFDIR | mode,
		Ino:  metadata.Ino,
	}

	return inode, fs.OK
}

func (n *FSNode) Rmdir(ctx context.Context, name string) syscall.Errno {
	n.log("Rmdir called with name: %s", name)

	// Construct the full path for the directory to be removed
	fullPath := path.Join(n.bfsNode.Path, name)

	// Generate the FsID for the directory
	fsId := GenerateFsID(fullPath)

	// Check if the directory is empty
	children, err := n.filesystem.CoordinatorClient.GetFsNodeChildren(ctx, fsId)
	if err != nil {
		return syscall.EIO
	}
	if len(children) > 0 {
		return syscall.ENOTEMPTY
	}

	// Remove the directory from the coordinator
	err = n.filesystem.CoordinatorClient.RemoveFsNode(ctx, fsId)
	if err != nil {
		return syscall.EIO
	}

	// Remove the directory from the parent's children
	err = n.filesystem.CoordinatorClient.RemoveFsNodeChild(ctx, n.bfsNode.ID, fsId)
	if err != nil {
		return syscall.EIO
	}

	return fs.OK
}

func (n *FSNode) Unlink(ctx context.Context, name string) syscall.Errno {
	n.log("Unlink called with name: %s", name)

	// Construct the full path for the file to be deleted
	fullPath := path.Join(n.bfsNode.Path, name)

	// Generate the FsID for the file
	fsId := GenerateFsID(fullPath)

	// Remove the file from the coordinator
	err := n.filesystem.CoordinatorClient.RemoveFsNode(ctx, fsId)
	if err != nil {
		return syscall.EIO
	}

	// Remove the file from the parent's children
	err = n.filesystem.CoordinatorClient.RemoveFsNodeChild(ctx, n.bfsNode.ID, fsId)
	if err != nil {
		return syscall.EIO
	}

	return fs.OK
}

func (n *FSNode) Rename(ctx context.Context, oldName string, newParent fs.InodeEmbedder, newName string, flags uint32) syscall.Errno {
	n.log("Rename called with oldName: %s, newName: %s, flags: %v", oldName, newName, flags)

	// Construct the full path for the old and new names
	oldFullPath := path.Join(n.bfsNode.Path, oldName)
	newFullPath := path.Join(n.bfsNode.Path, newName)

	// Generate the FsID for the old and new paths
	oldFsId := GenerateFsID(oldFullPath)
	newFsId := GenerateFsID(newFullPath)

	// Get the metadata for the old path
	metadata, err := n.filesystem.CoordinatorClient.GetFsNode(ctx, oldFsId)
	if err != nil {
		return syscall.ENOENT
	}

	// Update the metadata with the new name and path
	metadata.Name = newName
	metadata.Path = newFullPath

	// Set the updated metadata in the coordinator
	err = n.filesystem.CoordinatorClient.SetFsNode(ctx, newFsId, metadata)
	if err != nil {
		return syscall.EIO
	}

	// Remove the old node from the parent's children
	err = n.filesystem.CoordinatorClient.RemoveFsNodeChild(ctx, n.bfsNode.ID, oldFsId)
	if err != nil {
		return syscall.EIO
	}

	// Add the new node as a child of the parent
	err = n.filesystem.CoordinatorClient.AddFsNodeChild(ctx, n.bfsNode.ID, newFsId)
	if err != nil {
		return syscall.EIO
	}

	return fs.OK
}
