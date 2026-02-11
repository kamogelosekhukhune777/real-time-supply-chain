package messaging

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/kamogelosekhukhune777/real-time-supply-chain/pkg/logger"
	"github.com/nats-io/nats.go"
)

type Config struct {
	Log           *logger.Logger
	ClientName    string
	Scheme        string
	Hosts         []string
	Username      string
	Password      string
	Token         string
	ConnTimeout   time.Duration
	ReconnectWait time.Duration
	MaxReconnects int
}

func NewNATS(ctx context.Context, cfg Config) (*nats.Conn, nats.JetStreamContext, error) {
	opts := []nats.Option{
		nats.Name(cfg.ClientName),
		nats.Timeout(cfg.ConnTimeout),
		nats.MaxReconnects(cfg.MaxReconnects),
		nats.ReconnectWait(cfg.ReconnectWait),

		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			if err != nil {
				cfg.Log.Warn(ctx, "nats disconnected", "error", err)
			}
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			cfg.Log.Info(ctx, "nats reconnected", "url", nc.ConnectedUrl())
		}),
		nats.ClosedHandler(func(nc *nats.Conn) {
			if err := nc.LastError(); err != nil {
				cfg.Log.Error(ctx, "nats closed with error", "error", err)
			} else {
				cfg.Log.Info(ctx, "nats closed cleanly")
			}
		}),
	}

	if cfg.Token != "" {
		opts = append(opts, nats.Token(cfg.Token))
	} else if cfg.Username != "" {
		opts = append(opts, nats.UserInfo(cfg.Username, cfg.Password))
	}

	urls := make([]string, 0, len(cfg.Hosts))
	for _, h := range cfg.Hosts {
		urls = append(urls, fmt.Sprintf("%s://%s", cfg.Scheme, h))
	}

	nc, err := nats.Connect(strings.Join(urls, ","), opts...)
	if err != nil {
		return nil, nil, err
	}

	js, err := nc.JetStream(
		nats.PublishAsyncMaxPending(256),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("jetstream: %w", err)
	}

	return nc, js, nil
}

func EnsureStream(js nats.JetStreamContext, env string) error {
	streamName := "EVENTS"

	expected := &nats.StreamConfig{
		Name:     streamName,
		Subjects: []string{env + ".>"},
		Storage:  nats.FileStorage,
		MaxAge:   72 * time.Hour,
	}

	info, err := js.StreamInfo(streamName)
	if err == nil {
		// Validate critical invariants
		if info.Config.Storage != expected.Storage {
			return fmt.Errorf("stream %s has wrong storage type", streamName)
		}

		// Update if mutable fields changed
		if info.Config.MaxAge != expected.MaxAge ||
			!equalSubjects(info.Config.Subjects, expected.Subjects) {

			_, err := js.UpdateStream(expected)
			return err
		}

		return nil
	}

	_, err = js.AddStream(expected)
	return err
}

func equalSubjects(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	// Sort or use a map to ensure order doesn't trigger false updates
	m := make(map[string]struct{})
	for _, s := range a {
		m[s] = struct{}{}
	}
	for _, s := range b {
		if _, ok := m[s]; !ok {
			return false
		}
	}
	return true
}
