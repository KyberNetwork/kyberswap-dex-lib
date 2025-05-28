package vaultT1

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type SwapPath struct {
	Protocol common.Address `json:"protocol"`
	TokenIn  common.Address `json:"tokenIn"`
	TokenOut common.Address `json:"tokenOut"`
}

type SwapData struct {
	InAmt      *big.Int `json:"inAmt"`
	OutAmt     *big.Int `json:"outAmt"`
	WithAbsorb bool     `json:"withAbsorb"`
	Ratio      *big.Int `json:"ratio"`
}

type Swap struct {
	Path SwapPath `json:"path"`
	Data SwapData `json:"data"`
}

type PoolMeta struct {
	BlockNumber     uint64 `json:"blockNumber"`
	ApprovalAddress string `json:"approvalAddress,omitempty"`
}

type PoolExtra struct {
	WithAbsorb bool     `json:"withAbsorb"`
	Ratio      *big.Int `json:"ratio"`
}

type Gas struct {
	Liquidate int64
}

type StaticExtra struct {
	VaultLiquidationResolver string `json:"vaultLiquidationResolver"`
	HasNative                bool   `json:"hasNative"`
}
