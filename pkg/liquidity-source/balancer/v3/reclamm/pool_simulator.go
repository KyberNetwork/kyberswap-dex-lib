package reclamm

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
	} else if extra.Extra == nil {
		return nil, shared.ErrInvalidExtra
	} else if extra.Buffers == nil {
		extra.Buffers = make([]*shared.ExtraBuffer, len(entityPool.Tokens))
	}

	var staticExtra shared.StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	return base.NewPoolSimulator(params, extra.Extra, &staticExtra, &PoolSimulator{
		lastVirtualBalances:       extra.LastVirtualBalances,
		dailyPriceShiftBase:       extra.DailyPriceShiftBase,
		lastTimestamp:             extra.LastTimestamp,
		currentTimestamp:          extra.CurrentTimestamp,
		centerednessMargin:        extra.CenterednessMargin,
		startFourthRootPriceRatio: extra.StartFourthRootPriceRatio,
		endFourthRootPriceRatio:   extra.EndFourthRootPriceRatio,
		priceRatioUpdateStartTime: extra.PriceRatioUpdateStartTime,
		priceRatioUpdateEndTime:   extra.PriceRatioUpdateEndTime,
	}, nil)
}

type PoolSimulator struct {
	lastVirtualBalances       []*uint256.Int
	dailyPriceShiftBase       *uint256.Int
	lastTimestamp             *uint256.Int
	currentTimestamp          *uint256.Int
	centerednessMargin        *uint256.Int
	startFourthRootPriceRatio *uint256.Int
	endFourthRootPriceRatio   *uint256.Int
	priceRatioUpdateStartTime *uint256.Int
	priceRatioUpdateEndTime   *uint256.Int
}

func (p *PoolSimulator) BaseGas() int64 {
	return baseGas
}

func (p *PoolSimulator) OnSwap(param shared.PoolSwapParams) (*uint256.Int, error) {
	// Compute current virtual balances
	virtualBalancesResult, err := p.computeCurrentVirtualBalances(param.BalancesScaled18)
	if err != nil {
		return nil, err
	}

	// In SC it does: if (changed) _setLastVirtualBalances, but we don't need that as lastVirtualBalances isn't relevant going forward

	if param.Kind == shared.ExactIn {
		amountCalculatedScaled18, err := math.ReClammMath.ComputeOutGivenIn(
			param.BalancesScaled18,
			virtualBalancesResult.CurrentVirtualBalanceA,
			virtualBalancesResult.CurrentVirtualBalanceB,
			param.IndexIn,
			param.IndexOut,
			param.AmountGivenScaled18,
		)
		if err != nil {
			return nil, err
		}

		return amountCalculatedScaled18, nil
	}

	amountCalculatedScaled18, err := math.ReClammMath.ComputeInGivenOut(
		param.BalancesScaled18,
		virtualBalancesResult.CurrentVirtualBalanceA,
		virtualBalancesResult.CurrentVirtualBalanceB,
		param.IndexIn,
		param.IndexOut,
		param.AmountGivenScaled18,
	)
	if err != nil {
		return nil, err
	}

	return amountCalculatedScaled18, nil
}

// computeCurrentVirtualBalances computes the current virtual balances based on the pool state
func (p *PoolSimulator) computeCurrentVirtualBalances(balancesScaled18 []*uint256.Int) (*math.VirtualBalancesResult, error) {
	// Create price ratio state from pool simulator fields
	priceRatioState := &math.PriceRatioState{
		PriceRatioUpdateStartTime: p.priceRatioUpdateStartTime,
		PriceRatioUpdateEndTime:   p.priceRatioUpdateEndTime,
		StartFourthRootPriceRatio: p.startFourthRootPriceRatio,
		EndFourthRootPriceRatio:   p.endFourthRootPriceRatio,
	}

	// Call the math function to compute current virtual balances
	return math.ReClammMath.ComputeCurrentVirtualBalances(
		p.currentTimestamp,
		balancesScaled18,
		p.lastVirtualBalances[0], // lastVirtualBalanceA
		p.lastVirtualBalances[1], // lastVirtualBalanceB
		p.dailyPriceShiftBase,
		p.lastTimestamp,
		p.centerednessMargin,
		priceRatioState,
	)
}
