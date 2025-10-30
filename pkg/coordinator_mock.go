package blobcache

import (
	"context"
	"errors"
	"sync"
)

// MockCoordinator is a simple in-memory coordinator for testing
// Does not require Redis or any external dependencies
type MockCoordinator struct {
	hosts      map[string]map[string]*BlobCacheHost // locality -> hostId -> host
	fsNodes    map[string]*BlobFsMetadata           // id -> metadata
	fsChildren map[string][]string                  // parent id -> child ids
	locks      map[string]bool                      // lock keys
	mu         sync.RWMutex
}

func NewMockCoordinator() *MockCoordinator {
	return &MockCoordinator{
		hosts:      make(map[string]map[string]*BlobCacheHost),
		fsNodes:    make(map[string]*BlobFsMetadata),
		fsChildren: make(map[string][]string),
		locks:      make(map[string]bool),
	}
}

func (m *MockCoordinator) AddHostToIndex(ctx context.Context, locality string, host *BlobCacheHost) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if _, exists := m.hosts[locality]; !exists {
		m.hosts[locality] = make(map[string]*BlobCacheHost)
	}
	m.hosts[locality][host.HostId] = host
	return nil
}

func (m *MockCoordinator) SetHostKeepAlive(ctx context.Context, locality string, host *BlobCacheHost) error {
	return m.AddHostToIndex(ctx, locality, host)
}

func (m *MockCoordinator) GetAvailableHosts(ctx context.Context, locality string) ([]*BlobCacheHost, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	hosts := []*BlobCacheHost{}
	if localityHosts, exists := m.hosts[locality]; exists {
		for _, host := range localityHosts {
			hosts = append(hosts, host)
		}
	}
	return hosts, nil
}

func (m *MockCoordinator) GetRegionConfig(ctx context.Context, locality string) (BlobCacheServerConfig, error) {
	return BlobCacheServerConfig{}, errors.New("region config not found")
}

func (m *MockCoordinator) SetClientLock(ctx context.Context, hash string, host string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	key := "client-lock:" + hash
	if m.locks[key] {
		return errors.New("lock already held")
	}
	m.locks[key] = true
	return nil
}

func (m *MockCoordinator) RemoveClientLock(ctx context.Context, hash string, host string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	key := "client-lock:" + hash
	delete(m.locks, key)
	return nil
}

func (m *MockCoordinator) SetStoreFromContentLock(ctx context.Context, locality string, sourcePath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	key := "store-lock:" + locality + ":" + sourcePath
	if m.locks[key] {
		return errors.New("lock already held")
	}
	m.locks[key] = true
	return nil
}

func (m *MockCoordinator) RemoveStoreFromContentLock(ctx context.Context, locality string, sourcePath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	key := "store-lock:" + locality + ":" + sourcePath
	delete(m.locks, key)
	return nil
}

func (m *MockCoordinator) RefreshStoreFromContentLock(ctx context.Context, locality string, sourcePath string) error {
	// No-op for mock - locks don't expire
	return nil
}

func (m *MockCoordinator) SetFsNode(ctx context.Context, id string, metadata *BlobFsMetadata) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.fsNodes[id] = metadata
	return nil
}

func (m *MockCoordinator) GetFsNode(ctx context.Context, id string) (*BlobFsMetadata, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	metadata, exists := m.fsNodes[id]
	if !exists {
		return nil, errors.New("fs node not found")
	}
	return metadata, nil
}

func (m *MockCoordinator) RemoveFsNode(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	delete(m.fsNodes, id)
	return nil
}

func (m *MockCoordinator) RemoveFsNodeChild(ctx context.Context, pid, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if children, exists := m.fsChildren[pid]; exists {
		newChildren := []string{}
		for _, childId := range children {
			if childId != id {
				newChildren = append(newChildren, childId)
			}
		}
		m.fsChildren[pid] = newChildren
	}
	return nil
}

func (m *MockCoordinator) GetFsNodeChildren(ctx context.Context, id string) ([]*BlobFsMetadata, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	childIds, exists := m.fsChildren[id]
	if !exists {
		return []*BlobFsMetadata{}, nil
	}
	
	children := []*BlobFsMetadata{}
	for _, childId := range childIds {
		if metadata, exists := m.fsNodes[childId]; exists {
			children = append(children, metadata)
		}
	}
	return children, nil
}

func (m *MockCoordinator) AddFsNodeChild(ctx context.Context, pid, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if _, exists := m.fsChildren[pid]; !exists {
		m.fsChildren[pid] = []string{}
	}
	
	// Don't add duplicates
	for _, existingId := range m.fsChildren[pid] {
		if existingId == id {
			return nil
		}
	}
	
	m.fsChildren[pid] = append(m.fsChildren[pid], id)
	return nil
}
