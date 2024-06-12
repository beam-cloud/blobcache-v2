package blobcache

type BlobCacheConfig struct {
	Port                 uint   `key:"port" json:"port"`
	PersistencePath      string `key:"persistencePath" json:"persistence_path"`
	MaxCacheSizeMb       int64  `key:"maxCacheSizeMb" json:"max_cache_size_mb"`
	PageSizeBytes        int64  `key:"pageSizeBytes" json:"page_size_bytes"`
	GRPCMessageSizeBytes int    `key:"grpcMessageSizeBytes" json:"grpc_message_size_bytes"`
}

type BlobCacheHost struct {
}

type BlobCacheEntryChunk struct {
	Location BlobCacheHost
}

type BlobCacheEntry struct {
	Hash   string
	Size   uint64
	Chunks []BlobCacheEntryChunk
}
