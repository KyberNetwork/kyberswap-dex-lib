package efraprop

import (
	"math"
	"time"

	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ladder"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolSimulator struct {
	*ladder.PoolSimulator

	staticExtra StaticExtra
}

var (
	_ = pool.RegisterFactory(DexType, NewPoolSimulator)
	// A fill moves the side's cursor for good, so a split route through the same pool can
	// only get worse than one swap: share the limit across pools of this venue.
	_ = pool.RegisterUseSwapLimit(valueobject.ExchangeEfraProp)
)

func NewPoolSimulator(params pool.FactoryParams) (*PoolSimulator, error) {
	ep := params.EntityPool
	base, err := ladder.NewPoolSimulatorWith(ep,
		lo.Ternary[time.Duration](params.Opts.StaleCheck, maxAge, math.MaxInt64))
	if err != nil {
		return nil, err
	}
	base.Gas = defaultGas

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(ep.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	return &PoolSimulator{PoolSimulator: base, staticExtra: staticExtra}, nil
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	cloned.PoolSimulator = s.PoolSimulator.CloneState().(*ladder.PoolSimulator)
	return &cloned
}

// GetMetaInfo keeps ApprovalAddress (rather than ladder.PoolMeta's default shape) so callers
// that read pool.ApprovalInfo off it still resolve the venue to approve.
func (s *PoolSimulator) GetMetaInfo(_, _ string) any {
	return PoolMeta{
		BlockNumber:     s.Info.BlockNumber,
		Venue:           s.staticExtra.Venue,
		ApprovalAddress: s.staticExtra.Venue,
	}
}
