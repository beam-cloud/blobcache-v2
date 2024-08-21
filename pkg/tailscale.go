package blobcache

import (
	"context"
	"errors"
	"fmt"
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
	authCond *sync.Cond
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

	ts.authCond = sync.NewCond(&sync.Mutex{})
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

	go t.WaitForAuth(t.ctx)

	return t.server, nil
}

func (t *Tailscale) WaitForAuth(ctx context.Context) {
	if t.authDone {
		return
	}

	tailscaleClient, err := t.server.LocalClient()
	if err != nil {
		return
	}

	for {
		status, err := tailscaleClient.Status(ctx)
		if err != nil {
			return
		}

		if status.BackendState == ipn.Running.String() {
			t.authDone = true
			t.authCond.Broadcast() // Notify that ts auth was successful
			return
		}

		time.Sleep(500 * time.Millisecond)
	}
}

// DialWithTimeout returns a TCP connection to a tailscale service but times out after GRPCDialTimeoutS
func (t *Tailscale) DialWithTimeout(ctx context.Context, addr string) (net.Conn, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(t.cfg.GRPCDialTimeoutS)*time.Second)
	defer cancel()

	if t.server == nil {
		return nil, errors.New("server not initialized")
	}

	// Wait for authentication or timeout
	if !t.authDone {
		done := make(chan struct{})

		go func() {
			t.authCond.Wait()
			close(done)
		}()

		select {
		case <-timeoutCtx.Done():
			return nil, errors.New("timeout waiting for authentication")
		case <-done:
		}
	}

	conn, err := t.server.Dial(timeoutCtx, "tcp", addr)
	if err != nil {
		return nil, err
	}
	return conn, nil
}
