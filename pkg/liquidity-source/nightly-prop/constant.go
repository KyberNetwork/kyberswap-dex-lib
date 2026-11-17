package nightlyprop

import (
	"errors"
	"time"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	DexType = valueobject.ExchangeNightlyProp

	routerMethodGetRoutes      = "getRoutes"
	routerMethodGetPoolStateV1 = "getPoolStateV1"
	poolMethodGetTokens        = "getTokens"

	// defaultGas is router_partner_sell_cold_with_fee of the router gas snapshot
	// (evm_contracts/snapshots/NightlyRouterGasTest.json in the Nightly trading engine): a
	// swapWithFees whose walk stays in one level.
	defaultGas = 258_811
	// priceAgeMargin is the time from a quote to the block of its swap. The price age check uses
	// the quote's time plus this margin, so the route does not use a price that is stale when the
	// swap lands.
	priceAgeMargin = 2 * time.Second
)

var (
	ErrInvalidRouterAddress = errors.New("nightly-prop: invalid router address")
	ErrInvalidPoolRecord    = errors.New("nightly-prop: invalid pool record")
	ErrInvalidToken         = errors.New("nightly-prop: unknown or identical token")
	ErrInvalidAmount        = errors.New("nightly-prop: invalid amount")
	ErrNoRoute              = errors.New("nightly-prop: the router does not route the pair through the pool")
	ErrPaused               = errors.New("nightly-prop: the router or the pool is paused")
	ErrPriceUnavailable     = errors.New("nightly-prop: the pool has no oracle price")
)
