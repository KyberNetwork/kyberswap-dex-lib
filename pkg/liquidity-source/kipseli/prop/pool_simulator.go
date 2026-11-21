package prop

import (
	"math"
	"time"

	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ladder"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type PoolSimulator struct {
	*ladder.PoolSimulator

	staticExtra StaticExtra
}

// AmountInRange is promoted from the embedded *ladder.PoolSimulator. Asserted here so the
// promotion cannot be lost silently: embedding by value, or shadowing the method, stops
// compiling at this line instead of quietly making the pool unable to state its range.
var _ pool.IAmountInRange = (*PoolSimulator)(nil)

var (
	_ = pool.RegisterFactory(DexType, NewPoolSimulator)
	_ = pool.RegisterUseSwapLimit(DexType)
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

	return &PoolSimulator{
		PoolSimulator: base,
		staticExtra:   staticExtra,
	}, nil
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	cloned.PoolSimulator = s.PoolSimulator.CloneState().(*ladder.PoolSimulator)
	return &cloned
}

func (s *PoolSimulator) GetMetaInfo(_, _ string) any {
	return PoolMetaInfo{
		BlockNumber:     s.Info.BlockNumber,
		RouterAddress:   s.staticExtra.RouterAddress,
		ApprovalAddress: s.staticExtra.RouterAddress,
	}
}
