package setting

import (
	"context"

	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

type ISettingRepository interface {
	GetConfigs(_ context.Context, serviceCode string, currentHash string) (valueobject.RemoteConfig, error)
}
