package syntropiaprop

import (
	"math"
	"time"

	"github.com/goccy/go-json"
	"github.com/samber/lo"

	orderbook "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/order-book"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// PoolSimulator quotes from the levels the tracker read off the lens. The venue's pricing
// is not reimplemented here.
type PoolSimulator struct {
	*orderbook.PoolSimulator
	staticExtra StaticExtra
}

var (
	_ = pool.RegisterFactory(DexType, NewPoolSimulator)
	// Every pair is paid out of one vault.
	_ = pool.RegisterUseSwapLimit(DexType)
)

func NewPoolSimulator(params pool.FactoryParams) (*PoolSimulator, error) {
	base, err := orderbook.NewPoolSimulatorWith(params.EntityPool,
		lo.Ternary[time.Duration](params.Opts.StaleCheck, maxAge, math.MaxInt64))
	if err != nil {
		return nil, err
	}
	base.Gas = orderbook.Gas{Base: defaultGas}

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(params.EntityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	return &PoolSimulator{PoolSimulator: base, staticExtra: staticExtra}, nil
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	cloned.PoolSimulator = s.PoolSimulator.CloneState().(*orderbook.PoolSimulator)
	return &cloned
}

// GetMetaInfo names the pool, which implements IPropAMM: the executor transfers tokenIn to
// it and calls swap(tokenIn, tokenOut, amountIn, minAmountOut, recipient, deadline), or
// approves it and calls swapWithAllowance with the same arguments.
func (s *PoolSimulator) GetMetaInfo(_, _ string) any {
	return pool.MetaInfo{ApprovalAddress: s.staticExtra.Pool, BlockNumber: s.Info.BlockNumber}
}
