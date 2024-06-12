package blobcache

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"tailscale.com/tsnet"
)

type Tailscale struct {
	cfg      BlobCacheConfig
	server   *tsnet.Server
	mu       sync.Mutex
	hostname string
}

// NewTailscale creates a new Tailscale instance using tsnet
func NewTailscale(hostname string, cfg BlobCacheConfig) *Tailscale {
	return &Tailscale{
		hostname: hostname,
		cfg:      cfg,
		mu:       sync.Mutex{},
	}
}

func (t *Tailscale) logF(format string, v ...interface{}) {
	if t.cfg.Tailscale.Debug {
		log.Printf(format, v...)
	}
}

func (t *Tailscale) GetOrCreateServer() *tsnet.Server {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.server != nil {
		return t.server
	}

	t.server = &tsnet.Server{
		Hostname:   t.hostname,
		Dir:        fmt.Sprintf("%s/%s", t.cfg.Tailscale.StateDir, t.hostname),
		AuthKey:    t.cfg.Tailscale.AuthKey,
		ControlURL: t.cfg.Tailscale.ControlURL,
		Ephemeral:  t.cfg.Tailscale.Ephemeral,
	}

	t.server.Logf = t.logF
	return t.server
}

// Dial returns a TCP connection to a tailscale service
func (t *Tailscale) Dial(ctx context.Context, addr string) (net.Conn, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(t.cfg.Tailscale.DialTimeoutS*int(time.Second)))
	defer cancel()

	if t.server == nil {
		return nil, errors.New("server not initialized")
	}

	conn, err := t.server.Dial(timeoutCtx, "tcp", addr)
	if err != nil {
		return nil, err
	}
	return conn, nil
}
