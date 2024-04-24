package usecase

import (
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/buildroute"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/generatepath"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/getroute"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/poolfactory"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/poolmanager"
)

type Config struct {
	GetRoute   getroute.Config   `mapstructure:"getRoute" json:"getRoute"`
	BuildRoute buildroute.Config `mapstructure:"buildRoute"`

	IndexPools IndexPoolsConfig `mapstructure:"indexPools" json:"indexPools"`

	PoolFactory       poolfactory.Config  `mapstructure:"poolFactory" json:"poolFactory"`
	PoolManager       poolmanager.Config  `mapstructure:"poolManager" json:"poolManager"`
	GenerateBestPaths generatepath.Config `mapstructure:"generateBestPaths"`

	TrackExecutor TrackExecutorConfig `mapstructure:"trackExecutor"`
}

type (
	TokenAndAmounts struct {
		TokenAddress string   `mapstructure:"tokenAddress"`
		Amounts      []string `mapstructure:"amounts"`
	}

	GenerateBestPathsOptions struct {
		MaintainingIntervalSec int `mapstructure:"maintainingIntervalSec"`
	}
	SPFAFinderOptions struct {
		MaxHops                 uint32  `mapstructure:"maxHops"`
		DistributionPercent     uint32  `mapstructure:"distributionPercent"`
		MaxPathsInRoute         uint32  `mapstructure:"maxPathsInRoute"`
		MaxPathsToGenerate      uint32  `mapstructure:"maxPathsToGenerate"`
		MaxPathsToReturn        uint32  `mapstructure:"maxPathsToReturn"`
		MinPartUSD              float64 `mapstructure:"minPartUSD"`
		MinThresholdAmountInUSD float64 `mapstructure:"minThresholdAmountInUSD"`
		MaxThresholdAmountInUSD float64 `mapstructure:"maxThresholdAmountInUSD"`
	}
)

type (
	IndexPoolsConfig struct {
		WhitelistedTokenSet map[string]bool `mapstructure:"whitelistedTokenSet"`
		ChunkSize           int             `mapstructure:"chunkSize"`
		MaxGoroutines       int             `mapstructure:"maxGoroutines"`
		EnableRankByNative  bool            `mapstructure:"enableRankByNative"`
	}

	TrackExecutorConfig struct {
		SubgraphURL string `mapstructure:"subgraphURL"`
	}
)
