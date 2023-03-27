package entity

import (
	"strconv"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/utils"
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
	return utils.Join(p.Price, p.Liquidity, p.LpAddress, p.MarketPrice, string(p.PreferPriceSource))
}

// DecodePrice will decode price from the string
func DecodePrice(key, member string) Price {
	var p Price
	split := utils.Split(member)
	p.Address = key
	p.Price, _ = strconv.ParseFloat(split[0], 64)
	p.Liquidity, _ = strconv.ParseFloat(split[1], 64)
	p.LpAddress = split[2]
	if len(split) >= 4 {
		p.MarketPrice, _ = strconv.ParseFloat(split[3], 64)
	} else {
		p.MarketPrice = 0
	}

	// For example: 10000:100000:0x4535913573d299a6372ca43b90aa6be1cf68f779:120000:coingecko will return
	//{
	//  Address:           "key",
	//	Price:             10000,
	//	Liquidity:         100000,
	//	LpAddress:         "0x4535913573d299a6372ca43b90aa6be1cf68f779",
	//	MarketPrice:       120000,
	//	PreferPriceSource: PriceSourceCoingecko,
	//}
	if len(split) >= 5 {
		p.PreferPriceSource = parsePriceSource(split[4])
	} else {
		p.PreferPriceSource = PriceSourceCoingecko
	}

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

func parsePriceSource(source string) PriceSource {
	return PriceSource(source)
}
