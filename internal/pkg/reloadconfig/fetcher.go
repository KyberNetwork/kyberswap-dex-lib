package reloadconfig

import (
	"context"

	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

type Fetcher struct {
	cfg                 ReloadConfig
	reloadConfigUseCase IReloadConfigUseCase
}

func NewReloadConfigFetcher(
	cfg ReloadConfig,
	reloadConfigUseCase IReloadConfigUseCase,
) *Fetcher {
	return &Fetcher{
		cfg:                 cfg,
		reloadConfigUseCase: reloadConfigUseCase,
	}
}

func (f *Fetcher) Fetch(ctx context.Context) (valueobject.RemoteConfig, error) {
	return f.reloadConfigUseCase.Fetch(ctx, getServiceCode(f.cfg.ServiceName, f.cfg.ChainID))
}
