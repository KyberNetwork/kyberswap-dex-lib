package gyroscope2clp

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
	// TODO: implement
	return nil, nil
}

// _calculateQuadraticTerms
// https://github.com/gyrostable/concentrated-lps/blob/7e9bd3b20dd52663afca04ca743808b1d6a9521f/contracts/2clp/Gyro2CLPMath.sol#L53
func (l *gyro2CLPMath) _calculateQuadraticTerms(
	balances []*uint256.Int,
	sqrtAlpha *uint256.Int,
	sqrtBeta *uint256.Int,
) (*uint256.Int, *uint256.Int, *uint256.Int, *uint256.Int, error) {
	// TODO: implement
	return nil, nil, nil, nil, nil
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

func (l *gyro2CLPMath) _calcOutGivenIn(
	balanceIn, balanceOut, amountIn, virtualOffsetIn, virtualOffsetOut *uint256.Int,
) (*uint256.Int, error) {
	virtualParamIn, err := math.GyroFixedPoint.MulUp(
		virtualOffsetIn,
		new(uint256.Int).Add(math.GyroFixedPoint.ONE, number.Number_2),
	)
	if err != nil {
		return nil, err
	}

	virtInOver, err := math.GyroFixedPoint.Add(balanceIn, virtualParamIn)
	if err != nil {
		return nil, err
	}

	virtualParamOut, err := math.GyroFixedPoint.MulDown(
		virtualOffsetOut,
		new(uint256.Int).Sub(math.GyroFixedPoint.ONE, number.Number_1),
	)

	virtOutUnder, err := math.GyroFixedPoint.Add(balanceOut, virtualParamOut)
	if err != nil {
		return nil, err
	}

	tmp1, err := math.GyroFixedPoint.MulUp(virtOutUnder, amountIn)
	if err != nil {
		return nil, err
	}

	tmp2, err := math.GyroFixedPoint.Add(virtInOver, amountIn)
	if err != nil {
		return nil, err
	}

	return math.GyroFixedPoint.DivDown(tmp1, tmp2)
}
