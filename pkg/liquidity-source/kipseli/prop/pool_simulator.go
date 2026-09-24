package prop

import (
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ladder"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/titan"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type PoolSimulator struct {
	*ladder.PoolSimulator

	staticExtra StaticExtra
	so          titan.StateOverrides
}

var (
	_ = pool.RegisterFactory0(DexType, NewPoolSimulator)
	_ = pool.RegisterUseSwapLimit(DexType)
)

func NewPoolSimulator(ep entity.Pool) (*PoolSimulator, error) {
	base, err := ladder.NewPoolSimulatorWith(ep, maxAge)
	if err != nil {
		return nil, err
	}
	base.Gas = defaultGas

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(ep.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	// ladder.NewPoolSimulatorWith only reads Extra.Ladders, so our extra SO
	// field parses along for free without any stripping.
	var extra Extra
	_ = json.Unmarshal([]byte(ep.Extra), &extra)

	return &PoolSimulator{
		PoolSimulator: base,
		staticExtra:   staticExtra,
		so:            extra.SO,
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
		SO:              s.so,
	}
}
