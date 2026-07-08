package elfomofi

import (
	"github.com/goccy/go-json"

	orderbook "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/order-book"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

var _ = pool.RegisterFactory(DexType, NewPoolSimulator)

type PoolSimulator struct {
	*orderbook.PoolSimulator
	factoryAddress string
}

func NewPoolSimulator(params pool.FactoryParams) (*PoolSimulator, error) {
	poolSim, err := orderbook.NewPoolSimulator(params)
	if err != nil {
		return nil, err
	}
	poolSim.Gas = orderbook.Gas{Base: defaultGas}

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(params.EntityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	return &PoolSimulator{
		PoolSimulator:  poolSim,
		factoryAddress: staticExtra.FactoryAddress,
	}, nil
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	cloned.PoolSimulator = s.PoolSimulator.CloneState().(*orderbook.PoolSimulator)
	return &cloned
}

func (s *PoolSimulator) GetMetaInfo(_, _ string) any {
	return pool.ApprovalInfo{
		ApprovalAddress: s.factoryAddress,
	}
}
