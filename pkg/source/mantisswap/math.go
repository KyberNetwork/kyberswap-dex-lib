package mantisswap

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"math/big"
)

func GetAmountOut(
	from, to string,
	amount *big.Int,
	state *PoolState,
) (*big.Int, error) {
	if state.Paused {
		return nil, ErrPoolIsPaused
	}
	fromLp, ok := state.LPs[from]
	if !ok {
		return nil, ErrNoLp
	}
	toLp, ok := state.LPs[to]
	if !ok {
		return nil, ErrNoLp
	}

	toAmount, lpAmount, treasuryFees, err := getSwapAmount(fromLp, toLp, amount, false, big.NewInt(0), big.NewInt(0), state)
	if err != nil {
		return nil, err
	}

	if toAmount.Cmp(bignumber.ZeroBI) <= 0 {
		return nil, ErrZeroAmount
	}

	if err := updateAssetLiability(amount, true, big.NewInt(0), false, false, fromLp); err != nil {
		return nil, err
	}
	if err := updateAssetLiability(new(big.Int).Add(toAmount, treasuryFees), false, lpAmount, true, false, toLp); err != nil {
		return nil, err
	}

	return toAmount, nil
}

func getSwapAmount(
	fromLp, toLp *LP,
	amount *big.Int,
	isOneTap bool,
	fromAsset, fromLiability *big.Int,
	state *PoolState,
) (*big.Int, *big.Int, *big.Int, error) {
	if !state.SwapAllowed {
		return nil, nil, nil, ErrSwapNotAllowed
	}
	adjustedToAmount := new(big.Int).Div(
		new(big.Int).Mul(amount, bignumber.TenPowInt(toLp.Decimals)),
		bignumber.TenPowInt(fromLp.Decimals),
	)
	if !isOneTap {
		fromAsset = new(big.Int).Set(fromLp.Asset)
		fromLiability = new(big.Int).Set(fromLp.Liability)
	}
	toAsset := new(big.Int).Set(toLp.Asset)
	toLiability := new(big.Int).Set(toLp.Liability)
	if toAsset.Cmp(adjustedToAmount) < 0 {
		return nil, nil, nil, ErrLowAsset
	}

	swapSlippageFactor, err := getSwapSlippageFactor(
		new(big.Int).Div(new(big.Int).Mul(fromAsset, One18), fromLiability),
		new(big.Int).Div(new(big.Int).Mul(new(big.Int).Add(fromAsset, amount), One18), fromLiability),
		new(big.Int).Div(new(big.Int).Mul(toAsset, One18), toLiability),
		new(big.Int).Div(new(big.Int).Mul(new(big.Int).Sub(toAsset, adjustedToAmount), One18), toLiability),
		state,
	)
	if err != nil {
		return nil, nil, nil, err
	}

	toAmount := new(big.Int).Div(new(big.Int).Mul(adjustedToAmount, swapSlippageFactor), One18)

	nlr := getNetLiquidityRatio(state)
	swapFeeRatio := getSwapFeeRatio(nlr, state)
	feeAmount := new(big.Int).Div(new(big.Int).Mul(toAmount, swapFeeRatio), bignumber.TenPowInt(6))

	lpAmount := new(big.Int).Set(bignumber.ZeroBI)
	if !isOneTap {
		lpAmount = new(big.Int).Div(new(big.Int).Mul(toAmount, state.LpRatio), bignumber.TenPowInt(6))
	}
	toAmount = new(big.Int).Sub(toAmount, new(big.Int).Add(feeAmount, lpAmount))

	treasuryFees := new(big.Int).Div(
		new(big.Int).Mul(feeAmount, getTreasuryRatio(nlr)),
		big.NewInt(1e6),
	)

	return toAmount, lpAmount, treasuryFees, nil
}

func getSwapSlippageFactor(oldFromLR, newFromLR, oldToLR, newToLR *big.Int, state *PoolState) (*big.Int, error) {
	negativeFromSlippage := big.NewInt(0)
	negativeToSlippage := big.NewInt(0)
	basisPoint := bignumber.TenPowInt(18)
	if newFromLR.Cmp(oldFromLR) > 0 {
		oldFromLRSlippage, err := GetSlippage(oldFromLR, state)
		if err != nil {
			return nil, err
		}
		newFromLRSlippage, err := GetSlippage(newFromLR, state)
		if err != nil {
			return nil, err
		}
		negativeFromSlippage = new(big.Int).Div(
			new(big.Int).Mul(
				new(big.Int).Sub(oldFromLRSlippage, newFromLRSlippage),
				basisPoint,
			),
			new(big.Int).Sub(newFromLR, oldFromLR),
		)
	}
	if oldToLR.Cmp(newToLR) > 0 {
		newToLRSlippage, err := GetSlippage(newToLR, state)
		if err != nil {
			return nil, err
		}
		oldToLRSlippage, err := GetSlippage(oldToLR, state)
		if err != nil {
			return nil, err
		}
		negativeToSlippage = new(big.Int).Div(
			new(big.Int).Mul(
				new(big.Int).Sub(newToLRSlippage, oldToLRSlippage),
				basisPoint,
			),
			new(big.Int).Sub(oldToLR, newToLR),
		)
	}

	toFactorSigned := new(big.Int).Sub(new(big.Int).Add(basisPoint, negativeFromSlippage), negativeToSlippage)
	if toFactorSigned.Cmp(big.NewInt(2e18)) > 0 {
		toFactorSigned = big.NewInt(2e18)
	} else if toFactorSigned.Cmp(bignumber.ZeroBI) < 0 {
		toFactorSigned = big.NewInt(0)
	}

	return toFactorSigned, nil
}

func getSwapFeeRatio(nlr *big.Int, state *PoolState) *big.Int {
	var swapFee *big.Int
	if nlr.Cmp(new(big.Int).Mul(big.NewInt(96), bignumber.TenPowInt(16))) < 0 {
		swapFee = new(big.Int).Mul(bignumber.Four, state.BaseFee)
	} else if nlr.Cmp(One18) < 0 {
		swapFee = new(big.Int).Mul(bignumber.Two, state.BaseFee)
	} else {
		swapFee = new(big.Int).Set(state.BaseFee)
	}

	return new(big.Int).Sub(swapFee, state.LpRatio)
}

func GetSlippage(lr *big.Int, state *PoolState) (*big.Int, error) {
	if lr.Cmp(state.SlippageK) <= 0 {
		tmpExp, err := negativeExponential(new(big.Int).Mul(state.SlippageN, lr))
		if err != nil {
			return nil, err
		}
		return new(big.Int).Div(new(big.Int).Mul(state.SlippageA, tmpExp), big.NewInt(10)), nil
	} else if lr.Cmp(new(big.Int).Mul(bignumber.Two, state.SlippageK)) < 0 {
		tmpExp1, err := negativeExponential(
			new(big.Int).Mul(
				state.SlippageN,
				new(big.Int).Sub(new(big.Int).Mul(bignumber.Two, state.SlippageK), lr),
			))
		if err != nil {
			return nil, err
		}
		tmpExp2, err := negativeExponential(new(big.Int).Mul(state.SlippageN, state.SlippageK))
		if err != nil {
			return nil, err
		}
		tmpExp3, err := negativeExponential(new(big.Int).Mul(state.SlippageN, lr))
		if err != nil {
			return nil, err
		}

		return new(big.Int).Div(
			new(big.Int).Mul(
				state.SlippageA,
				new(big.Int).Sub(
					tmpExp1,
					new(big.Int).Mul(
						bignumber.Two,
						new(big.Int).Sub(tmpExp2, tmpExp3),
					),
				),
			),
			big.NewInt(10),
		), nil
	} else {
		tmpExp1, err := positiveExponential(
			new(big.Int).Mul(
				state.SlippageN,
				new(big.Int).Sub(lr, new(big.Int).Mul(bignumber.Two, state.SlippageK)),
			))
		if err != nil {
			return nil, err
		}
		tmpExp2, err := negativeExponential(new(big.Int).Mul(state.SlippageN, state.SlippageK))
		if err != nil {
			return nil, err
		}
		tmpExp3, err := negativeExponential(new(big.Int).Mul(state.SlippageN, lr))
		if err != nil {
			return nil, err
		}

		return new(big.Int).Div(
			new(big.Int).Mul(
				state.SlippageA,
				new(big.Int).Sub(
					tmpExp1,
					new(big.Int).Mul(
						bignumber.Two,
						new(big.Int).Sub(tmpExp2, tmpExp3),
					),
				),
			),
			big.NewInt(10),
		), nil
	}
}

func positiveExponential(x *big.Int) (*big.Int, error) {
	x = new(big.Int).Div(new(big.Int).Mul(x, One), One18)
	tmpExp, err := exp(x)
	if err != nil {
		return nil, err
	}
	return new(big.Int).Div(new(big.Int).Mul(tmpExp, One18), One), nil
}

func negativeExponential(x *big.Int) (*big.Int, error) {
	x = new(big.Int).Neg(new(big.Int).Div(new(big.Int).Mul(x, One), One18))
	expX, err := exp(x)
	if err != nil {
		return nil, err
	}
	return new(big.Int).Div(new(big.Int).Mul(expX, One18), One), nil
}

func exp(x *big.Int) (*big.Int, error) {
	maxPower := bignumber.NewBig10("2454971259878909886679")
	minPower := bignumber.NewBig10("-818323753292969962227")

	if x.Cmp(maxPower) > 0 {
		return nil, ErrLargerThanMaxPower
	}
	if x.Cmp(minPower) < 0 {
		return big.NewInt(0), nil
	}

	if x.Cmp(bignumber.ZeroBI) >= 0 {
		x = new(big.Int).Div(new(big.Int).Mul(x, One), Ln2)
	} else {
		x = new(big.Int).Div(new(big.Int).Mul(new(big.Int).Neg(x), One), Ln2)
		x = new(big.Int).Neg(x)
	}

	var shift, z *big.Int
	if x.Cmp(bignumber.ZeroBI) >= 0 {
		shift = new(big.Int).Div(x, One)
		z = new(big.Int).Mod(x, One)
	} else {
		shift = new(big.Int).Add(new(big.Int).Div(new(big.Int).Neg(x), One), bignumber.One)
		shift = new(big.Int).Neg(shift)
		z = new(big.Int).Sub(One, new(big.Int).Mod(new(big.Int).Neg(x), One))
	}

	zpow := new(big.Int).Set(z)
	result := new(big.Int).Set(One)
	result = new(big.Int).Add(result, new(big.Int).Div(new(big.Int).Mul(bignumber.NewBig("0xb17217f7d1cf79ab"), zpow), One))
	zpow = new(big.Int).Div(new(big.Int).Mul(zpow, z), One)
	result = new(big.Int).Add(result, new(big.Int).Div(new(big.Int).Mul(bignumber.NewBig("0x3d7f7bff058b1d50"), zpow), One))
	zpow = new(big.Int).Div(new(big.Int).Mul(zpow, z), One)
	result = new(big.Int).Add(result, new(big.Int).Div(new(big.Int).Mul(bignumber.NewBig("0xe35846b82505fc5"), zpow), One))
	zpow = new(big.Int).Div(new(big.Int).Mul(zpow, z), One)
	result = new(big.Int).Add(result, new(big.Int).Div(new(big.Int).Mul(bignumber.NewBig("0x276556df749cee5"), zpow), One))
	zpow = new(big.Int).Div(new(big.Int).Mul(zpow, z), One)
	result = new(big.Int).Add(result, new(big.Int).Div(new(big.Int).Mul(bignumber.NewBig("0x5761ff9e299cc4"), zpow), One))
	zpow = new(big.Int).Div(new(big.Int).Mul(zpow, z), One)
	result = new(big.Int).Add(result, new(big.Int).Div(new(big.Int).Mul(bignumber.NewBig("0xa184897c363c3"), zpow), One))
	zpow = new(big.Int).Div(new(big.Int).Mul(zpow, z), One)
	result = new(big.Int).Add(result, new(big.Int).Div(new(big.Int).Mul(bignumber.NewBig("0xffe5fe2c4586"), zpow), One))
	zpow = new(big.Int).Div(new(big.Int).Mul(zpow, z), One)
	result = new(big.Int).Add(result, new(big.Int).Div(new(big.Int).Mul(bignumber.NewBig("0x162c0223a5c8"), zpow), One))
	zpow = new(big.Int).Div(new(big.Int).Mul(zpow, z), One)
	result = new(big.Int).Add(result, new(big.Int).Div(new(big.Int).Mul(bignumber.NewBig("0x1b5253d395e"), zpow), One))
	zpow = new(big.Int).Div(new(big.Int).Mul(zpow, z), One)
	result = new(big.Int).Add(result, new(big.Int).Div(new(big.Int).Mul(bignumber.NewBig("0x1e4cf5158b"), zpow), One))
	zpow = new(big.Int).Div(new(big.Int).Mul(zpow, z), One)
	result = new(big.Int).Add(result, new(big.Int).Div(new(big.Int).Mul(bignumber.NewBig("0x1e8cac735"), zpow), One))
	zpow = new(big.Int).Div(new(big.Int).Mul(zpow, z), One)
	result = new(big.Int).Add(result, new(big.Int).Div(new(big.Int).Mul(bignumber.NewBig("0x1c3bd650"), zpow), One))
	zpow = new(big.Int).Div(new(big.Int).Mul(zpow, z), One)
	result = new(big.Int).Add(result, new(big.Int).Div(new(big.Int).Mul(bignumber.NewBig("0x1816193"), zpow), One))
	zpow = new(big.Int).Div(new(big.Int).Mul(zpow, z), One)
	result = new(big.Int).Add(result, new(big.Int).Div(new(big.Int).Mul(bignumber.NewBig("0x131496"), zpow), One))
	zpow = new(big.Int).Div(new(big.Int).Mul(zpow, z), One)
	result = new(big.Int).Add(result, new(big.Int).Div(new(big.Int).Mul(bignumber.NewBig("0xe1b7"), zpow), One))
	zpow = new(big.Int).Div(new(big.Int).Mul(zpow, z), One)
	result = new(big.Int).Add(result, new(big.Int).Div(new(big.Int).Mul(bignumber.NewBig("0x9c7"), zpow), One))

	if shift.Cmp(bignumber.ZeroBI) >= 0 {
		if new(big.Int).Rsh(result, uint(new(big.Int).Sub(big.NewInt(256), shift).Int64())).Cmp(bignumber.ZeroBI) > 0 {
			return new(big.Int).Sub(new(big.Int).Exp(bignumber.Two, big.NewInt(256), nil), bignumber.One), nil
		}
		return new(big.Int).Lsh(result, uint(shift.Int64())), nil
	}
	return new(big.Int).Rsh(result, uint(new(big.Int).Neg(shift).Int64())), nil
}

func updateAssetLiability(
	assetAmount *big.Int,
	assetIncrease bool,
	liabilityAmount *big.Int,
	liabilityIncrease, checkLimit bool,
	lp *LP,
) error {
	oldLiability := new(big.Int).Set(lp.Liability)

	if !checkLimit && new(big.Int).Add(oldLiability, liabilityAmount).Cmp(lp.LiabilityLimit) > 0 {
		return ErrLpLimitReach
	}

	if assetAmount.Cmp(bignumber.ZeroBI) > 0 {
		if assetIncrease {
			lp.Asset = new(big.Int).Add(lp.Asset, assetAmount)
		} else {
			if lp.Asset.Cmp(assetAmount) < 0 {
				return ErrBeNegative
			}
			lp.Asset = new(big.Int).Sub(lp.Asset, assetAmount)
		}
	}
	if liabilityAmount.Cmp(bignumber.ZeroBI) > 0 {
		if liabilityIncrease {
			lp.Liability = new(big.Int).Add(lp.Liability, liabilityAmount)
		} else {
			lp.Liability = new(big.Int).Sub(lp.Liability, liabilityAmount)
		}
	}

	return nil
}

func getNetLiquidityRatio(state *PoolState) *big.Int {
	totalAsset, totalLiability := getTotalAssetLiability(state)
	if totalLiability.Cmp(bignumber.ZeroBI) == 0 {
		return new(big.Int).Set(One18)
	} else {
		return new(big.Int).Div(new(big.Int).Mul(totalAsset, One18), totalLiability)
	}
}

func getTotalAssetLiability(state *PoolState) (*big.Int, *big.Int) {
	totalAsset := big.NewInt(0)
	totalLiability := big.NewInt(0)
	for _, lp := range state.LPs {
		price := lp.TokenOraclePrice
		lpAsset := new(big.Int).Set(lp.Asset)
		totalAsset = new(big.Int).Add(
			totalAsset,
			new(big.Int).Div(
				new(big.Int).Mul(lpAsset, price),
				bignumber.TenPowInt(lp.Decimals),
			),
		)
		totalLiability = new(big.Int).Add(
			totalLiability,
			new(big.Int).Div(
				new(big.Int).Mul(lp.Liability, price),
				bignumber.TenPowInt(lp.Decimals),
			),
		)
	}

	return totalAsset, totalLiability
}

func getTreasuryRatio(nlr *big.Int) *big.Int {
	if nlr.Cmp(One18) < 0 {
		return big.NewInt(0)
	} else if nlr.Cmp(big.NewInt(1.05e18)) < 0 {
		return big.NewInt(4e5)
	} else {
		return big.NewInt(8e5)
	}
}
