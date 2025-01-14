package blobcache

import (
	"fmt"
	"time"
)

const (
	BlobCacheHostPrefix   string = "blobcache-host"
	BlobCacheClientPrefix string = "blobcache-client"
	BlobCacheVersion      string = "dev"
)

const (
	defaultHostStorageCapacityThresholdPct float64 = 0.95
	defaultHostKeepAliveIntervalS          int     = 10
	defaultHostKeepAliveTimeoutS           int     = 60
)

type DiscoveryMode string

const (
	DiscoveryModeTailscale DiscoveryMode = "tailscale"
	DiscoveryModeMetadata  DiscoveryMode = "metadata"
)

type BlobCacheConfig struct {
	Token                           string          `key:"token" json:"token"`
	DebugMode                       bool            `key:"debugMode" json:"debug_mode"`
	TLSEnabled                      bool            `key:"tlsEnabled" json:"tls_enabled"`
	Port                            uint            `key:"port" json:"port"`
	HostStorageCapacityThresholdPct float64         `key:"hostStorageCapacityThresholdPct" json:"host_storage_capacity_threshold_pct"`
	ObjectTtlS                      int             `key:"objectTtlS" json:"object_ttl_s"`
	RoundTripThresholdMilliseconds  uint            `key:"rttThresholdMilliseconds" json:"rtt_threshold_ms"`
	MaxCachePct                     int64           `key:"maxCachePct" json:"max_cache_pct"`
	PageSizeBytes                   int64           `key:"pageSizeBytes" json:"page_size_bytes"`
	GRPCDialTimeoutS                int             `key:"grpcDialTimeoutS" json:"grpc_dial_timeout_s"`
	GRPCMessageSizeBytes            int             `key:"grpcMessageSizeBytes" json:"grpc_message_size_bytes"`
	Tailscale                       TailscaleConfig `key:"tailscale" json:"tailscale"`
	Metadata                        MetadataConfig  `key:"metadata" json:"metadata"`
	DiscoveryIntervalS              int             `key:"discoveryIntervalS" json:"discovery_interval_s"`
	DiscoveryMode                   string          `key:"discoveryMode" json:"discovery_mode"`
	BlobFs                          BlobFsConfig    `key:"blobfs" json:"blobfs"`
}

type TailscaleConfig struct {
	WaitForAuth  bool   `key:"waitForAuth" json:"wait_for_auth"`
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

type BlobFsConfig struct {
	Enabled            bool                 `key:"enabled" json:"enabled"`
	Prefetch           BlobFsPrefetchConfig `key:"prefetch" json:"prefetch"`
	MountPoint         string               `key:"mountPoint" json:"mount_point"`
	Sources            []SourceConfig       `key:"sources" json:"sources"`
	MaxBackgroundTasks int                  `key:"maxBackgroundTasks" json:"max_background_tasks"`
	MaxWriteKB         int                  `key:"maxWriteKB" json:"max_write_kb"`
	MaxReadAheadKB     int                  `key:"maxReadAheadKB" json:"max_read_ahead_kb"`
	DirectMount        bool                 `key:"directMount" json:"direct_mount"`
	DirectIO           bool                 `key:"directIO" json:"direct_io"`
	Options            []string             `key:"options" json:"options"`
}

type BlobFsPrefetchConfig struct {
	Enabled           bool     `key:"enabled" json:"enabled"`
	MinFileSizeBytes  uint64   `key:"minFileSizeBytes" json:"min_file_size_bytes"`
	EvictionIntervalS int      `key:"evictionIntervalS" json:"eviction_interval_s"`
	IdleTtlS          int      `key:"idleTtlS" json:"idle_ttl_s"`
	WindowSizeBytes   uint64   `key:"windowSizeBytes" json:"window_size_bytes"`
	IgnoreFileExt     []string `key:"ignoreFileExt" json:"ignore_file_ext"`
	DataTimeoutS      int      `key:"dataTimeoutS" json:"data_timeout_s"`
}

type SourceConfig struct {
	Mode           string           `key:"mode" json:"mode"`
	FilesystemName string           `key:"fsName" json:"filesystem_name"`
	FilesystemPath string           `key:"fsPath" json:"filesystem_path"`
	JuiceFS        JuiceFSConfig    `key:"juicefs" json:"juicefs"`
	MountPoint     MountPointConfig `key:"mountpoint" json:"mountpoint"`
}

type JuiceFSConfig struct {
	RedisURI   string `key:"redisURI" json:"redis_uri"`
	Bucket     string `key:"bucket" json:"bucket"`
	AccessKey  string `key:"accessKey" json:"access_key"`
	SecretKey  string `key:"secretKey" json:"secret_key"`
	CacheSize  int64  `key:"cacheSize" json:"cache_size"`
	BlockSize  int64  `key:"blockSize" json:"block_size"`
	Prefetch   int64  `key:"prefetch" json:"prefetch"`
	BufferSize int64  `key:"bufferSize" json:"buffer_size"`
}

type MountPointConfig struct {
	BucketName  string `key:"bucketName" json:"bucket_name"`
	AccessKey   string `key:"accessKey" json:"access_key"`
	SecretKey   string `key:"secretKey" json:"secret_key"`
	Region      string `key:"region" json:"region"`
	EndpointURL string `key:"endpointUrl" json:"endpoint_url"`
}

type BlobCacheHost struct {
	RTT              time.Duration `redis:"rtt" json:"rtt"`
	Addr             string        `redis:"addr" json:"addr"`
	PrivateAddr      string        `redis:"private_addr" json:"private_addr"`
	CapacityUsagePct float64       `redis:"capacity_usage_pct" json:"capacity_usage_pct"`
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
	Hash         string `redis:"hash" json:"hash"`
	Size         int64  `redis:"size" json:"size"`
	SourcePath   string `redis:"source_path" json:"source_path"`
	SourceOffset int64  `redis:"source_offset" json:"source_offset"`
}

type ErrEntryNotFound struct {
	Hash string
}

func (e *ErrEntryNotFound) Error() string {
	return fmt.Sprintf("entry not found: %s", e.Hash)
}

type ErrNodeNotFound struct {
	Id string
}

func (e *ErrNodeNotFound) Error() string {
	return fmt.Sprintf("blobfs node not found: %s", e.Id)
}

// BlobFS types
type FileSystemOpts struct {
	MountPoint string
	Verbose    bool
	Metadata   *BlobCacheMetadata
}

type FileSystem interface {
	Mount(opts FileSystemOpts) (func() error, <-chan error, error)
	Unmount() error
	Format() error
}

type FileSystemStorage interface {
	Metadata()
	Get(string)
	ReadFile(interface{} /* This could be any sort of FS node type, depending on the implementation */, []byte, int64)
}
