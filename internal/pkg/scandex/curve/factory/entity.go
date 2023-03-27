package factory

type PoolToken struct {
	Address   string `json:"address"`
	Precision string `json:"precision"`
	Rate      string `json:"rate"`
}

type PoolItem struct {
	ID               string      `json:"id"`
	Type             string      `json:"type"`
	Tokens           []PoolToken `json:"tokens"`
	LpToken          string      `json:"lpToken"`
	APrecision       string      `json:"aPrecision"`
	Version          int         `json:"version"`
	BasePool         string      `json:"basePool"`
	RateMultiplier   string      `json:"rateMultiplier"`
	UnderlyingTokens []string    `json:"underlyingTokens"`
}
