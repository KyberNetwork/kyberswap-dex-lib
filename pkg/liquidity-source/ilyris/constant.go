// Package ilyris integrates Ilyris BinPool, a discrete-bin AMM on Robinhood Chain.
package ilyris

import (
	"errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	DexType = valueobject.ExchangeIlyris

	// BinPoolLens.estimateSwapGas: base + per extra bin crossed.
	baseSwapGas    = 130_000
	perExtraBinGas = 41_500

	minBinID = -500_000
	maxBinID = 500_000
	minWord  = minBinID >> 8
	maxWord  = maxBinID >> 8

	feePrecision   = 1_000_000_000 // FEE_PRECISION
	maxFeeRate     = 100_000_000   // MAX_FEE_RATE
	bps            = 10_000
	volatilityUnit = 10_000 // VOLATILITY_UNIT: one bin of movement

	defaultNewPoolLimit = 100

	// Bitmap walk: words (256 bins each) per side per round, and the round cap.
	scanWords     = 8
	maxScanRounds = 16
	// getBinReserves ids per call; all calls share one multicall.
	binReservesChunk = 512
)

var (
	ErrInvalidToken          = errors.New("ilyris: invalid token")
	ErrInsufficientLiquidity = errors.New("ilyris: insufficient liquidity")
	ErrOverflow              = errors.New("ilyris: overflow")
	ErrSwapsPaused           = errors.New("ilyris: swaps paused by market guard")
	ErrCorporateActionFreeze = errors.New("ilyris: corporate-action freeze")
	ErrMalformedExtra        = errors.New("ilyris: malformed pool extra")
	ErrEmptyBook             = errors.New("ilyris: pool has no populated bins")
	ErrFeeMismatch           = errors.New("ilyris: derived fee rate differs from getTotalFeeRate")
)
