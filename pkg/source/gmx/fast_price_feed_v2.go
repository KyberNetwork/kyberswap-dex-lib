package gmx

import (
	"math/big"
)

type FastPriceFeedV2 struct {
	DisableFastPriceVoteCount     *big.Int                 `json:"disableFastPriceVoteCount"`
	IsSpreadEnabled               bool                     `json:"isSpreadEnabled"`
	LastUpdatedAt                 *big.Int                 `json:"lastUpdatedAt"`
	MaxDeviationBasisPoints       *big.Int                 `json:"maxDeviationBasisPoints"`
	MinAuthorizations             *big.Int                 `json:"minAuthorizations"`
	PriceDuration                 *big.Int                 `json:"priceDuration"`
	MaxPriceUpdateDelay           *big.Int                 `json:"maxPriceUpdateDelay"`
	SpreadBasisPointsIfChainError *big.Int                 `json:"spreadBasisPointsIfChainError"`
	SpreadBasisPointsIfInactive   *big.Int                 `json:"spreadBasisPointsIfInactive"`
	Prices                        map[string]*big.Int      `json:"prices"`
	PriceData                     map[string]PriceDataItem `json:"priceData"`
	MaxCumulativeDeltaDiffs       map[string]*big.Int      `json:"maxCumulativeDeltaDiffs"`
}

type PriceDataItem struct {
	RefPrice            *big.Int `json:"refPrice"`
	RefTime             uint64   `json:"refTime"`
	CumulativeRefDelta  uint64   `json:"cumulativeRefDelta"`
	CumulativeFastDelta uint64   `json:"cumulativeFastDelta"`
}

func (fp FastPriceFeedV2) GetVersion() int {
	return int(secondaryPriceFeedVersion2)
}

func NewFastPriceFeedV2() *FastPriceFeedV2 {
	return &FastPriceFeedV2{
		Prices:                  make(map[string]*big.Int),
		PriceData:               make(map[string]PriceDataItem),
		MaxCumulativeDeltaDiffs: make(map[string]*big.Int),
	}
}

const (
	fastPriceFeedMethodV2DisableFastPriceVoteCount     = "disableFastPriceVoteCount"
	fastPriceFeedMethodV2IsSpreadEnabled               = "isSpreadEnabled"
	fastPriceFeedMethodV2LastUpdatedAt                 = "lastUpdatedAt"
	fastPriceFeedMethodV2MaxDeviationBasisPoints       = "maxDeviationBasisPoints"
	fastPriceFeedMethodV2MinAuthorizations             = "minAuthorizations"
	fastPriceFeedMethodV2PriceDuration                 = "priceDuration"
	fastPriceFeedMethodV2MaxPriceUpdateDelay           = "maxPriceUpdateDelay"
	fastPriceFeedMethodV2SpreadBasisPointsIfChainError = "spreadBasisPointsIfChainError"
	fastPriceFeedMethodV2SpreadBasisPointsIfInactive   = "spreadBasisPointsIfInactive"
	fastPriceFeedMethodV2Prices                        = "prices"
	fastPriceFeedMethodV2MaxCumulativeDeltaDiffs       = "maxCumulativeDeltaDiffs"
	fastPriceFeedMethodV2GetPriceData                  = "getPriceData"
)
