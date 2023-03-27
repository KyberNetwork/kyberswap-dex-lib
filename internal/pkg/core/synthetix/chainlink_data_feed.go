package synthetix

import (
	"math/big"
)

type ChainlinkDataFeed struct {
	RoundID         *big.Int              `json:"roundId"`
	Answer          *big.Int              `json:"answer"`
	StartedAt       *big.Int              `json:"startedAt"`
	UpdatedAt       *big.Int              `json:"updatedAt"`
	AnsweredInRound *big.Int              `json:"answeredInRound"`
	Answers         map[string]*RoundData `json:"answers"`
}

type RoundData struct {
	RoundID         *big.Int `json:"roundId"`
	Answer          *big.Int `json:"answer"`
	StartedAt       *big.Int `json:"startedAt"`
	UpdatedAt       *big.Int `json:"updatedAt"`
	AnsweredInRound *big.Int `json:"answeredInRound"`
}

func (df *ChainlinkDataFeed) LatestRound() *big.Int {
	return df.RoundID
}

func (df *ChainlinkDataFeed) LatestAnswer() *big.Int {
	return df.Answer
}

func (df *ChainlinkDataFeed) LatestUpdatedAt() *big.Int {
	return df.UpdatedAt
}

// GetRoundData returns roundId, answer, startedAt, updatedAt, answeredInRound
func (df *ChainlinkDataFeed) GetRoundData(roundID *big.Int) (*big.Int, *big.Int, *big.Int, *big.Int, *big.Int) {
	roundIDStr := roundID.String()

	return roundID,
		df.Answers[roundIDStr].Answer,
		df.Answers[roundIDStr].StartedAt,
		df.Answers[roundIDStr].UpdatedAt,
		df.Answers[roundIDStr].AnsweredInRound
}
