package blobcache

import (
	"fmt"
	"log"
	"sync"

	"tailscale.com/tsnet"
)

type Tailscale struct {
	cfg    TailscaleConfig
	server *tsnet.Server
	mu     sync.Mutex
}

// NewTailscale creates a new Tailscale instance using tsnet
func NewTailscale(cfg TailscaleConfig) *Tailscale {
	return &Tailscale{
		cfg: cfg,
		mu:  sync.Mutex{},
	}
}

func (t *Tailscale) logF(format string, v ...interface{}) {
	if t.cfg.Debug {
		log.Printf(format, v...)
	}
}

func (t *Tailscale) GetOrCreateServer(hostname string) *tsnet.Server {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.server != nil {
		return t.server
	}

	server := &tsnet.Server{
		Hostname:   hostname,
		Dir:        fmt.Sprintf("%s/%s", t.cfg.StateDir, hostname),
		AuthKey:    t.cfg.AuthKey,
		ControlURL: t.cfg.ControlURL,
		Ephemeral:  t.cfg.Ephemeral,
	}

	server.Logf = t.logF
	return server
}

// // Dial returns a TCP connection to a tailscale service
// func (t *Tailscale) Dial(ctx context.Context, network, addr string) (net.Conn, error) {
// 	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
// 	defer cancel()

// 	// Connect to tailnet, if we aren't already
// 	if !t.initialized {
// 		t.mu.Lock()

// 		_, err := t.server.Up(timeoutCtx)
// 		if err != nil {
// 			t.mu.Unlock()
// 			return nil, err
// 		}

// 		t.initialized = true
// 		t.mu.Unlock()
// 	}

// 	conn, err := t.server.Dial(timeoutCtx, network, addr)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return conn, nil
// }
