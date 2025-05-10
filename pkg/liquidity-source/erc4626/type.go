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
		Deposit uint64 `json:"deposit,omitempty"`
		Redeem  uint64 `json:"redeem,omitempty"`
	}

	Extra struct {
		Gas        Gas          `json:"gas"`
		SwapTypes  SwapType     `json:"swapTypes"`
		MaxDeposit *uint256.Int `json:"maxDeposit,omitempty"`
		MaxRedeem  *uint256.Int `json:"maxRedeem,omitempty"`
	}

	Meta struct {
		BlockNumber uint64 `json:"blockNumber"`
	}

	PoolState struct {
		TotalSupply *big.Int
		TotalAssets *big.Int
		MaxDeposit  *big.Int
		MaxRedeem   *big.Int

		blockNumber uint64
	}

	PostSwapState struct {
		totalSupply *uint256.Int
		totalAssets *uint256.Int
	}
)
