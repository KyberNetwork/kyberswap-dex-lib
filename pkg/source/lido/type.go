package lido

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

type PoolItem struct {
	ID      string             `json:"id"`
	LpToken string             `json:"lpToken"`
	Tokens  []entity.PoolToken `json:"tokens"`
}

type StaticExtra struct {
	LpToken string `json:"lpToken"`
}

type Extra struct {
	StEthPerToken  *big.Int `json:"stEthPerToken"`  // Get amount of stETH for a one wstETH
	TokensPerStEth *big.Int `json:"tokensPerStEth"` // Get amount of wstETH for a one stETH
}
