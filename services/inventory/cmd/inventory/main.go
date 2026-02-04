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
	"strings"
	"syscall"
	"time"

	"github.com/ardanlabs/conf/v3"
	"github.com/kamogelosekhukhune777/real-time-supply-chain/pkg/logger"
	"github.com/kamogelosekhukhune777/real-time-supply-chain/pkg/otel"
	"github.com/kamogelosekhukhune777/real-time-supply-chain/pkg/sdks/sqldb"
	"github.com/kamogelosekhukhune777/real-time-supply-chain/services/inventory/internal/domain"
	"github.com/kamogelosekhukhune777/real-time-supply-chain/services/inventory/internal/messaging"
	"github.com/kamogelosekhukhune777/real-time-supply-chain/services/inventory/internal/messaging/store"
	"github.com/kamogelosekhukhune777/real-time-supply-chain/services/inventory/internal/service"
	"github.com/kamogelosekhukhune777/real-time-supply-chain/services/inventory/internal/stores/invcache"
	"github.com/kamogelosekhukhune777/real-time-supply-chain/services/inventory/internal/stores/invdb"
	"github.com/nats-io/nats.go"
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

	log = logger.NewWithEvents(os.Stdout, logger.LevelInfo, "INVENTORY", otel.GetTraceID, events)

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

	cfg := loadConfig()
	log.Info(ctx, "starting service", "version", cfg.Build)
	defer log.Info(ctx, "shutdown complete")

	expvar.NewString("build").Set(cfg.Build)

	// ---------------------------------------------------------------------------------------------------------------------------
	// Database

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
		return fmt.Errorf("db connect: %w", err)
	}
	defer db.Close()

	// ---------------------------------------------------------------------------------------------------------------------------
	// Redis

	rdb, err := newRedis(cfg)
	if err != nil {
		return fmt.Errorf("redis client:%w", err)
	}
	defer rdb.Close()

	// ---------------------------------------------------------------------------------------------------------------------------
	// Messaging (NATS + JetStream)

	nc, js, err := newNATS(cfg)
	if err != nil {
		return fmt.Errorf("nats connect :%w", err)
	}
	defer nc.Drain()

	// ---------------------------------------------------------------------------------------------------------------------------

	pub := messaging.NewPublisher(js)
	idemStore := store.NewRedisIdempotencyStore(rdb)

	// ---------------------------------------------------------------------------------------------------------------------------
	// Domain wiring

	invStore := invcache.NewStore(log, rdb, invdb.NewStore(log, db))
	invBus := domain.NewBusiness(log, invStore)

	// ---------------------------------------------------------------------------------------------------------------------------
	// Service

	svc := service.NewInventoryService(log, invBus, js, pub, idemStore)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		if err := svc.Start(ctx); err != nil {
			log.Error(ctx, "service stopped unexpectedly", "err", err)
			cancel()
		}
	}()

	log.Info(ctx, "service started")

	// ---------------------------------------------------------------------------------------------------------------------------
	// Graceful shutdown

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	select {
	case sig := <-sigCh:
		log.Info(ctx, "shutdown signal received", "signal", sig)
		cancel()
	case <-ctx.Done():
		log.Info(ctx, "context cancelled")
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer shutdownCancel()

	if err := svc.Stop(shutdownCtx); err != nil {
		log.Error(ctx, "service shutdown failed", "err", err)
	}

	if err := nc.Drain(); err != nil {
		log.Error(ctx, "nats drain failed", "err", err)
	}

	log.Info(ctx, "shutdown complete")
	return nil
}

func newRedis(cfg config) (*redis.Client, error) {
	opts := &redis.Options{
		Addr:        cfg.Redis.Hosts[0],
		Username:    cfg.Redis.Username,
		Password:    cfg.Redis.Password,
		DB:          cfg.Redis.DB,
		DialTimeout: cfg.Redis.DialTimeout,
	}

	if cfg.Redis.Scheme == "rediss" {
		opts.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	}

	rdb := redis.NewClient(opts)
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return nil, err
	}

	return rdb, nil
}

func newNATS(cfg config) (*nats.Conn, nats.JetStreamContext, error) {
	opts := []nats.Option{
		nats.Timeout(cfg.NATS.ConnTimeout),
		nats.ReconnectWait(cfg.NATS.ReconnectWait),
		nats.MaxReconnects(cfg.NATS.MaxReconnects),
	}

	if cfg.NATS.Token != "" {
		opts = append(opts, nats.Token(cfg.NATS.Token))
	} else if cfg.NATS.Username != "" {
		opts = append(opts, nats.UserInfo(cfg.NATS.Username, cfg.NATS.Password))
	}

	urls := make([]string, 0, len(cfg.NATS.Hosts))
	for _, h := range cfg.NATS.Hosts {
		urls = append(urls, fmt.Sprintf("%s://%s", cfg.NATS.Scheme, h))
	}

	nc, err := nats.Connect(strings.Join(urls, ","), opts...)
	if err != nil {
		return nil, nil, err
	}

	js, err := nc.JetStream()
	if err != nil {
		return nil, nil, err
	}

	return nc, js, nil
}

// ===================================================================================
// Config
// ===================================================================================

type config struct {
	conf.Version

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

	Redis struct {
		Scheme      string        `conf:"default:redis"`
		Hosts       []string      `conf:"default:inventory-redis:6379"`
		Username    string        `conf:"default:"`
		Password    string        `conf:"default:,mask"`
		DB          int           `conf:"default:0"`
		DialTimeout time.Duration `conf:"default:20s"`
	}

	ShutdownTimeout time.Duration `conf:"default:20s"`
}

func loadConfig() config {
	cfg := config{
		Version: conf.Version{
			Build: buildTag,
			Desc:  "Inventory Service",
		},
	}

	const prefix = "INVENTORY"
	help, err := conf.Parse(prefix, &cfg)
	if err != nil {
		if errors.Is(err, conf.ErrHelpWanted) {
			fmt.Println(help)
			os.Exit(0)
		}
		panic(err)
	}

	return cfg
}
