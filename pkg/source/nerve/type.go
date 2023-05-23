package nerve

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type PoolToken struct {
	Address   string `json:"address"`
	Precision string `json:"precision"`
}

type PoolItem struct {
	ID     string      `json:"id"`
	Name   string      `json:"name"`
	Tokens []PoolToken `json:"tokens"`
}

type SwapStorage struct {
	InitialA           *big.Int
	FutureA            *big.Int
	InitialATime       *big.Int
	FutureATime        *big.Int
	SwapFee            *big.Int
	AdminFee           *big.Int
	DefaultDepositFee  *big.Int
	DefaultWithdrawFee *big.Int
	Devaddr            common.Address
	LpToken            common.Address
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

type PoolStaticExtra struct {
	PrecisionMultipliers []string `json:"precisionMultipliers"`
}
