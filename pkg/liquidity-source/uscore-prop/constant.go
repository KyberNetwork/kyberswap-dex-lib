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
	maxRefinementRounds = 3
	maxSamplePoints     = 128
	// samplerGas covers USCoreSampler (~3.5M on the live pools); HyperEVM nodes otherwise may
	// default an eth_call to a small block's gas limit.
	samplerGas = 30_000_000
)

// quoteExactIn/quoteLadder status codes; any other code up to maxStatus means no quote.
const (
	statusOK           = 0
	statusInsufficient = 3 // the pool can't pay out this amount
	maxStatus          = 5
)

var (
	ErrInvalidState            = errors.New("uscore-prop: invalid pool state")
	ErrUnexpectedSamplerRevert = errors.New("uscore-prop: unexpected sampler revert")
)
