package server

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/KyberNetwork/reload"

	"github.com/KyberNetwork/router-service/internal/pkg/config"
	"github.com/KyberNetwork/router-service/internal/pkg/reloadconfig"
	"github.com/KyberNetwork/router-service/pkg/logger"
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
		isRunningProduction:  isProductionMode(cfg.Env),
	}
}

func (s *server) Run(ctx context.Context) error {
	logger.WithFields(ctx, logger.Fields{
		"grpc_addr":    s.cfg.GRPC.Host,
		"grpc_port":    s.cfg.GRPC.Port,
		"bind_address": s.cfg.Http.BindAddress,
		"http_prefix":  s.cfg.Http.Prefix,
		"http_mode":    s.cfg.Http.Mode,
		"env":          s.cfg.Env,
	}).Info("Starting server...")
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
		logger.Infof(ctx, "Starting reload manager")
		errCh <- s.reloadManager.Run(ctx)
	}()

	// Register notifier
	reloadChan := make(chan string)
	s.reloadManager.RegisterNotifier(reload.NotifierChan(reloadChan))

	go func() {
		logger.Infoln(ctx, "Starting reload config reporter")
		s.reloadConfigReporter.Report(ctx, reloadChan)
	}()
	for {
		select {
		case <-stop:
			time.Sleep(10 * time.Second)

			ctx, cancelFn := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancelFn()

			if err := s.httpServer.Shutdown(ctx); err != nil {
				logger.Errorf(ctx, "failed to stop server: %w", err)
			}

			return nil
		case err := <-errCh:
			return err
		}
	}
}

// TODO: need to improve app mode by enum.
func isProductionMode(env string) bool {
	return env == "production"
}
