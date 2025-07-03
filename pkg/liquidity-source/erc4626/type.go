package erc4626

import (
	"math/big"

	"github.com/holiman/uint256"
)

type SwapType uint8

const (
	None SwapType = iota
	Deposit
	Redeem
	Both
)

type (
	Gas struct {
		Deposit uint64 `json:"d,omitempty"`
		Redeem  uint64 `json:"r,omitempty"`
	}

	Extra struct {
		Gas         Gas          `json:"g"`
		SwapTypes   SwapType     `json:"sT,omitempty"`
		MaxDeposit  *uint256.Int `json:"mD,omitempty"`
		MaxRedeem   *uint256.Int `json:"mR,omitempty"`
		EntryFeeBps uint64       `json:"dF,omitempty"`
		ExitFeeBps  uint64       `json:"rF,omitempty"`
	}

	SwapInfo struct {
		assets *uint256.Int
	}

	Meta struct {
		BlockNumber uint64 `json:"blockNumber"`
	}

	PoolState struct {
		TotalSupply *big.Int
		TotalAssets *big.Int
		MaxDeposit  *big.Int
		MaxRedeem   *big.Int
		EntryFeeBps uint64
		ExitFeeBps  uint64

		blockNumber uint64
	}
)
