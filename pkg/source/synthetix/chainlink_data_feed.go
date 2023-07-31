package synthetix

import (
	"math/big"
)

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
