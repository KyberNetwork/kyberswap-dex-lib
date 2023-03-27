package curve

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/constant"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/utils/duration"
)

type PoolToken struct {
	Address   string `json:"address"`
	Precision string `json:"precision"`
	Rate      string `json:"rate"`
}

type PoolItem struct {
	ID               string      `json:"id"`
	Type             string      `json:"type"`
	Tokens           []PoolToken `json:"tokens"`
	LpToken          string      `json:"lpToken"`
	APrecision       string      `json:"aPrecision"`
	Version          int         `json:"version"`
	BasePool         string      `json:"basePool"`
	RateMultiplier   string      `json:"rateMultiplier"`
	UnderlyingTokens []string    `json:"underlyingTokens"`
}

type Properties struct {
	PoolPath                  string
	NewPoolJobIntervalSec     int64
	ReserveJobInterval        duration.Duration `json:"reserveJobInterval"`
	TotalSupplyJobIntervalSec int64
	NewPoolLimit              int
	UpdateReserveBulk         int
	ConcurrentBatches         int `json:"concurrentBatches"`
	AddressesFromProvider     []string
	IgnorePools               []string
}

func NewProperties(data map[string]interface{}) (properties Properties, err error) {
	bodyBytes, _ := json.Marshal(data)
	err = json.Unmarshal(bodyBytes, &properties)
	if properties.UpdateReserveBulk == 0 {
		properties.UpdateReserveBulk = 10
	}
	if properties.ReserveJobInterval.Duration == 0 {
		properties.ReserveJobInterval.Duration = 2 * time.Second
	}
	if properties.NewPoolJobIntervalSec == 0 {
		properties.NewPoolJobIntervalSec = 60
	}
	if properties.TotalSupplyJobIntervalSec == 0 {
		properties.TotalSupplyJobIntervalSec = 60
	}
	if properties.NewPoolLimit == 0 {
		properties.NewPoolLimit = 100
	}
	if properties.ConcurrentBatches == 0 {
		properties.ConcurrentBatches = constant.DefaultConcurrentBatches
	}

	return
}

var (
	ErrCanNotGetBalances = errors.New("cant get balances")
	ErrOneTokenHasNoRate = errors.New("1 token has no rate")
)
