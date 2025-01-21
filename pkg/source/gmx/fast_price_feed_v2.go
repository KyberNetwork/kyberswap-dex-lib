package gmx

import (
	"math/big"
	"time"
)

type FastPriceFeedV2 struct {
	DisableFastPriceVoteCount     *big.Int                 `json:"disableFastPriceVoteCount,omitempty"`
	IsSpreadEnabled               bool                     `json:"isSpreadEnabled,omitempty"`
	LastUpdatedAt                 *big.Int                 `json:"lastUpdatedAt,omitempty"`
	MaxDeviationBasisPoints       *big.Int                 `json:"maxDeviationBasisPoints,omitempty"`
	MinAuthorizations             *big.Int                 `json:"minAuthorizations,omitempty"`
	PriceDuration                 *big.Int                 `json:"priceDuration,omitempty"`
	MaxPriceUpdateDelay           *big.Int                 `json:"maxPriceUpdateDelay,omitempty"`
	SpreadBasisPointsIfChainError *big.Int                 `json:"spreadBasisPointsIfChainError,omitempty"`
	SpreadBasisPointsIfInactive   *big.Int                 `json:"spreadBasisPointsIfInactive,omitempty"`
	Prices                        map[string]*big.Int      `json:"prices,omitempty"`
	PriceData                     map[string]PriceDataItem `json:"priceData,omitempty"`
	MaxCumulativeDeltaDiffs       map[string]*big.Int      `json:"maxCumulativeDeltaDiffs,omitempty"`
}

type PriceDataItem struct {
	RefPrice            *big.Int `json:"refPrice"`
	RefTime             uint64   `json:"refTime"`
	CumulativeRefDelta  uint64   `json:"cumulativeRefDelta"`
	CumulativeFastDelta uint64   `json:"cumulativeFastDelta"`
}

func (pf *FastPriceFeedV2) GetVersion() int {
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

func (pf *FastPriceFeedV2) GetPrice(token string, refPrice *big.Int, maximise bool) *big.Int {
	if time.Now().Unix() > pf.LastUpdatedAt.Int64()+pf.MaxPriceUpdateDelay.Int64() {
		if maximise {
			price := new(big.Int).Add(BasisPointsDivisor, pf.SpreadBasisPointsIfChainError)
			return price.Div(price.Mul(refPrice, price), BasisPointsDivisor)
		}

		price := new(big.Int).Sub(BasisPointsDivisor, pf.SpreadBasisPointsIfChainError)
		return price.Div(price.Mul(refPrice, price), BasisPointsDivisor)
	}

	if time.Now().Unix() > pf.LastUpdatedAt.Int64()+pf.PriceDuration.Int64() {
		if maximise {
			price := new(big.Int).Add(BasisPointsDivisor, pf.SpreadBasisPointsIfInactive)
			return price.Div(price.Mul(refPrice, price), BasisPointsDivisor)
		}

		price := new(big.Int).Sub(BasisPointsDivisor, pf.SpreadBasisPointsIfInactive)
		return price.Div(price.Mul(refPrice, price), BasisPointsDivisor)
	}

	fastPrice := pf.Prices[token]
	if fastPrice.Sign() == 0 {
		return refPrice
	}

	var diffBasisPoints *big.Int
	if refPrice.Cmp(fastPrice) > 0 {
		diffBasisPoints = new(big.Int).Sub(refPrice, fastPrice)
	} else {
		diffBasisPoints = new(big.Int).Sub(fastPrice, refPrice)
	}

	diffBasisPoints = diffBasisPoints.Div(diffBasisPoints.Mul(diffBasisPoints, BasisPointsDivisor), refPrice)

	hasSpread := !pf.favorFastPrice(token) || diffBasisPoints.Cmp(pf.MaxDeviationBasisPoints) > 0

	if hasSpread {
		if maximise {
			if refPrice.Cmp(fastPrice) > 0 {
				return refPrice
			}

			return fastPrice
		}

		if refPrice.Cmp(fastPrice) < 0 {
			return refPrice
		}

		return fastPrice
	}

	return fastPrice
}

func (pf *FastPriceFeedV2) favorFastPrice(token string) bool {
	if pf.IsSpreadEnabled {
		return false
	}

	if pf.DisableFastPriceVoteCount.Cmp(pf.MinAuthorizations) >= 0 {
		return false
	}

	_, _, cumulativeRefDelta, cumulativeFastDelta := pf.getPriceData(token)

	if cumulativeFastDelta.Cmp(cumulativeRefDelta) > 0 &&
		new(big.Int).Sub(cumulativeFastDelta, cumulativeRefDelta).Cmp(pf.MaxCumulativeDeltaDiffs[token]) > 0 {
		return false
	}

	return true
}

func (pf *FastPriceFeedV2) getPriceData(token string) (*big.Int, *big.Int, *big.Int, *big.Int) {
	priceDataItem := pf.PriceData[token]

	return priceDataItem.RefPrice,
		new(big.Int).SetUint64(priceDataItem.RefTime),
		new(big.Int).SetUint64(priceDataItem.CumulativeRefDelta),
		new(big.Int).SetUint64(priceDataItem.CumulativeFastDelta)
}
