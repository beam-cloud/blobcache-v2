package blobcache

import (
	"context"

	proto "github.com/beam-cloud/blobcache-v2/proto"
)

func (cs *CacheService) GetAvailableHosts(ctx context.Context, req *proto.GetAvailableHostsRequest) (*proto.GetAvailableHostsResponse, error) {
	Logger.Debugf("GetAvailableHosts[ACK] - [%s]", req.Locality)

	hosts, err := cs.coordinator.GetAvailableHosts(ctx, req.Locality)
	if err != nil {
		return &proto.GetAvailableHostsResponse{Hosts: nil, Ok: false}, nil
	}

	protoHosts := make([]*proto.BlobCacheHost, 0)
	for _, host := range hosts {
		protoHosts = append(protoHosts, &proto.BlobCacheHost{HostId: host.HostId, Addr: host.Addr, PrivateIpAddr: host.PrivateAddr})
	}

	Logger.Debugf("GetAvailableHosts[OK] - [%s]", protoHosts)
	return &proto.GetAvailableHostsResponse{Hosts: protoHosts, Ok: true}, nil
}

func (cs *CacheService) SetClientLock(ctx context.Context, req *proto.SetClientLockRequest) (*proto.SetClientLockResponse, error) {
	Logger.Debugf("SetClientLock[ACK] - [%s]", req.Hash)

	err := cs.coordinator.SetClientLock(ctx, req.Hash, req.HostId)
	if err != nil {
		return &proto.SetClientLockResponse{Ok: false}, nil
	}

	return &proto.SetClientLockResponse{Ok: true}, nil
}

func (cs *CacheService) RemoveClientLock(ctx context.Context, req *proto.RemoveClientLockRequest) (*proto.RemoveClientLockResponse, error) {
	Logger.Debugf("RemoveClientLock[ACK] - [%s]", req.Hash)

	err := cs.coordinator.RemoveClientLock(ctx, req.Hash, req.HostId)
	if err != nil {
		return &proto.RemoveClientLockResponse{Ok: false}, nil
	}

	return &proto.RemoveClientLockResponse{Ok: true}, nil
}

func (cs *CacheService) SetStoreFromContentLock(ctx context.Context, req *proto.SetStoreFromContentLockRequest) (*proto.SetStoreFromContentLockResponse, error) {
	Logger.Infof("SetStoreFromContentLock[ACK] - [%s]", req.SourcePath)

	err := cs.coordinator.SetStoreFromContentLock(ctx, req.Locality, req.SourcePath)
	if err != nil {
		return &proto.SetStoreFromContentLockResponse{Ok: false}, nil
	}

	return &proto.SetStoreFromContentLockResponse{Ok: true}, nil
}

func (cs *CacheService) RemoveStoreFromContentLock(ctx context.Context, req *proto.RemoveStoreFromContentLockRequest) (*proto.RemoveStoreFromContentLockResponse, error) {
	Logger.Infof("RemoveStoreFromContentLock[ACK] - [%s]", req.SourcePath)

	err := cs.coordinator.RemoveStoreFromContentLock(ctx, req.Locality, req.SourcePath)
	if err != nil {
		return &proto.RemoveStoreFromContentLockResponse{Ok: false}, nil
	}

	return &proto.RemoveStoreFromContentLockResponse{Ok: true}, nil
}

func (cs *CacheService) RefreshStoreFromContentLock(ctx context.Context, req *proto.RefreshStoreFromContentLockRequest) (*proto.RefreshStoreFromContentLockResponse, error) {
	Logger.Infof("RefreshStoreFromContentLock[ACK] - [%s]", req.SourcePath)

	err := cs.coordinator.RefreshStoreFromContentLock(ctx, req.Locality, req.SourcePath)
	if err != nil {
		return &proto.RefreshStoreFromContentLockResponse{Ok: false}, nil
	}

	return &proto.RefreshStoreFromContentLockResponse{Ok: true}, nil
}

func (cs *CacheService) SetFsNode(ctx context.Context, req *proto.SetFsNodeRequest) (*proto.SetFsNodeResponse, error) {
	Logger.Debugf("SetFsNode[ACK] - [%s]", req.Id)

	metadata := &BlobFsMetadata{
		ID:        req.Metadata.Id,
		PID:       req.Metadata.Pid,
		Name:      req.Metadata.Name,
		Hash:      req.Metadata.Hash,
		Path:      req.Metadata.Path,
		Size:      req.Metadata.Size,
		Blocks:    req.Metadata.Blocks,
		Atime:     req.Metadata.Atime,
		Mtime:     req.Metadata.Mtime,
		Ctime:     req.Metadata.Ctime,
		Atimensec: req.Metadata.Atimensec,
		Mtimensec: req.Metadata.Mtimensec,
		Ctimensec: req.Metadata.Ctimensec,
		Mode:      req.Metadata.Mode,
		Nlink:     req.Metadata.Nlink,
		Rdev:      req.Metadata.Rdev,
		Blksize:   req.Metadata.Blksize,
		Padding:   req.Metadata.Padding,
		Uid:       req.Metadata.Uid,
		Gid:       req.Metadata.Gid,
		Gen:       req.Metadata.Gen,
	}

	err := cs.coordinator.SetFsNode(ctx, req.Id, metadata)
	if err != nil {
		return &proto.SetFsNodeResponse{Ok: false}, nil
	}

	return &proto.SetFsNodeResponse{Ok: true}, nil
}

func (cs *CacheService) GetFsNode(ctx context.Context, req *proto.GetFsNodeRequest) (*proto.GetFsNodeResponse, error) {
	Logger.Debugf("GetFsNode[ACK] - [%s]", req.Id)

	metadata, err := cs.coordinator.GetFsNode(ctx, req.Id)
	if err != nil {
		return &proto.GetFsNodeResponse{Metadata: nil, Ok: false}, nil
	}

	return &proto.GetFsNodeResponse{Metadata: metadata.ToProto(), Ok: true}, nil
}

func (cs *CacheService) GetFsNodeChildren(ctx context.Context, req *proto.GetFsNodeChildrenRequest) (*proto.GetFsNodeChildrenResponse, error) {
	Logger.Debugf("GetFsNodeChildren[ACK] - [%s]", req.Id)

	children, err := cs.coordinator.GetFsNodeChildren(ctx, req.Id)
	if err != nil {
		return &proto.GetFsNodeChildrenResponse{Children: nil, Ok: false}, nil
	}

	protoChildren := make([]*proto.BlobFsMetadata, 0)
	for _, child := range children {
		protoChildren = append(protoChildren, child.ToProto())
	}

	return &proto.GetFsNodeChildrenResponse{Children: protoChildren, Ok: true}, nil
}

func (cs *CacheService) AddFsNodeChild(ctx context.Context, req *proto.AddFsNodeChildRequest) (*proto.AddFsNodeChildResponse, error) {
	Logger.Debugf("AddFsNodeChild[ACK] - [%s]", req.Id)

	err := cs.coordinator.AddFsNodeChild(ctx, req.Pid, req.Id)
	if err != nil {
		return &proto.AddFsNodeChildResponse{Ok: false}, nil
	}

	return &proto.AddFsNodeChildResponse{Ok: true}, nil
}

func (cs *CacheService) AddHostToIndex(ctx context.Context, req *proto.AddHostToIndexRequest) (*proto.AddHostToIndexResponse, error) {
	Logger.Debugf("AddHostToIndex[ACK] - [%s]", req.Locality)

	host := &BlobCacheHost{
		HostId:      req.Host.HostId,
		Addr:        req.Host.Addr,
		PrivateAddr: req.Host.PrivateIpAddr,
	}

	err := cs.coordinator.AddHostToIndex(ctx, req.Locality, host)
	if err != nil {
		return &proto.AddHostToIndexResponse{Ok: false}, nil
	}

	return &proto.AddHostToIndexResponse{Ok: true}, nil
}

func (cs *CacheService) SetHostKeepAlive(ctx context.Context, req *proto.SetHostKeepAliveRequest) (*proto.SetHostKeepAliveResponse, error) {
	Logger.Debugf("SetHostKeepAlive[ACK] - [%s]", req.Locality)

	host := &BlobCacheHost{
		HostId:      req.Host.HostId,
		Addr:        req.Host.Addr,
		PrivateAddr: req.Host.PrivateIpAddr,
	}

	err := cs.coordinator.SetHostKeepAlive(ctx, req.Locality, host)
	if err != nil {
		return &proto.SetHostKeepAliveResponse{Ok: false}, nil
	}

	return &proto.SetHostKeepAliveResponse{Ok: true}, nil
}
