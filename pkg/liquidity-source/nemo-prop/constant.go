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

	// StableToken is NemoSwap's hardcoded USDC (Base): swapPrepaidWithFee takes
	// its fee in it and reverts InvalidStablePair on a pair without it.
	StableToken = "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913"
)

var (
	ErrInvalidMarkets    = errors.New("invalid getMarkets_v1 encoding")
	ErrInvalidStablePair = errors.New("pair has no stable token")
	ErrFeeTooLarge       = errors.New("stable fee consumes the whole stable side")
)
