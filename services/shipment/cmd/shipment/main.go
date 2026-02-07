package main

import (
	"context"
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
		DB struct {
			User         string `conf:"default:inventory"`
			Password     string `conf:"default:inventory,mask"`
			Host         string `conf:"default:inventory-postgres"`
			Name         string `conf:"default:inventory"`
			MaxIdleConns int    `conf:"default:0"`
			MaxOpenConns int    `conf:"default:0"`
			DisableTLS   bool   `conf:"default:true"`
		}
		ShutdownTimeout time.Duration `conf:"default:20s"`
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
	// Domain/Busines

	shipmentStorage := shipmentdb.NewStore(log, db)
	_ = shipment.NewBusiness(log, shipmentStorage)

	// ---------------------------------------------------------------------------------------------------------------------------

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	<-shutdown

	return nil
}
