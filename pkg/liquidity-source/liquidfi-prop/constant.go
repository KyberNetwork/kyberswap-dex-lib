package liquidfiprop

import (
	"errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	DexType = valueobject.ExchangeLiquidFiProp

	// defaultGas is the integer average of 4 Tenderly-simulated pool frames on the LiquidFi Entry
	// (BSC pairs 1 and 2, Base pair), excluding the one-time approve of the entry.
	defaultGas int64 = 149825

	methodPricing           = "pricing"
	methodVault             = "vault"
	methodGetSupportedPairs = "getSupportedPairs"
	methodPaused            = "paused"
	methodGetPairGlobalCfg  = "getPairGlobalConfig"
	methodGetPairCumulative = "getPairCumulatives"
)

var (
	ErrInvalidState    = errors.New("liquidfi-prop: invalid state")
	ErrInvalidToken    = errors.New("liquidfi-prop: invalid token")
	ErrZeroAmount      = errors.New("liquidfi-prop: zero amount")
	ErrZeroOutput      = errors.New("liquidfi-prop: zero output")
	ErrPaused          = errors.New("liquidfi-prop: paused")
	ErrStalePrice      = errors.New("liquidfi-prop: price expired")
	ErrInvalidMidPrice = errors.New("liquidfi-prop: invalid mid price")
	ErrInvalidSpread   = errors.New("liquidfi-prop: invalid spread")
	ErrInvalidDepth    = errors.New("liquidfi-prop: invalid depth config")
	ErrInvalidGrowth   = errors.New("liquidfi-prop: invalid growth factor")
	ErrExceedsCapacity = errors.New("liquidfi-prop: exceeds epoch capacity")
	ErrOverflow        = errors.New("liquidfi-prop: arithmetic overflow")
	ErrLnUndefined     = errors.New("liquidfi-prop: ln undefined")
	ErrExpOverflow     = errors.New("liquidfi-prop: exp overflow")
	ErrUnsupportedImpl = errors.New("liquidfi-prop: unsupported pricing implementation")
)
