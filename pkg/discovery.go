package blobcache

import (
	"context"

	"tailscale.com/client/tailscale"
)

type DiscoveryClient struct {
}

func (d *DiscoveryClient) Start() error {
	d.findNeighbors()
	return nil
}

func (d *DiscoveryClient) findNeighbors() error {
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

		// if strings.Contains(peer.HostName, serviceName) {
		// 	log.Printf("Found tailscale service<%s> @ %s\n", serviceName, peer.HostName)
		// 	return peer.HostName, nil
		// }
	}

	return nil
}
