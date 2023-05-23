package gmx

import "math/big"

type PriceFeed struct {
	RoundID *big.Int            `json:"roundId"`
	Answer  *big.Int            `json:"answer"`
	Answers map[string]*big.Int `json:"answers"`
}

type RoundData struct {
	RoundId         *big.Int
	Answer          *big.Int
	StartedAt       *big.Int
	UpdatedAt       *big.Int
	AnsweredInRound *big.Int
}

func NewPriceFeed() *PriceFeed {
	return &PriceFeed{
		Answers: make(map[string]*big.Int),
	}
}

const (
	priceFeedMethodLatestRoundData = "latestRoundData"
	priceFeedMethodGetRoundData    = "getRoundData"
	minRoundCount                  = 2
)
