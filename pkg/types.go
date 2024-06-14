package blobcache

import "time"

const (
	BlobCacheHostPrefix   string = "blobcache-host"
	BlobCacheClientPrefix string = "blobcache-client"
	BlobCacheVersion      string = "v0.1.0"
)

type BlobCacheConfig struct {
	Port                 uint            `key:"port" json:"port"`
	PersistencePath      string          `key:"persistencePath" json:"persistence_path"`
	MaxCacheSizeMb       int64           `key:"maxCacheSizeMb" json:"max_cache_size_mb"`
	PageSizeBytes        int64           `key:"pageSizeBytes" json:"page_size_bytes"`
	GRPCDialTimeoutS     int             `key:"grpcDialTimeoutS" json:"grpc_dial_timeout_s"`
	GRPCMessageSizeBytes int             `key:"grpcMessageSizeBytes" json:"grpc_message_size_bytes"`
	Tailscale            TailscaleConfig `key:"tailscale" json:"tailscale"`
	Metadata             MetadataConfig  `key:"metadata" json:"metadata"`
	DiscoveryIntervalS   int             `key:"discoveryIntervalS" json:"discovery_interval_s"`
}

type TailscaleConfig struct {
	ControlURL   string `key:"controlUrl" json:"control_url"`
	User         string `key:"user" json:"user"`
	AuthKey      string `key:"authKey" json:"auth_key"`
	HostName     string `key:"hostName" json:"host_name"`
	Debug        bool   `key:"debug" json:"debug"`
	Ephemeral    bool   `key:"ephemeral" json:"ephemeral"`
	StateDir     string `key:"stateDir" json:"state_dir"`
	DialTimeoutS int    `key:"dialTimeout" json:"dial_timeout"`
}

type MetadataConfig struct {
	RedisAddr       string `key:"redisAddr" json:"redis_addr"`
	RedisPasswd     string `key:"redisPasswd" json:"redis_passwd"`
	RedisTLSEnabled bool   `key:"redisTLSEnabled" json:"redis_tls_enabled"`
}

type BlobCacheHost struct {
	RTT  time.Duration
	Addr string
}

type ClientRequest struct {
	rt   ClientRequestType
	hash string
}

type ClientRequestType int

const (
	ClientRequestTypeStorage ClientRequestType = iota
	ClientRequestTypeRetrieval
)

type BlobCacheEntry struct {
	Hash    string `redis:"hash" json:"hash"`
	Size    int64  `redis:"size" json:"size"`
	Content []byte `redis:"content" json:"content"`
	Source  string `redis:"source" json:"source"`
}
