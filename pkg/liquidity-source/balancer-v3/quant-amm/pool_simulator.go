package quantamm

import (
	"fmt"
	"time"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v3/base"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v3/math"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v3/shared"
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

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	return base.NewPoolSimulator(entityPool, extra.Extra, staticExtra.StaticExtra, &PoolSimulator{
		weights:           extra.Weights,
		multipliers:       extra.Multipliers,
		lastUpdateTime:    extra.LastUpdateTime,
		lastInteropTime:   extra.LastInteropTime,
		maxTradeSizeRatio: staticExtra.MaxTradeSizeRatio,
	}, nil)
}

type PoolSimulator struct {
	weights           []*uint256.Int
	multipliers       []*uint256.Int
	lastUpdateTime    uint64
	lastInteropTime   uint64
	maxTradeSizeRatio *uint256.Int
}

func (p *PoolSimulator) BaseGas() int64 {
	return baseGas
}

func (p *PoolSimulator) OnSwap(param shared.PoolSwapParams) (*uint256.Int, error) {
	idxGiven, idxCalc, computeFn := param.IndexIn, param.IndexOut, math.WeightedMath.ComputeOutGivenExactIn
	if param.Kind == shared.ExactOut {
		idxGiven, idxCalc, computeFn = param.IndexOut, param.IndexIn, math.WeightedMath.ComputeInGivenExactOut
	}
	maxTradeSize, _ := math.FixPoint.MulDown(param.BalancesScaled18[idxGiven], p.maxTradeSizeRatio)
	fmt.Println(param.Kind, param.AmountGivenScaled18, maxTradeSize)
	if param.AmountGivenScaled18.Cmp(maxTradeSize) > 0 {
		return nil, ErrMaxTradeSizeRatioExceeded
	}

	multiplierTime := min(uint64(time.Now().Unix()), p.lastInteropTime)
	timeSinceLastUpdate := multiplierTime - p.lastUpdateTime
	tokenInWeight, tokenOutWeight, err := p.getNormalizedWeightPair(param.IndexIn, param.IndexOut, timeSinceLastUpdate)
	if err != nil {
		return nil, err
	}
	amountCalculated, err := computeFn(
		param.BalancesScaled18[param.IndexIn],
		tokenInWeight,
		param.BalancesScaled18[param.IndexOut],
		tokenOutWeight,
		param.AmountGivenScaled18,
	)
	if err != nil {
		return nil, err
	}

	maxTradeSize, _ = math.FixPoint.MulDown(param.BalancesScaled18[idxCalc], p.maxTradeSizeRatio)
	fmt.Println(">", param.Kind, amountCalculated, maxTradeSize)
	if amountCalculated.Cmp(maxTradeSize) > 0 {
		return nil, ErrMaxTradeSizeRatioExceeded
	}
	return amountCalculated, nil
}

func (p *PoolSimulator) getNormalizedWeightPair(idxIn, idxOut int, timeSinceLastUpdate uint64) (*uint256.Int, *uint256.Int,
	error) {
	if idxIn < 0 || idxIn >= len(p.weights) || idxOut < 0 || idxOut >= len(p.weights) {
		return nil, nil, shared.ErrInvalidToken
	}
	var uTimeSinceLastUpdate uint256.Int
	uTimeSinceLastUpdate.SetUint64(timeSinceLastUpdate)
	tokenInWeight := p.calculateBlockNormalisedWeight(p.weights[idxIn], p.multipliers[idxIn], &uTimeSinceLastUpdate)
	tokenOutWeight := p.calculateBlockNormalisedWeight(p.weights[idxOut], p.multipliers[idxOut], &uTimeSinceLastUpdate)
	return tokenInWeight, tokenOutWeight, nil
}

func (p *PoolSimulator) calculateBlockNormalisedWeight(weight, multiplier, uTimeSinceLastUpdate *uint256.Int) *uint256.Int {
	var multiplierScaled18 uint256.Int
	multiplierScaled18.Mul(multiplier, math.U1e18)
	if multiplier.Sign() > 0 {
		term, _ := math.FixPoint.MulDown(&multiplierScaled18, uTimeSinceLastUpdate)
		return term.Add(weight, term)
	}
	term, _ := math.FixPoint.MulDown(multiplierScaled18.Neg(&multiplierScaled18), uTimeSinceLastUpdate)
	return term.Sub(weight, term)
}
