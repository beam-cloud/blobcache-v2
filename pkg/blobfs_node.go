package blobcache

import (
	"context"
	"fmt"
	"log"
	"path"
	"syscall"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
)

type BlobFsNodeType string

const (
	DirNode     BlobFsNodeType = "dir"
	FileNode    BlobFsNodeType = "file"
	SymLinkNode BlobFsNodeType = "symlink"
)

type BlobFsNode struct {
	NodeType BlobFsNodeType
	Path     string
	ID       string // The unique identifier for the directory, used for content metadata.
	PID      string // The ID of the parent directory, used for access and file metadata.
	Name     string // The name of the node, used together with PID for access and file metadata.
	Attr     fuse.Attr
	Target   string
	DataLen  int64 // Length of the node's data, used for file metadata.
}

// IsDir returns true if the BlobFsNode represents a directory.
func (n *BlobFsNode) IsDir() bool {
	return n.NodeType == DirNode
}

// IsSymlink returns true if the BlobFsNode represents a symlink.
func (n *BlobFsNode) IsSymlink() bool {
	return n.NodeType == SymLinkNode
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

	return fs.OK
}

func (n *FSNode) Lookup(ctx context.Context, name string, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	n.log("Lookup called with name: %s", name)

	// Construct the full path from the root
	fullPath := path.Join(n.bfsNode.Path, name)
	log.Println("Full path: ", fullPath)

	directoryId := GenerateFsID("", name)
	log.Println("directory id: ", directoryId)

	childPath := path.Join(n.bfsNode.Path, name)
	log.Println("childPath: ", childPath)

	// out.Attr = child.Attr
	// childInode := n.NewInode(ctx, &FSNode{filesystem: n.filesystem, bfsNode: child, attr: child.Attr}, fs.StableAttr{Mode: child.Attr.Mode, Ino: child.Attr.Ino})

	return nil, syscall.ENOENT
}

func (n *FSNode) Opendir(ctx context.Context) syscall.Errno {
	n.log("Opendir called")
	return 0
}

func (n *FSNode) Open(ctx context.Context, flags uint32) (fh fs.FileHandle, fuseFlags uint32, errno syscall.Errno) {
	n.log("Open called with flags: %v", flags)
	return nil, 0, fs.OK
}

func (n *FSNode) Read(ctx context.Context, f fs.FileHandle, dest []byte, off int64) (fuse.ReadResult, syscall.Errno) {
	n.log("Read called with offset: %v", off)

	// Don't even try to read 0 byte files
	if n.bfsNode.DataLen == 0 {
		nRead := 0
		return fuse.ReadResultData(dest[:nRead]), fs.OK
	}

	nRead := 0

	// nRead, err := n.filesystem.Storage.ReadFile(n.bfsNode, dest, off)
	// if err != nil {
	// 	return nil, syscall.EIO
	// }

	return fuse.ReadResultData(dest[:nRead]), fs.OK
}

func (n *FSNode) Readlink(ctx context.Context) ([]byte, syscall.Errno) {
	n.log("Readlink called")

	if n.bfsNode.NodeType != SymLinkNode {
		return nil, syscall.EINVAL
	}

	// Use the symlink target path directly
	symlinkTarget := n.bfsNode.Target

	// In this case, we don't need to read the file
	return []byte(symlinkTarget), fs.OK
}

func (n *FSNode) Readdir(ctx context.Context) (fs.DirStream, syscall.Errno) {
	n.log("Readdir called")
	dirEntries := []fuse.DirEntry{}
	return fs.NewListDirStream(dirEntries), fs.OK
}

func (n *FSNode) Create(ctx context.Context, name string, flags uint32, mode uint32, out *fuse.EntryOut) (inode *fs.Inode, fh fs.FileHandle, fuseFlags uint32, errno syscall.Errno) {
	n.log("Create called with name: %s, flags: %v, mode: %v", name, flags, mode)
	return nil, nil, 0, syscall.EROFS
}

func (n *FSNode) Mkdir(ctx context.Context, name string, mode uint32, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	n.log("Mkdir called with name: %s, mode: %v", name, mode)
	return nil, syscall.EROFS
}

func (n *FSNode) Rmdir(ctx context.Context, name string) syscall.Errno {
	n.log("Rmdir called with name: %s", name)
	return syscall.EROFS
}

func (n *FSNode) Unlink(ctx context.Context, name string) syscall.Errno {
	n.log("Unlink called with name: %s", name)
	return syscall.EROFS
}

func (n *FSNode) Rename(ctx context.Context, oldName string, newParent fs.InodeEmbedder, newName string, flags uint32) syscall.Errno {
	n.log("Rename called with oldName: %s, newName: %s, flags: %v", oldName, newName, flags)
	return syscall.EROFS
}
