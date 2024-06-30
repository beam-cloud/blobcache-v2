package blobcache

import (
	"fmt"
	"time"

	"github.com/hanwen/go-fuse/v2/fuse"
)

const (
	BlobCacheHostPrefix   string = "blobcache-host"
	BlobCacheClientPrefix string = "blobcache-client"
	BlobCacheVersion      string = "v0.1.0"
)

type BlobCacheConfig struct {
	Token                          string          `key:"token" json:"token"`
	DebugMode                      bool            `key:"debugMode" json:"debug_mode"`
	TLSEnabled                     bool            `key:"tlsEnabled" json:"tls_enabled"`
	Port                           uint            `key:"port" json:"port"`
	RoundTripThresholdMilliseconds uint            `key:"rttThresholdMilliseconds" json:"rtt_threshold_ms"`
	MaxCacheSizeMb                 int64           `key:"maxCacheSizeMb" json:"max_cache_size_mb"`
	PageSizeBytes                  int64           `key:"pageSizeBytes" json:"page_size_bytes"`
	GRPCDialTimeoutS               int             `key:"grpcDialTimeoutS" json:"grpc_dial_timeout_s"`
	GRPCMessageSizeBytes           int             `key:"grpcMessageSizeBytes" json:"grpc_message_size_bytes"`
	Tailscale                      TailscaleConfig `key:"tailscale" json:"tailscale"`
	Metadata                       MetadataConfig  `key:"metadata" json:"metadata"`
	DiscoveryIntervalS             int             `key:"discoveryIntervalS" json:"discovery_interval_s"`
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

type RedisMode string

var (
	RedisModeSingle  RedisMode = "single"
	RedisModeCluster RedisMode = "cluster"
)

type RedisConfig struct {
	Addrs              []string      `key:"addrs" json:"addrs"`
	Mode               RedisMode     `key:"mode" json:"mode"`
	ClientName         string        `key:"clientName" json:"client_name"`
	EnableTLS          bool          `key:"enableTLS" json:"enable_tls"`
	InsecureSkipVerify bool          `key:"insecureSkipVerify" json:"insecure_skip_verify"`
	MinIdleConns       int           `key:"minIdleConns" json:"min_idle_conns"`
	MaxIdleConns       int           `key:"maxIdleConns" json:"max_idle_conns"`
	ConnMaxIdleTime    time.Duration `key:"connMaxIdleTime" json:"conn_max_idle_time"`
	ConnMaxLifetime    time.Duration `key:"connMaxLifetime" json:"conn_max_lifetime"`
	DialTimeout        time.Duration `key:"dialTimeout" json:"dial_timeout"`
	ReadTimeout        time.Duration `key:"readTimeout" json:"read_timeout"`
	WriteTimeout       time.Duration `key:"writeTimeout" json:"write_timeout"`
	MaxRedirects       int           `key:"maxRedirects" json:"max_redirects"`
	MaxRetries         int           `key:"maxRetries" json:"max_retries"`
	PoolSize           int           `key:"poolSize" json:"pool_size"`
	Username           string        `key:"username" json:"username"`
	Password           string        `key:"password" json:"password"`
	RouteByLatency     bool          `key:"routeByLatency" json:"route_by_latency"`
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

type ContentSource string

const (
	ContentSourceS3      ContentSource = "s3://"
	ContentSourceJuiceFS ContentSource = "jfs://"
)

type BlobCacheEntry struct {
	Hash    string `redis:"hash" json:"hash"`
	Size    int64  `redis:"size" json:"size"`
	Content []byte `redis:"content" json:"content"`
	Source  string `redis:"source" json:"source"`
}

type ErrEntryNotFound struct {
	Hash string
}

func (e *ErrEntryNotFound) Error() string {
	return fmt.Sprintf("entry not found: %s", e.Hash)
}

// BlobFS types
type FileSystemOpts struct {
	MountPoint string
	Verbose    bool
	Metadata   MetadataEngine
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
