package pamm

import (
	"encoding/json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/kipseli"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/kipseli/prop"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolSimulator struct {
	*prop.PoolSimulator
	so               map[string]kipseli.StateOverride
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

// CloneState shallow-copies the struct (so + lub are immutable post-init, safe
// to share) and recurses into the embedded prop simulator for state isolation.
func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	cloned.PoolSimulator = s.PoolSimulator.CloneState().(*prop.PoolSimulator)
	return &cloned
}

func (s *PoolSimulator) GetMetaInfo(_, _ string) any {
	return PoolMetaInfo{
		BlockNumber:      s.Info.BlockNumber,
		RouterAddress:    s.RouterAddress,
		SO:               s.so,
		LastUpdatedBlock: s.lastUpdatedBlock,
	}
}
