package tg

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/uploader"
	tdtg "github.com/gotd/td/tg"
)

// GotdConfig holds configuration for the gotd Telegram client.
type GotdConfig struct {
	AppID       int
	AppHash     string
	SessionPath string
	DeviceModel string
	AppVersion  string
	SystemInfo  string
}

// peerKey is a composite key for the peer cache to avoid collisions
// between different peer types sharing the same numeric ID.
type peerKey struct {
	Kind string
	ID   int64
}

// GotdClient implements Client using the gotd/td library.
type GotdClient struct {
	telegram *telegram.Client
	api      *tdtg.Client
	cfg      GotdConfig

	peerMu    sync.RWMutex
	peerCache map[peerKey]tdtg.InputPeerClass

	ready    chan struct{}
	stop     context.CancelFunc
	done     chan error
	closeErr error
	closed   sync.Once
}

// gotdDeviceConfig returns the DeviceConfig for gotd from our config.
func gotdDeviceConfig(cfg GotdConfig) telegram.DeviceConfig {
	device := cfg.DeviceModel
	if device == "" {
		device = "tgup"
	}
	return telegram.DeviceConfig{
		DeviceModel:   device,
		AppVersion:    cfg.AppVersion,
		SystemVersion: cfg.SystemInfo,
	}
}

// NewGotdClient creates a new gotd-based Telegram client.
func NewGotdClient(cfg GotdConfig) *GotdClient {
	sessionStore := &FileSessionStore{Path: cfg.SessionPath}
	client := telegram.NewClient(cfg.AppID, cfg.AppHash, telegram.Options{
		SessionStorage: sessionStore,
		Device:         gotdDeviceConfig(cfg),
		NoUpdates:      true,
	})

	return &GotdClient{
		telegram:  client,
		cfg:       cfg,
		peerCache: make(map[peerKey]tdtg.InputPeerClass),
	}
}

// Connect starts the MTProto connection and blocks until ready.
func (c *GotdClient) Connect(ctx context.Context) error {
	innerCtx, cancel := context.WithCancel(ctx)
	c.stop = cancel
	c.done = make(chan error, 1)
	c.ready = make(chan struct{})

	go func() {
		c.done <- c.telegram.Run(innerCtx, func(runCtx context.Context) error {
			c.api = c.telegram.API()
			close(c.ready)
			<-runCtx.Done()
			return runCtx.Err()
		})
	}()

	select {
	case <-c.ready:
		return nil
	case err := <-c.done:
		return fmt.Errorf("connect: %w", err)
	case <-ctx.Done():
		cancel()
		return ctx.Err()
	}
}

// Close shuts down the MTProto connection. Safe to call multiple times.
func (c *GotdClient) Close(ctx context.Context) error {
	c.closed.Do(func() {
		if c.stop != nil {
			c.stop()
		}
		if c.done != nil {
			select {
			case err := <-c.done:
				if !errors.Is(err, context.Canceled) {
					c.closeErr = err
				}
			case <-ctx.Done():
				c.closeErr = ctx.Err()
			}
		}
	})
	return c.closeErr
}

// IsAuthenticated checks if the current session is authenticated.
func (c *GotdClient) IsAuthenticated(ctx context.Context) bool {
	_, err := c.telegram.Self(ctx)
	return err == nil
}

// getAPI returns the raw API client, or an error if not connected.
func (c *GotdClient) getAPI() (*tdtg.Client, error) {
	if c.api == nil {
		return nil, fmt.Errorf("client not connected: call Connect first")
	}
	return c.api, nil
}

// newUploader creates a fresh uploader instance (not shared, safe for WithProgress).
func (c *GotdClient) newUploader() (*uploader.Uploader, error) {
	api, err := c.getAPI()
	if err != nil {
		return nil, err
	}
	return uploader.NewUploader(api), nil
}

func (c *GotdClient) cachePeer(kind string, id int64, peer tdtg.InputPeerClass) {
	c.peerMu.Lock()
	c.peerCache[peerKey{Kind: kind, ID: id}] = peer
	c.peerMu.Unlock()
}

func (c *GotdClient) getPeer(rt ResolvedTarget) (tdtg.InputPeerClass, error) {
	c.peerMu.RLock()
	peer, ok := c.peerCache[peerKey{Kind: rt.Kind, ID: rt.ID}]
	c.peerMu.RUnlock()
	if ok {
		return peer, nil
	}
	return nil, fmt.Errorf("peer not resolved: %s", rt.Raw)
}
