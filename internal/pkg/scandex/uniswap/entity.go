package uniswap

import (
	"encoding/json"
	"math/big"
	"time"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/duration"
)

type Pair struct {
	Address string
	Token0  string
	Token1  string
}

type Properties struct {
	SwapFee                   float64
	SubgraphAPI               string
	FactoryAddress            string
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

type Reserves struct {
	Reserve0           *big.Int
	Reserve1           *big.Int
	BlockTimestampLast uint32
}
