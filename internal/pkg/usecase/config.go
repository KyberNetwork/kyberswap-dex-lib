package usecase

import (
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/buildroute"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/getcustomroute"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/getroute"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/indexpools"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/poolfactory"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/poolmanager"
)

type Config struct {
	GetRoute       getroute.Config       `mapstructure:"getRoute" json:"getRoute"`
	GetCustomRoute getcustomroute.Config `mapstructure:"getCustomRoute" json:"getCustomRoute"`
	BuildRoute     buildroute.Config     `mapstructure:"buildRoute"`

	IndexPools                 indexpools.IndexPoolsConfig           `mapstructure:"indexPools" json:"indexPools"`
	TradeDataGenerator         indexpools.TradeDataGeneratorConfig   `mapstructure:"tradeDataGenerator"`
	UpdateLiquidityScoreConfig indexpools.UpdateLiquidityScoreConfig `mapstructure:"updateLiquidityScore"`

	PoolFactory poolfactory.Config `mapstructure:"poolFactory" json:"poolFactory"`
	PoolManager poolmanager.Config `mapstructure:"poolManager" json:"poolManager"`

	TrackExecutor TrackExecutorConfig `mapstructure:"trackExecutor"`
}

type (
	TrackExecutorConfig struct {
		SubgraphURL string `mapstructure:"subgraphURL"`
		StartBlock  uint64 `mapstructure:"startBlock"`
	}
)
