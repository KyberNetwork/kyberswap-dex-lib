package uniswapv4

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

const DexType = "uniswap-v4"
const EMPTY_ADDRESS = "0x0000000000000000000000000000000000000000"

const (
	graphSkipLimit  = 5000
	graphFirstLimit = 1000
)

var (
	// NativeTokenPlaceholderAddress is the address that UniswapV4 uses to represent native token in pools.
	NativeTokenPlaceholderAddress   = common.Address{}
	Q96                             = big.NewInt(1).Lsh(big.NewInt(1), 96)
	ErrCannotCalcAmountOutDueToHook = errors.New("cannot calculate amount out due to hook")
)
