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
)

var ErrInsufficientLiquidity = errors.New("insufficient liquidity")
