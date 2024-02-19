package reloadconfig

import (
	"context"
	"time"

	"github.com/KyberNetwork/router-service/pkg/logger"
)

const (
	EmptyString  = ""
	SignalReload = "reload"
)

type Reporter struct {
	cfg                 ReloadConfig
	reloadConfigUseCase IReloadConfigUseCase
}

func NewReloadConfigReporter(
	cfg ReloadConfig,
	reloadConfigUseCase IReloadConfigUseCase,
) *Reporter {
	return &Reporter{
		cfg:                 cfg,
		reloadConfigUseCase: reloadConfigUseCase,
	}
}

func (r *Reporter) Report(ctx context.Context, reloadChan chan string) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(r.cfg.Interval):
			// should not reload if HTTP URL is not set
			if r.cfg.HttpUrl == EmptyString {
				continue
			}

			shouldReload, err := r.reloadConfigUseCase.ShouldReload(ctx, getServiceCode(r.cfg.ServiceName, r.cfg.ChainID))
			if err != nil {
				logger.Errorf(ctx, "failed to check should reload, err: %v", err)
				continue
			}

			if shouldReload {
				reloadChan <- SignalReload
			}
		}
	}
}
