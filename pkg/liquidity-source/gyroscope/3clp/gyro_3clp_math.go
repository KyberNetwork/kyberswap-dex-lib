package gyro3clp

import (
	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/gyroscope/math"
)

var (
	number_0_5e18 = uint256.NewInt(0.5e18)
	number_1_5e18 = uint256.NewInt(1.5e18)
	number_2e18   = uint256.NewInt(2e18)
)

var Gyro3CLPMath *gyro3CLPMath

type gyro3CLPMath struct {
	_L_VS_LPLUS_MIN *uint256.Int
}

func init() {
	Gyro3CLPMath = &gyro3CLPMath{
		_L_VS_LPLUS_MIN: uint256.NewInt(1.3e18),
	}
}

// _calculateInvariant
// https://github.com/gyrostable/concentrated-lps/blob/7e9bd3b20dd52663afca04ca743808b1d6a9521f/contracts/3clp/Gyro3CLPMath.sol#L63C104-L63C111
func (l *gyro3CLPMath) _calculateInvariant(balances []*uint256.Int, root3Alpha *uint256.Int) (*uint256.Int, error) {
	// TODO: implement
	return nil, nil
}

func (l *gyro3CLPMath) _calculateCubicTerms(balances []*uint256.Int, root3Alpha *uint256.Int) {
	// TODO: implement
}

func (l *gyro3CLPMath) _calculateCubic(
	a,
	mb,
	mc,
	md,
	root3Alpha *uint256.Int,
) (*uint256.Int, error) {
	l_lower, rootEst, err := l._calculateCubicStartingPoint(a, mb, mc)
	if err != nil {
		return nil, err
	}

	rootEst = _runNewtonIteration(mb, mc, md, root3Alpha, l_lower, rootEst)

	return rootEst, nil
}

// _calculateCubicStartingPoint
// https://github.com/gyrostable/concentrated-lps/blob/7e9bd3b20dd52663afca04ca743808b1d6a9521f/contracts/3clp/Gyro3CLPMath.sol#L118
func (l *gyro3CLPMath) _calculateCubicStartingPoint(
	a,
	mb,
	mc *uint256.Int,
) (*uint256.Int, *uint256.Int, error) {

	radic := new(uint256.Int).Add(
		math.GyroFixedPoint.MulUpU(mb, mb),
		math.GyroFixedPoint.MulUpU(a, new(uint256.Int).Mul(mc, number.Number_3)),
	)

	tmp, err := math.GyroPoolMath.Sqrt(radic, number.Number_5)
	if err != nil {
		return nil, nil, err
	}

	lplus, err := math.GyroFixedPoint.DivUpU(
		new(uint256.Int).Add(mb, tmp),
		new(uint256.Int).Mul(a, number.Number_3),
	)
	if err != nil {
		return nil, nil, err
	}

	alpha := new(uint256.Int).Sub(math.GyroFixedPoint.ONE, a)

	var l0 *uint256.Int
	if alpha.Lt(number_0_5e18) {
		l0 = math.GyroFixedPoint.MulUpU(lplus, number_2e18)
	} else {
		l0 = math.GyroFixedPoint.MulUpU(lplus, number_1_5e18)
	}

	l_lower := math.GyroFixedPoint.MulUpU(lplus, l._L_VS_LPLUS_MIN)

	return l_lower, l0, nil
}

func (l *gyro3CLPMath) _runNewtonIteration(
	mb,
	mc,
	md,
	root3Alpha,
	l_lower,
	rootEst *uint256.Int,
) *uint256.Int {

}
