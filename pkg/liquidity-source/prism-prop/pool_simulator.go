package prismprop

import (
	"math"

	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	orderbook "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/order-book"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// PoolSimulator wraps order-book.PoolSimulator purely to carry prism-prop's
// own StaticExtra (the router address) through to GetMetaInfo, as
// pool.MetaInfo.ApprovalAddress -- prism-prop's swap(address,address,int256,
// uint256,address) (0x00799aff) is byte-for-byte identical to IFeltir.swap
// in ks-dex-aggregator-sc, so encoding reuses executeFeltir/
// PackPoolAddressFromApprovalInfo as-is; no prism-prop-specific executor
// code exists or is needed. The quoting/state logic itself is entirely
// order-book.PoolSimulator's; this type adds no behavior of its own.
type PoolSimulator struct {
	*orderbook.PoolSimulator
	routerAddress string
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	poolSim, err := orderbook.NewPoolSimulatorWith(entityPool, math.MaxInt64)
	if err != nil {
		return nil, err
	}
	poolSim.Gas = defaultGas

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	return &PoolSimulator{PoolSimulator: poolSim, routerAddress: staticExtra.RouterAddress}, nil
}

func (p *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *p
	cloned.PoolSimulator = p.PoolSimulator.CloneState().(*orderbook.PoolSimulator)
	return &cloned
}

func (p *PoolSimulator) GetMetaInfo(_, _ string) any {
	return pool.MetaInfo{ApprovalAddress: p.routerAddress, BlockNumber: p.Info.BlockNumber}
}
