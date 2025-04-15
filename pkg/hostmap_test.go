package blobcache

import (
	"testing"
	"time"
)

func TestClosestWithCapacity(t *testing.T) {
	hostMap := &HostMap{
		hosts: map[string]*BlobCacheHost{
			"host1": {RTT: 20 * time.Millisecond, CapacityUsagePct: 0.3},
			"host2": {RTT: 10 * time.Millisecond, CapacityUsagePct: 0.7},
			"host3": {RTT: 10 * time.Millisecond, CapacityUsagePct: 0.2},
			"host4": {RTT: 50 * time.Millisecond, CapacityUsagePct: 0.1},
		},
		cfg: BlobCacheGlobalConfig{
			HostStorageCapacityThresholdPct: 0.5,
		},
	}

	expectedHost := &BlobCacheHost{RTT: 10 * time.Millisecond, CapacityUsagePct: 0.2}

	host, err := hostMap.ClosestWithCapacity(5 * time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if host.RTT != expectedHost.RTT || host.CapacityUsagePct != expectedHost.CapacityUsagePct {
		t.Fatalf("expected host RTT: %v, CapacityUsagePct: %f; got RTT: %v, CapacityUsagePct: %f",
			expectedHost.RTT, expectedHost.CapacityUsagePct, host.RTT, host.CapacityUsagePct)
	}
}
