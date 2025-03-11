package uniswapv4

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

const (
	DexType      = "uniswap-v4"
	EmptyAddress = "0x0000000000000000000000000000000000000000"

	graphFirstLimit = 1000

	maxChangedTicks = 10
)

var (
	// NativeTokenAddress is the address that UniswapV4 uses to represent native token in pools.
	NativeTokenAddress = common.Address{}
	Q96                = new(big.Int).Lsh(bignumber.One, 96)
	ErrUnsupportedHook = errors.New("unsupported hook")

	ErrTooManyChangedTickes = errors.New("too many changed ticks")
)
