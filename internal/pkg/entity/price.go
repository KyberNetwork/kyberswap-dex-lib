package entity

import (
	"encoding/json"
	"math/big"
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
}

// for debug print
func (p *OnchainPrice) String() string {
	s, _ := json.Marshal(p)
	return string(s)
}
