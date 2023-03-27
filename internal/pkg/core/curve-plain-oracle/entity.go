package curveplainoracle

import "math/big"

type PoolStaticExtra struct {
	LpToken              string   `json:"lpToken"`
	APrecision           string   `json:"aPrecision"`
	PrecisionMultipliers []string `json:"precisionMultipliers"`
	Oracle               string   `json:"oracle"`
}
type Extra struct {
	Rates        []*big.Int `json:"rates"`
	InitialA     string     `json:"initialA"`
	FutureA      string     `json:"futureA"`
	InitialATime int64      `json:"initialATime"`
	FutureATime  int64      `json:"futureATime"`
	SwapFee      string     `json:"swapFee"`
	AdminFee     string     `json:"adminFee"`
}

type Meta struct {
	TokenInIndex  int  `json:"tokenInIndex"`
	TokenOutIndex int  `json:"tokenOutIndex"`
	Underlying    bool `json:"underlying"`
}

type Gas struct {
	Exchange int64
}
