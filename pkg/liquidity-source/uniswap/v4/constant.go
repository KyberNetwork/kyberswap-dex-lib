package uniswapv4

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"

	uniswapv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
)

const (
	DexType      = "uniswap-v4"
	EmptyAddress = "0x0000000000000000000000000000000000000000"

	graphFirstLimit = 1000

	maxChangedTicks = 10

	tickChunkSize = 100
)

var (
	// NativeTokenAddress is the address that UniswapV4 uses to represent native token in pools.
	NativeTokenAddress = common.Address{}

	ErrTooManyChangedTicks = errors.New("too many changed ticks")

	ErrInvalidAmountIn  = errors.New("invalid amount in")
	ErrInvalidAmountOut = errors.New("invalid amount out")
	ErrInvalidFee       = errors.New("invalid fee")

	defaultGas = uniswapv3.Gas{BaseGas: 129869, CrossInitTickGas: 15460}
)
