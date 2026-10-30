package thogprop

import (
	"errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	DexType           = valueobject.ExchangeThogProp
	methodGetPoolData = "getPoolData"
	maxTokens         = 28
	// Quote one block after the snapshot: 0.50 bp extra spread at the live policy.
	executionAgeLookahead uint64 = 1
	wmon                         = "0x3bd359c1119da7da1d913d1c4d2b7c461115433a"
	// Integer averages of Tenderly swapWithExtraFees frame gas through the
	// router on Monad (3 same-category, 14 cross-category swaps); see README.md.
	sameCategoryGas  = 273846
	crossCategoryGas = 564978
)

var (
	ErrInvalidState = errors.New("thog-prop: invalid schema-8/model-3 snapshot")
	ErrInvalidToken = errors.New("thog-prop: unknown or identical token")
	ErrUnavailable  = errors.New("thog-prop: quote unavailable")
	ErrOverflow     = errors.New("thog-prop: arithmetic overflow or division by zero")
	ErrBalance      = errors.New("thog-prop: inventory capacity exceeded")
)
