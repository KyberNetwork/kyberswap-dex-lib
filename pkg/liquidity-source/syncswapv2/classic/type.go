package syncswapv2classic

import (
	"github.com/holiman/uint256"
)

type ExtraClassicPool struct {
	SwapFee0To1  *uint256.Int `json:"swapFee0To1"`
	SwapFee1To0  *uint256.Int `json:"swapFee1To0"`
	VaultAddress string       `json:"vaultAddress"`
}
