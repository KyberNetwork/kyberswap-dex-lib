package nemoprop

import (
	"errors"
	"time"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	DexType = valueobject.ExchangeNemoProp

	// defaultGas is the whole hop in Kyber's executor: the input transfer to
	// the proxy plus NemoSwap.swapPrepaid. Tenderly measured 97k-101k on Base.
	defaultGas = 100_000

	// defaultFeedFresh is how long a snapshot quotes undecayed. The feed
	// server resends faster than this even when nothing changed, so an older
	// snapshot means the feed (or the tracker refresh) is behind and its
	// quotes start decaying.
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
	ErrInvalidMarkets = errors.New("invalid getMarkets_v1 encoding")
)
