package entity

import (
	"math/big"

	"github.com/goccy/go-json"
)

var two = big.NewFloat(2)

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

func (p *OnchainPrice) GetSellPriceUSD() float64 {
	if p != nil && p.USDPrice.Sell != nil {
		res, _ := p.USDPrice.Sell.Float64()
		return res
	}

	return 0.0
}

func (p *OnchainPrice) GetBuyPriceUSD() float64 {
	if p != nil && p.USDPrice.Buy != nil {
		res, _ := p.USDPrice.Buy.Float64()
		return res
	}

	return 0.0
}

func (p *OnchainPrice) GetMidPriceUSD() float64 {
	if p == nil {
		return 0.0
	}

	buy := p.USDPrice.Buy
	sell := p.USDPrice.Sell

	if buy != nil && sell != nil {
		var sum big.Float
		sum.Add(buy, sell)
		sum.Quo(&sum, two)
		res, _ := sum.Float64()
		return res
	}

	// only fall back sell price
	if sell != nil {
		res, _ := sell.Float64()
		return res
	}

	return 0.0
}

func (p *OnchainPrice) GetMidPriceNative() float64 {
	if p == nil {
		return 0.0
	}

	buy := p.NativePrice.Buy
	sell := p.NativePrice.Sell

	if buy != nil && sell != nil {
		var sum big.Float
		sum.Add(buy, sell)
		sum.Quo(&sum, two)
		res, _ := sum.Float64()
		return res
	}

	if buy != nil {
		res, _ := buy.Float64()
		return res
	}
	if sell != nil {
		res, _ := sell.Float64()
		return res
	}

	return 0.0
}

func (p *OnchainPrice) GetMidPriceNativeRaw() float64 {
	if p == nil {
		return 0.0
	}

	buy := p.NativePriceRaw.Buy
	sell := p.NativePriceRaw.Sell

	if buy != nil && sell != nil {
		var sum big.Float
		sum.Add(buy, sell)
		sum.Quo(&sum, two)
		res, _ := sum.Float64()
		return res
	}

	if buy != nil {
		res, _ := buy.Float64()
		return res
	}
	if sell != nil {
		res, _ := sell.Float64()
		return res
	}

	return 0.0
}
