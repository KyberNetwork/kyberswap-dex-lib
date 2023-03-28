package dodo

import (
	"encoding/json"
	"errors"
	"math/big"
	"time"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/duration"
)

type Token struct {
	Address  string `json:"id"`
	Name     string `json:"name"`
	Symbol   string `json:"symbol"`
	Decimals string `json:"decimals"`
}

type SubgraphPair struct {
	ID                 string `json:"id"`
	BaseToken          Token  `json:"baseToken"`
	QuoteToken         Token  `json:"quoteToken"`
	BaseLpToken        Token  `json:"baseLpToken"`
	I                  string `json:"i"`
	K                  string `json:"k"`
	LpFeeRate          string `json:"lpFeeRate"`
	MtFeeRate          string `json:"mtFeeRate"`
	BaseReserve        string `json:"baseReserve"`
	QuoteReserve       string `json:"quoteReserve"`
	IsTradeAllowed     bool   `json:"isTradeAllowed"`
	Type               string `json:"type"`
	CreatedAtTimestamp string `json:"createdAtTimestamp"`
}

type StaticExtra struct {
	PoolId           string   `json:"poolId"`
	LpToken          string   `json:"lpToken"`
	Type             string   `json:"type"`
	Tokens           []string `json:"tokens"`
	DodoV1SellHelper string   `json:"dodoV1SellHelper"`
}

type Extra struct {
	I              *big.Int   `json:"i"`
	K              *big.Int   `json:"k"`
	RStatus        int        `json:"rStatus"`
	MtFeeRate      *big.Float `json:"mtFeeRate"`
	LpFeeRate      *big.Float `json:"lpFeeRate"`
	Swappable      bool       `json:"swappable"`
	Reserves       []*big.Int `json:"reserves"`
	TargetReserves []*big.Int `json:"targetReserves"`
}

type PoolState struct {
	I  *big.Int `json:"i"`
	K  *big.Int `json:"K"`
	B  *big.Int `json:"B"`
	Q  *big.Int `json:"Q"`
	B0 *big.Int `json:"B0"`
	Q0 *big.Int `json:"Q0"`
	R  *big.Int `json:"R"`
}

type FeeRate struct {
	MtFeeRate *big.Int `json:"mtFeeRate"`
	LpFeeRate *big.Int `json:"lpFeeRate"`
}

type Properties struct {
	SubgraphAPI               string
	DodoV1SellHelper          string
	NewPoolJobIntervalSec     int64
	ReserveJobInterval        duration.Duration `json:"reserveJobInterval"`
	TotalSupplyJobIntervalSec int64
	NewPoolBulk               int
	UpdateReserveBulk         int
	ConcurrentBatches         int `json:"concurrentBatches"`
}

func NewProperties(data map[string]interface{}) (properties Properties, err error) {
	bodyBytes, err := json.Marshal(data)
	if err != nil {
		return
	}
	err = json.Unmarshal(bodyBytes, &properties)
	if err != nil {
		return properties, errors.New("cannot unmarshal Properties of scan configuration")
	}
	if properties.SubgraphAPI == "" {
		return properties, errors.New("missing the subgraph configuration")
	}
	if properties.DodoV1SellHelper == "" {
		return properties, errors.New("missing DodoV1SellHelper configuration")
	}
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
