package ironstable

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
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
	InitialA           *uint256.Int
	FutureA            *uint256.Int
	InitialATime       *uint256.Int
	FutureATime        *uint256.Int
	Fee                *uint256.Int
	AdminFee           *uint256.Int
	DefaultWithdrawFee *uint256.Int
	LpToken            common.Address
}
