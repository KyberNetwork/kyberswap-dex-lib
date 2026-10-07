package nemoprop

import (
	"context"
	"math/big"
	"sync/atomic"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ladder"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

// PoolTracker holds the process-wide live feed connection and, on every
// refresh, copies the latest snapshot's ladders for the pool's market into
// Extra (truncated at deliverable inventory and buffered) and its
// deliverable inventory into Reserves. Without a quotable snapshot covering
// the market, or with on-chain anchors older than the maximum age, the pool
// gets empty ladders and zero reserves, so it stops quoting.
type PoolTracker struct {
	cfg          *Config
	ethrpcClient *ethrpc.Client
	feed         *feedClient

	pricing atomic.Pointer[common.Address] // proxy.pricing(), re-read every refresh
}

var _ = pooltrack.RegisterFactoryCE0(DexType, NewPoolTracker)

func NewPoolTracker(cfg *Config, ethrpcClient *ethrpc.Client) *PoolTracker {
	return &PoolTracker{cfg: cfg, ethrpcClient: ethrpcClient, feed: acquireFeed(cfg)}
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	if len(p.Tokens) != 2 {
		return p, ladder.ErrInvalidToken
	}

	extra := Extra{FreshMs: t.cfg.freshMs(), MaxAgeMs: t.cfg.maxAgeMs(), DecayBps: t.cfg.DecayBps}
	reserves := [2]*uint256.Int{new(uint256.Int), new(uint256.Int)}
	now := time.Now()
	p.Timestamp = now.Unix()

	if snap := t.feed.latest(); snap != nil && now.Sub(snap.receivedAt).Milliseconds() < extra.MaxAgeMs {
		if m, ok := snap.market(p.Tokens[0].Address, p.Tokens[1].Address); ok {
			anchorsMs, err := t.anchorsUpdatedAtMs(ctx)
			if err != nil {
				return p, err
			}
			// Swaps execute against the on-chain anchors, not the feed. Past the
			// maximum age they have decayed away from any feed price: stop quoting.
			if now.UnixMilli()-anchorsMs < extra.MaxAgeMs {
				reserves = m.reserves
				extra.Ladders = shapeLadders(m.ladders, reserves, t.cfg.Buffer)
				extra.ReceivedAtMs = snap.receivedAt.UnixMilli()
				p.Timestamp = snap.receivedAt.Unix()
			}
		}
		if snap.blockNumber != 0 {
			p.BlockNumber = snap.blockNumber
		}
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}
	p.Extra = string(extraBytes)
	p.Reserves = ladder.ZeroUnquotedReserves(entity.PoolReserves{reserves[0].Dec(), reserves[1].Dec()},
		extra.Ladders)
	return p, nil
}

// anchorsUpdatedAtMs reads when Nemo last pushed its on-chain anchors. It
// re-reads proxy.pricing() in the same aggregate, so a pricing upgrade is
// picked up; that refresh reports 0 (stale) and the next reads the new one.
func (t *PoolTracker) anchorsUpdatedAtMs(ctx context.Context) (int64, error) {
	pricing := t.pricing.Load()
	if pricing == nil {
		var addr common.Address
		if _, err := t.ethrpcClient.NewRequest().SetContext(ctx).AddCall(&ethrpc.Call{
			ABI: nemoSwapABI, Target: t.cfg.Address, Method: "pricing",
		}, []any{&addr}).Call(); err != nil {
			return 0, err
		}
		pricing = &addr
		t.pricing.Store(pricing)
	}

	var current common.Address
	var updatedMs *big.Int
	if _, err := t.ethrpcClient.NewRequest().SetContext(ctx).AddCall(&ethrpc.Call{
		ABI: nemoSwapABI, Target: t.cfg.Address, Method: "pricing",
	}, []any{&current}).AddCall(&ethrpc.Call{
		ABI: nemoPricingABI, Target: hexutil.Encode(pricing[:]), Method: "lastUpdatedTimestampMs",
	}, []any{&updatedMs}).Aggregate(); err != nil {
		return 0, err
	}
	if current != *pricing {
		t.pricing.Store(&current)
		return 0, nil
	}
	if !updatedMs.IsInt64() {
		return 0, nil
	}
	return updatedMs.Int64(), nil
}
