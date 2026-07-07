package weighted

import (
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer/v3/base"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer/v3/math"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer/v3/shared"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

var _ = pool.RegisterFactory(DexType, NewPoolSimulator)

func NewPoolSimulator(params pool.FactoryParams) (*base.PoolSimulator, error) {
	entityPool := params.EntityPool
	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	var staticExtra shared.StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	return base.NewPoolSimulator(params, extra.Extra, &staticExtra, &PoolSimulator{
		normalizedWeights: extra.NormalizedWeights,
	}, nil)
}

type PoolSimulator struct {
	normalizedWeights []*uint256.Int
}

func (p *PoolSimulator) BaseGas() int64 {
	return baseGas
}

// OnSwap from https://etherscan.io/address/0xb9b144b5678ff6527136b2c12a86c9ee5dd12a85#code#F1#L150
func (p *PoolSimulator) OnSwap(param shared.PoolSwapParams) (amountOutScaled18 *uint256.Int, err error) {
	balanceTokenInScaled18 := param.BalancesScaled18[param.IndexIn]
	balanceTokenOutScaled18 := param.BalancesScaled18[param.IndexOut]

	weightIn, err := p.getNormalizedWeight(param.IndexIn)
	if err != nil {
		return nil, err
	}

	weightOut, err := p.getNormalizedWeight(param.IndexOut)
	if err != nil {
		return nil, err
	}

	return lo.Ternary(param.Kind == shared.ExactIn,
		math.WeightedMath.ComputeOutGivenExactIn, math.WeightedMath.ComputeInGivenExactOut,
	)(
		balanceTokenInScaled18,
		weightIn,
		balanceTokenOutScaled18,
		weightOut,
		param.AmountGivenScaled18,
	)
}

func (p *PoolSimulator) getNormalizedWeight(tokenIndex int) (*uint256.Int, error) {
	if tokenIndex > len(p.normalizedWeights) {
		return nil, ErrInvalidToken
	}

	return p.normalizedWeights[tokenIndex], nil
}
