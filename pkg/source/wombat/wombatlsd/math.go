package wombatlsd

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/wombat"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"math/big"
	"strings"
)

func QuotePotentialSwap(
	fromToken string,
	toToken string,
	fromAmount *big.Int,
	haircutRate, ampFactor, startCovRatio, endCovRatio *big.Int,
	assetMap map[string]wombat.Asset,
) (*big.Int, *big.Int, error) {
	if err := checkSameAddress(fromToken, toToken); err != nil {
		return nil, nil, err
	}
	if fromAmount.Cmp(bignumber.ZeroBI) == 0 {
		return nil, nil, ErrFromAmountIsZero
	}

	fromAsset, err := assetOf(fromToken, assetMap)
	if err != nil {
		return nil, nil, err
	}
	toAsset, err := assetOf(toToken, assetMap)
	if err != nil {
		return nil, nil, err
	}

	if fromAsset.IsPause {
		return nil, nil, ErrWombatAssetAlreadyPause
	}

	fromAmount = toWad(fromAmount, fromAsset.UnderlyingTokenDecimals)
	potentialOutcome, haircut, err := highCovRatioFeePoolV2QuoteFrom(fromAsset, toAsset, fromAmount, haircutRate, ampFactor, startCovRatio, endCovRatio)
	if err != nil {
		return nil, nil, err
	}
	potentialOutcome = fromWad(potentialOutcome, toAsset.UnderlyingTokenDecimals)
	if fromAmount.Cmp(bignumber.ZeroBI) > 0 {
		haircut = fromWad(haircut, toAsset.UnderlyingTokenDecimals)
	} else {
		haircut = fromWad(haircut, fromAsset.UnderlyingTokenDecimals)
	}

	return potentialOutcome, haircut, nil
}

func highCovRatioFeePoolV2QuoteFrom(
	fromAsset wombat.Asset,
	toAsset wombat.Asset,
	fromAmount, haircutRate, ampFactor, startCovRatio, endCovRatio *big.Int,
) (*big.Int, *big.Int, error) {
	actualToAmount, haircut, err := poolV2QuoteFrom(fromAsset, toAsset, fromAmount, haircutRate, ampFactor)
	if err != nil {
		return nil, nil, err
	}

	if fromAmount.Cmp(bignumber.ZeroBI) >= 0 {
		fromAssetCash := new(big.Int).Set(fromAsset.Cash)
		fromAssetLiability := new(big.Int).Set(fromAsset.Liability)
		finalFromAssetCovRatio := wdiv(new(big.Int).Add(fromAssetCash, fromAmount), fromAssetLiability)

		if finalFromAssetCovRatio.Cmp(startCovRatio) > 0 {
			fee, err := getHighCovRatioFee(wdiv(fromAssetCash, fromAssetLiability), finalFromAssetCovRatio, startCovRatio, endCovRatio)
			if err != nil {
				return nil, nil, err
			}
			highCovRatioFee := wmul(fee, actualToAmount)
			actualToAmount = new(big.Int).Sub(actualToAmount, highCovRatioFee)
			haircut = new(big.Int).Add(haircut, highCovRatioFee)
		}
	} else {
		toAssetCash := new(big.Int).Set(toAsset.Cash)
		toAssetLiability := new(big.Int).Set(toAsset.Liability)
		finalToAssetCovRatio := new(big.Int).Add(toAssetCash, wdiv(actualToAmount, toAssetLiability))

		if finalToAssetCovRatio.Cmp(startCovRatio) <= 0 {
			return actualToAmount, haircut, nil
		} else if wdiv(toAssetCash, toAssetLiability).Cmp(endCovRatio) >= 0 {
			return nil, nil, ErrCovRatioLimitExceeded
		}

		actualToAmount, err = findUpperBound(toAsset, fromAsset, new(big.Int).Neg(fromAmount), haircutRate, ampFactor, startCovRatio, endCovRatio)
		if err != nil {
			return nil, nil, err
		}
		_, haircut, err = highCovRatioFeePoolV2QuoteFrom(toAsset, fromAsset, actualToAmount, haircutRate, ampFactor, startCovRatio, endCovRatio)
		if err != nil {
			return nil, nil, err
		}
	}

	return actualToAmount, haircut, nil
}

func poolV2QuoteFrom(
	fromAsset wombat.Asset,
	toAsset wombat.Asset,
	fromAmount, haircutRate, ampFactor *big.Int,
) (*big.Int, *big.Int, error) {
	var haircut, actualToAmount *big.Int

	var newFromAmount = new(big.Int).Set(fromAmount)
	if newFromAmount.Cmp(bignumber.ZeroBI) < 0 {
		newFromAmount = wdiv(newFromAmount, new(big.Int).Sub(WADI, haircutRate))
	}

	fromCash := new(big.Int).Set(fromAsset.Cash)
	fromLiability := new(big.Int).Set(fromAsset.Liability)
	toCash := new(big.Int).Set(toAsset.Cash)

	scaleFactor := dynamicPoolV2QuoteFactor(fromAsset, toAsset)
	if scaleFactor.Cmp(WAD) != 0 {
		fromCash = new(big.Int).Div(new(big.Int).Mul(fromCash, scaleFactor), big.NewInt(1e18))
		fromLiability = new(big.Int).Div(new(big.Int).Mul(fromLiability, scaleFactor), big.NewInt(1e18))
		newFromAmount = new(big.Int).Div(new(big.Int).Mul(newFromAmount, scaleFactor), big.NewInt(1e18))
	}

	idealToAmount, err := swapQuoteFunc(
		fromCash, toCash,
		fromLiability, toAsset.Liability,
		newFromAmount, ampFactor,
	)
	if err != nil {
		return nil, nil, err
	}

	if (newFromAmount.Cmp(bignumber.ZeroBI) > 0 && toCash.Cmp(idealToAmount) < 0) ||
		(newFromAmount.Cmp(bignumber.ZeroBI) < 0 && fromAsset.Cash.Cmp(new(big.Int).Neg(newFromAmount)) < 0) {
		return nil, nil, ErrCashNotEnough
	}

	if newFromAmount.Cmp(bignumber.ZeroBI) > 0 {
		haircut = wmul(idealToAmount, haircutRate)
		actualToAmount = new(big.Int).Sub(idealToAmount, haircut)
	} else {
		actualToAmount = new(big.Int).Set(idealToAmount)
		haircut = wmul(new(big.Int).Neg(newFromAmount), haircutRate)
	}

	return actualToAmount, haircut, nil
}

func getHighCovRatioFee(initCovRatio, finalCovRatio, startCovRatio, endCovRatio *big.Int) (*big.Int, error) {
	if finalCovRatio.Cmp(endCovRatio) > 0 {
		return nil, ErrCovRatioLimitExceeded
	} else if finalCovRatio.Cmp(startCovRatio) <= 0 || finalCovRatio.Cmp(initCovRatio) <= 0 {
		return big.NewInt(0), nil
	}

	a := big.NewInt(0)
	if initCovRatio.Cmp(startCovRatio) > 0 {
		a = new(big.Int).Mul(
			new(big.Int).Sub(initCovRatio, startCovRatio),
			new(big.Int).Sub(initCovRatio, startCovRatio))
	}
	b := new(big.Int).Mul(
		new(big.Int).Sub(finalCovRatio, startCovRatio),
		new(big.Int).Sub(finalCovRatio, startCovRatio))
	fee := wdiv(
		new(big.Int).Div(new(big.Int).Div(
			new(big.Int).Sub(b, a),
			new(big.Int).Sub(finalCovRatio, initCovRatio),
		), bignumber.Two),
		new(big.Int).Sub(endCovRatio, startCovRatio),
	)

	return fee, nil
}

func findUpperBound(fromAsset, toAsset wombat.Asset, toAmount, hairCutRate, ampFactor, startCovRatio, endCovRatio *big.Int) (*big.Int, error) {
	decimals := fromAsset.UnderlyingTokenDecimals
	toWadFactor := toWad(big.NewInt(1), decimals)
	high := new(big.Int).Sub(wmul(fromAsset.Liability, endCovRatio), fromWad(fromAsset.Cash, decimals))
	low := big.NewInt(1)

	quote, _, err := highCovRatioFeePoolV2QuoteFrom(fromAsset, toAsset, new(big.Int).Mul(high, toWadFactor), hairCutRate, ampFactor, startCovRatio, endCovRatio)
	if err != nil {
		return nil, err
	}
	if quote.Cmp(toAmount) < 0 {
		return nil, ErrCovRatioLimitExceeded
	}
	for low.Cmp(high) < 0 {
		mid := new(big.Int).Div(new(big.Int).Add(low, high), bignumber.Two)
		quote, _, err := highCovRatioFeePoolV2QuoteFrom(fromAsset, toAsset, new(big.Int).Mul(mid, toWadFactor), hairCutRate, ampFactor, startCovRatio, endCovRatio)
		if err != nil {
			return nil, err
		}
		if quote.Cmp(toAmount) >= 0 {
			high = new(big.Int).Set(mid)
		} else {
			low = new(big.Int).Add(mid, bignumber.One)
		}
	}

	return new(big.Int).Mul(high, toWadFactor), nil
}

func dynamicPoolV2QuoteFactor(fromAsset, toAsset wombat.Asset) *big.Int {
	fromAssetRelativePrice := new(big.Int).Set(fromAsset.RelativePrice)
	// theoretically we should multiply toCash, toLiability and idealToAmount by toAssetRelativePrice
	// however we simplify the calculation by dividing "from amounts" by toAssetRelativePrice
	toAssetRelativePrice := new(big.Int).Set(toAsset.RelativePrice)

	return new(big.Int).Div(new(big.Int).Mul(bignumber.BONE, fromAssetRelativePrice), toAssetRelativePrice)
}

func swapQuoteFunc(ax, ay, lx, ly, dx, a *big.Int) (*big.Int, error) {
	if lx.Cmp(bignumber.ZeroBI) == 0 || ly.Cmp(bignumber.ZeroBI) == 0 {
		return nil, ErrCoreUnderflow
	}
	d := new(big.Int).Sub(
		new(big.Int).Add(ax, ay),
		wmul(
			a,
			new(big.Int).Add(
				new(big.Int).Div(new(big.Int).Mul(lx, lx), ax),
				new(big.Int).Div(new(big.Int).Mul(ly, ly), ay)),
		),
	)
	rx := wdiv(new(big.Int).Add(ax, dx), lx)
	b := new(big.Int).Sub(
		new(big.Int).Div(
			new(big.Int).Mul(
				lx,
				new(big.Int).Sub(
					rx,
					wdiv(a, rx)),
			),
			ly,
		),
		wdiv(d, ly),
	)
	ry := solveQuad(b, a)
	dy := new(big.Int).Sub(wmul(ly, ry), ay)
	if dy.Cmp(bignumber.ZeroBI) < 0 {
		return new(big.Int).Neg(dy), nil
	} else {
		return new(big.Int).Set(dy), nil
	}
}

func solveQuad(b, c *big.Int) *big.Int {
	return new(big.Int).Div(
		new(big.Int).Sub(
			signedSafeMathSqrt(
				new(big.Int).Add(
					new(big.Int).Mul(b, b),
					new(big.Int).Mul(new(big.Int).Mul(c, bignumber.Four), WADI)),
				b,
			),
			b,
		),
		bignumber.Two,
	)
}

func checkSameAddress(from, to string) error {
	if strings.EqualFold(from, to) {
		return ErrTheSameAddress
	}

	return nil
}

func assetOf(token string, assetMap map[string]wombat.Asset) (wombat.Asset, error) {
	asset, ok := assetMap[token]
	if !ok {
		return wombat.Asset{}, ErrAssetIsNotExist
	}

	return asset, nil
}

// ----------------- DSMATH
func wmul(x, y *big.Int) *big.Int {
	return new(big.Int).Div(
		new(big.Int).Add(
			new(big.Int).Mul(x, y),
			new(big.Int).Div(WAD, bignumber.Two)),
		WAD,
	)
}

func wdiv(x, y *big.Int) *big.Int {
	return new(big.Int).Div(
		new(big.Int).Add(
			new(big.Int).Mul(x, WAD),
			new(big.Int).Div(y, bignumber.Two)),
		y,
	)
}

func toWad(x *big.Int, d uint8) *big.Int {
	if d < 18 {
		return new(big.Int).Mul(x, bignumber.TenPowInt(18-d))
	} else if d > 18 {
		return new(big.Int).Div(x, bignumber.TenPowInt(d-18))
	}

	return x
}

func fromWad(x *big.Int, d uint8) *big.Int {
	if d < 18 {
		return new(big.Int).Div(x, bignumber.TenPowInt(18-d))
	} else if d > 18 {
		return new(big.Int).Mul(x, bignumber.TenPowInt(d-18))
	}

	return x
}

func signedSafeMathSqrt(y, guess *big.Int) *big.Int {
	var z *big.Int
	if y.Cmp(bignumber.Three) > 0 {
		if guess.Cmp(bignumber.ZeroBI) > 0 && guess.Cmp(y) <= 0 {
			z = new(big.Int).Set(guess)
		} else if guess.Cmp(bignumber.ZeroBI) < 0 && new(big.Int).Neg(guess).Cmp(y) <= 0 {
			z = new(big.Int).Neg(guess)
		} else {
			z = new(big.Int).Set(y)
		}
		x := new(big.Int).Div(
			new(big.Int).Add(
				new(big.Int).Div(y, z),
				z,
			),
			bignumber.Two,
		)
		for x.Cmp(z) != 0 {
			z = new(big.Int).Set(x)
			x = new(big.Int).Div(
				new(big.Int).Add(
					new(big.Int).Div(y, x),
					x,
				),
				bignumber.Two,
			)
		}
	} else if y.Cmp(bignumber.ZeroBI) != 0 {
		z = new(big.Int).Set(bignumber.One)
	}

	return z
}

func addCash(cash, amount *big.Int) *big.Int {
	return new(big.Int).Add(cash, amount)
}

func removeCash(cash, amount *big.Int) *big.Int {
	return new(big.Int).Sub(cash, amount)
}
