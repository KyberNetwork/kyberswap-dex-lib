package server

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/KyberNetwork/reload"
	"github.com/rs/zerolog/log"

	"github.com/KyberNetwork/router-service/internal/pkg/config"
	"github.com/KyberNetwork/router-service/internal/pkg/reloadconfig"
	"github.com/KyberNetwork/router-service/pkg/util/env"
)

type server struct {
	httpServer           *http.Server
	cfg                  *config.Config
	reloadConfigReporter *reloadconfig.Reporter
	reloadManager        *reload.Manager
	isRunningProduction  bool
}

func NewServer(httpServer *http.Server,
	cfg *config.Config,
	reloadConfigReporter *reloadconfig.Reporter,
	reloadManager *reload.Manager) *server {
	return &server{
		httpServer:           httpServer,
		cfg:                  cfg,
		reloadConfigReporter: reloadConfigReporter,
		reloadManager:        reloadManager,
		isRunningProduction:  env.IsProductionMode(),
	}
}

func (s *server) Run(ctx context.Context) error {
	log.Ctx(ctx).Info().
		Str("grpc_addr", s.cfg.GRPC.Host).
		Int("grpc_port", s.cfg.GRPC.Port).
		Str("bind_address", s.cfg.Http.BindAddress).
		Str("http_prefix", s.cfg.Http.Prefix).
		Str("http_mode", s.cfg.Http.Mode).
		Str("env", s.cfg.Env).
		Msg("Starting server...")
	return s.run(ctx)
}

func (s *server) run(ctx context.Context) error {
	stop := make(chan os.Signal, 1)
	errCh := make(chan error)
	signal.Notify(stop, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil {
			errCh <- err
		}
	}()

	go func() {
		errCh <- s.reloadManager.Run(ctx)
	}()

	// Register notifier
	reloadChan := make(chan string)
	s.reloadManager.RegisterNotifier(reload.NotifierChan(reloadChan))

	go func() {
		s.reloadConfigReporter.Report(ctx, reloadChan)
	}()
	for {
		select {
		case <-stop:
			if s.isRunningProduction {
				time.Sleep(10 * time.Second)
			}

			ctx, cancelFn := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancelFn()

			if err := s.httpServer.Shutdown(ctx); err != nil {
				log.Ctx(ctx).Err(err).Msg("failed to stop server")
			}

			return nil
		case err := <-errCh:
			return err
		}
	}
}
