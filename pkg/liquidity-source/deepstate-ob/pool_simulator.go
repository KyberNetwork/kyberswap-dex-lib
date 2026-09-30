package deepstateob

import (
	"math"
	"time"

	"github.com/goccy/go-json"
	"github.com/samber/lo"

	orderbook "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/order-book"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// PoolSimulator wraps the generic order-book simulator (flat price/size
// levels, walked to compute a quote) exactly like pkg/liquidity-source/kuru-ob
// does. All Deepstate-specific work (radix-tree decode, tick-to-price
// conversion, decimal normalization) happens once in pool_tracker.go when
// building Extra.LevelsFrom; from here on the swap math is identical to any
// other flat-levels order book.
type PoolSimulator struct {
	*orderbook.PoolSimulator
	router string
	epoch  string
}

var _ = pool.RegisterFactory(DexType, NewPoolSimulator)

func NewPoolSimulator(params pool.FactoryParams) (*PoolSimulator, error) {
	entityPool := params.EntityPool
	poolSim, err := orderbook.NewPoolSimulatorWith(entityPool,
		lo.Ternary[time.Duration](params.Opts.StaleCheck, orderbook.MaxAge, math.MaxInt64))
	if err != nil {
		return nil, err
	}
	poolSim.Gas = defaultGas

	var staticExtra StaticExtra
	_ = json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra)

	var extra Extra
	_ = json.Unmarshal([]byte(entityPool.Extra), &extra)

	return &PoolSimulator{PoolSimulator: poolSim, router: staticExtra.Router, epoch: extra.Epoch}, nil
}

func (p *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *p
	cloned.PoolSimulator = p.PoolSimulator.CloneState().(*orderbook.PoolSimulator)
	return &cloned
}

func (p *PoolSimulator) GetMetaInfo(tokenIn, _ string) any {
	return MetaInfo{
		Router:      p.router,
		Epoch:       p.epoch,
		IsBid:       p.GetTokenIndex(tokenIn) == 1, // swapping FROM token1 fills the ask tree via isBid=true
		BlockNumber: p.Info.BlockNumber,
	}
}
