package blobcache

import (
	"context"
	"crypto/tls"
	"errors"
	"net"

	proto "github.com/beam-cloud/blobcache-v2/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

type CoordinatorClientRemote struct {
	host   string
	cfg    BlobCacheGlobalConfig
	client proto.BlobCacheClient
}

func NewCoordinatorClientRemote(cfg BlobCacheGlobalConfig, token string) (CoordinatorClient, error) {
	transportCredentials := grpc.WithTransportCredentials(insecure.NewCredentials())

	isTLS := cfg.TLSEnabled
	if isTLS {
		h2creds := credentials.NewTLS(&tls.Config{NextProtos: []string{"h2"}})
		transportCredentials = grpc.WithTransportCredentials(h2creds)
	}

	var dialFunc func(context.Context, string) (net.Conn, error) = nil
	addr := cfg.CoordinatorHost

	dialFunc = DialWithTimeout

	maxMessageSize := cfg.GRPCMessageSizeBytes
	var dialOpts = []grpc.DialOption{
		transportCredentials,
		grpc.WithContextDialer(dialFunc),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(maxMessageSize),
			grpc.MaxCallSendMsgSize(maxMessageSize),
		),
	}

	if token != "" {
		dialOpts = append(dialOpts, grpc.WithUnaryInterceptor(GrpcAuthInterceptor(token)))
	}

	conn, err := grpc.Dial(addr, dialOpts...)
	if err != nil {
		return nil, err
	}

	return &CoordinatorClientRemote{cfg: cfg, host: cfg.CoordinatorHost, client: proto.NewBlobCacheClient(conn)}, nil
}

func (c *CoordinatorClientRemote) AddHostToIndex(ctx context.Context, locality string, host *BlobCacheHost) error {
	r, err := c.client.AddHostToIndex(ctx, &proto.AddHostToIndexRequest{Locality: locality, Host: &proto.BlobCacheHost{HostId: host.HostId, Addr: host.Addr, PrivateIpAddr: host.PrivateAddr}})
	if err != nil {
		return err
	}

	if !r.Ok {
		return errors.New("failed to add host to index")
	}

	return nil
}

func (c *CoordinatorClientRemote) SetHostKeepAlive(ctx context.Context, locality string, host *BlobCacheHost) error {
	r, err := c.client.SetHostKeepAlive(ctx, &proto.SetHostKeepAliveRequest{Locality: locality, Host: &proto.BlobCacheHost{HostId: host.HostId, Addr: host.Addr, PrivateIpAddr: host.PrivateAddr}})
	if err != nil {
		return err
	}

	if !r.Ok {
		return errors.New("failed to set host keep alive")
	}

	return nil
}

func (c *CoordinatorClientRemote) SetStoreFromContentLock(ctx context.Context, sourcePath string) error {
	r, err := c.client.SetStoreFromContentLock(ctx, &proto.SetStoreFromContentLockRequest{SourcePath: sourcePath})
	if err != nil {
		return err
	}

	if !r.Ok {
		return errors.New("failed to set store from content lock")
	}

	return nil
}

func (c *CoordinatorClientRemote) RemoveStoreFromContentLock(ctx context.Context, sourcePath string) error {
	r, err := c.client.RemoveStoreFromContentLock(ctx, &proto.RemoveStoreFromContentLockRequest{SourcePath: sourcePath})
	if err != nil {
		return err
	}

	if !r.Ok {
		return errors.New("failed to remove store from content lock")
	}

	return nil
}

func (c *CoordinatorClientRemote) RefreshStoreFromContentLock(ctx context.Context, sourcePath string) error {
	r, err := c.client.RefreshStoreFromContentLock(ctx, &proto.RefreshStoreFromContentLockRequest{SourcePath: sourcePath})
	if err != nil {
		return err
	}

	if !r.Ok {
		return errors.New("failed to refresh store from content lock")
	}

	return nil
}

func (c *CoordinatorClientRemote) GetAvailableHosts(ctx context.Context, locality string) ([]*BlobCacheHost, error) {
	response, err := c.client.GetAvailableHosts(ctx, &proto.GetAvailableHostsRequest{Locality: locality})
	if err != nil {
		return nil, err
	}

	hosts := make([]*BlobCacheHost, 0)
	for _, host := range response.Hosts {
		hosts = append(hosts, &BlobCacheHost{
			HostId:      host.HostId,
			Addr:        host.Addr,
			PrivateAddr: host.PrivateIpAddr,
		})
	}

	return hosts, nil
}

func (c *CoordinatorClientRemote) SetClientLock(ctx context.Context, hash string, hostId string) error {
	r, err := c.client.SetClientLock(ctx, &proto.SetClientLockRequest{Hash: hash, HostId: hostId})
	if err != nil {
		return err
	}

	if !r.Ok {
		return errors.New("failed to set client lock")
	}

	return nil
}

func (c *CoordinatorClientRemote) RemoveClientLock(ctx context.Context, hash string, hostId string) error {
	r, err := c.client.RemoveClientLock(ctx, &proto.RemoveClientLockRequest{Hash: hash, HostId: hostId})
	if err != nil {
		return err
	}

	if !r.Ok {
		return errors.New("failed to remove client lock")
	}

	return nil
}

func (c *CoordinatorClientRemote) SetFsNode(ctx context.Context, id string, metadata *BlobFsMetadata) error {
	r, err := c.client.SetFsNode(ctx, &proto.SetFsNodeRequest{
		Id: id,
		Metadata: &proto.BlobFsMetadata{
			Id:        metadata.ID,
			Pid:       metadata.PID,
			Name:      metadata.Name,
			Path:      metadata.Path,
			Hash:      metadata.Hash,
			Size:      metadata.Size,
			Blocks:    metadata.Blocks,
			Atime:     metadata.Atime,
			Mtime:     metadata.Mtime,
			Ctime:     metadata.Ctime,
			Atimensec: metadata.Atimensec,
			Mtimensec: metadata.Mtimensec,
			Ctimensec: metadata.Ctimensec,
			Mode:      metadata.Mode,
			Nlink:     metadata.Nlink,
			Rdev:      metadata.Rdev,
			Blksize:   metadata.Blksize,
			Padding:   metadata.Padding,
			Uid:       metadata.Uid,
			Gid:       metadata.Gid,
			Gen:       metadata.Gen,
		},
	})

	if err != nil {
		return err
	}

	if !r.Ok {
		return errors.New("failed to set fs node")
	}

	return nil
}

func (c *CoordinatorClientRemote) GetFsNode(ctx context.Context, id string) (*BlobFsMetadata, error) {
	response, err := c.client.GetFsNode(ctx, &proto.GetFsNodeRequest{Id: id})
	if err != nil {
		return nil, err
	}

	if !response.Ok {
		return nil, errors.New("failed to get fs node")
	}

	return &BlobFsMetadata{
		ID:        response.Metadata.Id,
		PID:       response.Metadata.Pid,
		Name:      response.Metadata.Name,
		Path:      response.Metadata.Path,
		Hash:      response.Metadata.Hash,
		Size:      response.Metadata.Size,
		Blocks:    response.Metadata.Blocks,
		Atime:     response.Metadata.Atime,
		Mtime:     response.Metadata.Mtime,
		Ctime:     response.Metadata.Ctime,
		Atimensec: response.Metadata.Atimensec,
		Mtimensec: response.Metadata.Mtimensec,
		Ctimensec: response.Metadata.Ctimensec,
		Mode:      response.Metadata.Mode,
		Nlink:     response.Metadata.Nlink,
		Rdev:      response.Metadata.Rdev,
		Blksize:   response.Metadata.Blksize,
		Padding:   response.Metadata.Padding,
		Uid:       response.Metadata.Uid,
		Gid:       response.Metadata.Gid,
		Gen:       response.Metadata.Gen,
	}, nil
}

func (c *CoordinatorClientRemote) GetFsNodeChildren(ctx context.Context, id string) ([]*BlobFsMetadata, error) {
	response, err := c.client.GetFsNodeChildren(ctx, &proto.GetFsNodeChildrenRequest{Id: id})
	if err != nil {
		return nil, err
	}

	if !response.Ok {
		return nil, errors.New("failed to get fs node children")
	}

	children := make([]*BlobFsMetadata, 0)
	for _, child := range response.Children {
		children = append(children, &BlobFsMetadata{
			ID:        child.Id,
			PID:       child.Pid,
			Name:      child.Name,
			Path:      child.Path,
			Hash:      child.Hash,
			Size:      child.Size,
			Blocks:    child.Blocks,
			Atime:     child.Atime,
			Mtime:     child.Mtime,
			Ctime:     child.Ctime,
			Atimensec: child.Atimensec,
			Mtimensec: child.Mtimensec,
			Ctimensec: child.Ctimensec,
			Mode:      child.Mode,
			Nlink:     child.Nlink,
			Rdev:      child.Rdev,
			Blksize:   child.Blksize,
			Padding:   child.Padding,
			Uid:       child.Uid,
			Gid:       child.Gid,
			Gen:       child.Gen,
		})
	}
	return children, nil
}
