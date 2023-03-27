package curveAave

type PoolStaticExtra struct {
	LpToken              string   `json:"lpToken"`
	UnderlyingTokens     []string `json:"underlyingTokens"`
	PrecisionMultipliers []string `json:"precisionMultipliers"`
}

type Extra struct {
	InitialA            string `json:"initialA"`
	FutureA             string `json:"futureA"`
	InitialATime        int64  `json:"initialATime"`
	FutureATime         int64  `json:"futureATime"`
	SwapFee             string `json:"swapFee"`
	AdminFee            string `json:"adminFee"`
	OffpegFeeMultiplier string `json:"offpegFeeMultiplier"`
}

type Meta struct {
	TokenInIndex  int  `json:"tokenInIndex"`
	TokenOutIndex int  `json:"tokenOutIndex"`
	Underlying    bool `json:"underlying"`
}

type Gas struct {
	ExchangeUnderlying int64
}
