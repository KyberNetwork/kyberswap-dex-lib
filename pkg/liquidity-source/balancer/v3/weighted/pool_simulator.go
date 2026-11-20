package weighted

import (
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

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
		minTokenBalances:  extra.MinTokenBalances,
	}, nil)
}

type PoolSimulator struct {
	normalizedWeights []*uint256.Int
	minTokenBalances  []*uint256.Int
}

func (p *PoolSimulator) BaseGas() int64 {
	return baseGas
}

// OnSwap mirrors WeightedPool.onSwap for both v1 and v2 deployments:
//   - v1: https://etherscan.io/address/0xb9b144b5678ff6527136b2c12a86c9ee5dd12a85#code#F1#L150
//   - v2 (when minTokenBalances is present): rounds balanceTokenInScaled18 up by +1 on both
//     ExactIn and ExactOut swaps and enforces getMinTokenBalances() before and after the swap.
func (p *PoolSimulator) OnSwap(param shared.PoolSwapParams) (*uint256.Int, error) {
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

	isV2 := len(p.minTokenBalances) > 0
	var balanceInPlusOne uint256.Int
	if isV2 {
		if _, overflow := balanceInPlusOne.AddOverflow(balanceTokenInScaled18, math.U1); overflow {
			return nil, math.ErrAddOverflow
		}
		balanceTokenInScaled18 = &balanceInPlusOne

		if err := p.ensureMinimumBalance(param.IndexIn, balanceTokenInScaled18); err != nil {
			return nil, err
		}
	}

	var calculatedAmountScaled18, amountOutScaled18 *uint256.Int
	if param.Kind == shared.ExactIn {
		calculatedAmountScaled18, err = math.WeightedMath.ComputeOutGivenExactIn(
			balanceTokenInScaled18,
			weightIn,
			balanceTokenOutScaled18,
			weightOut,
			param.AmountGivenScaled18,
		)
		amountOutScaled18 = calculatedAmountScaled18
	} else {
		calculatedAmountScaled18, err = math.WeightedMath.ComputeInGivenExactOut(
			balanceTokenInScaled18,
			weightIn,
			balanceTokenOutScaled18,
			weightOut,
			param.AmountGivenScaled18,
		)
		amountOutScaled18 = param.AmountGivenScaled18
	}
	if err != nil {
		return nil, err
	}

	if isV2 {
		var endingBalanceOut uint256.Int
		if _, underflow := endingBalanceOut.SubOverflow(balanceTokenOutScaled18, amountOutScaled18); underflow {
			return nil, math.ErrSubOverflow
		}
		if err := p.ensureMinimumBalance(param.IndexOut, &endingBalanceOut); err != nil {
			return nil, err
		}
	}

	return calculatedAmountScaled18, nil
}

func (p *PoolSimulator) getNormalizedWeight(tokenIndex int) (*uint256.Int, error) {
	if tokenIndex < 0 || tokenIndex >= len(p.normalizedWeights) {
		return nil, ErrInvalidToken
	}

	return p.normalizedWeights[tokenIndex], nil
}

func (p *PoolSimulator) ensureMinimumBalance(tokenIndex int, balanceScaled18 *uint256.Int) error {
	if tokenIndex < 0 || tokenIndex >= len(p.minTokenBalances) {
		return ErrInvalidToken
	}

	if minBal := p.minTokenBalances[tokenIndex]; minBal != nil && balanceScaled18.Lt(minBal) {
		return ErrTokenBalanceBelowMin
	}

	return nil
}
