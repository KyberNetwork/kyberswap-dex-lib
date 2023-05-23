package metavault

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
	return int(SecondaryPriceFeedVersion1)
}

func NewFastPriceFeedV1() *FastPriceFeedV1 {
	return &FastPriceFeedV1{
		Prices: make(map[string]*big.Int),
	}
}

const (
	FastPriceFeedMethodV1DisableFastPriceVoteCount = "disableFastPriceVoteCount"
	FastPriceFeedMethodV1IsSpreadEnabled           = "isSpreadEnabled"
	FastPriceFeedMethodV1LastUpdatedAt             = "lastUpdatedAt"
	FastPriceFeedMethodV1MaxDeviationBasisPoints   = "maxDeviationBasisPoints"
	FastPriceFeedMethodV1MinAuthorizations         = "minAuthorizations"
	FastPriceFeedMethodV1PriceDuration             = "priceDuration"
	FastPriceFeedMethodV1Prices                    = "prices"
	FastPriceFeedMethodV1VolBasisPoints            = "volBasisPoints"
)
