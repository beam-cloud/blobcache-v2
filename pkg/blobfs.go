package blobcache

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
)

type BlobFsMetadata struct {
	PID       string `redis:"pid" json:"pid"`
	ID        string `redis:"id" json:"id"`
	Name      string `redis:"name" json:"name"`
	Path      string `redis:"path" json:"path"`
	Hash      string `redis:"hash" json:"hash"`
	Ino       uint64 `redis:"ino" json:"ino"`
	Size      uint64 `redis:"size" json:"size"`
	Blocks    uint64 `redis:"blocks" json:"blocks"`
	Atime     uint64 `redis:"atime" json:"atime"`
	Mtime     uint64 `redis:"mtime" json:"mtime"`
	Ctime     uint64 `redis:"ctime" json:"ctime"`
	Atimensec uint32 `redis:"atimensec" json:"atimensec"`
	Mtimensec uint32 `redis:"mtimensec" json:"mtimensec"`
	Ctimensec uint32 `redis:"ctimensec" json:"ctimensec"`
	Mode      uint32 `redis:"mode" json:"mode"`
	Nlink     uint32 `redis:"nlink" json:"nlink"`
	Rdev      uint32 `redis:"rdev" json:"rdev"`
	Blksize   uint32 `redis:"blksize" json:"blksize"`
	Padding   uint32 `redis:"padding" json:"padding"`
	Uid       uint32 `redis:"uid" json:"uid"`
	Gid       uint32 `redis:"gid" json:"gid"`
}

type StorageLayer interface {
}

// Generates a directory ID based on parent ID and name.
func GenerateFsID(name string) string {
	hash := sha256.Sum256([]byte(name))
	return hex.EncodeToString(hash[:])
}

// SHA1StringToUint64 converts the first 8 bytes of a SHA-1 hash string to a uint64
func SHA1StringToUint64(hash string) (uint64, error) {
	bytes, err := hex.DecodeString(hash[:16]) // first 8 bytes (16 hex characters)
	if err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint64(bytes), nil
}

type BlobFsSystemOpts struct {
	Verbose  bool
	Metadata *BlobCacheMetadata
	Config   BlobCacheConfig
	Client   *BlobCacheClient
}

type BlobFs struct {
	ctx      context.Context
	root     *FSNode
	verbose  bool
	Metadata *BlobCacheMetadata
	Client   *BlobCacheClient
	Config   BlobCacheConfig
}

func Mount(ctx context.Context, opts BlobFsSystemOpts) (func() error, <-chan error, error) {
	mountPoint := opts.Config.BlobFs.MountPoint
	Logger.Infof("Mounting to %s\n", mountPoint)

	if _, err := os.Stat(mountPoint); os.IsNotExist(err) {
		err = os.MkdirAll(mountPoint, 0755)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create mount point directory: %v", err)
		}

		Logger.Info("Mount point directory created.")
	} else if isFuseMount(mountPoint) {
		if err := forceUnmount(mountPoint); err != nil {
			return nil, nil, fmt.Errorf("failed to unmount existing FUSE mount: %v", err)
		}
	}

	blobfs, err := NewFileSystem(ctx, opts)
	if err != nil {
		return nil, nil, fmt.Errorf("could not create filesystem: %v", err)
	}

	root, _ := blobfs.Root()
	attrTimeout := time.Second * 60
	entryTimeout := time.Second * 60
	fsOptions := &fs.Options{
		AttrTimeout:  &attrTimeout,
		EntryTimeout: &entryTimeout,
	}
	server, err := fuse.NewServer(fs.NewNodeFS(root, fsOptions), mountPoint, &fuse.MountOptions{
		MaxBackground:        512,
		DisableXAttrs:        true,
		EnableSymlinkCaching: true,
		SyncRead:             false,
		RememberInodes:       true,
		MaxReadAhead:         1 << 17,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("could not create server: %v", err)
	}

	serverError := make(chan error, 1)
	startServer := func() error {
		go func() {
			go server.Serve()

			if err := server.WaitMount(); err != nil {
				serverError <- err
				return
			}

			server.Wait()
			close(serverError)
		}()

		return nil
	}

	return startServer, serverError, nil
}

// NewFileSystem initializes a new BlobFs with root metadata.
func NewFileSystem(ctx context.Context, opts BlobFsSystemOpts) (*BlobFs, error) {
	metadata := opts.Metadata

	bfs := &BlobFs{
		ctx:      ctx,
		verbose:  opts.Verbose,
		Config:   opts.Config,
		Client:   opts.Client,
		Metadata: metadata,
	}

	rootID := GenerateFsID("/")
	rootPID := "" // Root node has no parent
	rootPath := "/"

	dirMeta, err := metadata.GetFsNode(bfs.ctx, rootID)
	if err != nil || dirMeta == nil {
		log.Printf("Root node metadata not found, creating it now...\n")

		dirMeta = &BlobFsMetadata{PID: rootPID, ID: rootID, Path: rootPath, Ino: 1, Mode: fuse.S_IFDIR | 0755}

		err := metadata.SetFsNode(bfs.ctx, rootID, dirMeta)
		if err != nil {
			log.Fatalf("Unable to create blobfs root node dir metdata: %+v\n", err)
		}
	}

	// Create the actual root filesystem node required by FUSE
	attr := fuse.Attr{
		Ino:  1,
		Mode: dirMeta.Mode,
	}

	rootNode := &FSNode{
		filesystem: bfs,
		attr:       attr,

		bfsNode: &BlobFsNode{
			Path: dirMeta.Path,
			ID:   dirMeta.ID,
			PID:  dirMeta.PID,
			Attr: attr,
		},
	}

	bfs.root = rootNode
	return bfs, nil
}

func (bfs *BlobFs) Root() (fs.InodeEmbedder, error) {
	if bfs.root == nil {
		return nil, fmt.Errorf("root not initialized")
	}
	return bfs.root, nil
}

func isFuseMount(mountPoint string) bool {
	cmd := exec.Command("findmnt", "-n", "-o", "FSTYPE", mountPoint)
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(output), "fuse")
}

func forceUnmount(mountPoint string) error {
	cmd := exec.Command("fusermount", "-uz", mountPoint)
	if _, err := cmd.CombinedOutput(); err != nil {
		return err
	}
	return nil
}
