package valantisstex

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

type SwapFeeModuleData struct {
	FeeInBips       *big.Int
	InternalContext []byte
}

type Extra struct {
	WithdrawalModule  common.Address `json:"withdrawalModule"`
	SwapFeeInBipsZtoO *uint256.Int   `json:"swapFeeInBipsZtoO"`
	SwapFeeInBipsOtoZ *uint256.Int   `json:"swapFeeInBipsOtoZ"`
	Rate0To1          *uint256.Int   `json:"r01"`
	Rate1To0          *uint256.Int   `json:"r10"`
	Gas               [2]uint64      `json:"gas"`
}

type StaticExtra struct {
	SwapFeeModule      common.Address `json:"swapFeeModule"`
	DefaultSwapFeeBips *uint256.Int   `json:"defaultSwapFeeBips"`
	StexAMM            common.Address `json:"stexAMM"`
}

type MetaInfo struct {
	BlockNumber uint64 `json:"bN"`
	IsZeroToOne bool   `json:"isZeroToOne"`
}
