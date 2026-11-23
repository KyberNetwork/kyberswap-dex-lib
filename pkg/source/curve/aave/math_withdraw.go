package aave

import (
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

// calculateWithdrawOneTokenDy returns dy and the new y for a one-coin withdraw, in legacy order.
func calculateWithdrawOneTokenDy(
	balances []*uint256.Int,
	precisionMultipliers []*uint256.Int,
	futureATime int64,
	futureA *uint256.Int,
	initialATime int64,
	initialA *uint256.Int,
	swapFee *uint256.Int,
	lpSupply *uint256.Int,
	tokenIndex int,
	tokenAmount *uint256.Int,
) (*uint256.Int, *uint256.Int, error) {
	var numTokens = len(balances)
	if tokenIndex >= numTokens {
		return nil, nil, ErrTokenIndexesOutOfRange
	}
	xp, err := _xp(balances, precisionMultipliers)
	if err != nil {
		return nil, nil, err
	}
	var preciseA = _getAPrecise(futureATime, futureA, initialATime, initialA)
	d0, err := getD(xp, preciseA)
	if err != nil {
		return nil, nil, err
	}
	// d1 = d0 - tokenAmount*d0/lpSupply; legacy goes negative here (signed getYD), uint256 cannot hold it
	var sub uint256.Int
	sub.Mul(tokenAmount, d0)
	sub.Div(&sub, lpSupply)
	if d0.Cmp(&sub) < 0 {
		return nil, nil, ErrNegativeResult
	}
	var d1 uint256.Int
	d1.Sub(d0, &sub)

	if tokenAmount.Cmp(xp[tokenIndex]) > 0 {
		return nil, nil, ErrWithdrawMoreThanAvailable
	}

	newY, err := getYD(preciseA, tokenIndex, xp, &d1)
	if err != nil {
		return nil, nil, err
	}
	var feePerToken = _feePerToken(swapFee, numTokens)
	var xpReduced = make([]*uint256.Int, numTokens)
	for i := range numTokens {
		var norm, tmp uint256.Int
		tmp.Mul(xp[i], &d1)
		tmp.Div(&tmp, d0)
		if i == tokenIndex {
			norm.Sub(&tmp, newY)
		} else {
			norm.Sub(xp[i], &tmp)
		}
		var reduce uint256.Int
		reduce.Mul(&norm, feePerToken)
		reduce.Div(&reduce, FeeDenominator)
		xpReduced[i] = new(uint256.Int).Sub(xp[i], &reduce)
	}
	yd, err := getYD(preciseA, tokenIndex, xpReduced, &d1)
	if err != nil {
		return nil, nil, err
	}
	// legacy dy = xpReduced[tokenIndex] - yd, then (dy - 1) / multiplier; negative when yd exceeds xpReduced
	if xpReduced[tokenIndex].Cmp(yd) < 0 {
		return nil, nil, ErrNegativeResult
	}
	var dy uint256.Int
	dy.Sub(xpReduced[tokenIndex], yd)
	if dy.IsZero() {
		return nil, nil, ErrNegativeResult
	}
	dy.Sub(&dy, big256.U1)
	dy.Div(&dy, precisionMultipliers[tokenIndex])
	return &dy, newY, nil
}

// CalculateWithdrawOneCoin returns dy and the second value legacy returns as its fee. Legacy returns newY there
// (calculateWithdrawOneTokenDy's second result), not dy0 - dy, so the port returns newY too.
func (t *PoolSimulator) CalculateWithdrawOneCoin(tokenAmount *uint256.Int, i int) (*uint256.Int, *uint256.Int, error) {
	return calculateWithdrawOneTokenDy(
		t.Info.Reserves,
		t.Multipliers,
		t.FutureATime, t.FutureA,
		t.InitialATime, t.InitialA,
		t.Info.SwapFee,
		t.LpSupply,
		i,
		tokenAmount,
	)
}

// RemoveLiquidityOneCoin burns tokenAmount LP into one coin and updates the reserve and LP supply.
func (t *PoolSimulator) RemoveLiquidityOneCoin(tokenAmount *uint256.Int, i int) (*uint256.Int, error) {
	dy, dyFee, err := t.CalculateWithdrawOneCoin(tokenAmount, i)
	if err != nil {
		return nil, err
	}
	var admin uint256.Int
	admin.Mul(dyFee, t.AdminFee)
	admin.Div(&admin, FeeDenominator)
	var take uint256.Int
	take.Add(dy, &admin)
	// legacy lets the reserve and LP supply go negative; uint256 cannot hold that, so check before writing state
	if t.Info.Reserves[i].Cmp(&take) < 0 || t.LpSupply.Cmp(tokenAmount) < 0 {
		return nil, ErrNegativeResult
	}
	t.Info.Reserves[i] = new(uint256.Int).Sub(t.Info.Reserves[i], &take)
	t.LpSupply = new(uint256.Int).Sub(t.LpSupply, tokenAmount)
	return dy, nil
}

// calculateTokenAmount is the LP amount minted (deposit) or burned (withdraw) for a set of amounts.
func calculateTokenAmount(
	balances []*uint256.Int,
	precisionMultipliers []*uint256.Int,
	futureATime int64,
	futureA *uint256.Int,
	initialATime int64,
	initialA *uint256.Int,
	withdrawFee *uint256.Int,
	lpSupply *uint256.Int,
	amounts []*uint256.Int,
	deposit bool,
) (*uint256.Int, error) {
	var numTokens = len(balances)
	var a = _getAPrecise(futureATime, futureA, initialATime, initialA)
	xp, err := _xp(balances, precisionMultipliers)
	if err != nil {
		return nil, err
	}
	d0, err := getD(xp, a)
	if err != nil {
		return nil, err
	}
	var balances1 = make([]*uint256.Int, numTokens)
	for i := range numTokens {
		balances1[i] = new(uint256.Int)
		if deposit {
			balances1[i].Add(balances[i], amounts[i])
		} else {
			if balances[i].Cmp(amounts[i]) < 0 {
				return nil, ErrWithdrawMoreThanAvailable
			}
			balances1[i].Sub(balances[i], amounts[i])
		}
	}
	xp1, err := _xp(balances1, precisionMultipliers)
	if err != nil {
		return nil, err
	}
	d1, err := getD(xp1, a)
	if err != nil {
		return nil, err
	}
	if deposit {
		var diff uint256.Int
		diff.Sub(d1, d0)
		diff.Mul(&diff, lpSupply)
		return diff.Div(&diff, d0), nil
	}
	var diff uint256.Int
	diff.Sub(d0, d1)
	diff.Mul(&diff, lpSupply)
	diff.Div(&diff, d0)
	diff.Mul(&diff, FeeDenominator)
	var den uint256.Int
	den.Sub(FeeDenominator, withdrawFee)
	return diff.Div(&diff, &den), nil
}

// CalculateTokenAmount returns the LP minted for a deposit, or burned for a withdraw, of the given amounts.
func (t *PoolSimulator) CalculateTokenAmount(amounts []*uint256.Int, deposit bool) (*uint256.Int, error) {
	return calculateTokenAmount(
		t.Info.Reserves,
		t.Multipliers,
		t.FutureATime, t.FutureA,
		t.InitialATime, t.InitialA,
		new(uint256.Int), // withdraw fee not used in deposit case
		t.LpSupply,
		amounts,
		deposit,
	)
}

// AddLiquidity deposits amounts, updates the reserves and LP supply, and returns the LP minted.
func (t *PoolSimulator) AddLiquidity(amounts []*uint256.Int) (*uint256.Int, error) {
	var nCoins = len(amounts)
	var nCoinsU256 = uint256.NewInt(uint64(nCoins))
	var amp = _getAPrecise(t.FutureATime, t.FutureA, t.InitialATime, t.InitialA)
	// snapshot the live balances: AddLiquidity replaces t.Info.Reserves[i] below
	var oldBalances = make([]*uint256.Int, nCoins)
	for i := range nCoins {
		oldBalances[i] = new(uint256.Int).Set(t.Info.Reserves[i])
	}
	D0, err := t.getDPrecision(oldBalances, amp)
	if err != nil {
		return nil, err
	}
	var tokenSupply = t.LpSupply
	var newBalances = make([]*uint256.Int, nCoins)
	for i := range nCoins {
		newBalances[i] = new(uint256.Int).Add(oldBalances[i], amounts[i])
	}
	D1, err := t.getDPrecision(newBalances, amp)
	if err != nil {
		return nil, err
	}
	if D1.Cmp(D0) <= 0 {
		return nil, ErrD1LowerThanD0
	}
	var mintAmount uint256.Int
	if tokenSupply.Sign() > 0 {
		var ys uint256.Int
		ys.Add(D0, D1)
		ys.Div(&ys, nCoinsU256)
		var fee uint256.Int
		fee.Mul(t.Info.SwapFee, nCoinsU256)
		fee.Div(&fee, new(uint256.Int).Mul(big256.U4, uint256.NewInt(uint64(nCoins-1))))
		feeMul := t.OffpegFeeMultiplier
		for i := range nCoins {
			// legacy stores the pre-fee balance: "cannot determine real amount transferred, so use this"
			t.Info.Reserves[i] = new(uint256.Int).Set(newBalances[i])
			var ideal, difference, xs, dynFee, feeI uint256.Int
			ideal.Mul(D1, oldBalances[i])
			ideal.Div(&ideal, D0)
			if ideal.Cmp(newBalances[i]) > 0 {
				difference.Sub(&ideal, newBalances[i])
			} else {
				difference.Sub(newBalances[i], &ideal)
			}
			xs.Add(oldBalances[i], newBalances[i])
			dyn, err := _dynamicFee(&xs, &ys, &fee, feeMul)
			if err != nil {
				return nil, err
			}
			dynFee.Set(dyn)
			feeI.Mul(&dynFee, &difference)
			feeI.Div(&feeI, FeeDenominator)
			if newBalances[i].Cmp(&feeI) < 0 {
				return nil, ErrNegativeResult
			}
			newBalances[i] = new(uint256.Int).Sub(newBalances[i], &feeI)
		}
		D2, err := t.getDPrecision(newBalances, amp)
		if err != nil {
			return nil, err
		}
		var diff uint256.Int
		diff.Sub(D2, D0)
		mintAmount.Mul(tokenSupply, &diff)
		mintAmount.Div(&mintAmount, D0)
	} else {
		for i := range nCoins {
			t.Info.Reserves[i] = new(uint256.Int).Set(newBalances[i])
		}
		mintAmount.Set(D1)
	}
	t.LpSupply = new(uint256.Int).Add(t.LpSupply, &mintAmount)
	return &mintAmount, nil
}
