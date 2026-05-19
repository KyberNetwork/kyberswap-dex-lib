package nadswap

import (
	"errors"

	"github.com/holiman/uint256"
)

const (
	DexType = "nadswap"

	BPS       = 10000
	LpFeeRate = 25 // BPS

	buyGas  = 150000
	sellGas = 150000

	factoryMethodAllPairs       = "allPairs"
	factoryMethodAllPairsLength = "allPairsLength"
	factoryMethodFeeCollector   = "feeCollector"

	pairMethodToken0      = "token0"
	pairMethodToken1      = "token1"
	pairMethodGetReserves = "getReserves"

	feeCollectorMethodGetFeeConfig = "getFeeConfig"
)

var (
	ErrInvalidToken          = errors.New("invalid token")
	ErrInsufficientInput     = errors.New("insufficient input amount")
	ErrInsufficientOutput    = errors.New("insufficient output amount")
	ErrInsufficientLiquidity = errors.New("insufficient liquidity")
	ErrInvalidFeeRate        = errors.New("invalid fee rate")
	ErrOverflow              = errors.New("overflow")

	uBPS       = uint256.NewInt(BPS)
	uLpFeeRate = uint256.NewInt(LpFeeRate)
)
