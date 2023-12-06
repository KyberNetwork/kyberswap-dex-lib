package gyro2clp

import (
	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/gyroscope/math"
)

var Gyro2CLPMath *gyro2CLPMath

type gyro2CLPMath struct {
}

func init() {
	Gyro2CLPMath = &gyro2CLPMath{}
}

// _calculateInvariant
// https://github.com/gyrostable/concentrated-lps/blob/7e9bd3b20dd52663afca04ca743808b1d6a9521f/contracts/2clp/Gyro2CLPMath.sol#L29
func (l *gyro2CLPMath) _calculateInvariant(
	balances []*uint256.Int,
	sqrtAlpha *uint256.Int,
	sqrtBeta *uint256.Int,
) (*uint256.Int, error) {
	a, mb, bSquare, mc, err := l._calculateQuadraticTerms(balances, sqrtAlpha, sqrtBeta)
	if err != nil {
		return nil, err
	}

	return l._calculateQuadratic(a, mb, bSquare, mc)
}

// _calculateQuadraticTerms
// https://github.com/gyrostable/concentrated-lps/blob/7e9bd3b20dd52663afca04ca743808b1d6a9521f/contracts/2clp/Gyro2CLPMath.sol#L53
func (l *gyro2CLPMath) _calculateQuadraticTerms(
	balances []*uint256.Int,
	sqrtAlpha *uint256.Int,
	sqrtBeta *uint256.Int,
) (*uint256.Int, *uint256.Int, *uint256.Int, *uint256.Int, error) {
	a, err := math.NewCalculator(math.GyroFixedPoint.ONE).
		SubWith(math.NewCalculator(sqrtAlpha).DivDown(sqrtBeta)).
		Result()
	if err != nil {
		return nil, nil, nil, nil, err
	}

	bterm0, err := math.GyroFixedPoint.DivDown(balances[1], sqrtBeta)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	bterm1, err := math.GyroFixedPoint.DivDown(balances[0], sqrtAlpha)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	mb, err := math.GyroFixedPoint.Add(bterm0, bterm1)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	mc, err := math.GyroFixedPoint.MulDown(balances[0], balances[1])
	if err != nil {
		return nil, nil, nil, nil, err
	}

	bSquare, err := math.NewCalculator(balances[0]).
		MulDown(balances[0]).
		MulDown(sqrtAlpha).
		MulDown(sqrtAlpha).
		Result()
	if err != nil {
		return nil, nil, nil, nil, err
	}

	bSq2, err := math.NewCalculator(balances[0]).
		MulDown(balances[1]).
		MulDown(new(uint256.Int).Mul(number.Number_2, math.GyroFixedPoint.ONE)).
		MulDown(sqrtAlpha).
		DivDown(sqrtBeta).
		Result()
	if err != nil {
		return nil, nil, nil, nil, err
	}

	bSq3, err := math.NewCalculator(balances[1]).
		MulDown(balances[1]).
		DivDownWith(math.NewCalculator(sqrtBeta).MulUp(sqrtBeta)).
		Result()

	bSquare, err = math.NewCalculator(bSquare).Add(bSq2).Add(bSq3).Result()
	if err != nil {
		return nil, nil, nil, nil, err
	}

	return a, mb, bSquare, mc, nil
}

// _calculateQuadratic
// https://github.com/gyrostable/concentrated-lps/blob/7e9bd3b20dd52663afca04ca743808b1d6a9521f/contracts/2clp/Gyro2CLPMath.sol#L91
func (l *gyro2CLPMath) _calculateQuadratic(
	a,
	mb,
	bSquare,
	mc *uint256.Int,
) (*uint256.Int, error) {
	denominator, err := math.GyroFixedPoint.MulUp(a, new(uint256.Int).Mul(number.Number_2, math.GyroFixedPoint.ONE))
	if err != nil {
		return nil, err
	}

	tmp, err := math.GyroFixedPoint.MulDown(mc, new(uint256.Int).Mul(number.Number_4, math.GyroFixedPoint.ONE))
	if err != nil {
		return nil, err
	}

	addTerm, err := math.GyroFixedPoint.MulDown(tmp, a)
	if err != nil {
		return nil, err
	}

	radicand, err := math.GyroFixedPoint.Add(bSquare, addTerm)
	if err != nil {
		return nil, err
	}

	sqrResult, err := math.GyroPoolMath.Sqrt(radicand, number.Number_5)
	if err != nil {
		return nil, err
	}

	numerator, err := math.GyroFixedPoint.Add(mb, sqrResult)
	if err != nil {
		return nil, err
	}

	return math.GyroFixedPoint.DivDown(numerator, denominator)
}

// _calcOutGivenIn
// https://github.com/gyrostable/concentrated-lps/blob/7e9bd3b20dd52663afca04ca743808b1d6a9521f/contracts/2clp/Gyro2CLPMath.sol#L123
func (l *gyro2CLPMath) _calcOutGivenIn(
	balanceIn, balanceOut, amountIn, virtualOffsetIn, virtualOffsetOut *uint256.Int,
) (*uint256.Int, error) {
	virtInOver, err := math.NewCalculator(balanceIn).
		AddWith(
			math.NewCalculator(virtualOffsetIn).MulUp(new(uint256.Int).Add(math.GyroFixedPoint.ONE, number.Number_2)),
		).Result()

	virtOutUnder, err := math.NewCalculator(balanceOut).
		AddWith(
			math.NewCalculator(virtualOffsetOut).MulDown(new(uint256.Int).Sub(math.GyroFixedPoint.ONE, number.Number_1)),
		).Result()
	if err != nil {
		return nil, err
	}

	return math.NewCalculator(virtOutUnder).
		MulDown(amountIn).
		DivDownWith(math.NewCalculator(virtInOver).Add(amountIn)).
		Result()
}

// _calculateVirtualParameter0
// https://github.com/gyrostable/concentrated-lps/blob/7e9bd3b20dd52663afca04ca743808b1d6a9521f/contracts/2clp/Gyro2CLPMath.sol#L197
func (l *gyro2CLPMath) _calculateVirtualParameter0(invariant, _sqrtBeta *uint256.Int) (*uint256.Int, error) {
	return math.GyroFixedPoint.DivDown(invariant, _sqrtBeta)
}

// _calculateVirtualParameter1
// https://github.com/gyrostable/concentrated-lps/blob/7e9bd3b20dd52663afca04ca743808b1d6a9521f/contracts/2clp/Gyro2CLPMath.sol#L203
func (l *gyro2CLPMath) _calculateVirtualParameter1(invariant, _sqrtAlpha *uint256.Int) (*uint256.Int, error) {
	return math.GyroFixedPoint.MulDown(invariant, _sqrtAlpha)
}
