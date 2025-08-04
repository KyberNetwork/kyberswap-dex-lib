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
		DepositRate *uint256.Int `json:"dR,omitempty"`
		RedeemRate  *uint256.Int `json:"rR,omitempty"`
	}

	Meta struct {
		BlockNumber uint64 `json:"blockNumber"`
	}

	PoolState struct {
		MaxDeposit  *big.Int
		MaxRedeem   *big.Int
		TotalAssets *big.Int
		TotalSupply *big.Int
		DepositRate *big.Int
		RedeemRate  *big.Int

		blockNumber uint64
	}
)
