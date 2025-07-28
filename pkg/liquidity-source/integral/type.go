package integral

import (
	"math/big"

	"github.com/holiman/uint256"
)

type Gas struct {
	Swap int64
}

type Extra struct {
	RelayerAddress string `json:"relayerAddress"`

	IsEnabled bool `json:"isEnabled"`

	Price         *uint256.Int `json:"price"`
	InvertedPrice *uint256.Int `json:"invertedPrice"`
	SwapFee       *uint256.Int `json:"swapFee"`

	Token0LimitMin *uint256.Int `json:"t0LiMi"`
	Token0LimitMax *uint256.Int `json:"t0LiMa"`
	Token1LimitMin *uint256.Int `json:"t1LiMi"`
	Token1LimitMax *uint256.Int `json:"t1LiMa"`

	Token0LimitMaxMultiplier *uint256.Int `json:"t0LiMaMu"`
	Token1LimitMaxMultiplier *uint256.Int `json:"t1LiMaMu"`
}

type SwapInfo struct {
	RelayerAddress string   `json:"relayerAddress"`
	NewReserve0    *big.Int `json:"-"`
	NewReserve1    *big.Int `json:"-"`
}

type MetaInfo struct {
	ApprovalAddress string `json:"approvalAddress,omitempty"`
}

type PoolState struct {
	Price     *big.Int `json:"price"`
	Fee       *big.Int `json:"fee"`
	LimitMin0 *big.Int `json:"limitMin0"`
	LimitMax0 *big.Int `json:"limitMax0"`
	LimitMin1 *big.Int `json:"limitMin1"`
	LimitMax1 *big.Int `json:"limitMax1"`
}

type PriceByPair struct {
	XDecimals uint8
	YDecimals uint8
	Price     *big.Int
}
