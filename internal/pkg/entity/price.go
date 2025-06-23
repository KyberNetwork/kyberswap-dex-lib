package entity

import (
	"math/big"

	"github.com/goccy/go-json"
)

type Price struct {
	// how many quote we need to sell to buy 1 token
	Buy *big.Float

	// how many quote we'll get after selling 1 token
	Sell *big.Float
}

type OnchainPrice struct {
	Decimals uint8

	// price in native token unit
	NativePrice Price

	// raw price (wei) against native token
	NativePriceRaw Price

	// this is derived from token's price in Native token unit, and price of Native token in USD unit
	USDPrice Price
}

// for debug print
func (p *OnchainPrice) String() string {
	s, _ := json.Marshal(p)
	return string(s)
}

func (p *OnchainPrice) GetBuyPriceIfAny() float64 {
	buyPrice := p.USDPrice.Buy
	if buyPrice == nil {
		buyPrice = p.USDPrice.Sell
	}

	if buyPrice == nil {
		return 0
	}

	value, _ := buyPrice.Float64()
	return value
}

func (p *OnchainPrice) GetSellPriceIfAny() float64 {
	sellPrice := p.USDPrice.Sell
	if sellPrice == nil {
		sellPrice = p.USDPrice.Buy
	}

	if sellPrice == nil {
		return 0
	}

	value, _ := sellPrice.Float64()
	return value
}
