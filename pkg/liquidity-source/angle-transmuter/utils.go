package angletransmuter

import (
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

var (
	BASE_9  = u256.TenPow(9)
	BASE_12 = u256.TenPow(12)
	BASE_18 = u256.TenPow(18)

	MAX_BURN_FEE = uint256.NewInt(999_000_000)

	newBASE18 = func() *uint256.Int {
		return new(uint256.Int).Set(BASE_18)
	}
)

func _quoteMintExactInput(
	oracleValue *uint256.Int,
	amountIn *uint256.Int,
	fees Fees,
	stablecoinsIssued *uint256.Int,
	otherStablecoinSupply *uint256.Int,
	stablecoinCap *uint256.Int,
	collatDecimal uint8,
) (*uint256.Int, error) {
	amountOut := new(uint256.Int).Mul(oracleValue, amountIn)
	convertDecimalTo(amountOut, 18+collatDecimal, 18)
	amountOut, err := _quoteFees(fees, MintExactInput, amountOut, stablecoinsIssued, otherStablecoinSupply)
	if err != nil {
		return nil, err
	}
	if stablecoinCap != nil && stablecoinCap.Sign() >= 0 && new(uint256.Int).Add(amountOut, stablecoinsIssued).Gt(stablecoinCap) {
		return nil, ErrInvalidSwap
	}
	return amountOut, nil
}

// nolint
func _quoteMintExactOutput(
	oracleValue *uint256.Int,
	amountOut *uint256.Int,
	fees Fees,
	stablecoinsIssued *uint256.Int,
	otherStablecoinSupply *uint256.Int,
	stablecoinCap *uint256.Int,
) (*uint256.Int, error) {

	if stablecoinCap != nil && stablecoinCap.Sign() >= 0 && new(uint256.Int).Add(amountOut, stablecoinsIssued).Gt(stablecoinCap) {
		return nil, ErrInvalidSwap
	}
	amountIn, err := _quoteFees(fees, MintExactOutput, amountOut, stablecoinsIssued, otherStablecoinSupply)
	if err != nil {
		return nil, err
	}
	amountIn.Div(amountIn, oracleValue)
	return amountIn, nil
}

// nolint
func _quoteBurnExactOutput(
	oracleValue *uint256.Int,
	ratio *uint256.Int,
	amountOut *uint256.Int,
	fees Fees,
	stablecoinsIssued *uint256.Int,
	otherStablecoinSupply *uint256.Int,
) (*uint256.Int, error) {
	amountIn, overflow := new(uint256.Int).MulDivOverflow(amountOut, oracleValue, ratio)
	if overflow {
		return nil, ErrMulOverflow
	}
	amountIn, err := _quoteFees(fees, BurnExactOutput, amountIn, stablecoinsIssued, otherStablecoinSupply)
	if err != nil {
		return nil, err
	}
	return amountIn, nil
}

func _quoteBurnExactInput(
	oracleValue *uint256.Int,
	ratio *uint256.Int,
	amountIn *uint256.Int,
	fees Fees,
	stablecoinsIssued *uint256.Int,
	otherStablecoinSupply *uint256.Int,
	collatDecimal uint8,
) (*uint256.Int, error) {
	amountOut, err := _quoteFees(fees, BurnExactInput, amountIn, stablecoinsIssued, otherStablecoinSupply)
	if err != nil {
		return nil, err
	}
	_, overflow := amountOut.MulDivOverflow(amountOut, ratio, oracleValue)
	if overflow {
		return nil, ErrMulOverflow
	}
	convertDecimalTo(amountOut, 18, collatDecimal)
	return amountOut, nil
}

func _quoteFees(
	fees Fees,
	quoteType QuoteType,
	amountStable *uint256.Int,
	stablecoinsIssued *uint256.Int,
	otherStablecoinSupply *uint256.Int,
) (*uint256.Int, error) {
	var err error
	isMint := _isMint(quoteType)
	isExact := _isExact(quoteType)

	n := lo.Ternary(isMint, len(fees.XFeeMint), len(fees.XFeeBurn))

	currentExposure := new(uint256.Int).Div(
		new(uint256.Int).Mul(stablecoinsIssued, BASE_9),
		new(uint256.Int).Add(otherStablecoinSupply, stablecoinsIssued),
	)

	amount := uint256.NewInt(0)

	i := findLowerBound(isMint,
		lo.Ternary(isMint, lo.Map(fees.XFeeMint, func(item *uint256.Int, index int) *uint256.Int {
			return new(uint256.Int).Mul(item, BASE_9)
		}), lo.Map(fees.XFeeBurn, func(item *uint256.Int, index int) *uint256.Int {
			return new(uint256.Int).Mul(item, BASE_9)
		})),
		BASE_9,
		currentExposure,
	)
	var lowerExposure, upperExposure, lowerFees, upperFees *uint256.Int
	amountToNextBreakPoint := new(uint256.Int)
	for i < n-1 {
		if isMint {
			lowerExposure = fees.XFeeMint[i]
			upperExposure = fees.XFeeMint[i+1]
			lowerFees = fees.YFeeMint[i]
			upperFees = fees.YFeeMint[i+1]
			amountToNextBreakPoint.Sub(
				new(uint256.Int).Div(
					new(uint256.Int).Mul(otherStablecoinSupply, upperExposure),
					new(uint256.Int).Sub(BASE_9, upperExposure),
				),
				stablecoinsIssued,
			)
		} else {
			lowerExposure = fees.XFeeBurn[i]
			upperExposure = fees.XFeeBurn[i+1]
			lowerFees = fees.YFeeBurn[i]
			upperFees = fees.YFeeBurn[i+1]

			amountToNextBreakPoint.Sub(
				stablecoinsIssued,
				new(uint256.Int).Div(
					new(uint256.Int).Mul(otherStablecoinSupply, upperExposure),
					new(uint256.Int).Sub(BASE_9, upperExposure),
				),
			)
		}
		currentFees, amountFromPrevBreakPoint := new(uint256.Int), new(uint256.Int)
		if new(uint256.Int).Mul(lowerExposure, BASE_9).Eq(currentExposure) {
			currentFees = lowerFees
		} else if lowerFees.Eq(upperFees) {
			currentFees = lowerFees
		} else {
			if isMint {
				amountFromPrevBreakPoint.Sub(
					stablecoinsIssued,
					new(uint256.Int).Div(
						new(uint256.Int).Mul(otherStablecoinSupply, lowerExposure),
						new(uint256.Int).Sub(BASE_9, lowerExposure),
					),
				)
			} else {
				amountFromPrevBreakPoint.Sub(
					new(uint256.Int).Div(
						new(uint256.Int).Mul(otherStablecoinSupply, lowerExposure),
						new(uint256.Int).Sub(BASE_9, lowerExposure),
					),
					stablecoinsIssued,
				)
			}
			currentFees.Add(
				lowerFees,
				new(uint256.Int).Div(
					new(uint256.Int).Mul(new(uint256.Int).Sub(upperFees, lowerFees), amountFromPrevBreakPoint),
					new(uint256.Int).Add(amountToNextBreakPoint, amountFromPrevBreakPoint),
				),
			)
		}

		amountToNextBreakPointNormalizer := new(uint256.Int)
		if isExact {
			amountToNextBreakPointNormalizer.Set(amountToNextBreakPoint)
		} else if isMint {
			amountToNextBreakPointNormalizer, err = _invertFeeMint(amountToNextBreakPoint, new(uint256.Int).Div(new(uint256.Int).Add(upperFees, currentFees), uint256.NewInt(2)))
			if err != nil {
				return nil, err
			}
		} else {
			fee := new(uint256.Int)
			fee.Add(upperFees, currentFees).Div(fee, u256.U2)
			amountToNextBreakPointNormalizer, err = _applyFeeBurn(amountToNextBreakPoint, fee)
			if err != nil {
				return nil, err
			}
		}
		if !amountToNextBreakPointNormalizer.Lt(amountStable) {
			midFee := new(uint256.Int)
			if isExact {
				temp := new(uint256.Int)
				midFee.Add(
					currentFees,
					temp.Div(
						temp.Mul(amountStable, temp.Sub(upperFees, currentFees)),
						new(uint256.Int).Mul(amountToNextBreakPointNormalizer, u256.U2)),
				)
			} else {
				ac4 := new(uint256.Int)
				ac4.Div(
					ac4.Mul(
						BASE_9,
						ac4.Mul(
							ac4.Mul(u256.U2, amountStable),
							new(uint256.Int).Sub(upperFees, currentFees),
						),
					),
					amountToNextBreakPoint,
				)
				midFee.Add(midFee, midFee.Add(BASE_9, currentFees))
				if isMint {
					midFee.Div(
						midFee.Sub(
							midFee.Add(
								midFee.Sqrt(
									midFee.Add(
										midFee.Exp(
											midFee.Add(BASE_9, currentFees),
											u256.U2,
										), ac4,
									),
								),
								currentFees,
							),
							BASE_9,
						),
						u256.U2,
					)
				} else {
					baseMinusCurrentSquared := new(uint256.Int)
					baseMinusCurrentSquared.Exp(baseMinusCurrentSquared.Sub(BASE_9, currentFees), u256.U2)
					// Mathematically, this condition is always verified, but rounding errors may make this
					// mathematical invariant break, in which case we consider that the square root is null
					if baseMinusCurrentSquared.Lt(ac4) {
						midFee.Div(
							midFee.Add(currentFees, BASE_9),
							u256.U2,
						)
					} else {
						midFee.Div(
							midFee.Sub(
								midFee.Add(currentFees, BASE_9),
								new(uint256.Int).Sqrt(new(uint256.Int).Sub(baseMinusCurrentSquared, ac4)),
							),
							u256.U2,
						)
					}
				}
			}
			res, err := _computeFee(quoteType, amountStable, midFee)
			if err != nil {
				return nil, err
			}
			return new(uint256.Int).Add(amount, res), nil
		} else {
			amountStable.Sub(amountStable, amountToNextBreakPointNormalizer)
			var temp *uint256.Int
			if !isExact {
				temp = amountToNextBreakPoint
			} else if isMint {
				temp, err = _invertFeeMint(amountToNextBreakPoint, new(uint256.Int).Div(new(uint256.Int).Add(upperFees, currentFees), u256.U2))
				if err != nil {
					return nil, err
				}
			} else {
				temp, err = _applyFeeBurn(amountToNextBreakPoint, new(uint256.Int).Div(new(uint256.Int).Add(upperFees, currentFees), u256.U2))
				if err != nil {
					return nil, err
				}
			}
			amount.Add(amount, temp)
			currentExposure.Mul(upperExposure, BASE_9)
			i++
			if isMint {
				stablecoinsIssued.Add(stablecoinsIssued, amountToNextBreakPoint)
			} else {
				stablecoinsIssued.Sub(stablecoinsIssued, amountToNextBreakPoint)
			}
		}
	}
	fee, err := _computeFee(quoteType, amountStable, lo.TernaryF(isMint,
		func() *uint256.Int { return fees.YFeeMint[n-1] }, func() *uint256.Int { return fees.YFeeBurn[n-1] },
	))
	if err != nil {
		return nil, err
	}
	amount.Add(amount, fee)
	return amount, nil
}

func _isMint(quoteType QuoteType) bool {
	return quoteType == MintExactInput || quoteType == MintExactOutput
}

func _isExact(quoteType QuoteType) bool {
	return quoteType == MintExactOutput || quoteType == BurnExactInput
}

func findLowerBound(
	increasingArray bool,
	array []*uint256.Int,
	_ *uint256.Int,
	element *uint256.Int,
) int {
	if len(array) == 0 {
		return 0
	}
	low := 1
	high := len(array)

	if (increasingArray && !array[high-1].Gt(element)) ||
		(!increasingArray && !array[high-1].Lt(element)) {
		return high - 1
	}

	for low < high {
		mid := (low + high) / 2

		if increasingArray && array[mid].Gt(element) ||
			(!increasingArray && array[mid].Lt(element)) {
			high = mid
		} else {
			low = mid + 1
		}
	}

	return low - 1
}

func _applyFeeMint(amountIn, fees *uint256.Int) (*uint256.Int, error) {
	res := new(uint256.Int)
	if fees.Sign() >= 0 {
		// Consider that if fees are above `BASE_12` this is equivalent to infinite fees
		if !fees.Lt(BASE_12) {
			return nil, ErrInvalidSwap
		}
		// (amountIn * BASE_9) / (BASE_9 + castedFees);
		res.Div(
			res.Mul(amountIn, BASE_9),
			new(uint256.Int).Add(BASE_9, fees),
		)
		return res, nil
	}
	// (amountIn * BASE_9) / (BASE_9 - Math.abs(-fees));
	res.Div(
		res.Mul(amountIn, BASE_9),
		new(uint256.Int).Sub(BASE_9, new(uint256.Int).Abs(fees)),
	)
	return res, nil
}

func _invertFeeMint(amountOut, fees *uint256.Int) (*uint256.Int, error) {
	res := new(uint256.Int)
	if fees.Sign() >= 0 {
		// Consider that if fees are above `BASE_12` this is equivalent to infinite fees
		if !fees.Lt(BASE_12) {
			return nil, ErrInvalidSwap
		}
		// (amountOut * (BASE_9 + castedFees)) / BASE_9;
		res.Div(
			res.Mul(
				amountOut,
				new(uint256.Int).Add(BASE_9, fees),
			),
			BASE_9,
		)
		return res, nil
	}
	// (amountOut * (BASE_9 - Math.abs(-fees))) / BASE_9;
	res.Div(
		res.Mul(
			amountOut,
			new(uint256.Int).Sub(BASE_9, new(uint256.Int).Abs(fees)),
		),
		BASE_9,
	)
	return res, nil
}

func _applyFeeBurn(amountIn, fees *uint256.Int) (*uint256.Int, error) {
	res := new(uint256.Int)
	if fees.Sign() >= 0 {
		if !fees.Lt(MAX_BURN_FEE) {
			return nil, ErrInvalidSwap
		}
		// ((BASE_9 - castedFees) * amountIn) / BASE_9;
		res.Div(
			res.Mul(new(uint256.Int).Sub(BASE_9, fees), amountIn),
			BASE_9,
		)
		return res, nil
	}

	// ((BASE_9 + Math.abs(-fees)) * amountIn) / BASE_9;
	res.Div(
		res.Mul(new(uint256.Int).Add(BASE_9, new(uint256.Int).Abs(fees)), amountIn),
		BASE_9,
	)
	return res, nil
}

func _invertFeeBurn(amountOut, fees *uint256.Int) (*uint256.Int, error) {
	res := new(uint256.Int)
	if fees.Sign() >= 0 {
		if !fees.Lt(MAX_BURN_FEE) {
			return nil, ErrInvalidSwap
		}
		// (amountOut * BASE_9) / (BASE_9 - castedFees);
		res.Div(
			res.Mul(amountOut, BASE_9),
			new(uint256.Int).Sub(BASE_9, fees),
		)
		return res, nil
	}
	// (amountOut * BASE_9) / (BASE_9 + Math.abs(-fees));
	res.Div(
		res.Mul(amountOut, BASE_9),
		new(uint256.Int).Add(BASE_9, new(uint256.Int).Abs(fees)),
	)
	return res, nil
}

func _computeFee(
	quoteType QuoteType,
	amount *uint256.Int,
	fees *uint256.Int,
) (*uint256.Int, error) {
	if quoteType == MintExactInput {
		return _applyFeeMint(amount, fees)
	}
	if quoteType == MintExactOutput {
		return _invertFeeMint(amount, fees)
	}
	if quoteType == BurnExactInput {
		return _applyFeeBurn(amount, fees)
	}
	return _invertFeeBurn(amount, fees)
}

func convertDecimalTo(amount *uint256.Int, fromDecimals, toDecimals uint8) *uint256.Int {
	if fromDecimals > toDecimals {
		return amount.Div(amount, u256.TenPow(fromDecimals-toDecimals))
	} else if fromDecimals < toDecimals {
		return amount.Mul(amount, u256.TenPow(toDecimals-fromDecimals))
	}
	return amount
}
