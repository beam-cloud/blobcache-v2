package blobcache_fs

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
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
	EntryList  []string             `json:"entryList"`  // List of file/directory names under this directory (Represents )
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
	log.Printf("Mounting to %s\n", opts.MountPoint)

	if _, err := os.Stat(opts.MountPoint); os.IsNotExist(err) {
		err = os.MkdirAll(opts.MountPoint, 0755)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create mount point directory: %v", err)
		}

		log.Println("Mount point directory created.")
	}

	infiniFs, err := NewFileSystem(BlobFsSystemOpts{Verbose: opts.Verbose})
	if err != nil {
		return nil, nil, fmt.Errorf("could not create filesystem: %v", err)
	}

	root, _ := infiniFs.Root()
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
	cfs := &BlobFs{
		verbose:  opts.Verbose,
		Metadata: opts.Metadata,
		// Storage:  storage,
	}

	rootID := GenerateDirectoryID("", "/", 0)
	rootPID := "" // Root node has no parent

	// Fetch or create root directory access metadata
	rootAccessMetadata, err := cfs.Metadata.GetDirectoryAccessMetadata(rootPID, "/")
	if err != nil {
		// Default permissions for root directory (e.g., drwxr-xr-x)
		rootAccessMetadata = &DirectoryAccessMetadata{
			PID:        rootPID,
			ID:         rootID,
			Permission: fuse.S_IFDIR | 0755, // Octal notation for permissions
		}
		if err := cfs.Metadata.SaveDirectoryAccessMetadata(rootAccessMetadata); err != nil {
			return nil, fmt.Errorf("failed to save root directory access metadata: %w", err)
		}
	}

	log.Println("root access metadata: ", rootAccessMetadata)

	// Fetch or create root directory content metadata
	// If there is no directory content metadata, this is sort of the equivlanet of a "format"
	// in that the root directory content metadata that's saved will no longer have any entries
	_, err = cfs.Metadata.GetDirectoryContentMetadata(rootID)
	if err != nil {
		rootContentMetadata := &DirectoryContentMetadata{
			Id:         rootID,
			EntryList:  []string{},                 // Empty list, no entries yet
			Timestamps: make(map[string]time.Time), // Empty timestamps
		}

		if err := cfs.Metadata.SaveDirectoryContentMetadata(rootContentMetadata); err != nil {
			return nil, fmt.Errorf("failed to save root directory content metadata: %w", err)
		}
	}

	// Create the actual root filesystem node required by FUSE
	rootNode := &FSNode{
		filesystem: cfs,
		attr: fuse.Attr{
			Ino:  1,
			Mode: rootAccessMetadata.Permission,
		},

		bfsNode: &BlobFsNode{
			NodeType: DirNode,
			Path:     "/",
			ID:       rootID,
			PID:      rootPID,
		},
	}

	cfs.root = rootNode
	return cfs, nil
}

func (cfs *BlobFs) Root() (fs.InodeEmbedder, error) {
	if cfs.root == nil {
		return nil, fmt.Errorf("root not initialized")
	}
	return cfs.root, nil
}
