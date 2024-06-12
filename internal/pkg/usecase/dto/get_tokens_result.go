package dto

type (
	GetTokensResult struct {
		Tokens []*GetTokensResultToken `json:"tokens"`
	}

	GetTokensResultToken struct {
		Address  string                `json:"address"`
		Name     string                `json:"name"`
		Decimals uint8                 `json:"decimals"`
		Symbol   string                `json:"symbol"`
		Price    *GetTokensResultPrice `json:"price,omitempty"`
		Hash     string                `json:"hash"`
	}

	GetTokensResultPrice struct {
		Price             float64 `json:"price"`
		Liquidity         float64 `json:"liquidity"`
		LpAddress         string  `json:"lpAddress"`
		MarketPrice       float64 `json:"marketPrice"`
		PreferPriceSource string  `json:"preferPriceSource"`
	}
)
