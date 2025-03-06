package entity

import "math/big"

type AlphaFee struct {
	Token     string
	Amount    *big.Int
	AmountUsd float64
	Pool      string
	AMMAmount *big.Int
}
