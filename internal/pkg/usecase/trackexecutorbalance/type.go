package trackexecutor

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type SubgraphAggregatorResponse struct {
	ExecutorExchanges []ExchangeEvent `json:"executorExchanges"`
}

type ExchangeEvent struct {
	Executor    string `json:"executor"`
	Pair        string `json:"pair"`
	Token       string `json:"token"`
	BlockNumber string `json:"blockNumber"`
}

type PoolInfo struct {
	entity    *entity.Pool
	simulator poolpkg.IPoolSimulator
}
