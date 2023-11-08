package fxdx

import (
	"math/big"
	"time"

	"github.com/KyberNetwork/blockchain-toolkit/integer"
)

type FastPriceFeed struct {
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
	RefTime             *big.Int `json:"refTime"`
	CumulativeRefDelta  *big.Int `json:"cumulativeRefDelta"`
	CumulativeFastDelta *big.Int `json:"cumulativeFastDelta"`
}

func NewFastPriceFeed() *FastPriceFeed {
	return &FastPriceFeed{
		Prices:                  make(map[string]*big.Int),
		PriceData:               make(map[string]PriceDataItem),
		MaxCumulativeDeltaDiffs: make(map[string]*big.Int),
	}
}

const (
	fastPriceFeedMethodDisableFastPriceVoteCount     = "disableFastPriceVoteCount"
	fastPriceFeedMethodIsSpreadEnabled               = "isSpreadEnabled"
	fastPriceFeedMethodLastUpdatedAt                 = "lastUpdatedAt"
	fastPriceFeedMethodMaxDeviationBasisPoints       = "maxDeviationBasisPoints"
	fastPriceFeedMethodMinAuthorizations             = "minAuthorizations"
	fastPriceFeedMethodPriceDuration                 = "priceDuration"
	fastPriceFeedMethodMaxPriceUpdateDelay           = "maxPriceUpdateDelay"
	fastPriceFeedMethodSpreadBasisPointsIfChainError = "spreadBasisPointsIfChainError"
	fastPriceFeedMethodSpreadBasisPointsIfInactive   = "spreadBasisPointsIfInactive"
	fastPriceFeedMethodPrices                        = "prices"
	fastPriceFeedMethodMaxCumulativeDeltaDiffs       = "maxCumulativeDeltaDiffs"
	fastPriceFeedMethodGetPriceData                  = "getPriceData"
)

func (pf *FastPriceFeed) GetPrice(token string, refPrice *big.Int, maximise bool) *big.Int {
	if new(big.Int).SetInt64(time.Now().Unix()).Cmp(new(big.Int).Add(pf.LastUpdatedAt, pf.MaxPriceUpdateDelay)) > 0 {
		if maximise {
			return new(big.Int).Div(new(big.Int).Mul(refPrice, new(big.Int).Add(BasisPointsDivisor, pf.SpreadBasisPointsIfChainError)), BasisPointsDivisor)
		}

		return new(big.Int).Div(new(big.Int).Mul(refPrice, new(big.Int).Sub(BasisPointsDivisor, pf.SpreadBasisPointsIfChainError)), BasisPointsDivisor)
	}

	if new(big.Int).SetInt64(time.Now().Unix()).Cmp(new(big.Int).Add(pf.LastUpdatedAt, pf.PriceDuration)) > 0 {
		if maximise {
			return new(big.Int).Div(new(big.Int).Mul(refPrice, new(big.Int).Add(BasisPointsDivisor, pf.SpreadBasisPointsIfInactive)), BasisPointsDivisor)
		}

		return new(big.Int).Div(new(big.Int).Mul(refPrice, new(big.Int).Sub(BasisPointsDivisor, pf.SpreadBasisPointsIfInactive)), BasisPointsDivisor)
	}

	fastPrice := pf.Prices[token]
	if fastPrice.Cmp(integer.Zero()) == 0 {
		return refPrice
	}

	var diffBasisPoints *big.Int
	if refPrice.Cmp(fastPrice) > 0 {
		diffBasisPoints = new(big.Int).Sub(refPrice, fastPrice)
	} else {
		diffBasisPoints = new(big.Int).Sub(fastPrice, refPrice)
	}

	diffBasisPoints = new(big.Int).Div(new(big.Int).Mul(diffBasisPoints, BasisPointsDivisor), refPrice)

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

func (pf *FastPriceFeed) favorFastPrice(token string) bool {
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

func (pf *FastPriceFeed) getPriceData(token string) (*big.Int, *big.Int, *big.Int, *big.Int) {
	priceDataItem := pf.PriceData[token]

	return priceDataItem.RefPrice,
		priceDataItem.RefTime,
		priceDataItem.CumulativeRefDelta,
		priceDataItem.CumulativeFastDelta
}
