package syntropiaprop

import (
	"errors"
	"time"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	DexType = valueobject.ExchangeSyntropiaProp

	// defaultGas is the pool's swap call on a Base fork with real WETH and USDC, tokenIn pushed
	// first: 92.0k buying WETH and 104.7k selling it, plus 6k when the swap is written against
	// the pair's hourly budget.
	defaultGas = 111_000

	// maxAge bounds quoting a snapshot. The maker replaces each price about once a second and
	// a price expires at most 60s after it was set, so an older snapshot prices a replaced curve.
	maxAge = 15 * time.Second
)

var (
	ErrInvalidConfig = errors.New("syntropia-prop: invalid config")
	ErrInvalidPool   = errors.New("syntropia-prop: pool is not a pair of this venue")
	ErrInvalidBook   = errors.New("syntropia-prop: lens returned an invalid book")
)
