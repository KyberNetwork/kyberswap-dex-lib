package curvecompound

type PoolStaticExtra struct {
	LpToken              string   `json:"lpToken"`
	UnderlyingTokens     []string `json:"underlyingTokens"`
	PrecisionMultipliers []string `json:"precisionMultipliers"`
}

type Extra struct {
	A        string   `json:"a"`
	SwapFee  string   `json:"swapFee"`
	AdminFee string   `json:"adminFee"`
	Rates    []string `json:"rates"`
}

type Meta struct {
	TokenInIndex  int  `json:"tokenInIndex"`
	TokenOutIndex int  `json:"tokenOutIndex"`
	Underlying    bool `json:"underlying"`
}

type Gas struct {
	Exchange           int64
	ExchangeUnderlying int64
}
