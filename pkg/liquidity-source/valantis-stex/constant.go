package valantisstex

import (
	"errors"

	"github.com/holiman/uint256"

	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	DexType = valueobject.ExchangeValantisStex

	defaultGas = 116059

	resolution = 96
)

var (
	maxSwapFeeBips = u256.UBasisPoint

	Q96 = uint256.MustFromHex("0x1000000000000000000000000")
)

var (
	ErrInvalidToken                            = errors.New("invalid token")
	ErrZeroSwap                                = errors.New("zero swap")
	ErrSovereignPoolSwapExcessiveSwapFee       = errors.New("swap excessive swap fee")
	ErrInvalidSpotPriceAfterSwap               = errors.New("invalid spot price after swap")
	ErrSqrtPX96MustGtZero                      = errors.New("sqrtPX96 must be greater than zero")
	ErrLiquidityMustGtZero                     = errors.New("liquidity must be greater than zero")
	ErrGetNextSqrtPriceFromAmount0RoundingUp   = errors.New("getNextSqrtPriceFromAmount0RoundingUp: overflow or underflow")
	ErrGetNextSqrtPriceFromAmount1RoundingDown = errors.New("getNextSqrtPriceFromAmount1RoundingDown: sqrtPX96 <= quotient")
	ErrInsufficientReserve                     = errors.New("insufficient reserve")
	ErrAmountInFilledGtAmountInWithoutFee      = errors.New("amountIn filled greater than amountIn without fee")
)
