package entity

type PriceSource string

const (
	PriceSourceKyberswap PriceSource = "kyberswap"
	PriceSourceCoingecko PriceSource = "coingecko"

	// UpperLimitPrice is the upper limit price for a token in $, if the price is higher than this, we should ignore it
	UpperLimitPrice = 1000000
)

type Price struct {
	Address           string      `json:"address"`
	Price             float64     `json:"price"`
	Liquidity         float64     `json:"liquidity"`
	LpAddress         string      `json:"lpAddress"`
	MarketPrice       float64     `json:"marketPrice"`
	PreferPriceSource PriceSource `json:"preferPriceSource"`
}

// GetPreferredPrice returns the preferred price and whether it is a market price.
// When MarketPrice is zero and Price does not exceed UpperLimitPrice, it uses the pool price;
// otherwise, the default preferred source is the market price from CoinGecko.
func (p Price) GetPreferredPrice() (float64, bool) {
	// We don't always have market price, so it's better to have this fallback
	if p.MarketPrice == 0 && p.Price <= UpperLimitPrice {
		return p.Price, false
	}

	switch p.PreferPriceSource {
	case PriceSourceKyberswap:
		return p.Price, false
	case PriceSourceCoingecko:
		return p.MarketPrice, true
	default:
		return p.MarketPrice, true
	}
}
