package uscoreprop

import (
	"errors"
	"time"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	DexType = valueobject.ExchangeUSCoreProp
	// defaultGas is the measured executeGeneric cost (82-91k on HyperEVM), excluding the one-time approve.
	defaultGas = 90_000
	// MaxAge rejects ladders older than ~10 HyperEVM blocks (1s each): the pool prices off a live
	// oracle. pool-service must refresh well inside it (2s interval).
	MaxAge              = 10 * time.Second
	multicall3          = "0xcA11bde05977b3631167028862bE2a173976CA11"
	maxRefinementRounds = 3
	maxSamplePoints     = 96
)

// quoteExactIn/quoteLadder status codes; any other code up to maxStatus means no quote.
const (
	statusOK           = 0
	statusInsufficient = 3 // the pool can't pay out this amount
	maxStatus          = 5
)

var ErrInvalidState = errors.New("uscore-prop: invalid pool state")
