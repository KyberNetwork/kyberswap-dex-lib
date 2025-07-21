package trackexecutor

import (
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

type Config struct {
	ChainID                 valueobject.ChainID `mapstructure:"chainId"`
	AggregatorSubgraphURL   string              `mapstructure:"subgraphURL"`
	PoolApprovalSubgraphURL string              `mapstructure:"poolApprovalSubgraphURL"`
	StartBlock              uint64              `mapstructure:"startBlock"`
	ExecutorAddresses       []string            `mapstructure:"executorAddresses"`
}
