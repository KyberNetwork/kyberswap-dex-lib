package hashflow

import (
	"math/big"
	"strings"
)

const (
	PoolIDPrefix    = "hf"
	PoolIDSeparator = ":"
)

type (
	Pair struct {
		MarketMaker          string
		Tokens               []string
		Decimals             []uint8
		ZeroToOnePriceLevels []PriceLevel
		OneToZeroPriceLevels []PriceLevel
	}
	PriceLevel struct {
		Level *big.Float `json:"level"`
		Price *big.Float `json:"price"`
	}

	StaticExtra struct {
		MarketMaker string `json:"marketMaker"`
	}
	Extra struct {
		ZeroToOnePriceLevels []PriceLevel `json:"zeroToOnePriceLevels"`
		OneToZeroPriceLevels []PriceLevel `json:"oneToZeroPriceLevels"`
	}

	PoolID struct {
		MarketMaker string
		Token0      string
		Token1      string
	}
)

func (p PoolID) String() string {
	return strings.Join([]string{PoolIDPrefix, p.MarketMaker, p.Token0, p.Token1}, PoolIDSeparator)
}

func ParsePoolID(str string) (PoolID, bool) {
	splits := strings.Split(str, PoolIDSeparator)

	if len(splits) != 4 {
		return PoolID{}, false
	}

	return PoolID{
		MarketMaker: splits[1],
		Token0:      splits[2],
		Token1:      splits[3],
	}, true
}
