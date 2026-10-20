package pancakeprop

import (
	"errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	DexType = valueobject.ExchangePancakeProp

	// defaultGas is the integer average of 9 Tenderly-simulated executeGeneric frames on BSC (both pairs
	// and directions, flat and curve fills), excluding the one-time approve of the entry.
	defaultGas int64 = 142928

	methodPricing           = "pricing"
	methodVault             = "vault"
	methodGetSupportedPairs = "getSupportedPairs"
	methodPaused            = "paused"
	methodGetPairGlobalCfg  = "getPairGlobalConfig"
	methodGetPairCumulative = "getPairCumulatives"

	// Mapping slots of the pricing implementation (unverified source, found by tracing).
	slotPairPrice = 4 // pairId => packed price word
)

var (
	ErrInvalidState    = errors.New("pancake-prop: invalid state")
	ErrInvalidToken    = errors.New("pancake-prop: invalid token")
	ErrZeroAmount      = errors.New("pancake-prop: zero amount")
	ErrZeroOutput      = errors.New("pancake-prop: zero output")
	ErrPaused          = errors.New("pancake-prop: paused")
	ErrStalePrice      = errors.New("pancake-prop: price expired")
	ErrInvalidMidPrice = errors.New("pancake-prop: invalid mid price")
	ErrInvalidSpread   = errors.New("pancake-prop: invalid spread")
	ErrInvalidDepth    = errors.New("pancake-prop: invalid depth config")
	ErrInvalidGrowth   = errors.New("pancake-prop: invalid growth factor")
	ErrExceedsCapacity = errors.New("pancake-prop: exceeds epoch capacity")
	ErrOverflow        = errors.New("pancake-prop: arithmetic overflow")
	ErrLnUndefined     = errors.New("pancake-prop: ln undefined")
	ErrExpOverflow     = errors.New("pancake-prop: exp overflow")
	ErrUnsupportedImpl = errors.New("pancake-prop: unsupported pricing implementation")
)
