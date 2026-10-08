package nemoprop

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ladder"
)

// StaticExtra is stored once per pool. Address is the NemoSwap proxy every
// market of this pool is settled through. The market itself is the pool's
// token pair: [base, market asset].
type StaticExtra struct {
	Address string `json:"a"`
}

// Extra is a feed snapshot's shaped ladders for one market, plus what the
// simulator needs to age them: when Nemo priced the snapshot and
// the freshness, decay and maximum-age settings in force.
type Extra struct {
	ladder.Extra
	PricedAtMs int64 `json:"t,omitempty"`
	FreshMs    int64 `json:"fr,omitempty"`
	MaxAgeMs   int64 `json:"ma,omitempty"`
	DecayBps   int64 `json:"d,omitempty"`
}
