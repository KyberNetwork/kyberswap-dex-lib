package shared

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type QuoterConfig struct {
	QuoterAddress string
}

type PoolKey struct {
	Currency0   common.Address `abi:"currency0"`
	Currency1   common.Address `abi:"currency1"`
	Hooks       common.Address `abi:"hooks"`
	PoolManager common.Address `abi:"poolManager"`
	Fee         *big.Int       `abi:"fee"`
	Parameters  common.Hash    `abi:"parameters"`
}

type QuoteExactSingleParams struct {
	PoolKey     PoolKey  `abi:"poolKey"`
	ZeroForOne  bool     `abi:"zeroForOne"`
	ExactAmount *big.Int `abi:"exactAmount"`
	HookData    []byte   `abi:"hookData"`
}

type QuoteResult struct {
	AmountOut   *big.Int `abi:"amountOut"`
	GasEstimate *big.Int `abi:"gasEstimate"`
}
