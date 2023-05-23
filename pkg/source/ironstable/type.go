package ironstable

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type PoolToken struct {
	Address  string `json:"address"`
	Decimals uint8  `json:"decimals"`
}

type Pool struct {
	ID      string      `json:"id"`
	Tokens  []PoolToken `json:"tokens"`
	SwapFee float64     `json:"swapFee"`
}

type PoolStaticExtra struct {
	LpToken              string   `json:"lpToken"`
	PrecisionMultipliers []string `json:"precisionMultipliers"`
}

type Extra struct {
	InitialA           string `json:"initialA"`
	FutureA            string `json:"futureA"`
	InitialATime       int64  `json:"initialATime"`
	FutureATime        int64  `json:"futureATime"`
	SwapFee            string `json:"swapFee"`
	AdminFee           string `json:"adminFee"`
	DefaultWithdrawFee string `json:"defaultWithdrawFee"`
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
