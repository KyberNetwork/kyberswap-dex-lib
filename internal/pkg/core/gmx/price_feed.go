package gmx

import (
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/constant"

	"math/big"
)

// PriceFeed
// https://github.com/gmx-io/gmx-contracts/blob/master/contracts/oracle/PriceFeed.sol
type PriceFeed struct {
	RoundID *big.Int            `json:"roundId"`
	Answer  *big.Int            `json:"answer"`
	Answers map[string]*big.Int `json:"answers"`
}

func (pf *PriceFeed) LatestRound() *big.Int {
	return pf.RoundID
}

func (pf *PriceFeed) LatestAnswer() *big.Int {
	return pf.Answer
}

// GetRoundData returns roundId, answer, startedAt, updatedAt, answeredInRound
func (pf *PriceFeed) GetRoundData(roundID *big.Int) (*big.Int, *big.Int, *big.Int, *big.Int, *big.Int) {
	return roundID, pf.Answers[roundID.String()], constant.Zero, constant.Zero, constant.Zero
}
