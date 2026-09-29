package curve

import (
	"errors"
	"strings"

	"github.com/holiman/uint256"
)

const (
	DexType = "arcade-fun"

	// Pool addresses are synthetic: every launch trades on the ArcadeHook contract
	// that created it, keyed by its V4 PoolId (which hashes the hook address, so it is
	// unique across hook generations). The prefix keeps them distinct from the
	// uniswap-v4 pool the same PoolId names once the launch graduates.
	poolAddressPrefix = "arcade-fun-"

	// LaunchMode / Status values of ArcadeHook.CurveState.
	modePump        = 0
	statusCurving   = 0
	statusGraduated = 2

	basisPoints    = 10_000
	tradeFeeBps    = 100 // ArcadeV4Curve.TRADE_FEE_BPS
	feeDenominator = basisPoints

	// Gas of ArcadeHook.buy / sell measured against a real v4-core PoolManager in
	// Foundry: first buy 113,548 (143,450 with the anti-sniper tax transfers),
	// graduating buy 376,147 (it also seeds and locks the V4 position), sell 26,159
	// on warm storage. Padded for cold storage on a live chain.
	buyGas           = 200_000
	graduatingBuyGas = 600_000
	sellGas          = 100_000
)

// ArcadeV4Curve constants (USDC 6 decimals, launch token 18 decimals).
var (
	virtualUsdcReserve  = uint256.MustFromDecimal("5500000000")                   // 5_500e6
	virtualTokenReserve = uint256.MustFromDecimal("1094200000000000000000000000") // 1_094_200_000e18
	curveSupply         = uint256.MustFromDecimal("777000000000000000000000000")  // 777_000_000e18
	kConstant           = new(uint256.Int).Mul(virtualUsdcReserve, virtualTokenReserve)

	uTradeFeeBps    = uint256.NewInt(tradeFeeBps)
	uFeeDenominator = uint256.NewInt(feeDenominator)
)

var (
	ErrInvalidToken    = errors.New("arcade-fun: invalid token")
	ErrZeroAmount      = errors.New("arcade-fun: zero amount")
	ErrNotCurving      = errors.New("arcade-fun: launch is not on its bonding curve")
	ErrPaused          = errors.New("arcade-fun: hook is paused")
	ErrZeroOutput      = errors.New("arcade-fun: amount too small, the curve returns nothing")
	ErrSellExceedsSold = errors.New("arcade-fun: selling more tokens than the curve has issued")
	ErrOverflow        = errors.New("arcade-fun: overflow")
	ErrNotTracked      = errors.New("arcade-fun: pool state not tracked yet")
)

// PoolAddress is the synthetic pool address of a launch, from its PoolId.
func PoolAddress(poolID string) string {
	return poolAddressPrefix + strings.ToLower(poolID)
}

// PoolIDFromAddress recovers the PoolId from a synthetic pool address.
func PoolIDFromAddress(address string) string {
	return strings.TrimPrefix(strings.ToLower(address), poolAddressPrefix)
}
