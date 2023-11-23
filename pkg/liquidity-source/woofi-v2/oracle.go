package woofiv2

import (
	"math/big"

	"github.com/holiman/uint256"
)

type Oracle struct {
	LatestRoundData RoundData `json:"latestRoundData"`
}

type RoundData struct {
	RoundID         *uint256.Int `json:"roundId"`
	Answer          *big.Int     `json:"answer"`
	StartedAt       *uint256.Int `json:"startedAt"`
	UpdatedAt       *uint256.Int `json:"updatedAt"`
	AnsweredInRound *uint256.Int `json:"answeredInRound"`
}

func (o *Oracle) GetLatestRoundData() (*uint256.Int, *big.Int, *uint256.Int, *uint256.Int, *uint256.Int) {
	return o.LatestRoundData.RoundID, o.LatestRoundData.Answer, o.LatestRoundData.StartedAt, o.LatestRoundData.UpdatedAt, o.LatestRoundData.AnsweredInRound
}
