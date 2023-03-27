package balancer

import (
	"encoding/json"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/constant"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/utils/duration"
)

type SubgraphPair struct {
	ID       string `json:"id"`
	SwapFee  string `json:"swapFee"`
	PoolType string `json:"poolType"`
	Address  string `json:"address"`
	Tokens   []struct {
		Address  string `json:"address"`
		Weight   string `json:"weight"`
		Decimals int    `json:"decimals"`
	} `json:"tokens"`
}
type StaticExtra struct {
	VaultAddress  string `json:"vaultAddress"`
	PoolId        string `json:"poolId"`
	TokenDecimals []int  `json:"tokenDecimals"`
}
type Extra struct {
	AmplificationParameter AmplificationParameter `json:"amplificationParameter"`
	ScalingFactors         []*big.Int             `json:"scalingFactors,omitempty"`
}
type PoolToken struct {
	Tokens          []common.Address
	Balances        []*big.Int
	LastChangeBlock *big.Int
}
type AmplificationParameter struct {
	Value      *big.Int `json:"value"`
	IsUpdating bool     `json:"isUpdating"`
	Precision  *big.Int `json:"precision"`
}

type Properties struct {
	SubgraphAPI               string
	NewPoolJobIntervalSec     int64
	ReserveJobInterval        duration.Duration `json:"reserveJobInterval"`
	TotalSupplyJobIntervalSec int64
	NewPoolBulk               int
	UpdateReserveBulk         int
	ConcurrentBatches         int `json:"concurrentBatches"`
}

func NewProperties(data map[string]interface{}) (properties Properties, err error) {
	bodyBytes, _ := json.Marshal(data)
	err = json.Unmarshal(bodyBytes, &properties)
	if properties.NewPoolBulk == 0 {
		properties.NewPoolBulk = 100
	}
	if properties.UpdateReserveBulk == 0 {
		properties.UpdateReserveBulk = 100
	}
	if properties.ReserveJobInterval.Duration == 0 {
		properties.ReserveJobInterval.Duration = 600 * time.Second
	}
	if properties.TotalSupplyJobIntervalSec == 0 {
		properties.TotalSupplyJobIntervalSec = 600
	}
	if properties.ConcurrentBatches == 0 {
		properties.ConcurrentBatches = constant.DefaultConcurrentBatches
	}

	return
}
