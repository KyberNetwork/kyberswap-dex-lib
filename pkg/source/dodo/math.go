package dodo

import (
	"errors"
	"math/big"
)

// https://github.com/DODOEX/contractV2/blob/1c8d393ae1ed7a9c7effeceb58a6db4579637e8e/contracts/DODOPrivatePool/impl/DPPTrader.sol#L201
func QuerySellBase(amount *big.Float, state *PoolSimulatorState) (result *big.Float, mtFee *big.Float, err error) {
	if state.RStatus == rStatusOne {
		result, err = ROneSellBase(amount, state)
		if err != nil {
			return nil, nil, err
		}
	} else if state.RStatus == rStatusAboveOne {
		backToOnePayBase := new(big.Float).Sub(state.B0, state.B)
		backToOneReceiveQuote := new(big.Float).Sub(state.Q, state.Q0)

		if amount.Cmp(backToOnePayBase) < 0 {
			result, err = RAboveSellBase(amount, state)
			if err != nil {
				return nil, nil, err
			}

			if result.Cmp(backToOneReceiveQuote) > 0 {
				result = backToOneReceiveQuote
			}
		} else if amount.Cmp(backToOnePayBase) == 0 {
			result = backToOneReceiveQuote
		} else {
			rOneSellBase, err := ROneSellBase(new(big.Float).Sub(amount, backToOnePayBase), state)
			if err != nil {
				return nil, nil, err
			}
			result = new(big.Float).Add(backToOneReceiveQuote, rOneSellBase)
		}
	} else {
		result, err = RBelowSellBase(amount, state)
		if err != nil {
			return nil, nil, err
		}
	}

	mtFee = new(big.Float).Mul(result, state.mtFeeRate)
	lpFee := new(big.Float).Mul(result, state.lpFeeRate)

	result = new(big.Float).Sub(new(big.Float).Sub(result, mtFee), lpFee)

	return result, mtFee, nil
}

// https://github.com/DODOEX/contractV2/blob/1c8d393ae1ed7a9c7effeceb58a6db4579637e8e/contracts/DODOPrivatePool/impl/DPPTrader.sol#L223
func QuerySellQuote(amount *big.Float, state *PoolSimulatorState) (result *big.Float, mtFee *big.Float, err error) {
	if state.RStatus == rStatusOne {
		result, err = ROneSellQuote(amount, state)
		if err != nil {
			return nil, nil, err
		}
	} else if state.RStatus == rStatusAboveOne {
		result, err = RAboveSellQuote(amount, state)
		if err != nil {
			return nil, nil, err
		}
	} else {
		backToOneReceiveBase := new(big.Float).Sub(state.B, state.B0)
		backToOnePayQuote := new(big.Float).Sub(state.Q0, state.Q)

		if amount.Cmp(backToOnePayQuote) < 0 {
			result, err = RBelowSellQuote(amount, state)
			if err != nil {
				return nil, nil, err
			}

			if result.Cmp(backToOneReceiveBase) > 0 {
				result = backToOneReceiveBase
			}
		} else if amount.Cmp(backToOnePayQuote) == 0 {
			result = backToOneReceiveBase
		} else {
			rOneSellQuote, err := ROneSellQuote(new(big.Float).Sub(amount, backToOnePayQuote), state)
			if err != nil {
				return nil, nil, err
			}
			result = new(big.Float).Add(backToOneReceiveBase, rOneSellQuote)
		}
	}

	mtFee = new(big.Float).Mul(result, state.mtFeeRate)
	lpFee := new(big.Float).Mul(result, state.lpFeeRate)

	result = new(big.Float).Sub(new(big.Float).Sub(result, mtFee), lpFee)

	return result, mtFee, nil
}

func ROneSellBase(amount *big.Float, state *PoolSimulatorState) (result *big.Float, err error) {
	result, err = solveQuadraticFunctionForTrade(state.Q0, state.Q0, amount, state.OraclePrice, state.k)

	return
}

func ROneSellQuote(amount *big.Float, state *PoolSimulatorState) (result *big.Float, err error) {
	result, err = solveQuadraticFunctionForTrade(
		state.B0, state.B0, amount, new(big.Float).Quo(big.NewFloat(1), state.OraclePrice), state.k,
	)
	return
}

func RAboveSellBase(amount *big.Float, state *PoolSimulatorState) (result *big.Float, err error) {
	result, err = integrate(state.B0, new(big.Float).Add(state.B, amount), state.B, state.OraclePrice, state.k)
	return
}

func RAboveSellQuote(amount *big.Float, state *PoolSimulatorState) (result *big.Float, err error) {
	result, err = solveQuadraticFunctionForTrade(
		state.B0, state.B, amount, new(big.Float).Quo(big.NewFloat(1), state.OraclePrice), state.k,
	)
	return
}

func RBelowSellQuote(amount *big.Float, state *PoolSimulatorState) (result *big.Float, err error) {
	result, err = integrate(
		state.Q0, new(big.Float).Add(state.Q, amount), state.Q, new(big.Float).Quo(big.NewFloat(1), state.OraclePrice),
		state.k,
	)
	return
}

func RBelowSellBase(amount *big.Float, state *PoolSimulatorState) (result *big.Float, err error) {
	result, err = solveQuadraticFunctionForTrade(state.Q0, state.Q, amount, state.OraclePrice, state.k)
	return
}

func integrate(V0, V1, V2, i, k *big.Float) (*big.Float, error) {
	if V0.Cmp(big.NewFloat(0)) <= 0 {
		return nil, errors.New("TARGET_IS_ZERO")
	}

	fairAmount := new(big.Float).Mul(i, new(big.Float).Sub(V1, V2))

	if k.Cmp(big.NewFloat(0)) == 0 {
		return fairAmount, nil
	}

	penalty := new(big.Float).Mul(new(big.Float).Quo(new(big.Float).Quo(new(big.Float).Mul(V0, V0), V1), V2), k)
	return new(big.Float).Mul(fairAmount, new(big.Float).Add(new(big.Float).Sub(big.NewFloat(1), k), penalty)), nil
}

func solveQuadraticFunctionForTrade(V0, V1, delta, i, k *big.Float) (*big.Float, error) {
	if V0.Cmp(big.NewFloat(0)) <= 0 {
		return big.NewFloat(0), errors.New("TARGET_IS_ZERO")
	}

	if delta.Cmp(big.NewFloat(0)) == 0 {
		return delta, nil
	}

	if k.Cmp(big.NewFloat(0)) == 0 {
		if new(big.Float).Mul(delta, i).Cmp(V1) == 1 {
			return V1, nil
		} else {
			return new(big.Float).Mul(delta, i), nil
		}
	}

	if k.Cmp(big.NewFloat(1)) == 0 {
		tmp := new(big.Float).Quo(new(big.Float).Mul(new(big.Float).Mul(i, delta), V1), new(big.Float).Mul(V0, V0))
		result := new(big.Float).Quo(new(big.Float).Mul(V1, tmp), new(big.Float).Add(tmp, big.NewFloat(1)))

		return result, nil
	}

	part2 := new(big.Float).Add(
		new(big.Float).Mul(new(big.Float).Quo(new(big.Float).Mul(k, V0), V1), V0), new(big.Float).Mul(i, delta),
	)
	bAbs := new(big.Float).Mul(new(big.Float).Sub(big.NewFloat(1), k), V1)

	var bSig bool

	if bAbs.Cmp(part2) >= 0 {
		bAbs = new(big.Float).Sub(bAbs, part2)
		bSig = false
	} else {
		bAbs = new(big.Float).Sub(part2, bAbs)
		bSig = true
	}

	squareRoot := new(big.Float).Mul(
		new(big.Float).Mul(
			new(big.Float).Mul(
				new(big.Float).Mul(
					big.NewFloat(4), new(big.Float).Sub(big.NewFloat(1), k),
				), k,
			), V0,
		), V0,
	)

	squareRoot = new(big.Float).Add(new(big.Float).Mul(bAbs, bAbs), squareRoot)
	squareRoot.Sqrt(squareRoot)

	denominator := new(big.Float).Mul(big.NewFloat(2), new(big.Float).Sub(big.NewFloat(1), k))
	var numerator *big.Float

	if bSig {
		numerator = new(big.Float).Sub(squareRoot, bAbs)
	} else {
		numerator = new(big.Float).Add(bAbs, squareRoot)
	}

	return new(big.Float).Sub(V1, new(big.Float).Quo(numerator, denominator)), nil
}

//func solveQuadraticFunctionForTarget(V1, delta, i, k *big.Float) *big.Float {
//	if V1.Cmp(big.NewFloat(0)) == 0 {
//		return big.NewFloat(0)
//	}
//	if k.Cmp(big.NewFloat(0)) == 0 {
//		return new(big.Float).Add(V1, new(big.Float).Mul(i, delta))
//	}
//
//	ki := new(big.Float).Mul(new(big.Float).Mul(k, big.NewFloat(4)), i)
//	var sqrt big.Float
//
//	if ki.Cmp(big.NewFloat(0)) == 0 {
//		sqrt = *constant.BoneFloat
//	} else if new(big.Float).Quo(new(big.Float).Mul(ki, delta), ki).Cmp(delta) == 0 {
//		sqrt.Sqrt(
//			new(big.Float).Add(
//				new(big.Float).Quo(new(big.Float).Mul(ki, delta), V1), constant.TenPowDecimals(36),
//			),
//		)
//	} else {
//		sqrt.Sqrt(
//			new(big.Float).Add(
//				new(big.Float).Mul(new(big.Float).Quo(ki, V1), delta), constant.TenPowDecimals(36),
//			),
//		)
//	}
//
//	premium := new(big.Float).Add(
//		new(big.Float).Quo(
//			new(big.Float).Sub(&sqrt, big.NewFloat(1)), new(big.Float).Mul(k, big.NewFloat(2)),
//		), big.NewFloat(1),
//	)
//
//	return new(big.Float).Mul(V1, premium)
//}

func UpdateStateSellBase(amountIn *big.Float, amountOut *big.Float, state *PoolSimulatorState) {
	// state.B = state.B + amountInF
	// state.Q = state.Q - outputAmountF
	state.B = new(big.Float).Add(state.B, amountIn)
	state.Q = new(big.Float).Sub(state.Q, amountOut)

	if state.RStatus == rStatusOne {
		state.RStatus = rStatusBelowOne
	} else if state.RStatus == rStatusAboveOne {
		backToOnePayBase := new(big.Float).Sub(state.B0, state.B)

		if amountIn.Cmp(backToOnePayBase) < 0 {
			state.RStatus = rStatusAboveOne
		} else if amountIn.Cmp(backToOnePayBase) == 0 {
			state.RStatus = rStatusOne
		} else {
			state.RStatus = rStatusBelowOne
		}
	} else {
		state.RStatus = rStatusBelowOne
	}
}

func UpdateStateSellQuote(amountIn *big.Float, amountOut *big.Float, state *PoolSimulatorState) {
	// state.B = state.B - amountOutF
	// state.Q = state.Q + amountInF
	state.B = new(big.Float).Sub(state.B, amountOut)
	state.Q = new(big.Float).Add(state.Q, amountIn)

	if state.RStatus == rStatusOne {
		state.RStatus = rStatusAboveOne
	} else if state.RStatus == rStatusAboveOne {
		state.RStatus = rStatusAboveOne
	} else {
		backToOnePayQuote := new(big.Float).Sub(state.Q0, state.Q)

		if amountIn.Cmp(backToOnePayQuote) < 0 {
			state.RStatus = rStatusBelowOne
		} else if amountIn.Cmp(backToOnePayQuote) == 0 {
			state.RStatus = rStatusOne
		} else {
			state.RStatus = rStatusAboveOne
		}
	}
}
