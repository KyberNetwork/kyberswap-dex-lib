package carbon

import (
	"errors"
	"time"

	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	DexType = valueobject.ExchangeCarbon

	orderIdxToken0 = 0
	orderIdxToken1 = 1

	one           = 1 << 48
	ppmResolution = 1000000

	defaultTradingFeePpm = 4000

	defaultSingleTradeActionGas   = 45669
	defaultTradeBySourceAmountGas = 63037

	maxStrategiesPerBatch = 200

	// dust filter: per side (token0-order / token1-order), an order is only tracked
	// if it clears both a liquidity floor (relative to the largest order on that side)
	// and a rate floor (relative to the best-priced order on that side).
	dustLiquidityPct = 1
	dustRatePct      = 10

	// full re-scan of every on-chain strategy happens at most this often; between
	// scans only already-tracked strategies are refreshed plus any newly created ones.
	fullScanInterval = time.Minute
)

var (
	uOne           = uint256.NewInt(one)
	oneSquared     = new(uint256.Int).Mul(uOne, uOne)
	uPmmResolution = uint256.NewInt(ppmResolution)

	uHundred          = uint256.NewInt(100)
	uDustLiquidityPct = uint256.NewInt(dustLiquidityPct)
	uDustRatePct      = uint256.NewInt(dustRatePct)

	ErrInvalidToken          = errors.New("invalid token")
	ErrInvalidSwap           = errors.New("invalid swap")
	ErrInsufficientLiquidity = errors.New("insufficient liquidity")
	ErrNoStrategies          = errors.New("no strategies available")
	ErrZeroAmount            = errors.New("zero amount")
)
