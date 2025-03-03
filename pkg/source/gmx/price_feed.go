package gmx

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PriceFeed struct {
	RoundID *big.Int            `json:"roundId,omitempty"`
	Answer  *big.Int            `json:"answer,omitempty"`
	Answers map[string]*big.Int `json:"answers,omitempty"`
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

func (pf *PriceFeed) LatestRound() *big.Int {
	return pf.RoundID
}

func (pf *PriceFeed) LatestAnswer() *big.Int {
	return pf.Answer
}

// GetRoundData returns roundId, answer, startedAt, updatedAt, answeredInRound
func (pf *PriceFeed) GetRoundData(roundID *big.Int) (*big.Int, *big.Int, *big.Int, *big.Int, *big.Int) {
	return roundID, pf.Answers[roundID.String()], bignumber.ZeroBI, bignumber.ZeroBI, bignumber.ZeroBI
}
