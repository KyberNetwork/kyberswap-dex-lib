package types

import (
	"math/big"

	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

var (
	ZeroEncodingSwap = EncodingSwap{}

	// ZeroCollectAmount executor doesn't have to transfer token to the pool when executing the swap.
	// amount has already been transferred by executor (if it's the first swap of a path) or previous pool
	ZeroCollectAmount = big.NewInt(0)
)

type EncodingSwap struct {
	Pool              string
	TokenIn           string
	TokenOut          string
	SwapAmount        *big.Int
	AmountOut         *big.Int
	LimitReturnAmount *big.Int
	Exchange          valueobject.Exchange
	PoolLength        int
	PoolType          string
	PoolExtra         interface{}
	Extra             interface{}

	// Flags indicates behavior for Executor contract with Gas optimization feature.
	// Reference: https://www.notion.so/kybernetwork/SC-KS-DEX-Aggregator-Changelog-v3-0-0-28fa34a3736c4b1e943fbd62f5ddb277
	Flags []EncodingSwapFlag

	// CollectAmount there are two possible values:
	// - ZeroCollectAmount: executor doesn't have to transfer token to the pool when executing the swap.
	// amount has already been transferred by executor (if it's the first swap of a path) or previous pool
	// - NonZeroCollectAmount: executor will re-calculate swap amount and transfer this swap amount to the pool
	CollectAmount *big.Int

	// Recipient address of wallet or contract to be received token out after swapped
	// There are three types of recipients:
	// - next pool address
	// - aggregation executor contract address
	// - user wallet (to) address
	Recipient string
}

func (s EncodingSwap) IsZero() bool {
	return len(s.Pool) == 0
}
