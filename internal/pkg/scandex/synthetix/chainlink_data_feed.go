package synthetix

import "math/big"

type ChainlinkDataFeed struct {
	RoundID         *big.Int             `json:"roundId"`
	Answer          *big.Int             `json:"answer"`
	StartedAt       *big.Int             `json:"startedAt"`
	UpdatedAt       *big.Int             `json:"updatedAt"`
	AnsweredInRound *big.Int             `json:"answeredInRound"`
	Answers         map[string]RoundData `json:"answers"`
}

type RoundData struct {
	RoundId         *big.Int
	Answer          *big.Int
	StartedAt       *big.Int
	UpdatedAt       *big.Int
	AnsweredInRound *big.Int
}

func NewChainlinkDataFeed() *ChainlinkDataFeed {
	return &ChainlinkDataFeed{
		Answers: make(map[string]RoundData),
	}
}
