package lido

import (
	"encoding/json"
	"math/big"
	"time"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/constant"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/entity"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/utils/duration"
)

type StaticExtra struct {
	LpToken string `json:"lpToken"`
}

type Extra struct {
	StEthPerToken  *big.Int `json:"stEthPerToken"`  // Get amount of stETH for a one wstETH
	TokensPerStEth *big.Int `json:"tokensPerStEth"` // Get amount of wstETH for a one stETH
}

type PoolItem struct {
	ID      string             `json:"id"`
	Type    string             `json:"type"`
	Name    string             `json:"name"`
	LpToken string             `json:"lpToken"`
	Tokens  []entity.PoolToken `json:"tokens"`
}

type Properties struct {
	PoolPath                  string            `json:"poolPath"`
	NewPoolJobIntervalSec     int64             `json:"newPoolJobIntervalSec"`
	ReserveJobInterval        duration.Duration `json:"reserveJobInterval"`
	TotalSupplyJobIntervalSec int64             `json:"totalSupplyJobIntervalSec"`
	NewPoolBulk               int               `json:"newPoolBulk"`
	UpdateReserveBulk         int               `json:"updateReserveBulk"`
	ConcurrentBatches         int               `json:"concurrentBatches"`
}

func NewProperties(data map[string]interface{}) (properties Properties, err error) {
	bodyBytes, err := json.Marshal(data)
	if err != nil {
		return
	}

	err = json.Unmarshal(bodyBytes, &properties)
	if err != nil {
		return
	}

	if properties.NewPoolBulk == 0 {
		properties.NewPoolBulk = 10
	}

	if properties.UpdateReserveBulk == 0 {
		properties.UpdateReserveBulk = 10
	}

	if properties.ReserveJobInterval.Duration == 0 {
		properties.ReserveJobInterval.Duration = 10 * time.Second
	}

	if properties.TotalSupplyJobIntervalSec == 0 {
		properties.TotalSupplyJobIntervalSec = 10
	}

	if properties.ConcurrentBatches == 0 {
		properties.ConcurrentBatches = constant.DefaultConcurrentBatches
	}

	return
}
