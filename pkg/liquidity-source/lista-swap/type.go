package listaswap

import "math/big"

type StaticExtra struct {
	LpToken              string   `json:"lpToken"`
	APrecision           string   `json:"aPrecision"`
	PrecisionMultipliers []string `json:"precisionMultipliers"`
	Rates                []string `json:"rates"`
	IsNativeCoins        []bool   `json:"isNativeCoins"`
}

type Extra struct {
	InitialA           string      `json:"initialA"`
	FutureA            string      `json:"futureA"`
	InitialATime       int64       `json:"initialATime"`
	FutureATime        int64       `json:"futureATime"`
	SwapFee            string      `json:"swapFee"`
	AdminFee           string      `json:"adminFee"`
	OraclePrices       [2]*big.Int `json:"oraclePrices"`
	PriceDiffThreshold [2]*big.Int `json:"priceDiffThreshold"`
}

type Meta struct {
	TokenInIndex  int  `json:"tokenInIndex"`
	TokenOutIndex int  `json:"tokenOutIndex"`
	Underlying    bool `json:"underlying,omitempty"`

	TokenInIsNative  bool `json:"tokenInIsNative,omitempty"`
	TokenOutIsNative bool `json:"tokenOutIsNative,omitempty"`
}
