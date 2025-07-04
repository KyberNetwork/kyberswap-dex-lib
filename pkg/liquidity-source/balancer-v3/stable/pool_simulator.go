package stable

import (
	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v3/base"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v3/hooks"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v3/math"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v3/shared"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*base.PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	var staticExtra shared.StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	var hook hooks.IHook
	var err error
	switch staticExtra.HookType {
	case shared.StableSurgeHookType:
		hook, err = hooks.NewStableSurgeHook(extra.MaxSurgeFeePercentage, extra.SurgeThresholdPercentage)
	}

	if err != nil {
		logger.WithFields(logger.Fields{
			"poolID":      entityPool.Address,
			"hookType":    staticExtra.HookType,
			"hooksConfig": extra.HooksConfig,
		}).Errorf("failed to create hook: %v", err)
		return nil, err
	}

	return base.NewPoolSimulator(entityPool, extra.Extra, &staticExtra, &PoolSimulator{
		currentAmp: extra.AmplificationParameter,
	}, hook)
}

type PoolSimulator struct {
	currentAmp *uint256.Int
}

func (p *PoolSimulator) BaseGas() int64 {
	return baseGas
}

// OnSwap from https://etherscan.io/address/0xc1d48bb722a22cc6abf19facbe27470f08b3db8c#code#F1#L169
func (p *PoolSimulator) OnSwap(param shared.PoolSwapParams) (*uint256.Int, error) {
	invariant, err := p.computeInvariant(param.BalancesScaled18, shared.RoundDown)
	if err != nil {
		return nil, err
	}

	return lo.Ternary(param.Kind == shared.ExactIn,
		math.StableMath.ComputeOutGivenExactIn, math.StableMath.ComputeInGivenExactOut,
	)(
		p.currentAmp,
		param.BalancesScaled18,
		param.IndexIn,
		param.IndexOut,
		param.AmountGivenScaled18,
		invariant,
	)
}

func (p *PoolSimulator) computeInvariant(balancesLiveScaled18 []*uint256.Int, rounding shared.Rounding) (*uint256.Int,
	error) {
	invariant, err := math.StableMath.ComputeInvariant(p.currentAmp, balancesLiveScaled18)
	if err != nil {
		return nil, err
	}

	if invariant.Sign() > 0 && rounding == shared.RoundUp {
		return invariant.AddUint64(invariant, 1), nil
	}

	return invariant, nil
}
