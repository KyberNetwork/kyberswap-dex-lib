package prop

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
	_ = pool.RegisterUseSwapLimit(valueobject.Exchange1010Prop)
)

func NewPoolSimulator(params pool.FactoryParams) (*PoolSimulator, error) {
	p := params.EntityPool
	base, err := ladder.NewPoolSimulatorWith(p,
		lo.Ternary[time.Duration](params.Opts.StaleCheck, ladder.MaxAge, math.MaxInt64))
	if err != nil {
		return nil, err
	}
	base.Gas = defaultGas

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	return &PoolSimulator{PoolSimulator: base, staticExtra: staticExtra}, nil
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	cloned.PoolSimulator = s.PoolSimulator.CloneState().(*ladder.PoolSimulator)
	return &cloned
}

// GetMetaInfo keeps ApprovalAddress (rather than ladder.PoolMeta's default
// shape) so callers that read pool.ApprovalInfo off it still resolve the
// router to approve.
func (s *PoolSimulator) GetMetaInfo(_, _ string) any {
	return pool.MetaInfo{ApprovalAddress: s.staticExtra.RouterAddress, BlockNumber: s.Info.BlockNumber}
}
