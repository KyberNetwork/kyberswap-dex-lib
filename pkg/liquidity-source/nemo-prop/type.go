package nemoprop

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ladder"
)

// StaticExtra is stored once per pool. Address is the NemoSwap proxy every
// market of this pool is settled through. The market itself is the pool's
// token pair: [base, market asset].
type StaticExtra struct {
	Address string `json:"a"`
}

// Extra carries the feed settings simulators need to connect on their own.
// Prices never live in pool state: every quote reads the live feed.
type Extra struct {
	Feed FeedSettings `json:"f"`
}

// FeedSettings travel in pool state so any process building a simulator
// can open (or re-authenticate) the shared feed connection. Comparable on
// purpose: simulators compare it by value on every quote.
type FeedSettings struct {
	URL       string `json:"url,omitempty"`
	AuthToken string `json:"auth,omitempty"`
	ChainID   int64  `json:"chainId,omitempty"`
	FreshMs   int64  `json:"freshMs,omitempty"`
	MaxAgeMs  int64  `json:"maxAgeMs,omitempty"`
	Buffer    int64  `json:"buffer,omitempty"`
	DecayBps  int64  `json:"decayBps,omitempty"`
}

// SwapInfo carries the exact snapshot state a quote was priced on, so
// UpdateBalance applies the swap there even if a newer snapshot has arrived.
// Its fields stay out of JSON so route responses and logs carry nothing
// about feed cadence or staleness.
type SwapInfo struct {
	Seq uint64 `json:"-"`
	// Decay is DecayBps times milliseconds past the fresh window when quoted.
	Decay uint64 `json:"-"`

	state *ladder.PoolSimulator
	// ladderOut is the undecayed ladder output, which the ladder's
	// consumption tracking must see.
	ladderOut *big.Int
}
