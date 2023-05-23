package oneswap

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type Metadata struct {
	Offset int `json:"offset"`
}

type SwapStorage struct {
	InitialA           *big.Int
	FutureA            *big.Int
	InitialATime       *big.Int
	FutureATime        *big.Int
	SwapFee            *big.Int
	AdminFee           *big.Int
	DefaultWithdrawFee *big.Int
	LpToken            common.Address
}

type StaticExtra struct {
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
