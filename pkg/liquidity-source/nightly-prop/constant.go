package nightlyprop

import (
	"errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	DexType = valueobject.ExchangeNightlyProp

	routerMethodGetRoutes      = "getRoutes"
	routerMethodGetPoolStateV1 = "getPoolStateV1"
	poolMethodGetTokens        = "getTokens"

	// Estimated gas for a swap within one price level.
	defaultGas = 258_833
)

var (
	ErrInvalidRouterAddress = errors.New("nightly-prop: invalid router address")
	ErrInvalidKyberExecutor = errors.New("nightly-prop: invalid Kyber executor address")
	ErrInvalidPoolRecord    = errors.New("nightly-prop: invalid pool record")
	ErrInvalidToken         = errors.New("nightly-prop: unknown or identical token")
	ErrInvalidAmount        = errors.New("nightly-prop: invalid amount")
	ErrNoRoute              = errors.New("nightly-prop: the router does not route the pair through the pool")
	ErrPaused               = errors.New("nightly-prop: the router or the pool is paused")
	ErrMissingRole          = errors.New("nightly-prop: the pool does not let the router swap, or the oracle does not let the pool read prices")
	ErrPartnerNotEnabled    = errors.New("nightly-prop: the partner has no PARTNER_ROLE or no fee recipient on the router")
	ErrPriceUnavailable     = errors.New("nightly-prop: the pool has no oracle price")
)
