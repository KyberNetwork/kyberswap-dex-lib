package zkerafinance

import (
	"math/big"
)

var (
	maximizeTrue  = "true"
	maximizeFalse = "false"
)

type PriceFeed struct {
	LatestAnswers map[string]*big.Int `json:"latestAnswers"`
}

func NewPriceFeed() *PriceFeed {
	return &PriceFeed{
		LatestAnswers: make(map[string]*big.Int),
	}
}

const priceFeedMethodLatestAnswer = "latestAnswer"

func (pf *PriceFeed) LatestAnswer(maximize bool) *big.Int {
	if maximize {
		return pf.LatestAnswers[maximizeTrue]
	}
	return pf.LatestAnswers[maximizeFalse]
}
