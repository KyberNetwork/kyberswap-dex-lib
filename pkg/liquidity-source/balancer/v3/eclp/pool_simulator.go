package eclp

import (
	"github.com/KyberNetwork/int256"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer/v3/base"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer/v3/math"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer/v3/shared"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*base.PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	} else if extra.Extra == nil {
		return nil, shared.ErrInvalidExtra
	} else if extra.Buffers == nil {
		extra.Buffers = make([]*shared.ExtraBuffer, len(entityPool.Tokens))
	}

	var staticExtra shared.StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	return base.NewPoolSimulator(entityPool, extra.Extra, &staticExtra, &PoolSimulator{
		eclpParams: extra.ECLPParams,
	}, nil)
}

type PoolSimulator struct {
	eclpParams *ECLPParams
}

func (p *PoolSimulator) BaseGas() int64 {
	return baseGas
}

// OnSwap https://arbiscan.io/address/0xc09a98b0138d8cfceff0e4ef672e8bd30ec6eda9#code#F1#L156
func (p *PoolSimulator) OnSwap(param shared.PoolSwapParams) (amountOutScaled18 *uint256.Int, err error) {
	eclpParams, derivedECLPParams := &p.eclpParams.Params, &p.eclpParams.D
	invariant := &math.Vector2{}
	{
		currentInvariant, invErr, err := math.GyroECLPMath.CalculateInvariantWithError(
			param.BalancesScaled18, eclpParams, derivedECLPParams,
		)
		if err != nil {
			return nil, err
		}

		invariant.X = new(int256.Int)
		invariant.X = invariant.X.Add(
			currentInvariant,
			invariant.X.Mul(math.I2, invErr),
		)

		invariant.Y = currentInvariant
	}

	if param.Kind == shared.ExactIn {
		amountOutScaled18, err = math.GyroECLPMath.CalcOutGivenIn(
			param.BalancesScaled18,
			param.AmountGivenScaled18,
			param.IndexIn == 0,
			eclpParams,
			derivedECLPParams,
			invariant,
		)
	} else {
		amountOutScaled18, err = math.GyroECLPMath.CalcInGivenOut(
			param.BalancesScaled18,
			param.AmountGivenScaled18,
			param.IndexIn == 0,
			eclpParams,
			derivedECLPParams,
			invariant,
		)
	}

	if err != nil {
		return nil, err
	}

	return
}
