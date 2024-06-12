package blobcache

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"tailscale.com/tsnet"
)

type Tailscale struct {
	server      *tsnet.Server
	debug       bool
	initialized bool
	mu          sync.Mutex
}

// NewTailscale creates a new Tailscale instance using tsnet
func NewTailscale(cfg TailscaleConfig) *Tailscale {
	ts := &Tailscale{
		server: &tsnet.Server{
			Dir:        cfg.StateDir,
			Hostname:   cfg.HostName,
			AuthKey:    cfg.AuthKey,
			ControlURL: cfg.ControlURL,
			Ephemeral:  cfg.Ephemeral,
		},
		debug:       cfg.Debug,
		initialized: false,
		mu:          sync.Mutex{},
	}

	ts.server.Logf = ts.logF
	return ts
}

func (t *Tailscale) logF(format string, v ...interface{}) {
	if t.debug {
		log.Printf(format, v...)
	}
}

// Serve connects to a tailnet and serves a local service
func (t *Tailscale) Serve(ctx context.Context) (net.Listener, error) {
	log.Printf("Connecting to tailnet <ControlURL=%s>\n", t.server.ControlURL)

	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	addr := fmt.Sprintf(":%d", 2049)
	listener, err := t.server.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	_, err = t.server.Up(timeoutCtx)
	if err != nil {
		return nil, err
	}

	log.Printf("Connected to tailnet - listening on addr: %s\n", addr)
	return listener, nil
}

// Dial returns a TCP connection to a tailscale service
func (t *Tailscale) Dial(ctx context.Context, network, addr string) (net.Conn, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Connect to tailnet, if we aren't already
	if !t.initialized {
		t.mu.Lock()

		_, err := t.server.Up(timeoutCtx)
		if err != nil {
			t.mu.Unlock()
			return nil, err
		}

		t.initialized = true
		t.mu.Unlock()
	}

	conn, err := t.server.Dial(timeoutCtx, network, addr)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func (t *Tailscale) GetServer() *tsnet.Server {
	return t.server
}

// Stops the Tailscale server
func (t *Tailscale) Close() error {
	return t.server.Close()
}
