package blobcache_fs

import "github.com/hanwen/go-fuse/v2/fuse"

type FileSystemOpts struct {
	MountPoint string
	Verbose    bool
}

type FileSystem interface {
	Mount(opts FileSystemOpts) (func() error, <-chan error, error)
	Unmount() error
	Format() error
}

type FileSystemStorage interface {
	Metadata()
	Get(string)
	ListDirectory(string)
	ReadFile(interface{} /* This could be any sort of FS node type, depending on the implementation */, []byte, int64)
}

type MetadataEngine interface {
	GetDirectoryContentMetadata(id string) (*DirectoryContentMetadata, error)
	GetDirectoryAccessMetadata(pid, name string) (*DirectoryAccessMetadata, error)
	GetFileMetadata(pid, name string) (*FileMetadata, error)
	SaveDirectoryContentMetadata(contentMeta *DirectoryContentMetadata) error
	SaveDirectoryAccessMetadata(accessMeta *DirectoryAccessMetadata) error
	ListDirectory(string) []fuse.DirEntry
}
