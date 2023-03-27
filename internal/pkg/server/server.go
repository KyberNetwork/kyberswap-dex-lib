package server

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/KyberNetwork/reload"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/config"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/reloadconfig"
	"github.com/KyberNetwork/kyberswap-aggregator/pkg/logger"
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
	logger.WithFields(logger.Fields{
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
		logger.Infof("Starting reload manager")
		errCh <- s.reloadManager.Run(ctx)
	}()

	// Register notifier
	reloadChan := make(chan string)
	s.reloadManager.RegisterNotifier(reload.NotifierChan(reloadChan))

	go func() {
		logger.Infoln("Starting reload config reporter")
		s.reloadConfigReporter.Report(ctx, reloadChan)
	}()
	for {
		select {
		case <-stop:
			ctx, cancelFn := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancelFn()

			if err := s.httpServer.Shutdown(ctx); err != nil {
				logger.Errorf("failed to stop server: %w", err)
			}

			if s.isRunningProduction {
				logger.Infoln("Shutting down. Wait for 15 seconds")
				time.Sleep(15 * time.Second)
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
