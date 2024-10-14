package usecase

import (
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/buildroute"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/getcustomroute"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/getroute"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/poolfactory"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/poolmanager"
)

type Config struct {
	GetRoute       getroute.Config       `mapstructure:"getRoute" json:"getRoute"`
	GetCustomRoute getcustomroute.Config `mapstructure:"getCustomRoute" json:"getCustomRoute"`
	BuildRoute     buildroute.Config     `mapstructure:"buildRoute"`

	IndexPools IndexPoolsConfig `mapstructure:"indexPools" json:"indexPools"`

	PoolFactory poolfactory.Config `mapstructure:"poolFactory" json:"poolFactory"`
	PoolManager poolmanager.Config `mapstructure:"poolManager" json:"poolManager"`

	TrackExecutor TrackExecutorConfig `mapstructure:"trackExecutor"`
}

type (
	IndexPoolsConfig struct {
		WhitelistedTokenSet map[string]bool `mapstructure:"whitelistedTokenSet"`
		ChunkSize           int             `mapstructure:"chunkSize"`
		MaxGoroutines       int             `mapstructure:"maxGoroutines"`
		EnableRankByNative  bool            `mapstructure:"enableRankByNative"`

		// If the pool has 0 TVL, and the direct index length is less than this value,
		// we will still add the pool to the indexes.
		MaxDirectIndexLenForZeroTvl int `mapstructure:"maxDirectIndexLenForZeroTvl"`
	}

	TrackExecutorConfig struct {
		SubgraphURL string `mapstructure:"subgraphURL"`
	}
)
