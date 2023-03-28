package usecase

import (
	"context"

	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

const (
	EmptyConfigHash = ""
)

type ReloadConfigUseCase struct {
	configFetcherRepo IConfigFetcherRepository
	currentConfigHash string
}

func NewReloadConfigUseCase(
	configFetcherRepo IConfigFetcherRepository,
) *ReloadConfigUseCase {
	return &ReloadConfigUseCase{
		configFetcherRepo: configFetcherRepo,
	}
}

func (u *ReloadConfigUseCase) ShouldReload(ctx context.Context, serviceCode string) (bool, error) {
	if u.currentConfigHash == EmptyConfigHash {
		return true, nil
	}

	remoteCfg, err := u.configFetcherRepo.GetConfigs(ctx, serviceCode, u.currentConfigHash)

	if err != nil {
		return false, err
	}

	if remoteCfg.Hash != u.currentConfigHash {
		return true, nil
	}

	return false, nil
}

func (u *ReloadConfigUseCase) Fetch(ctx context.Context, serviceCode string) (valueobject.RemoteConfig, error) {
	remoteCfg, err := u.configFetcherRepo.GetConfigs(ctx, serviceCode, u.currentConfigHash)
	if err != nil {
		return valueobject.RemoteConfig{}, err
	}

	// Keep the current hash to use next time
	u.currentConfigHash = remoteCfg.Hash

	return remoteCfg, nil
}
