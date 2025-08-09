package unibtc

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type PoolMeta struct {
	BlockNumber uint64 `json:"blockNumber"`
}

type PoolExtra struct {
	Paused           bool           `json:"paused"`
	TokensPaused     []bool         `json:"tokensPaused"`
	TokensAllowed    []bool         `json:"tokensAllowed"`
	Caps             []*big.Int     `json:"caps"`
	ExchangeRateBase *big.Int       `json:"exchangeRateBase"`
	SupplyFeeder     common.Address `json:"supplyFeeder"`
	TokenUsedCaps    []*big.Int     `json:"tokenUsedCaps"`
}

type Gas struct {
	Mint int64
}
