package blobcache

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"tailscale.com/client/tailscale"
)

type DiscoveryClient struct {
}

func (d *DiscoveryClient) Start() error {
	d.findNeighbors()
	return nil
}

func (d *DiscoveryClient) findNeighbors() error {
	serviceName := "blobcache-"
	client := tailscale.LocalClient{}

	status, err := client.Status(context.Background())
	if err != nil {
		return err
	}

	// Iterate through the peers to find a matching service
	for _, peer := range status.Peer {
		if !peer.Online {
			continue
		}

		if strings.Contains(peer.HostName, serviceName) {
			log.Printf("Found service<%s> @ %s\n", serviceName, peer.HostName)
			d.checkService(peer.HostName)
		}
	}

	return nil
}

// checkService attempts to connect to the gRPC service and verifies its availability
func (d *DiscoveryClient) checkService(host string) bool {
	const port = 12345
	address := fmt.Sprintf("%s:%d", host, port)

	conn, err := grpc.Dial(address, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(1*time.Second))
	if err != nil {
		log.Printf("Failed to connect to gRPC service: %v", err)
		return false
	}
	defer conn.Close()

	// Check the connection state
	if conn.GetState() != connectivity.Ready {
		log.Printf("gRPC service not ready: %s", host)
		return false
	}

	return true
}
