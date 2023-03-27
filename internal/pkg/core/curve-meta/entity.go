package curveMeta

type PoolStaticExtra struct {
	LpToken              string   `json:"lpToken"`
	BasePool             string   `json:"basePool"`
	RateMultiplier       string   `json:"rateMultiplier"`
	APrecision           string   `json:"aPrecision"`
	UnderlyingTokens     []string `json:"underlyingTokens"`
	PrecisionMultipliers []string `json:"precisionMultipliers"`
	Rates                []string `json:"rates"`
}

type Extra struct {
	//BasePool       string `json:"basePool"`
	//RateMultiplier string `json:"rateMultiplier"`
	InitialA     string `json:"initialA"`
	FutureA      string `json:"futureA"`
	InitialATime int64  `json:"initialATime"`
	FutureATime  int64  `json:"futureATime"`
	SwapFee      string `json:"swapFee"`
	AdminFee     string `json:"adminFee"`
	//LpToken        string `json:"lpToken"`
	//APrecision     string `json:"aPrecision"`
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
