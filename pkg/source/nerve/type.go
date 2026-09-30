package nerve

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
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
	InitialA           *uint256.Int
	FutureA            *uint256.Int
	InitialATime       *uint256.Int
	FutureATime        *uint256.Int
	SwapFee            *uint256.Int
	AdminFee           *uint256.Int
	DefaultDepositFee  *uint256.Int
	DefaultWithdrawFee *uint256.Int
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
