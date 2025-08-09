package integral

import (
	"math/big"

	"github.com/holiman/uint256"
)

type Gas struct {
	Swap int64
}

type Extra struct {
	IsEnabled      bool   `json:"isEnabled"`
	RelayerAddress string `json:"relayerAddress"`

	Price         *uint256.Int `json:"price,omitempty"`
	InvertedPrice *uint256.Int `json:"invertedPrice,omitempty"`
	SwapFee       *uint256.Int `json:"swapFee,omitempty"`

	Token0LimitMin *uint256.Int `json:"t0LiMi,omitempty"`
	Token1LimitMin *uint256.Int `json:"t1LiMi,omitempty"`

	Token0LimitMaxMultiplier *uint256.Int `json:"t0LiMaMu,omitempty"`
	Token1LimitMaxMultiplier *uint256.Int `json:"t1LiMaMu,omitempty"`
}

type SwapInfo struct {
	RelayerAddress string `json:"relayerAddress"`
}

type MetaInfo struct {
	ApprovalAddress string `json:"approvalAddress,omitempty"`
}

type PoolState struct {
	Price     *big.Int
	Fee       *big.Int
	LimitMin0 *big.Int
	LimitMax0 *big.Int
	LimitMin1 *big.Int
	LimitMax1 *big.Int
}

type PriceByPair struct {
	XDecimals uint8
	YDecimals uint8
	Price     *big.Int
}
