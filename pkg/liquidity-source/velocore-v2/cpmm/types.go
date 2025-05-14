package cpmm

import (
	"encoding/hex"
	"math/big"
	"strings"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type Gas struct {
	Swap int64
}

type Metadata struct {
	Offset int `json:"offset"`
}

type bytes32 [32]byte

func (b *bytes32) unwrapToken() string {
	last20Bytes := b[12:]
	address := "0x" + hex.EncodeToString(last20Bytes)
	return strings.ToLower(address)
}

type StaticExtra struct {
	Weights          []*big.Int `json:"weights"`
	PoolTokenNumber  uint       `json:"poolTokenNumber"`
	NativeTokenIndex int        `json:"nativeTokenIndex"`
	Vault            string     `json:"vault"`
}

type Extra struct {
	ChainID       valueobject.ChainID `json:"chainId"`
	Fee1e9        uint32              `json:"fee1e9"`
	FeeMultiplier *big.Int            `json:"feeMultiplier"`
}

type Meta struct {
	Vault            string `json:"vault"`
	NativeTokenIndex int    `json:"nativeTokenIndex"`
	BlockNumber      uint64 `json:"blockNumber"`
	ApprovalAddress  string `json:"approvalAddress"`
}

type SwapInfo struct {
	IsFeeMultiplierUpdated bool   `json:"-"`
	FeeMultiplier          string `json:"-"`
}

// internal types

type velocoreExecuteResult struct {
	Tokens                 []string
	R                      []*big.Int
	FeeMultiplier          *big.Int
	IsFeeMultiplierUpdated bool
}
