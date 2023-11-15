package zkerafinance

import (
	"math/big"
)

type PriceFeed struct {
	LatestAnswers map[bool]*big.Int `json:"latestAnswers"`
}

func NewPriceFeed() *PriceFeed {
	return &PriceFeed{
		LatestAnswers: make(map[bool]*big.Int),
	}
}

const priceFeedMethodLatestAnswer = "latestAnswer"

func (pf *PriceFeed) LatestAnswer(maximize bool) *big.Int {
	return pf.LatestAnswers[maximize]
}
