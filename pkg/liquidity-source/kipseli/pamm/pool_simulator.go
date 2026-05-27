package pamm

import (
	"encoding/json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/kipseli/prop"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolSimulator struct {
	*prop.PoolSimulator
	so               map[string]map[string]string
	lastUpdatedBlock uint64
}

var (
	_ = pool.RegisterFactory0(DexType, NewPoolSimulator)
	_ = pool.RegisterUseSwapLimit(valueobject.ExchangeKipseliPamm)
)

func NewPoolSimulator(p entity.Pool) (*PoolSimulator, error) {
	inner, err := prop.NewPoolSimulator(p)
	if err != nil {
		return nil, err
	}
	sim := &PoolSimulator{PoolSimulator: inner}
	var extra Extra
	if err := json.Unmarshal([]byte(p.Extra), &extra); err == nil {
		sim.so = extra.SO
		sim.lastUpdatedBlock = extra.LastUpdatedBlock
	}
	return sim, nil
}

func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	res, err := s.PoolSimulator.CalcAmountOut(params)
	if err == nil && res != nil {
		res.Gas = defaultGas
	}
	return res, err
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	return &PoolSimulator{
		PoolSimulator:    s.PoolSimulator.CloneState().(*prop.PoolSimulator),
		so:               s.so,
		lastUpdatedBlock: s.lastUpdatedBlock,
	}
}

func (s *PoolSimulator) GetMetaInfo(_, _ string) any {
	return PoolMetaInfo{
		BlockNumber:      s.Info.BlockNumber,
		RouterAddress:    s.RouterAddress,
		SO:               s.so,
		LastUpdatedBlock: s.lastUpdatedBlock,
	}
}
