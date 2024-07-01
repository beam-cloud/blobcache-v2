package blobcache

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"time"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
)

type DirectoryAccessMetadata struct {
	PID        string `json:"pid"`        // ID of the parent directory, may be empty or a special value for root.
	ID         string `json:"id"`         // ID of the directory.
	Permission uint32 `json:"permission"` // Permissions for the directory, stored as an integer.
}

type DirectoryContentMetadata struct {
	Id         string               `json:"id"`         // ID of the directory.
	EntryList  []string             `json:"entryList"`  // List of file/directory names under this directory
	Timestamps map[string]time.Time `json:"timestamps"` // Timestamps for each entry.
}

type FileMetadata struct {
	ID       string `json:"id"`       // ID of the file
	PID      string `json:"pid"`      // ID of the parent directory.
	Name     string `json:"name"`     // Name of the file.
	FileData []byte `json:"fileData"` // You might store just a reference to S3 here.
}

type StorageLayer interface {
}

// Generates a directory ID based on the birth triple.
func GenerateDirectoryID(parentID, name string, version int) string {
	birthTriple := fmt.Sprintf("%s:%s:%d", parentID, name, version)
	hash := sha256.Sum256([]byte(birthTriple))
	return hex.EncodeToString(hash[:])
}

type BlobFsSystemOpts struct {
	Verbose  bool
	Metadata MetadataEngine
}

type BlobFs struct {
	root     *FSNode
	verbose  bool
	Metadata MetadataEngine
	Storage  StorageLayer
}

func Mount(opts FileSystemOpts) (func() error, <-chan error, error) {
	Logger.Infof("Mounting to %s\n", opts.MountPoint)

	if _, err := os.Stat(opts.MountPoint); os.IsNotExist(err) {
		err = os.MkdirAll(opts.MountPoint, 0755)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create mount point directory: %v", err)
		}

		Logger.Info("Mount point directory created.")
	}

	blobfs, err := NewFileSystem(BlobFsSystemOpts{Verbose: opts.Verbose, Metadata: opts.Metadata})
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
func NewFileSystem(opts BlobFsSystemOpts) (*BlobFs, error) {
	bfs := &BlobFs{
		verbose:  opts.Verbose,
		Metadata: opts.Metadata,
		// Storage:  storage,
	}

	rootID := GenerateDirectoryID("", "/", 0)
	rootPID := "" // Root node has no parent

	// Create the actual root filesystem node required by FUSE
	rootNode := &FSNode{
		filesystem: bfs,
		attr: fuse.Attr{
			Ino:  1,
			Mode: fuse.S_IFDIR | 0755,
		},

		bfsNode: &BlobFsNode{
			NodeType: DirNode,
			Path:     "/",
			ID:       rootID,
			PID:      rootPID,
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
