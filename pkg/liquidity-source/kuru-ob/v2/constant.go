package kuruobv2

import (
	"errors"

	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

const (
	DexType = "kuru-ob-v2"

	bookDepth       = 50 // price levels read per side
	marketStateLive = 0
	marketBatchSize = 100

	// ponytail: V1 gas until a V2 executor helper (deposit+swap+withdraw) exists to measure.
	gasBase  = 221703
	gasLevel = 84155
)

var feeDenominator = uint256.NewInt(10_000_000) // takerFeePps is in PPS

var (
	ErrInvalidToken          = errors.New("invalid token")
	ErrInvalidStaticExtra    = errors.New("invalid static extra")
	ErrZeroAmountOut         = errors.Join(pool.ErrSwapAmountBelowMin, errors.New("zero amount out"))
	ErrInsufficientLiquidity = errors.New("insufficient liquidity")
	ErrOverflow              = errors.New("overflow")
)
