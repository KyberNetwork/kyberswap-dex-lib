package pools

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"

	ekubomath "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/v3/math"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/v3/quoting"
)

// Launch is getLaunch's static tuple for the packed-metadata/tick-rate revision.
// It must not be decoded against earlier ZeroSeedLaunch deployments.
type Launch struct {
	Controller   common.Address `json:"controller"`
	TradingStart uint64         `json:"tradingStart,string"`
	DecayRateX32 uint64         `json:"decayRateX32,string"`
	InitialFee   uint64         `json:"initialFee,string"`
	FinalFee     uint64         `json:"finalFee,string"`
	TickLower    int32          `json:"tickLower"`
	TickUpper    int32          `json:"tickUpper"`
}

type ZeroSeedLaunchPoolState struct {
	UnderlyingPoolState *ConcentratedPoolState `json:"underlyingPoolState"`
	Launch              Launch                 `json:"launch"`
	BlockTimestamp      uint64                 `json:"blockTimestamp"`
}

type ZeroSeedLaunchPool struct {
	*ConcentratedPool
	launch         Launch
	blockTimestamp uint64
}

func NewZeroSeedLaunchPool(key *ConcentratedPoolKey, state *ZeroSeedLaunchPoolState) (*ZeroSeedLaunchPool, error) {
	if state == nil || state.UnderlyingPoolState == nil || state.BlockTimestamp == 0 {
		return nil, fmt.Errorf("missing launch snapshot or block timestamp")
	}
	l := state.Launch
	if key.Config.Fee != 0 || l.Controller == (common.Address{}) || l.FinalFee > l.InitialFee ||
		l.TickLower < ekubomath.MinTick || l.TickUpper > ekubomath.MaxTick || l.TickLower >= l.TickUpper {
		return nil, fmt.Errorf("invalid launch configuration")
	}
	s := state.UnderlyingPoolState
	if s.SqrtRatio == nil || s.Liquidity == nil || s.SqrtRatio.Lt(ekubomath.ToSqrtRatio(l.TickLower)) ||
		s.SqrtRatio.Gt(ekubomath.ToSqrtRatio(l.TickUpper)) || s.TickBounds[0] > l.TickLower || s.TickBounds[1] < l.TickUpper {
		return nil, fmt.Errorf("incomplete launch range snapshot")
	}
	return &ZeroSeedLaunchPool{NewConcentratedPool(key, s), l, state.BlockTimestamp}, nil
}

func (p *ZeroSeedLaunchPool) GetState() any {
	return &ZeroSeedLaunchPoolState{p.ConcentratedPoolState, p.launch, p.blockTimestamp}
}

func (p *ZeroSeedLaunchPool) CloneSwapStateOnly() Pool {
	cloned := *p
	cloned.ConcentratedPool = p.ConcentratedPool.CloneSwapStateOnly().(*ConcentratedPool)
	return &cloned
}

func (p *ZeroSeedLaunchPool) Quote(amount *uint256.Int, isToken1 bool) (*quoting.Quote, error) {
	if p.blockTimestamp < p.launch.TradingStart {
		return nil, fmt.Errorf("launch trading has not started at snapshot timestamp")
	}
	var magnitude uint256.Int
	magnitude.Abs(amount)
	var signedBoundary uint256.Int
	signedBoundary.SetUint64(1).Lsh(&signedBoundary, 127)
	if (amount.Sign() >= 0 && magnitude.Cmp(&signedBoundary) >= 0) || magnitude.Gt(&signedBoundary) {
		return nil, ekubomath.ErrOverflow
	}
	limitTick := p.launch.TickLower
	if ekubomath.IsPriceIncreasing(amount, isToken1) {
		limitTick = p.launch.TickUpper
	}
	limit := ekubomath.ToSqrtRatio(limitTick)
	quote, err := p.ConcentratedPool.quoteWithLimit(amount, isToken1, limit)
	if err != nil {
		return nil, err
	}
	if quote.CalculatedAmount.Gt(&signedBoundary) {
		return nil, ekubomath.ErrOverflow
	}
	rate := ekubomath.DecayedLaunchFee(p.launch.InitialFee, p.launch.FinalFee, p.launch.DecayRateX32, p.blockTimestamp-p.launch.TradingStart)
	if amount.Sign() >= 0 {
		fee := ekubomath.ComputeFee(quote.CalculatedAmount, rate)
		quote.CalculatedAmount.Sub(quote.CalculatedAmount, fee)
		quote.FeesPaid = fee
	} else {
		gross, err := ekubomath.AmountBeforeFee(quote.CalculatedAmount, rate)
		if err != nil || gross.BitLen() > 127 {
			return nil, ekubomath.ErrOverflow
		}
		quote.FeesPaid = new(uint256.Int).Sub(gross, quote.CalculatedAmount)
		quote.CalculatedAmount = gross
	}
	extension := p.GetKey().Extension()
	quote.SwapInfo.Forward = &extension
	quote.SwapInfo.SkipAhead = 1
	quote.SwapInfo.PriceLimit = ekubomath.FixedSqrtRatioToFloat(limit, false)
	// The frozen contracts' cold-router full-domain test bounds each execution
	// below 1.9M. Keep this conservative until the provider executor is calibrated.
	quote.Gas = 1_900_000
	return quote, nil
}
