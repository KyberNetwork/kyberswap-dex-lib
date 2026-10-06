package nemoprop

import (
	"strings"
	"time"

	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ladder"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// PoolSimulator quotes one (base, market) of a NemoSwap proxy from the feed
// ladders the tracker wrote into pool state. Like every ladder adapter it
// does no I/O: the live feed connection lives in the tracker only.
type PoolSimulator struct {
	*ladder.PoolSimulator

	proxy string
}

var (
	_ = pool.RegisterFactory(DexType, NewPoolSimulator)
	_ = pool.RegisterUseSwapLimit(valueobject.ExchangeNemoProp)
)

// NewPoolSimulator ages the snapshot once, at build time, from when the
// tracker received it. Past the fresh window its ladder outputs decay like
// the contract's quotes do between pushes. A snapshot past the maximum age,
// or past the fresh window with decay disabled, is rejected with
// ladder.ErrStale under StaleCheck; otherwise the simulator is built with
// empty ladders and zero reserves, as the tracker writes once a snapshot
// ages out, so older prices never quote.
func NewPoolSimulator(params pool.FactoryParams) (*PoolSimulator, error) {
	p := params.EntityPool
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}
	var extra Extra
	if err := json.Unmarshal([]byte(p.Extra), &extra); err != nil {
		return nil, err
	}

	scale, quotable := extra.decayAt(time.Now())
	if params.Opts.StaleCheck && !quotable {
		return nil, ladder.ErrStale
	}
	switch {
	case !quotable:
		extra.Ladders = [2][]ladder.Point{}
		p.Reserves = entity.PoolReserves{"0", "0"}
	case scale < 1:
		for dir := range extra.Ladders {
			decayed := make([]ladder.Point, len(extra.Ladders[dir]))
			for i, pt := range extra.Ladders[dir] {
				decayed[i] = ladder.Point{pt.AmountIn(), pt.AmountOut() * scale}
			}
			extra.Ladders[dir] = decayed
		}
	}
	if !quotable || scale < 1 {
		extraBytes, err := json.Marshal(extra.Extra)
		if err != nil {
			return nil, err
		}
		p.Extra = string(extraBytes)
	}

	base, err := ladder.NewPoolSimulator(p)
	if err != nil {
		return nil, err
	}
	base.Gas = defaultGas
	return &PoolSimulator{PoolSimulator: base, proxy: strings.ToLower(staticExtra.Address)}, nil
}

// decayAt returns the factor ladder outputs scale by at now,
// 1/(1 + DecayBps/10000 * secondsStale), and whether the snapshot is still
// quotable: within the maximum age, and fresh unless decay is enabled. A
// pool without a snapshot has no ladders, so it quotes nothing either way.
func (e *Extra) decayAt(now time.Time) (float64, bool) {
	if e.ReceivedAtMs == 0 {
		return 1, false
	}
	age := now.UnixMilli() - e.ReceivedAtMs
	if age >= e.MaxAgeMs {
		return 1, false
	}
	stale := age - e.FreshMs
	if stale <= 0 {
		return 1, true
	}
	if e.DecayBps <= 0 {
		return 1, false
	}
	return float64(decayScale) / float64(decayScale+e.DecayBps*stale), true
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	cloned.PoolSimulator = s.PoolSimulator.CloneState().(*ladder.PoolSimulator)
	return &cloned
}

// GetMetaInfo keeps ApprovalAddress (rather than ladder.PoolMeta's default
// shape) so callers that read pool.ApprovalInfo off it resolve the proxy.
func (s *PoolSimulator) GetMetaInfo(_, _ string) any {
	return pool.MetaInfo{ApprovalAddress: s.proxy, BlockNumber: s.Info.BlockNumber}
}
