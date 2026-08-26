package prismprop

import (
	"math"
	"time"

	"github.com/goccy/go-json"
	"github.com/samber/lo"

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

var _ = pool.RegisterFactory(DexType, NewPoolSimulator)

// NewPoolSimulator honors params.Opts.StaleCheck against orderbook.MaxAge,
// same as order-book's own NewPoolSimulator -- the book updates almost every
// block (confirmed live), so a stale/stuck tracker must not keep serving
// quotes indefinitely.
func NewPoolSimulator(params pool.FactoryParams) (*PoolSimulator, error) {
	maxAge := lo.Ternary[time.Duration](params.Opts.StaleCheck, orderbook.MaxAge, math.MaxInt64)
	poolSim, err := orderbook.NewPoolSimulatorWith(params.EntityPool, maxAge)
	if err != nil {
		return nil, err
	}
	poolSim.Gas = defaultGas

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(params.EntityPool.StaticExtra), &staticExtra); err != nil {
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
