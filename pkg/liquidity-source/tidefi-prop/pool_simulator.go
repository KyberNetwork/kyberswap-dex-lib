package tidefiprop

import (
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ladder"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolSimulator struct {
	*ladder.PoolSimulator

	staticExtra StaticExtra
}

var (
	_ = pool.RegisterFactory0(DexType, NewPoolSimulator)
	_ = pool.RegisterUseSwapLimit(valueobject.ExchangeTideFiProp)
)

func NewPoolSimulator(p entity.Pool) (*PoolSimulator, error) {
	base, err := ladder.NewPoolSimulatorWith(p, MaxAge)
	if err != nil {
		return nil, err
	}
	base.Gas = defaultGas

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	return &PoolSimulator{PoolSimulator: base, staticExtra: staticExtra}, nil
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	cloned.PoolSimulator = s.PoolSimulator.CloneState().(*ladder.PoolSimulator)
	return &cloned
}

// GetMetaInfo keeps ApprovalAddress (rather than ladder.PoolMeta's default
// shape) so callers that read pool.ApprovalInfo off it still resolve the
// swapper contract to approve.
func (s *PoolSimulator) GetMetaInfo(_, _ string) any {
	return pool.MetaInfo{ApprovalAddress: s.staticExtra.Address, BlockNumber: s.Info.BlockNumber}
}
