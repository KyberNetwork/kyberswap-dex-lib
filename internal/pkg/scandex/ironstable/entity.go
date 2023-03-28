package ironstable

import (
	"encoding/json"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/duration"
)

type PoolToken struct {
	Address   string `json:"address"`
	Precision string `json:"precision"`
}

type PoolItem struct {
	ID      string      `json:"id"`
	LpToken string      `json:"lpToken"`
	Tokens  []PoolToken `json:"tokens"`
}

type Properties struct {
	PoolPath              string
	NewPoolJobIntervalSec int64
	ReserveJobInterval    duration.Duration `json:"reserveJobInterval"`
	NewPoolLimit          int
	UpdateReserveBulk     int
	ConcurrentBatches     int `json:"concurrentBatches"`
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
	if properties.NewPoolLimit == 0 {
		properties.NewPoolLimit = 100
	}
	if properties.ConcurrentBatches == 0 {
		properties.ConcurrentBatches = constant.DefaultConcurrentBatches
	}

	return
}

type SwapStorage struct {
	InitialA           *big.Int
	FutureA            *big.Int
	InitialATime       *big.Int
	FutureATime        *big.Int
	Fee                *big.Int
	AdminFee           *big.Int
	DefaultWithdrawFee *big.Int
	LpToken            common.Address
}

type Balances []*big.Int
