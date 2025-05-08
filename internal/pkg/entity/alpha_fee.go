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

type AlphaFeeV2 struct {
	AMMAmount      *big.Int
	SwapReductions []AlphaFeeV2SwapReduction
}

type AlphaFeeV2SwapReduction struct {
	// Index of the alpha fee swap in the route summary,
	// count from left-to-right, up-to-down in the route's path.
	ExecutedId      int
	PoolAddress     string
	TokenIn         string
	TokenOut        string
	ReduceAmount    *big.Int
	ReduceAmountUsd float64
}
