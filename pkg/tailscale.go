package blobcache

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"tailscale.com/ipn"
	"tailscale.com/tsnet"
)

type Tailscale struct {
	ctx      context.Context
	cfg      BlobCacheConfig
	server   *tsnet.Server
	mu       sync.Mutex
	hostname string
	authDone bool
}

// NewTailscale creates a new Tailscale instance using tsnet
func NewTailscale(ctx context.Context, hostname string, cfg BlobCacheConfig) *Tailscale {
	ts := &Tailscale{
		ctx:      ctx,
		hostname: hostname,
		cfg:      cfg,
		mu:       sync.Mutex{},
		authDone: false,
	}

	return ts
}

func (t *Tailscale) logF(format string, v ...interface{}) {
	if t.cfg.Tailscale.Debug {
		Logger.Infof(format, v...)
	}
}

func (t *Tailscale) GetOrCreateServer() (*tsnet.Server, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.server != nil {
		return t.server, nil
	}

	t.server = &tsnet.Server{
		Hostname:   t.hostname,
		Dir:        fmt.Sprintf("%s/%s", t.cfg.Tailscale.StateDir, t.hostname),
		AuthKey:    t.cfg.Tailscale.AuthKey,
		ControlURL: t.cfg.Tailscale.ControlURL,
		Ephemeral:  t.cfg.Tailscale.Ephemeral,
	}

	t.server.Logf = t.logF
	t.server.UserLogf = t.logF

	if err := t.server.Start(); err != nil {
		return nil, err
	}

	return t.server, nil
}

func (t *Tailscale) WaitForAuth(ctx context.Context, timeout time.Duration) error {
	if t.authDone {
		return nil
	}

	tailscaleClient, err := t.server.LocalClient()
	if err != nil {
		return err
	}

	log.Println("blobcache: waiting for tailscale auth")

	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for {
		select {
		case <-timeoutCtx.Done():
			return fmt.Errorf("blobcache: tailscale auth timed out after %v", timeout)
		default:
			status, err := tailscaleClient.Status(timeoutCtx)
			if err != nil {
				continue
			}

			if status.BackendState == ipn.Running.String() {
				t.authDone = true
				log.Println("blobcache: tailscale auth completed")
				return nil
			}

			time.Sleep(500 * time.Millisecond)
		}
	}
}

// DialWithTimeout returns a TCP connection to a tailscale service but times out after GRPCDialTimeoutS
func (t *Tailscale) DialWithTimeout(ctx context.Context, addr string) (net.Conn, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(t.cfg.GRPCDialTimeoutS)*time.Second)
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
