package messaging

import (
	"context"
	"fmt"
	"time"

	"github.com/kamogelosekhukhune777/real-time-supply-chain/pkg/logger"
	"github.com/nats-io/nats.go"
)

type Config struct {
	Log           *logger.Logger
	URL           string
	ConnTimeout   time.Duration
	MaxReconnects int
	ReconnectWait time.Duration
}

func NewNATS(ctx context.Context, cfg Config) (*nats.Conn, nats.JetStreamContext, error) {
	opts := []nats.Option{
		nats.Name("inventory-service"),
		nats.Timeout(cfg.ConnTimeout),
		nats.MaxReconnects(cfg.MaxReconnects),
		nats.ReconnectWait(cfg.ReconnectWait),

		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			if err != nil {
				cfg.Log.Warn(ctx, "nats disconnect", "error", err)
			}
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			// log info
			cfg.Log.Info(ctx, "NATS reconnected", nc.ConnectedUrl())
		}),
		nats.ClosedHandler(func(nc *nats.Conn) {
			// log fatal / shutdown
			cfg.Log.Info(ctx, "nats closed", "error", nc.LastError())
		}),
	}

	nc, err := nats.Connect(cfg.URL, opts...)
	if err != nil {
		return nil, nil, fmt.Errorf("nats connect: %w", err)
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
	// Fetch current state first to avoid accidental overrides
	info, err := js.StreamInfo(streamName)

	cfg := &nats.StreamConfig{
		Name:     streamName,
		Subjects: []string{env + ".>"},
		Storage:  nats.FileStorage,
		MaxAge:   72 * time.Hour,
	}

	if err == nil {
		// Only update if something actually changed to save IO
		if info.Config.MaxAge != cfg.MaxAge {
			_, err = js.UpdateStream(cfg)
			return err
		}
		return nil
	}

	_, err = js.AddStream(cfg)
	return err
}
