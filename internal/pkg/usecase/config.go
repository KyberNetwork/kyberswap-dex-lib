package usecase

import (
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/getroute"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/poolfactory"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/poolmanager"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

type Config struct {
	GetRoute   getroute.Config  `mapstructure:"getRoute" json:"getRoute"`
	BuildRoute BuildRouteConfig `mapstructure:"buildRoute" json:"buildRoute"`

	IndexPools IndexPoolsConfig `mapstructure:"indexPools" json:"indexPools"`

	PoolFactory poolfactory.Config `mapstructure:"poolFactory" json:"poolFactory"`
	PoolManager poolmanager.Config `mapstructure:"poolManager" json:"poolManager"`
}

type (
	BuildRouteConfig struct {
		ChainID valueobject.ChainID `mapstructure:"chainId"`
	}
)

type (
	IndexPoolsConfig struct {
		WhitelistedTokenSet map[string]bool `mapstructure:"whitelistedTokenSet"`
		ChunkSize           int             `mapstructure:"chunkSize"`
	}
)
