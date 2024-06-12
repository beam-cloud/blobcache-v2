package blobcache

type BlobCacheConfig struct {
	Port                 uint            `key:"port" json:"port"`
	PersistencePath      string          `key:"persistencePath" json:"persistence_path"`
	MaxCacheSizeMb       int64           `key:"maxCacheSizeMb" json:"max_cache_size_mb"`
	PageSizeBytes        int64           `key:"pageSizeBytes" json:"page_size_bytes"`
	GRPCMessageSizeBytes int             `key:"grpcMessageSizeBytes" json:"grpc_message_size_bytes"`
	Tailscale            TailscaleConfig `key:"tailscale" json:"tailscale"`
	Metadata             MetadataConfig  `key:"metadata" json:"metadata"`
}

type TailscaleConfig struct {
	ControlURL string `key:"controlUrl" json:"control_url"`
	User       string `key:"user" json:"user"`
	AuthKey    string `key:"authKey" json:"auth_key"`
	HostName   string `key:"hostName" json:"host_name"`
	Debug      bool   `key:"debug" json:"debug"`
	Ephemeral  bool   `key:"ephemeral" json:"ephemeral"`
	StateDir   string `key:"stateDir" json:"state_dir"`
}

type MetadataConfig struct {
	RedisAddr   string `key:"redisAddr" json:"redis_addr"`
	RedisPasswd string `key:"redisPasswd" json:"redis_passwd"`
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
