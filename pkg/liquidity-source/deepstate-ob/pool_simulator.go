package deepstateob

import (
	"math"

	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
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

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	poolSim, err := orderbook.NewPoolSimulatorWith(entityPool, math.MaxInt64)
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
