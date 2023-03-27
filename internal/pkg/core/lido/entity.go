package lido

import (
	"math/big"
)

type StaticExtra struct {
	LpToken string `json:"lpToken"`
}

type Extra struct {
	StEthPerToken  *big.Int `json:"stEthPerToken"`  // Get amount of stETH for a one wstETH
	TokensPerStEth *big.Int `json:"tokensPerStEth"` // Get amount of wstETH for a one stETH
}

type Gas struct {
	Wrap   int64
	Unwrap int64
}
