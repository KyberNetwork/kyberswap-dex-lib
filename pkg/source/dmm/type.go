//go:generate go run github.com/tinylib/msgp -unexported -tests=false -v
//msgp:tuple Gas
//msgp:ignore PoolModelReserves Extra Metadata TradeInfo ExtraField

package dmm

import (
	"math/big"
)

type PoolModelReserves []string

type Extra struct {
	VReserves      PoolModelReserves `json:"vReserves"`
	FeeInPrecision string            `json:"feeInPrecision"`
}

type Gas struct {
	SwapBase    int64
	SwapNonBase int64
}

type Metadata struct {
	Offset int `json:"offset"`
}

type TradeInfo struct {
	Reserve0       *big.Int
	Reserve1       *big.Int
	VReserve0      *big.Int
	VReserve1      *big.Int
	FeeInPrecision *big.Int
}

type ExtraField struct {
	VReserves      []string `json:"vReserves"`
	FeeInPrecision string   `json:"feeInPrecision"`
}
