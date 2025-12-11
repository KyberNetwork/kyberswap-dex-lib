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

type AMMState struct {
	SqrtSpotPriceX96 *uint256.Int `json:"sqrtSpotPriceX96"`
	SqrtPriceLowX96  *uint256.Int `json:"sqrtPriceLowX96"`
	SqrtPriceHighX96 *uint256.Int `json:"sqrtPriceHighX96"`
}

type Extra struct {
	EffectiveAMMLiquidity *uint256.Int `json:"effectiveAMMLiquidity"`
	AMMState              AMMState     `json:"ammState"`
	SwapFeeInBipsZtoO     *uint256.Int `json:"swapFeeInBipsZtoO"`
	SwapFeeInBipsOtoZ     *uint256.Int `json:"swapFeeInBipsOtoZ"`
}

type StaticExtra struct {
	SwapFeeModule      common.Address `json:"swapFeeModule"`
	DefaultSwapFeeBips *uint256.Int   `json:"defaultSwapFeeBips"`
}

type MetaInfo struct {
	BlockNumber uint64 `json:"bN"`
}
