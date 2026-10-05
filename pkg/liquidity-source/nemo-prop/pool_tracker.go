package nemoprop

import (
	"context"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ladder"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

// PoolTracker does no RPC. Its job is to put the feed settings into every
// pool's Extra, which is how router processes learn to connect, and to
// mirror the feed's deliverable inventory into Reserves for pool-service,
// zeroed once the feed stops quoting the market. Prices are never stored:
// simulators read them from the live feed.
type PoolTracker struct {
	cfg *Config
}

var _ = pooltrack.RegisterFactoryCE0(DexType, NewPoolTracker)

func NewPoolTracker(cfg *Config, _ *ethrpc.Client) *PoolTracker {
	return &PoolTracker{cfg: cfg}
}

func (t *PoolTracker) GetNewPoolState(
	_ context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return p, err
	}
	if len(p.Tokens) != 2 {
		return p, ladder.ErrInvalidToken
	}

	now := time.Now()
	settings := t.feedSettings()
	extraBytes, err := json.Marshal(Extra{Feed: settings})
	if err != nil {
		return p, err
	}
	p.Extra = string(extraBytes)
	p.Timestamp = now.Unix()

	if key := feedKey(settings, staticExtra.Address); key != "" {
		snap, _ := acquireFeed(key, settings, staticExtra.Address, now.Unix()).quotable(now)
		if snap != nil {
			// A live snapshot without this market (e.g. dropped by a pricing
			// upgrade) means nothing is deliverable through this pool.
			p.Reserves = entity.PoolReserves{"0", "0"}
			if state := snap.market(p.Tokens[0].Address, p.Tokens[1].Address); state != nil {
				p.Reserves = entity.PoolReserves{state.Reserve0.Dec(), state.Reserve1.Dec()}
			}
			if snap.blockNumber != 0 {
				p.BlockNumber = snap.blockNumber
			}
		}
	}
	return p, nil
}

func (t *PoolTracker) feedSettings() FeedSettings {
	if t.cfg.Feed.URL == "" {
		return FeedSettings{}
	}
	return FeedSettings{
		URL:       t.cfg.Feed.URL,
		AuthToken: t.cfg.Feed.AuthToken,
		ChainID:   int64(t.cfg.ChainID),
		FreshMs:   t.cfg.Feed.FreshMs,
		MaxAgeMs:  t.cfg.Feed.MaxAgeMs,
		Buffer:    t.cfg.Buffer,
		DecayBps:  t.cfg.DecayBps,
	}
}
