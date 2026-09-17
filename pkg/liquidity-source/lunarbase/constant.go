package lunarbase

import (
	"github.com/pkg/errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

const (
	DexType = "lunarbase"

	defaultGas = 120000

	// fQ24 is 2^24 — used to render Q24 directional fees as fractional
	// `SwapFee` on the entity.
	fQ24 = 1 << 24
)

var (
	ErrSnapshotBehind = errors.New("RPC snapshot is older than the observed block")

	ErrStalePool             = errors.WithMessage(pool.ErrUnsupported, "stale pool")
	ErrInvalidToken          = errors.New("invalid token")
	ErrPoolPaused            = errors.New("pool is paused")
	ErrZeroPrice             = errors.New("pool price is zero")
	ErrInsufficientLiquidity = errors.New("insufficient liquidity")
	ErrQuoteFailed           = errors.New("quote failed")
)
