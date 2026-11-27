package liquidcore

import (
	"math"
	"time"

	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ladder"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type PoolSimulator struct {
	*ladder.PoolSimulator
}

// AmountInRange is promoted from the embedded *ladder.PoolSimulator. Asserted here so the
// promotion cannot be lost silently: embedding by value, or shadowing the method, stops
// compiling at this line instead of quietly making the pool unable to state its range.
var _ pool.IAmountInRange = (*PoolSimulator)(nil)

var _ = pool.RegisterFactory(DexType, NewPoolSimulator)

func NewPoolSimulator(params pool.FactoryParams) (*PoolSimulator, error) {
	return NewPoolSimulatorWith(params.EntityPool, lo.Ternary(params.Opts.StaleCheck, MaxAge, math.MaxInt64))
}

func NewPoolSimulatorWith(ep entity.Pool, maxAge time.Duration) (*PoolSimulator, error) {
	base, err := ladder.NewPoolSimulatorWith(ep, maxAge)
	if err != nil {
		return nil, err
	}
	base.Gas = defaultGas

	return &PoolSimulator{PoolSimulator: base}, nil
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	cloned.PoolSimulator = s.PoolSimulator.CloneState().(*ladder.PoolSimulator)
	return &cloned
}
