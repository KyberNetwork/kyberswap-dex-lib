package velocorev2cpmm

import (
	"encoding/hex"
	"math/big"
)

type Metadata struct {
	Offset int `json:"offset"`
}

type bytes32 [32]byte

func (b *bytes32) unwrapToken() string {
	last20Bytes := b[12:]
	return "0x" + hex.EncodeToString(last20Bytes)
}

type StaticExtra struct {
	PoolTokenNumber uint `json:"poolTokenNumber"`
}

type Extra struct {
	Fee1e9        uint32 `json:"fee1e9"`
	FeeMultiplier string `json:"feeMultiplier"`
}

type Meta struct {
	Fee1e9        uint32 `json:"fee1e9"`
	FeeMultiplier string `json:"feeMultiplier"`
}

type SwapInfo struct {
	IsFeeMultiplierUpdated bool   `json:"isFeeMultiplierUpdated"`
	FeeMultiplier          string `json:"feeMultiplier"`
}

// internal types

type velocoreExecuteResult struct {
	Tokens                 []string
	R                      []*big.Int
	FeeMultiplier          *big.Int
	IsFeeMultiplierUpdated bool
}
