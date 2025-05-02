package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/horockey/service_discovery/internal/config"
	"github.com/horockey/service_discovery/internal/controller/http_controller"
	"github.com/horockey/service_discovery/internal/extractor/health_upds/http_check_health_upds"
	"github.com/horockey/service_discovery/internal/gateway/nodes_updates/http_broadcast_nodes_updates"
	"github.com/horockey/service_discovery/internal/repository/nodes/badger_nodes"
	"github.com/horockey/service_discovery/internal/usecase/discovery"
	"github.com/rs/zerolog"
)

func main() {
	logger := zerolog.New(zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.RFC3339,
	}).With().Timestamp().Logger()

	cfg, err := config.New(logger)
	if err != nil {
		logger.
			Fatal().
			Err(fmt.Errorf("creating config: %w", err)).
			Send()
	}

	if err := os.MkdirAll(cfg.BadgerDir, os.ModePerm); err != nil {
		logger.
			Fatal().
			Err(fmt.Errorf("making dir %s: %w", cfg.BadgerDir, err)).
			Send()
	}

	db, err := badger.Open(badger.DefaultOptions(cfg.BadgerDir))
	if err != nil {
		logger.
			Fatal().
			Err(fmt.Errorf("creating badger instance: %w", err)).
			Send()
	}
	defer db.Close()

	nodesRepo := badger_nodes.New(
		db,
		time.Duration(cfg.DownNodesRmIvlMSec)*time.Millisecond,
	)

	updsGw, err := http_broadcast_nodes_updates.New(
		runtime.NumCPU(),
		cfg.APIKey,
		logger.With().Str("scope", "http_updates_gateway").Logger(),
	)
	if err != nil {
		logger.
			Fatal().
			Err(fmt.Errorf("creating updates gateway: %w", err)).
			Send()
	}

	updsExtr, err := http_check_health_upds.New(
		nodesRepo,
		100,
		time.Duration(cfg.HealthcheckIvlMsec)*time.Millisecond,
		cfg.APIKey,
		logger.With().Str("scope", "healthcheck_extractor").Logger(),
	)
	if err != nil {
		logger.
			Fatal().
			Err(fmt.Errorf("creating healthcheck extractor: %w", err)).
			Send()
	}

	uc := discovery.New(
		nodesRepo,
		updsExtr,
		updsGw,
		logger.With().Str("scope", "usecase").Logger(),
	)

	ctrl := http_controller.New(
		cfg.BaseURL,
		uc,
		cfg.APIKey,
		logger.With().Str("scope", "http_controller").Logger(),
	)

	ctx, cancel := signal.NotifyContext(
		context.Background(),
		syscall.SIGTERM,
		syscall.SIGABRT,
		syscall.SIGINT,
		syscall.SIGQUIT,
		syscall.SIGKILL,
	)
	defer cancel()

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := updsGw.Start(ctx); err != nil && !errors.Is(err, context.Canceled) {
			logger.
				Error().
				Err(fmt.Errorf("running updates gateway: %w", err)).
				Send()
			cancel()
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := updsExtr.Start(ctx); err != nil && !errors.Is(err, context.Canceled) {
			logger.
				Error().
				Err(fmt.Errorf("running updates extractor: %w", err)).
				Send()
			cancel()
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := uc.Start(ctx); err != nil && !errors.Is(err, context.Canceled) {
			logger.
				Error().
				Err(fmt.Errorf("running usecase: %w", err)).
				Send()
			cancel()
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := ctrl.Start(ctx); err != nil && !errors.Is(err, context.Canceled) {
			logger.
				Error().
				Err(fmt.Errorf("running http controller: %w", err)).
				Send()
			cancel()
		}
	}()

	logger.Info().Msg("Service started")
	wg.Wait()
	logger.Info().Msg("Service stopped")
}
