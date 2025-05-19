package trackexecutor

import (
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

type Config struct {
	ChainID           valueobject.ChainID `mapstructure:"chainId"`
	SubgraphURL       string              `mapstructure:"subgraphURL"`
	StartBlock        uint64              `mapstructure:"startBlock"`
	ExecutorAddresses []string            `mapstructure:"executorAddresses"`
}
