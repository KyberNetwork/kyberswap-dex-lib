package xsolvbtc

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type PoolMeta struct {
	BlockNumber uint64 `json:"blockNumber"`
}

type PoolExtra struct {
	DepositAllowed  bool
	WithdrawFeeRate uint64
	MaxMultiplier   *big.Int
	Oracle          common.Address
	Nav             *big.Int
}

type Gas struct {
	Deposit  int64
	Withdraw int64
}

type SwapInfo struct {
	IsDeposit bool `json:"isDeposit"`
}
