package gyro3clp

// import (
// 	"errors"

// 	"github.com/KyberNetwork/blockchain-toolkit/number"
// 	"github.com/holiman/uint256"

// 	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/gyroscope/math"
// )

// var (
// 	ErrInvariantTooLarge  = errors.New("INVARIANT_TOO_LARGE")
// 	ErrInvariantUnderflow = errors.New("INVARIANT_UNDERFLOW")
// )

// var (
// 	number_0_5e18 = uint256.NewInt(0.5e18)
// 	number_1_5e18 = uint256.NewInt(1.5e18)
// 	number_2e18   = uint256.NewInt(2e18)
// 	number_1e12   = uint256.NewInt(1e12)
// 	number_1e16   = uint256.NewInt(1e16)
// )

// var Gyro3CLPMath *gyro3CLPMath

// type gyro3CLPMath struct {
// 	_L_VS_LPLUS_MIN              *uint256.Int
// 	_L_MAX                       *uint256.Int
// 	_L_THRESHOLD_SIMPLE_NUMERICS *uint256.Int
// }

// func init() {
// 	Gyro3CLPMath = &gyro3CLPMath{
// 		_L_VS_LPLUS_MIN:              uint256.NewInt(1.3e18),
// 		_L_MAX:                       uint256.NewInt(1e34),
// 		_L_THRESHOLD_SIMPLE_NUMERICS: uint256.NewInt(2e31),
// 	}
// }

// // _calculateInvariant
// // https://github.com/gyrostable/concentrated-lps/blob/7e9bd3b20dd52663afca04ca743808b1d6a9521f/contracts/3clp/Gyro3CLPMath.sol#L63C104-L63C111
// func (l *gyro3CLPMath) _calculateInvariant(balances []*uint256.Int, root3Alpha *uint256.Int) (*uint256.Int, error) {
// 	// TODO: implement
// 	return nil, nil
// }

// func (l *gyro3CLPMath) _calculateCubicTerms(balances []*uint256.Int, root3Alpha *uint256.Int) {
// 	// TODO: implement
// }

// func (l *gyro3CLPMath) _calculateCubic(
// 	a,
// 	mb,
// 	mc,
// 	md,
// 	root3Alpha *uint256.Int,
// ) (*uint256.Int, error) {
// 	l_lower, rootEst, err := l._calculateCubicStartingPoint(a, mb, mc)
// 	if err != nil {
// 		return nil, err
// 	}

// 	rootEst = _runNewtonIteration(mb, mc, md, root3Alpha, l_lower, rootEst)

// 	return rootEst, nil
// }

// // _calculateCubicStartingPoint
// // https://github.com/gyrostable/concentrated-lps/blob/7e9bd3b20dd52663afca04ca743808b1d6a9521f/contracts/3clp/Gyro3CLPMath.sol#L118
// func (l *gyro3CLPMath) _calculateCubicStartingPoint(
// 	a,
// 	mb,
// 	mc *uint256.Int,
// ) (*uint256.Int, *uint256.Int, error) {

// 	radic := new(uint256.Int).Add(
// 		math.GyroFixedPoint.MulUpU(mb, mb),
// 		math.GyroFixedPoint.MulUpU(a, new(uint256.Int).Mul(mc, number.Number_3)),
// 	)

// 	tmp, err := math.GyroPoolMath.Sqrt(radic, number.Number_5)
// 	if err != nil {
// 		return nil, nil, err
// 	}

// 	lplus, err := math.GyroFixedPoint.DivUpU(
// 		new(uint256.Int).Add(mb, tmp),
// 		new(uint256.Int).Mul(a, number.Number_3),
// 	)
// 	if err != nil {
// 		return nil, nil, err
// 	}

// 	alpha := new(uint256.Int).Sub(math.GyroFixedPoint.ONE, a)

// 	var l0 *uint256.Int
// 	if alpha.Lt(number_0_5e18) {
// 		l0 = math.GyroFixedPoint.MulUpU(lplus, number_2e18)
// 	} else {
// 		l0 = math.GyroFixedPoint.MulUpU(lplus, number_1_5e18)
// 	}

// 	l_lower := math.GyroFixedPoint.MulUpU(lplus, l._L_VS_LPLUS_MIN)

// 	return l_lower, l0, nil
// }

// func (l *gyro3CLPMath) _runNewtonIteration(
// 	mb,
// 	mc,
// 	md,
// 	root3Alpha,
// 	l_lower,
// 	rootEst *uint256.Int,
// ) *uint256.Int {
// 	deltaAbsPrev := number.Zero

// 	for iteration := 0; iteration < 255; iteration++ {

// 	}
// }

// func (l *gyro3CLPMath) _calcNewtonDelta(
// 	mb,
// 	mc,
// 	md,
// 	root3Alpha,
// 	l_lower,
// 	rootEst *uint256.Int,
// ) (*uint256.Int, bool, error) {
// 	if rootEst.Gt(l._L_MAX) {
// 		return nil, false, ErrInvariantTooLarge
// 	}

// 	if rootEst.Lt(l_lower) {
// 		return nil, false, ErrInvariantUnderflow
// 	}

// 	rootEst2 := math.GyroFixedPoint.MulDownU(rootEst, rootEst)
// 	dfRootEst, err := math.GyroFixedPoint.MulDown(new(uint256.Int).Mul(rootEst, number.Number_3), rootEst)
// 	if err != nil {
// 		return nil, false, err
// 	}

// 	// dfRootEst = dfRootEst - dfRootEst.mulDownU(root3Alpha).mulDownU(root3Alpha).mulDownU(root3Alpha);
// 	dfRootEst = new(uint256.Int).Sub(
// 		dfRootEst,
// 		math.GyroFixedPoint.MulDownU(
// 			math.GyroFixedPoint.MulDownU(
// 				math.GyroFixedPoint.MulDownU(dfRootEst, root3Alpha),
// 				root3Alpha,
// 			),
// 			root3Alpha,
// 		),
// 	)

// 	// dfRootEst = dfRootEst - 2 * rootEst.mulDownU(mb) - mc;
// 	dfRootEst = new(uint256.Int).Sub(
// 		new(uint256.Int).Sub(
// 			dfRootEst,
// 			new(uint256.Int).Mul(number.Number_2, math.GyroFixedPoint.MulDownU(rootEst, mb)),
// 		),
// 		mc,
// 	)

// 	var deltaMinus, deltaPlus *uint256.Int

// 	if rootEst.Gt(l._L_THRESHOLD_SIMPLE_NUMERICS) {
// 		deltaMinus = math.GyroFixedPoint.MulDownLargeSmallU(rootEst2, rootEst)
// 		deltaMinus = new(uint256.Int).Sub(
// 			deltaMinus,
// 			math.GyroFixedPoint.MulDownLargeSmallU(
// 				math.GyroFixedPoint.MulDownLargeSmallU(
// 					math.GyroFixedPoint.MulDownLargeSmallU(deltaMinus, root3Alpha),
// 					root3Alpha,
// 				),
// 				root3Alpha,
// 			),
// 		)
// 		deltaMinus, err = math.GyroFixedPoint.DivDownLargeU(deltaMinus, dfRootEst)
// 		if err != nil {
// 			return nil, false, err
// 		}

// 		deltaPlus := math.GyroFixedPoint.MulDownLargeSmallU(rootEst2, mb)

// 		deltaPlus = new(uint256.Int).Add(
// 			deltaPlus,
// 			math.GyroFixedPoint.MulDownU(mc, rootEst),
// 		)

// 		deltaPlus, err = math.GyroFixedPoint.DivDownLargeU_2(deltaPlus, dfRootEst, number_1e12, number_1e16)
// 		if err != nil {
// 			return nil, false, err
// 		}

// 		tmp, err := math.GyroFixedPoint.DivDownU(md, dfRootEst)
// 		if err != nil {
// 			return nil, false, err
// 		}

// 		deltaPlus = new(uint256.Int).Add(
// 			deltaPlus,
// 			tmp,
// 		)
// 	} else {
// 		deltaMinus = math.GyroFixedPoint.MulDownU(rootEst2, rootEst)
// 		deltaMinus = new(uint256.Int).Sub(
// 			deltaMinus,
// 			math.GyroFixedPoint.MulDownU(
// 				math.GyroFixedPoint.MulDownU(
// 					math.GyroFixedPoint.MulDownU(deltaMinus, root3Alpha),
// 					root3Alpha,
// 				),
// 				root3Alpha,
// 			),
// 		)

// 		deltaMinus, err = math.GyroFixedPoint.DivDownU(deltaMinus, dfRootEst)
// 		if err != nil {
// 			return nil, false, err
// 		}

// 		deltaPlus = math.GyroFixedPoint.MulDownU(rootEst2, mb)
// 		deltaPlus, err = math.GyroFixedPoint.DivDownU(
// 			new(uint256.Int).Add(deltaPlus, math.GyroFixedPoint.MulDownU(rootEst, mc)),
// 			dfRootEst,
// 		)
// 		if err != nil {
// 			return nil, false, err
// 		}

// 		tmp, err := math.GyroFixedPoint.DivDownU(md, dfRootEst)
// 		if err != nil {
// 			return nil, false, err
// 		}

// 		deltaPlus = new(uint256.Int).Add(
// 			deltaPlus,
// 			tmp,
// 		)
// 	}

// 	deltaIsPos := !deltaPlus.Lt(deltaMinus)

// 	var deltaAbs *uint256.Int

// 	if deltaIsPos {
// 		deltaAbs = new(uint256.Int).Sub(deltaPlus, deltaMinus)
// 	} else {
// 		deltaAbs = new(uint256.Int).Sub(deltaMinus, deltaPlus)
// 	}

// 	return deltaAbs, deltaIsPos, nil
// }
