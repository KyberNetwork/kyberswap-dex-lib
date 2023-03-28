package reloadconfig

import (
	"context"

	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

type IReloadConfigUseCase interface {
	ShouldReload(ctx context.Context, serviceCode string) (bool, error)
	Fetch(ctx context.Context, serviceCode string) (valueobject.RemoteConfig, error)
}
