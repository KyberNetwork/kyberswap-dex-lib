package syncswapv2stable

import (
	"github.com/holiman/uint256"
)

type ExtraStablePool struct {
	SwapFee0To1 *uint256.Int `json:"swapFee0To1"`
	SwapFee1To0 *uint256.Int `json:"swapFee1To0"`

	Token0PrecisionMultiplier *uint256.Int `json:"token0PrecisionMultiplier"`
	Token1PrecisionMultiplier *uint256.Int `json:"token1PrecisionMultiplier"`

	VaultAddress string       `json:"vaultAddress"`
	A            *uint256.Int `json:"a"`
}
