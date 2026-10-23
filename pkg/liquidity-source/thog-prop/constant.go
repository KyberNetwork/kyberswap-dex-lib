package thogprop

import (
	"errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	DexType           = valueobject.ExchangeThogProp
	methodGetPoolData = "getPoolData"
	maxTokens         = 32
	wmon              = "0x3bd359c1119da7da1d913d1c4d2b7c461115433a"
	// Measured against the deployed proxy on a Monad fork; includes ERC-20 transfers.
	defaultGas = 660000
)

var (
	ErrInvalidState = errors.New("thog-prop: invalid schema-7 snapshot")
	ErrInvalidToken = errors.New("thog-prop: unknown or identical token")
	ErrUnavailable  = errors.New("thog-prop: quote unavailable")
	ErrOverflow     = errors.New("thog-prop: arithmetic overflow or division by zero")
	ErrBalance      = errors.New("thog-prop: inventory capacity exceeded")
)
