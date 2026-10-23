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

	// Keeps quote*10^decimals far from 2^256 in the walk.
	maxQuoteDecimals = 30

	// KuruV2Helper executeKuruV2 (deposit, swap, withdraw), forge fork test on Monad block 115874578,
	// warm account. gasLevel = int average of (gas - avg 1-level gas) / (levels - 1) over 16
	// multi-level swaps; gasBase = avg 1-level gas (166017, 26 swaps) - gasLevel.
	gasBase  = 100679
	gasLevel = 65338
)

var feeDenominator = uint256.NewInt(10_000_000) // takerFeePps is in PPS

var (
	ErrInvalidToken          = errors.New("invalid token")
	ErrInvalidStaticExtra    = errors.New("invalid static extra")
	ErrZeroAmountOut         = errors.Join(pool.ErrSwapAmountBelowMin, errors.New("zero amount out"))
	ErrInsufficientLiquidity = errors.New("insufficient liquidity")
	ErrOverflow              = errors.New("overflow")
)
