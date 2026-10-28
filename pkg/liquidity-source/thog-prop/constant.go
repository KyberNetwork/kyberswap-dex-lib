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
	// Estimates include call overhead beyond the model-3 harness measurements.
	// Same-category swaps skip covariance-risk work; see README.md for provenance.
	sameCategoryGas  = 350000
	crossCategoryGas = 550000
)

var (
	ErrInvalidState = errors.New("thog-prop: invalid schema-8/model-3 snapshot")
	ErrInvalidToken = errors.New("thog-prop: unknown or identical token")
	ErrUnavailable  = errors.New("thog-prop: quote unavailable")
	ErrOverflow     = errors.New("thog-prop: arithmetic overflow or division by zero")
	ErrBalance      = errors.New("thog-prop: inventory capacity exceeded")
)
