package gmxglp

import (
	"math/big"
	"time"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
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

func (pf *FastPriceFeedV1) GetPrice(token string, refPrice *big.Int, maximise bool) *big.Int {
	if new(big.Int).SetInt64(time.Now().Unix()).Cmp(new(big.Int).Add(pf.LastUpdatedAt, pf.PriceDuration)) > 0 {
		return refPrice
	}

	fastPrice := pf.Prices[token]
	if fastPrice.Cmp(bignumber.ZeroBI) == 0 {
		return refPrice
	}

	maxPrice := new(big.Int).Div(new(big.Int).Mul(refPrice, new(big.Int).Add(BasisPointsDivisor, pf.MaxDeviationBasisPoints)), BasisPointsDivisor)
	minPrice := new(big.Int).Div(new(big.Int).Mul(refPrice, new(big.Int).Sub(BasisPointsDivisor, pf.MaxDeviationBasisPoints)), BasisPointsDivisor)

	if pf.favorFastPrice() {
		if fastPrice.Cmp(minPrice) >= 0 && fastPrice.Cmp(maxPrice) <= 0 {
			if maximise {
				if refPrice.Cmp(fastPrice) > 0 {
					volPrice := new(big.Int).Div(new(big.Int).Mul(fastPrice, new(big.Int).Add(BasisPointsDivisor, pf.VolBasisPoints)), BasisPointsDivisor)

					if volPrice.Cmp(refPrice) > 0 {
						return refPrice
					} else {
						return volPrice
					}
				}

				return fastPrice
			}

			if refPrice.Cmp(fastPrice) < 0 {
				volPrice := new(big.Int).Div(new(big.Int).Mul(fastPrice, new(big.Int).Sub(BasisPointsDivisor, pf.VolBasisPoints)), BasisPointsDivisor)

				if volPrice.Cmp(refPrice) < 0 {
					return refPrice
				} else {
					return volPrice
				}
			}

			return fastPrice
		}
	}

	if maximise {
		if refPrice.Cmp(fastPrice) > 0 {
			return refPrice
		}

		if fastPrice.Cmp(maxPrice) > 0 {
			return maxPrice
		} else {
			return fastPrice
		}
	}

	if refPrice.Cmp(fastPrice) < 0 {
		return refPrice
	}

	if fastPrice.Cmp(minPrice) < 0 {
		return minPrice
	} else {
		return fastPrice
	}
}

func (pf *FastPriceFeedV1) favorFastPrice() bool {
	if pf.IsSpreadEnabled {
		return false
	}

	if pf.DisableFastPriceVoteCount.Cmp(pf.MinAuthorizations) >= 0 {
		return false
	}

	return true
}
