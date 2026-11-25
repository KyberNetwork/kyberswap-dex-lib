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

	// The router fixture measured 38,774 extra gas for a second write and 3,425 for a skipped level.
	extraWrittenLevelGas = 39_000
	skippedLevelGas      = 3_500
	// The router_partner_sell_cold_with_fee snapshot skips level 0 and writes level -1.
	singleLevelGas = 258_833 - skippedLevelGas
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
