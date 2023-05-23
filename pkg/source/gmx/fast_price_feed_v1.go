package gmx

import (
	"math/big"
)

type FastPriceFeedV1 struct {
	DisableFastPriceVoteCount *big.Int            `json:"disableFastPriceVoteCount"`
	IsSpreadEnabled           bool                `json:"isSpreadEnabled"`
	LastUpdatedAt             *big.Int            `json:"lastUpdatedAt"`
	MaxDeviationBasisPoints   *big.Int            `json:"maxDeviationBasisPoints"`
	MinAuthorizations         *big.Int            `json:"minAuthorizations"`
	PriceDuration             *big.Int            `json:"priceDuration"`
	VolBasisPoints            *big.Int            `json:"volBasisPoints"`
	Prices                    map[string]*big.Int `json:"prices"`
}

func (fp FastPriceFeedV1) GetVersion() int {
	return int(secondaryPriceFeedVersion1)
}

func NewFastPriceFeedV1() *FastPriceFeedV1 {
	return &FastPriceFeedV1{
		Prices: make(map[string]*big.Int),
	}
}

const (
	fastPriceFeedMethodV1DisableFastPriceVoteCount = "disableFastPriceVoteCount"
	fastPriceFeedMethodV1IsSpreadEnabled           = "isSpreadEnabled"
	fastPriceFeedMethodV1LastUpdatedAt             = "lastUpdatedAt"
	fastPriceFeedMethodV1MaxDeviationBasisPoints   = "maxDeviationBasisPoints"
	fastPriceFeedMethodV1MinAuthorizations         = "minAuthorizations"
	fastPriceFeedMethodV1PriceDuration             = "priceDuration"
	fastPriceFeedMethodV1Prices                    = "prices"
	fastPriceFeedMethodV1VolBasisPoints            = "volBasisPoints"
)
