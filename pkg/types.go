package blobcache

import (
	"os"
	"time"

	proto "github.com/beam-cloud/blobcache-v2/proto"
)

const (
	BlobCacheHostPrefix string = "blobcache-host"
	BlobCacheVersion    string = "dev"
)

const (
	defaultHostStorageCapacityThresholdPct float64 = 0.95
	defaultHostKeepAliveIntervalS          int     = 10
	defaultHostKeepAliveTimeoutS           int     = 60
)

type BlobCacheConfig struct {
	Server BlobCacheServerConfig `key:"server" json:"server"`
	Client BlobCacheClientConfig `key:"client" json:"client"`
	Global BlobCacheGlobalConfig `key:"global" json:"global"`
}

type BlobCacheGlobalConfig struct {
	DefaultLocality                 string  `key:"defaultLocality" json:"default_locality"`
	CoordinatorHost                 string  `key:"coordinatorHost" json:"coordinator_host"`
	ServerPort                      uint    `key:"serverPort" json:"server_port"`
	DiscoveryIntervalS              int     `key:"discoveryIntervalS" json:"discovery_interval_s"`
	RoundTripThresholdMilliseconds  uint    `key:"rttThresholdMilliseconds" json:"rtt_threshold_ms"`
	HostStorageCapacityThresholdPct float64 `key:"hostStorageCapacityThresholdPct" json:"host_storage_capacity_threshold_pct"`
	GRPCDialTimeoutS                int     `key:"grpcDialTimeoutS" json:"grpc_dial_timeout_s"`
	GRPCMessageSizeBytes            int     `key:"grpcMessageSizeBytes" json:"grpc_message_size_bytes"`
	DebugMode                       bool    `key:"debugMode" json:"debug_mode"`
	PrettyLogs                      bool    `key:"prettyLogs" json:"pretty_logs"`
}

func (c *BlobCacheGlobalConfig) GetLocality() string {
	locality := os.Getenv("BLOBCACHE_LOCALITY")
	if locality != "" {
		return locality
	}

	return c.DefaultLocality
}

type BlobCacheServerMode string

const (
	BlobCacheServerModeCoordinator BlobCacheServerMode = "coordinator"
	BlobCacheServerModeSlave       BlobCacheServerMode = "slave"
)

type BlobCacheServerConfig struct {
	Mode                 BlobCacheServerMode `key:"mode" json:"mode"`
	DiskCacheDir         string              `key:"diskCacheDir" json:"disk_cache_dir"`
	DiskCacheMaxUsagePct float64             `key:"diskCacheMaxUsagePct" json:"disk_cache_max_usage_pct"`
	Token                string              `key:"token" json:"token"`
	PrettyLogs           bool                `key:"prettyLogs" json:"pretty_logs"`
	ObjectTtlS           int                 `key:"objectTtlS" json:"object_ttl_s"`
	MaxCachePct          int64               `key:"maxCachePct" json:"max_cache_pct"`
	PageSizeBytes        int64               `key:"pageSizeBytes" json:"page_size_bytes"`
	Metadata             MetadataConfig      `key:"metadata" json:"metadata"`
	Sources              []SourceConfig      `key:"sources" json:"sources"`

	// Allows a coordinator to override a slave server's config for a specific locality/region
	Regions map[string]RegionConfig `key:"regions" json:"regions"`
}

func (c *BlobCacheServerConfig) ToProto() *proto.BlobCacheServerConfig {
	protoConfig := &proto.BlobCacheServerConfig{
		Mode:                 string(c.Mode),
		DiskCacheDir:         c.DiskCacheDir,
		DiskCacheMaxUsagePct: float32(c.DiskCacheMaxUsagePct),
		MaxCachePct:          c.MaxCachePct,
		PageSizeBytes:        c.PageSizeBytes,
		ObjectTtlS:           int64(c.ObjectTtlS),
		PrettyLogs:           c.PrettyLogs,
		Token:                c.Token,
		Sources:              make([]*proto.SourceConfig, 0),
	}

	for _, source := range c.Sources {
		protoSource := &proto.SourceConfig{
			Mode:           string(source.Mode),
			FilesystemName: source.FilesystemName,
			FilesystemPath: source.FilesystemPath,
		}

		switch source.Mode {
		case SourceModeMountPoint:
			protoSource.Mountpoint = &proto.MountPointConfig{
				BucketName:     source.MountPoint.BucketName,
				AccessKey:      source.MountPoint.AccessKey,
				SecretKey:      source.MountPoint.SecretKey,
				Region:         source.MountPoint.Region,
				EndpointUrl:    source.MountPoint.EndpointURL,
				ForcePathStyle: source.MountPoint.ForcePathStyle,
			}

		case SourceModeJuiceFS:
			protoSource.Juicefs = &proto.JuiceFSConfig{
				RedisUri:  source.JuiceFS.RedisURI,
				Bucket:    source.JuiceFS.Bucket,
				AccessKey: source.JuiceFS.AccessKey,
				SecretKey: source.JuiceFS.SecretKey,
			}

		}

		protoConfig.Sources = append(protoConfig.Sources, protoSource)
	}

	return protoConfig
}

func BlobCacheServerConfigFromProto(protoConfig *proto.BlobCacheServerConfig) BlobCacheServerConfig {
	cfg := BlobCacheServerConfig{
		Mode:                 BlobCacheServerMode(protoConfig.Mode),
		DiskCacheDir:         protoConfig.DiskCacheDir,
		DiskCacheMaxUsagePct: float64(protoConfig.DiskCacheMaxUsagePct),
		MaxCachePct:          protoConfig.MaxCachePct,
		PageSizeBytes:        protoConfig.PageSizeBytes,
		ObjectTtlS:           int(protoConfig.ObjectTtlS),
		PrettyLogs:           protoConfig.PrettyLogs,
		Token:                protoConfig.Token,
		Sources:              make([]SourceConfig, len(protoConfig.Sources)),
	}

	for i, protoSource := range protoConfig.Sources {
		localSource := SourceConfig{
			Mode:           protoSource.Mode,
			FilesystemName: protoSource.FilesystemName,
			FilesystemPath: protoSource.FilesystemPath,
		}

		switch protoSource.Mode {
		case SourceModeMountPoint:
			localSource.MountPoint = MountPointConfig{
				BucketName:     protoSource.Mountpoint.BucketName,
				AccessKey:      protoSource.Mountpoint.AccessKey,
				SecretKey:      protoSource.Mountpoint.SecretKey,
				Region:         protoSource.Mountpoint.Region,
				EndpointURL:    protoSource.Mountpoint.EndpointUrl,
				ForcePathStyle: protoSource.Mountpoint.ForcePathStyle,
			}

		case SourceModeJuiceFS:
			localSource.JuiceFS = JuiceFSConfig{
				RedisURI:  protoSource.Juicefs.RedisUri,
				Bucket:    protoSource.Juicefs.Bucket,
				AccessKey: protoSource.Juicefs.AccessKey,
				SecretKey: protoSource.Juicefs.SecretKey,
			}
		}

		cfg.Sources[i] = localSource
	}

	return cfg
}

type RegionConfig struct {
	ServerConfig BlobCacheServerConfig `key:"server" json:"server"`
}

type BlobCacheClientConfig struct {
	Token                 string       `key:"token" json:"token"`
	MinRetryLengthBytes   int64        `key:"minRetryLengthBytes" json:"min_retry_length_bytes"`
	MaxGetContentAttempts int          `key:"maxGetContentAttempts" json:"max_get_content_attempts"`
	NTopHosts             int          `key:"nTopHosts" json:"n_top_hosts"`
	BlobFs                BlobFsConfig `key:"blobfs" json:"blobfs"`
}

type BlobCacheMetadataMode string

const (
	BlobCacheMetadataModeDefault BlobCacheMetadataMode = "default"
	BlobCacheMetadataModeLocal   BlobCacheMetadataMode = "local"
)

type ValkeyConfig struct {
	PrimaryName     string                `key:"primaryName" json:"primary_name"`
	Password        string                `key:"password" json:"password"`
	TLS             bool                  `key:"tls" json:"tls"`
	Host            string                `key:"host" json:"host"`
	Port            int                   `key:"port" json:"port"`
	ExistingPrimary ValkeyExistingPrimary `key:"existingPrimary" json:"existingPrimary"`
}

type ValkeyExistingPrimary struct {
	Host string `key:"host" json:"host"`
	Port int    `key:"port" json:"port"`
}

type MetadataConfig struct {
	Mode         BlobCacheMetadataMode `key:"mode" json:"mode"`
	ValkeyConfig ValkeyConfig          `key:"valkey" json:"valkey"`

	// Default config
	RedisAddr       string    `key:"redisAddr" json:"redis_addr"`
	RedisPasswd     string    `key:"redisPasswd" json:"redis_passwd"`
	RedisTLSEnabled bool      `key:"redisTLSEnabled" json:"redis_tls_enabled"`
	RedisMode       RedisMode `key:"redisMode" json:"redis_mode"`
	RedisMasterName string    `key:"redisMasterName" json:"redis_master_name"`
}

type RedisMode string

var (
	RedisModeSingle   RedisMode = "single"
	RedisModeCluster  RedisMode = "cluster"
	RedisModeSentinel RedisMode = "sentinel"
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
	MasterName         string        `key:"masterName" json:"master_name"`
	SentinelPassword   string        `key:"sentinelPassword" json:"sentinel_password"`
}

type BlobFsConfig struct {
	Enabled            bool     `key:"enabled" json:"enabled"`
	MountPoint         string   `key:"mountPoint" json:"mount_point"`
	MaxBackgroundTasks int      `key:"maxBackgroundTasks" json:"max_background_tasks"`
	MaxWriteKB         int      `key:"maxWriteKB" json:"max_write_kb"`
	MaxReadAheadKB     int      `key:"maxReadAheadKB" json:"max_read_ahead_kb"`
	DirectMount        bool     `key:"directMount" json:"direct_mount"`
	DirectIO           bool     `key:"directIO" json:"direct_io"`
	Options            []string `key:"options" json:"options"`
}

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
	Gen       uint64 `redis:"gen" json:"gen"`
}

func (m *BlobFsMetadata) ToProto() *proto.BlobFsMetadata {
	return &proto.BlobFsMetadata{
		Id:        m.ID,
		Pid:       m.PID,
		Name:      m.Name,
		Path:      m.Path,
		Hash:      m.Hash,
		Size:      m.Size,
		Blocks:    m.Blocks,
		Atime:     m.Atime,
		Mtime:     m.Mtime,
		Ctime:     m.Ctime,
		Atimensec: m.Atimensec,
		Mtimensec: m.Mtimensec,
		Ctimensec: m.Ctimensec,
		Mode:      m.Mode,
		Nlink:     m.Nlink,
		Rdev:      m.Rdev,
		Blksize:   m.Blksize,
		Padding:   m.Padding,
		Uid:       m.Uid,
		Gid:       m.Gid,
		Gen:       m.Gen,
	}
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
	BucketName     string `key:"bucketName" json:"bucket_name"`
	AccessKey      string `key:"accessKey" json:"access_key"`
	SecretKey      string `key:"secretKey" json:"secret_key"`
	Region         string `key:"region" json:"region"`
	EndpointURL    string `key:"endpointUrl" json:"endpoint_url"`
	ForcePathStyle bool   `key:"forcePathStyle" json:"force_path_style"`
}

type BlobCacheHost struct {
	RTT              time.Duration `redis:"rtt" json:"rtt"`
	HostId           string        `redis:"host_id" json:"host_id"`
	Addr             string        `redis:"addr" json:"addr"`
	PrivateAddr      string        `redis:"private_addr" json:"private_addr"`
	CapacityUsagePct float64       `redis:"capacity_usage_pct" json:"capacity_usage_pct"`
}

// Bytes is needed for the rendezvous hasher
func (h *BlobCacheHost) Bytes() []byte {
	return []byte(h.HostId)
}

func (h *BlobCacheHost) ToProto() *proto.BlobCacheHost {
	return &proto.BlobCacheHost{
		HostId:           h.HostId,
		Addr:             h.Addr,
		PrivateIpAddr:    h.PrivateAddr,
		CapacityUsagePct: float32(h.CapacityUsagePct),
	}
}

type ClientRequest struct {
	rt        ClientRequestType
	hash      string
	key       string // This key is used for rendezvous hashing / deterministic routing (it's usually the same as the hash)
	hostIndex int
}

type ClientRequestType int

const (
	ClientRequestTypeStorage ClientRequestType = iota
	ClientRequestTypeRetrieval
)

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
