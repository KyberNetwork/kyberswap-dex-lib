package entity

import "math/big"

type AlphaFee struct {
	Token     string
	Amount    *big.Int
	AmountUsd float64
	Pool      string
	AMMAmount *big.Int

	// index to charged alpha fee in routeSummary
	PathId int
	SwapId int
}
