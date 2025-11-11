package uniswapv4

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	uniswapv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
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
	Q96                = new(big.Int).Lsh(bignumber.One, 96)

	ErrTooManyChangedTicks = errors.New("too many changed ticks")

	defaultGas = uniswapv3.Gas{BaseGas: 129869, CrossInitTickGas: 15460}
)
