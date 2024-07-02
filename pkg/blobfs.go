package blobcache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
)

type DirMetadata struct {
	Inode uint64 `redis:"inode" json:"inode"`
	PID   string `redis:"pid" json:"pid"`
	ID    string `redis:"id" json:"id"`
	Name  string `redis:"name" json:"name"`
	Mode  uint32 `redis:"mode" json:"mode"`
}

type FileMetadata struct {
	Inode uint64 `redis:"inode" json:"inode"`
	ID    string `redis:"pid" json:"pid"`
	PID   string `redis:"id" json:"id"`
	Name  string `redis:"name" json:"name"`
	Mode  uint32 `redis:"mode" json:"mode"`
}

type StorageLayer interface {
}

// Generates a directory ID based on parent ID and name.
func GenerateFsID(parentID, name string) string {
	hash := sha256.Sum256([]byte(fmt.Sprintf("%s:%s", parentID, name)))
	return hex.EncodeToString(hash[:])
}

type BlobFsSystemOpts struct {
	Verbose    bool
	Metadata   MetadataEngine
	MountPoint string
}

type BlobFs struct {
	ctx      context.Context
	root     *FSNode
	verbose  bool
	Metadata MetadataEngine
	Storage  StorageLayer
}

func Mount(ctx context.Context, opts BlobFsSystemOpts) (func() error, <-chan error, error) {
	Logger.Infof("Mounting to %s\n", opts.MountPoint)

	if _, err := os.Stat(opts.MountPoint); os.IsNotExist(err) {
		err = os.MkdirAll(opts.MountPoint, 0755)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create mount point directory: %v", err)
		}

		Logger.Info("Mount point directory created.")
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
	server, err := fuse.NewServer(fs.NewNodeFS(root, fsOptions), opts.MountPoint, &fuse.MountOptions{
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
		verbose:  opts.Verbose,
		Metadata: metadata,
		ctx:      ctx,
		// Storage:  storage,
	}

	rootID := GenerateFsID("", "/")
	rootPID := "" // Root node has no parent

	dirMeta, err := metadata.GetDirMetadata(bfs.ctx, rootID)
	if err != nil || dirMeta == nil {
		log.Printf("Root node metadata not found, creating it now...\n")

		dirMeta = &DirMetadata{PID: rootPID, ID: rootID, Mode: fuse.S_IFDIR | 0755, Inode: 1}

		err := metadata.SetDirMetadata(bfs.ctx, rootID, dirMeta)
		if err != nil {
			log.Fatalf("Unable to create blobfs root node dir metdata: %+v\n", err)
		}
	}

	// Create the actual root filesystem node required by FUSE
	rootNode := &FSNode{
		filesystem: bfs,
		attr: fuse.Attr{
			Ino:  1,
			Mode: dirMeta.Mode,
		},

		bfsNode: &BlobFsNode{
			NodeType: DirNode,
			Path:     "/",
			ID:       dirMeta.ID,
			PID:      dirMeta.PID,
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
