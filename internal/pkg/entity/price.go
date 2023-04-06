package entity

import (
	"encoding/json"
)

const PriceKey = "prices"

type PriceSource string

const (
	PriceSourceKyberswap PriceSource = "kyberswap"
	PriceSourceCoingecko PriceSource = "coingecko"
)

type Price struct {
	Address           string      `json:"address"`
	Price             float64     `json:"price"`
	Liquidity         float64     `json:"liquidity"`
	LpAddress         string      `json:"lpAddress"`
	MarketPrice       float64     `json:"marketPrice"`
	PreferPriceSource PriceSource `json:"preferPriceSource"`
}

func (p Price) Encode() string {
	bytes, _ := json.Marshal(p)

	return string(bytes)
}

// DecodePrice will decode price from the string
func DecodePrice(key, member string) Price {
	var p Price
	err := json.Unmarshal([]byte(member), &p)
	if err != nil {
		return Price{}
	}

	p.Address = key

	return p
}

// GetPreferredPrice returns the preferred price + if the value is market price or not
// Default is price from Coingecko price source
func (p Price) GetPreferredPrice() (float64, bool) {
	// We don't always have market price, so it's better to have this fallback
	if p.MarketPrice == 0 {
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
