package main

import (
	"context"
	"crypto/tls"
	"errors"
	"expvar"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/ardanlabs/conf/v3"
	"github.com/kamogelosekhukhune777/real-time-supply-chain/pkg/logger"
	"github.com/kamogelosekhukhune777/real-time-supply-chain/pkg/otel"
	"github.com/kamogelosekhukhune777/real-time-supply-chain/pkg/sdks/sqldb"
	"github.com/kamogelosekhukhune777/real-time-supply-chain/services/shipment/internal/domain/shipment"
	"github.com/kamogelosekhukhune777/real-time-supply-chain/services/shipment/internal/domain/shipment/stores/shipmentdb"
	shipmentcache "github.com/kamogelosekhukhune777/real-time-supply-chain/services/shipment/internal/domain/shipment/stores/shpimentcache"
	"github.com/kamogelosekhukhune777/real-time-supply-chain/services/shipment/internal/messaging"
	"github.com/kamogelosekhukhune777/real-time-supply-chain/services/shipment/internal/messaging/idempotencystore"
	"github.com/redis/go-redis/v9"
)

var buildTag = "develop"

func main() {
	var log *logger.Logger

	events := logger.Events{
		Error: func(ctx context.Context, r logger.Record) {
			log.Info(ctx, "******* SEND ALERT *******")
		},
	}

	log = logger.NewWithEvents(os.Stdout, logger.LevelInfo, "SHIPMENT", otel.GetTraceID, events)

	ctx := context.Background()
	if err := run(ctx, log); err != nil {
		log.Error(ctx, "startup failed", "err", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, log *logger.Logger) error {

	// ---------------------------------------------------------------------------------------------------------------------------
	log.Info(ctx, "startup", "GOMAXPROCS", runtime.GOMAXPROCS(0))

	// ---------------------------------------------------------------------------------------------------------------------------
	// Configuration

	cfg := struct {
		conf.Version
		Redis
		NATS struct {
			Scheme        string        `conf:"default:nats"`
			Hosts         []string      `conf:"default:localhost:4222"`
			Username      string        `conf:"default:"`
			Password      string        `conf:"default:,mask"`
			Token         string        `conf:"default:,mask"`
			ConnTimeout   time.Duration `conf:"default:20s"`
			ReconnectWait time.Duration `conf:"default:20s"`
			MaxReconnects int           `conf:"default:9"`
		}
		DB struct {
			User         string `conf:"default:inventory"`
			Password     string `conf:"default:inventory,mask"`
			Host         string `conf:"default:inventory-postgres"`
			Name         string `conf:"default:inventory"`
			MaxIdleConns int    `conf:"default:0"`
			MaxOpenConns int    `conf:"default:0"`
			DisableTLS   bool   `conf:"default:true"`
		}
		Tempo struct {
			Host            string        `conf:"default:tempo:4317"`
			ServiceName     string        `conf:"default:sales"`
			Probability     float64       `conf:"default:0.05"`
			ShutdownTimeout time.Duration `conf:"default:20s"`
		}
	}{
		Version: conf.Version{
			Build: buildTag,
			Desc:  "Shipment ...",
		},
	}

	const prefix = "SHIPMENT"
	help, err := conf.Parse(prefix, &cfg)
	if err != nil {
		if errors.Is(err, conf.ErrHelpWanted) {
			fmt.Println(help)
			return nil
		}
		return fmt.Errorf("parsing config: %w", err)
	}

	// ---------------------------------------------------------------------------------------------------------------------------
	// App Starting

	log.Info(ctx, "starting service", "version", cfg.Build)
	defer log.Info(ctx, "shutdown complete")

	out, err := conf.String(&cfg)
	if err != nil {
		return fmt.Errorf("generating config for output: %w", err)
	}
	log.Info(ctx, "startup", "config", out)

	log.BuildInfo(ctx)

	expvar.NewString("build").Set(cfg.Build)

	// ---------------------------------------------------------------------------------------------------------------------------
	// Database Support

	log.Info(ctx, "startup", "status", "initializing database support", "hostport", cfg.DB.Host)

	db, err := sqldb.Open(sqldb.Config{
		User:         cfg.DB.User,
		Password:     cfg.DB.Password,
		Host:         cfg.DB.Host,
		Name:         cfg.DB.Name,
		MaxIdleConns: cfg.DB.MaxIdleConns,
		MaxOpenConns: cfg.DB.MaxOpenConns,
		DisableTLS:   cfg.DB.DisableTLS,
	})
	if err != nil {
		return fmt.Errorf("connecting to db: %w", err)
	}

	defer db.Close()

	// ---------------------------------------------------------------------------------------------------------------------------
	// Redis

	rdb, err := newRedis(cfg.Redis)
	if err != nil {
		return fmt.Errorf("redis client:%w", err)
	}
	defer rdb.Close()

	// -------------------------------------------------------------------------
	// Start Tracing Support

	log.Info(ctx, "startup", "status", "initializing tracing support")

	traceProvider, teardown, err := otel.InitTracing(otel.Config{
		ServiceName: cfg.Tempo.ServiceName,
		Host:        cfg.Tempo.Host,
		Probability: cfg.Tempo.Probability,
	})
	if err != nil {
		return fmt.Errorf("starting tracing: %w", err)
	}

	defer teardown(context.Background())

	tracer := traceProvider.Tracer(cfg.Tempo.ServiceName)

	// ---------------------------------------------------------------------------------------------------------------------------
	// Messaging (NATS + JetStream)

	nCfg := messaging.Config{
		Log:           log,
		ClientName:    "shipment-service",
		Scheme:        cfg.NATS.Scheme,
		Hosts:         cfg.NATS.Hosts,
		Username:      cfg.NATS.Username,
		Password:      cfg.NATS.Password,
		Token:         cfg.NATS.Token,
		ConnTimeout:   cfg.NATS.ConnTimeout,
		ReconnectWait: cfg.NATS.ReconnectWait,
		MaxReconnects: cfg.NATS.MaxReconnects,
	}

	nc, js, err := messaging.NewNATS(ctx, nCfg)
	if err != nil {
		return fmt.Errorf("nats connect :%w", err)
	}
	defer nc.Drain()

	env := "prod"
	env = buildTag
	if err := messaging.EnsureStream(js, env); err != nil {
		return fmt.Errorf("Could not ensure JetStream: %w", err)
	}

	// ---------------------------------------------------------------------------------------------------------------------------
	// Messaging

	_ = messaging.NewPublisher(log, js, tracer)
	_ = idempotencystore.NewRedisIdempotencyStore(rdb)

	// ---------------------------------------------------------------------------------------------------------------------------
	// Domain/Busines

	shipmentStorage := shipmentcache.NewStore(log, rdb, shipmentdb.NewStore(log, db))
	_ = shipment.NewBusiness(log, shipmentStorage)

	// ---------------------------------------------------------------------------------------------------------------------------

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	<-shutdown

	return nil
}

type Redis struct {
	Scheme      string        `conf:"default:redis"`
	Hosts       []string      `conf:"default:inventory-redis:6379"`
	Username    string        `conf:"default:"`
	Password    string        `conf:"default:,mask"`
	DB          int           `conf:"default:0"`
	DialTimeout time.Duration `conf:"default:20s"`
}

func newRedis(cfg Redis) (*redis.Client, error) {
	opts := &redis.Options{
		Addr:        cfg.Hosts[0],
		Username:    cfg.Username,
		Password:    cfg.Password,
		DB:          cfg.DB,
		DialTimeout: cfg.DialTimeout,
	}

	if cfg.Scheme == "redis" {
		opts.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	}

	rdb := redis.NewClient(opts)
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return nil, err
	}

	return rdb, nil
}
