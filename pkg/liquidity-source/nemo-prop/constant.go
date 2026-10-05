package nemoprop

import (
	"errors"
	"time"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	DexType = valueobject.ExchangeNemoProp

	// defaultGas is NemoSwap.swapPrepaid, the entrypoint Kyber's executor
	// calls after transferring the input to the proxy. Measured at 50k-62k
	// on Base (vault paying itself), plus up to 20k for a taker's first
	// balance of the output token.
	defaultGas = 80_000

	// defaultFeedFresh is how long a snapshot quotes undecayed. The feed
	// server resends faster than this even when nothing changed, so an older
	// snapshot means the feed is unhealthy and its quotes start decaying.
	defaultFeedFresh = time.Second

	// defaultFeedMaxAge bounds how long after receipt a snapshot quotes at
	// all, decayed or not, so a feed gone quiet can't keep quoting stale
	// prices and stale inventory. It leaves room for a reconnect.
	defaultFeedMaxAge = 5 * time.Second

	// decayScale is 10000 bps * 1000 ms: decay is DecayBps per second of
	// staleness, measured in milliseconds.
	decayScale = 10_000 * 1_000

	// marketEntrySize is one getMarkets_v1 entry: [asset: 160 bits][unit: 96 bits].
	marketEntrySize = 32
)

var (
	// ErrNoFeed is returned when there is no quotable feed snapshot
	// covering the pool's market. Nemo is quoted from its live feed only.
	ErrNoFeed = errors.New("no quotable nemo feed snapshot")

	ErrInvalidMarkets = errors.New("invalid getMarkets_v1 encoding")
)
