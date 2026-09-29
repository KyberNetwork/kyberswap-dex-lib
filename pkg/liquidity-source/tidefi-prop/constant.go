package tidefiprop

import (
	"errors"
	"time"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	DexType = valueobject.ExchangeTideFiProp

	defaultGas = 150_000

	// MaxAge gates how long a sampled ladder stays routable: TideFi's own
	// quote() prices off a per-token config with an on-chain expiry
	// (observed ~60s validity window), so a snapshot that's gone stale
	// shouldn't stay routable indefinitely -- same reasoning as
	// manta-prop/fermi-prop's freshness gate. Kept well under the observed
	// 60s on-chain expiry.
	MaxAge = 15 * time.Second

	// swapFeePpm is TideFi's per-caller fee tier (parts per million of
	// FEE_DENOMINATOR=1e6 on-chain), checked server-side against
	// msg.sender -- any other value reverts. Confirmed on-chain via quote()
	// from KyberSwap's executor address: only 0 validates until TideFi
	// assigns a nonzero tier.
	swapFeePpm = 0

	// takerAPITimeout bounds the whole discovery connect+read: the Taker
	// API pushes tidefi_markets immediately on connect, no request needed.
	takerAPITimeout = 10 * time.Second
)

var ErrInsufficientLiquidity = errors.New("insufficient liquidity")
