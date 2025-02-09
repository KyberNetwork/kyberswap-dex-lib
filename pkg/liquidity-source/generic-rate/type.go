package generic_rate

import (
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

type PoolItem struct {
	ID             string             `json:"id"`
	Type           string             `json:"type"`
	Tokens         []entity.PoolToken `json:"tokens"`
	SwapDirections [][2]int           `json:"swapDirections,omitempty"`
	RateProvider   string             `json:"rateProvider,omitempty"`
}

type Extra struct {
	Paused         bool                     `json:"paused"`
	BlockTimestamp uint64                   `json:"blockTimestamp"`
	SwapFuncArgs   []*uint256.Int           `json:"swapFuncArgs"`
	SwapFuncByPair map[int]map[int]SwapFunc `json:"swapFuncByPair"`
}

type StaticExtra struct {
	RateProvider string `json:"rateProvider,omitempty"`
}

type SwapFuncData map[int]map[int]SwapFunc
