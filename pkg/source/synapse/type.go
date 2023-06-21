package synapse

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type SwapStorage struct {
	InitialA     *big.Int
	FutureA      *big.Int
	InitialATime *big.Int
	FutureATime  *big.Int
	SwapFee      *big.Int
	AdminFee     *big.Int
	LpToken      common.Address
}

type Extra struct {
	InitialA     string `json:"initialA"`
	FutureA      string `json:"futureA"`
	InitialATime int64  `json:"initialATime"`
	FutureATime  int64  `json:"futureATime"`
	SwapFee      string `json:"swapFee"`
	AdminFee     string `json:"adminFee"`
}

type PoolItem struct {
	ID      string      `json:"id"`
	LpToken string      `json:"lpToken"`
	Tokens  []PoolToken `json:"tokens"`
}

type PoolToken struct {
	Address   string `json:"address"`
	Precision string `json:"precision"`
}

type StaticExtra struct {
	LpToken              string   `json:"lpToken"`
	PrecisionMultipliers []string `json:"precisionMultipliers"`
}
