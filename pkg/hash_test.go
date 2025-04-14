package blobcache

import (
	"fmt"
	"reflect"
	"testing"
)

var sampleKeys = []string{
	"352DAB08-C1FD-4462-B573-7640B730B721",
	"382080D3-B847-4BB5-AEA8-644C3E56F4E1",
	"2B340C12-7958-4DBE-952C-67496E15D0C8",
	"BE05F82B-902E-4868-8CC9-EE50A6C64636",
	"C7ECC571-E924-4523-A313-951DFD5D8073",
}

type getTestcase struct {
	key          string
	expectedHost *BlobCacheHost
}

func TestHashGet(t *testing.T) {
	hostMap := NewHostMap(BlobCacheGlobalConfig{}, nil)

	hostMap.Set(&BlobCacheHost{Host: "a"})
	hostMap.Set(&BlobCacheHost{Host: "b"})
	hostMap.Set(&BlobCacheHost{Host: "c"})
	hostMap.Set(&BlobCacheHost{Host: "d"})
	hostMap.Set(&BlobCacheHost{Host: "e"})

	hash := NewRendezvousHasher()
	hash.Add(hostMap.GetAll()...)

	gotHost := hash.Get("foo")
	if gotHost != nil && gotHost.Host != "e" {
		t.Errorf("got: %#v, expected: %#v", gotHost, &BlobCacheHost{Host: "e"})
	}

	hash.Add(hostMap.GetAll()...)

	testcases := []getTestcase{
		{"", &BlobCacheHost{Host: "d"}},
		{"foo", &BlobCacheHost{Host: "e"}},
		{"bar", &BlobCacheHost{Host: "c"}},
	}

	for _, testcase := range testcases {
		gotHost := hash.Get(testcase.key)
		if gotHost.Host != testcase.expectedHost.Host {
			t.Errorf("got: %#v, expected: %#v", gotHost, testcase.expectedHost)
		}
	}
}

type getNTestcase struct {
	count         int
	key           string
	expectedHosts []*BlobCacheHost
}

func Test_Hash_GetN(t *testing.T) {
	hostMap := NewHostMap(BlobCacheGlobalConfig{}, nil)

	hash := NewRendezvousHasher()

	hostMap.Set(&BlobCacheHost{Host: "a"})
	hostMap.Set(&BlobCacheHost{Host: "b"})
	hostMap.Set(&BlobCacheHost{Host: "c"})
	hostMap.Set(&BlobCacheHost{Host: "d"})
	hostMap.Set(&BlobCacheHost{Host: "e"})

	hash.Add(hostMap.GetAll()...)

	testcases := []getNTestcase{
		{1, "foo", []*BlobCacheHost{{Host: "e"}}},
		{2, "bar", []*BlobCacheHost{{Host: "c"}, {Host: "e"}}},
		{3, "baz", []*BlobCacheHost{{Host: "d"}, {Host: "a"}, {Host: "b"}}},
		{2, "biz", []*BlobCacheHost{{Host: "b"}, {Host: "a"}}},
		{0, "boz", []*BlobCacheHost{}},
		{100, "floo", []*BlobCacheHost{{Host: "d"}, {Host: "a"}, {Host: "b"}, {Host: "c"}, {Host: "e"}}},
	}

	for _, testcase := range testcases {
		gotHosts := hash.GetN(testcase.count, testcase.key)
		if !reflect.DeepEqual(gotHosts, testcase.expectedHosts) {
			t.Errorf("got: %#v, expected: %#v", gotHosts, testcase.expectedHosts)
		}
	}
}

func TestHashRemove(t *testing.T) {
	hostMap := NewHostMap(BlobCacheGlobalConfig{}, nil)

	hostMap.Set(&BlobCacheHost{Host: "a"})
	hostMap.Set(&BlobCacheHost{Host: "b"})
	hostMap.Set(&BlobCacheHost{Host: "c"})
	hostMap.Set(&BlobCacheHost{Host: "d"})
	hostMap.Set(&BlobCacheHost{Host: "e"})

	hash := NewRendezvousHasher()
	hash.Add(hostMap.GetAll()...)

	var keyForB string
	for i := 0; i < 10000; i++ {
		randomKey := fmt.Sprintf("key-%d", i)
		if hash.Get(randomKey).Host == "b" {
			keyForB = randomKey
			break
		}
	}

	if keyForB == "" {
		t.Fatalf("Failed to find a key that maps to 'b'")
	}

	hash.Remove(hostMap.Get("b"))

	// Check if the key now maps to a different node
	newNode := hash.Get(keyForB)
	if newNode.Host == "b" {
		t.Errorf("Key %s still maps to removed node 'b'", keyForB)
	}

	if newNode == nil {
		t.Errorf("Key %s does not map to any node after removing 'b'", keyForB)
	}
}
