package thogprop

import (
	"errors"
	"time"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	DexType           = valueobject.ExchangeThogProp
	methodGetPoolData = "getPoolData"
	maxTokens         = 28
	// Multicall3 at its canonical address (deployed on Monad); the lens fallback reads through it.
	multicall3                       = "0xcA11bde05977b3631167028862bE2a173976CA11"
	selectorGetCurrentBlockTimestamp = "0x0f28c97d"
	// Quote one block after the snapshot: 0.50 bp extra spread at the live policy.
	executionAgeLookahead uint64 = 1
	wmon                         = "0x3bd359c1119da7da1d913d1c4d2b7c461115433a"
	// Integer averages of Tenderly swapWithExtraFees frame gas through the
	// router on Monad (3 same-category, 14 cross-category swaps); see README.md.
	sameCategoryGas                = 273846
	crossCategoryGas               = 564978
	defaultBlockTimeMs      int64  = 300
	defaultMaxSnapshotAgeMs int64  = 15_000
	clockSkewAllowanceMs    int64  = 1_000
	defaultRequestTimeoutMs int64  = 2_000
	refreshRetryDelay              = 300 * time.Millisecond
	snapshotLensGas         uint64 = 3_000_000
)

var (
	ErrInvalidState  = errors.New("thog-prop: invalid schema-8/model-3 snapshot")
	ErrInvalidToken  = errors.New("thog-prop: unknown or identical token")
	ErrUnavailable   = errors.New("thog-prop: quote unavailable")
	ErrOverflow      = errors.New("thog-prop: arithmetic overflow or division by zero")
	ErrBalance       = errors.New("thog-prop: inventory capacity exceeded")
	ErrStaleSnapshot = errors.New("thog-prop: stale snapshot")
)
