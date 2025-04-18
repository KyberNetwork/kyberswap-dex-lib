package entity

import "math/big"

type AlphaFee struct {
	AlphaFeeToken string
	Amount        *big.Int
	AmountUsd     float64
	AMMAmount     *big.Int

	// index to charged alpha fee in routeSummary -> deprecated
	// PathId int
	// SwapId int

	// now we use new field executedId which is the order where sc execute the swap contains alpha fee
	Pool       string
	ExecutedId int32
	TokenIn    string
}
