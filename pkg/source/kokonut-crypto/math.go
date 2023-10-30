package kokonutcrypto

import (
	"errors"
	"fmt"
	"github.com/daoleno/uniswapv3-sdk/constants"
	"math/big"
	"time"

	constant "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

func sortArray(A0 []*big.Int) []*big.Int {
	var nCoins = len(A0)
	var ret = make([]*big.Int, nCoins)
	ret[0] = new(big.Int).Set(A0[0])
	ret[1] = new(big.Int).Set(A0[1])
	if A0[0].Cmp(A0[1]) < 0 {
		ret[0] = new(big.Int).Set(A0[1])
		ret[1] = new(big.Int).Set(A0[0])
	}

	return ret
}

func geometricMean(unsortedX []*big.Int, sort bool) (*big.Int, error) {
	var nCoins = len(unsortedX)
	var nCoinsBi = big.NewInt(int64(nCoins))
	var x = []*big.Int{new(big.Int).Set(unsortedX[0]), new(big.Int).Set(unsortedX[1])}
	if sort {
		x = sortArray(unsortedX)
	}
	var D = new(big.Int).Set(x[0])
	var diff = constant.ZeroBI
	for i := 0; i < 255; i += 1 {
		var DPrev = new(big.Int).Set(D)
		D = new(big.Int).Div(
			new(big.Int).Add(
				D,
				new(big.Int).Div(new(big.Int).Mul(x[0], x[1]), D),
			),
			nCoinsBi,
		)
		if D.Cmp(DPrev) > 0 {
			diff = new(big.Int).Sub(D, DPrev)
		} else {
			diff = new(big.Int).Sub(DPrev, D)
		}
		if diff.Cmp(constant.One) <= 0 || new(big.Int).Mul(diff, constant.BONE).Cmp(D) < 0 {
			return D, nil
		}
	}
	return nil, ErrDidNotCoverage
}

func newtonD(mA *big.Int, mGamma *big.Int, xUnsorted []*big.Int) (*big.Int, error) {
	if mA.Cmp(new(big.Int).Sub(MinA, constant.One)) <= 0 {
		return nil, ErrUnsafeValuesA
	}
	if mA.Cmp(new(big.Int).Add(MaxA, constant.One)) >= 0 {
		return nil, ErrUnsafeValuesA
	}
	if mGamma.Cmp(MinGamma) < 0 || mGamma.Cmp(MaxGamma) > 0 {
		return nil, ErrUnsafeValuesGamma
	}
	var nCoins = len(xUnsorted)
	var nCoinsBi = big.NewInt(int64(nCoins))
	var x = sortArray(xUnsorted)
	if x[0].Cmp(constant.TenPowInt(9)) < 0 || x[0].Cmp(constant.TenPowInt(33)) > 0 {
		return nil, ErrUnsafeValuesXi
	}
	if new(big.Int).Mul(x[1], constant.TenPowInt(18)).Cmp(new(big.Int).Sub(constant.TenPowInt(14), constant.One)) <= 0 {
		return nil, ErrUnsafeValuesXi
	}

	var mean, err = geometricMean(x, false)
	if err != nil {
		return nil, err
	}
	var D = new(big.Int).Mul(nCoinsBi, mean)
	var S = new(big.Int).Add(x[0], x[1])

	for i := 0; i < 255; i += 1 {
		var DPrev = D
		K0 := new(big.Int).Div(
			new(big.Int).Mul(
				new(big.Int).Div(
					new(big.Int).Mul(
						new(big.Int).Mul(constant.BONE, new(big.Int).Mul(nCoinsBi, nCoinsBi)),
						x[0],
					),
					D,
				),
				x[1],
			),
			D,
		)

		var _g1k0 = new(big.Int).Add(mGamma, constant.BONE)
		if _g1k0.Cmp(K0) > 0 {
			_g1k0 = new(big.Int).Add(new(big.Int).Sub(_g1k0, K0), constant.One)
		} else {
			_g1k0 = new(big.Int).Add(new(big.Int).Sub(K0, _g1k0), constant.One)
		}
		var mul1 = new(big.Int).Div(
			new(big.Int).Mul(
				new(big.Int).Mul(
					new(big.Int).Div(
						new(big.Int).Mul(
							new(big.Int).Div(new(big.Int).Mul(constant.BONE, D), mGamma), _g1k0,
						), mGamma,
					),
					_g1k0,
				),
				AMultiplier,
			), mA,
		)
		var mul2 = new(big.Int).Div(
			new(big.Int).Mul(
				new(big.Int).Mul(
					new(big.Int).Mul(constant.Two, constant.BONE), nCoinsBi,
				), K0,
			), _g1k0,
		)
		var negFprime = new(big.Int).Sub(
			new(big.Int).Add(
				new(big.Int).Add(S, new(big.Int).Div(new(big.Int).Mul(S, mul2), constant.BONE)),
				new(big.Int).Div(new(big.Int).Mul(mul1, nCoinsBi), K0),
			),
			new(big.Int).Div(new(big.Int).Mul(mul2, D), constant.BONE),
		)

		// D -= f / fprime
		var DPlus = new(big.Int).Div(new(big.Int).Mul(D, new(big.Int).Add(negFprime, S)), negFprime)
		var DMinus = new(big.Int).Div(new(big.Int).Mul(D, D), negFprime)
		if constant.BONE.Cmp(K0) > 0 {
			DMinus = new(big.Int).Add(
				DMinus,
				new(big.Int).Div(
					new(big.Int).Mul(
						new(big.Int).Div(
							new(big.Int).Mul(D, new(big.Int).Div(mul1, negFprime)), constant.BONE,
						), new(big.Int).Sub(constant.BONE, K0),
					),
					K0,
				),
			)
		} else {
			DMinus = new(big.Int).Sub(
				DMinus,
				new(big.Int).Div(
					new(big.Int).Mul(
						new(big.Int).Div(
							new(big.Int).Mul(D, new(big.Int).Div(mul1, negFprime)), constant.BONE,
						), new(big.Int).Sub(K0, constant.BONE),
					),
					K0,
				),
			)
		}
		if DPlus.Cmp(DMinus) > 0 {
			D = new(big.Int).Sub(DPlus, DMinus)
		} else {
			D = new(big.Int).Div(new(big.Int).Sub(DMinus, DPlus), constant.Two)
		}

		var diff *big.Int
		if D.Cmp(DPrev) > 0 {
			diff = new(big.Int).Sub(D, DPrev)
		} else {
			diff = new(big.Int).Sub(DPrev, D)
		}
		var temp = constant.TenPowInt(16)
		if D.Cmp(temp) > 0 {
			temp = D
		}
		if new(big.Int).Mul(diff, constant.TenPowInt(14)).Cmp(temp) < 0 {
			for _, _x := range x {
				var frac = new(big.Int).Div(new(big.Int).Mul(_x, constant.BONE), D)
				if frac.Cmp(constant.TenPowInt(16)) < 0 || frac.Cmp(constant.TenPowInt(20)) > 0 {
					return nil, ErrUnsafeValuesXi
				}
			}
			return D, nil
		}
	}
	return nil, ErrDidNotCoverage
}

func newtonY(mA *big.Int, mGamma *big.Int, xj *big.Int, D *big.Int) (*big.Int, error) {
	// Reference: https://github.com/curvefi/curve-crypto-contract/blob/master/contracts/two/CurveCryptoSwap2ETH.vy#L346-L349
	if mA.Cmp(new(big.Int).Sub(MinA, constant.One)) <= 0 || mA.Cmp(new(big.Int).Add(MaxA, constant.One)) >= 0 {
		return nil, ErrUnsafeValuesA
	}

	if mGamma.Cmp(new(big.Int).Sub(MinGamma, constant.One)) <= 0 || mGamma.Cmp(new(big.Int).Add(MaxGamma, constant.One)) >= 0 {
		return nil, ErrUnsafeValuesGamma
	}

	if D.Cmp(new(big.Int).Sub(constant.TenPowInt(17), constant.One)) <= 0 {
		return nil, ErrUnsafeValueD
	}
	if D.Cmp(new(big.Int).Add(new(big.Int).Mul(constant.TenPowInt(15), constant.TenPowInt(18)), constant.One)) >= 0 {
		return nil, ErrUnsafeValueD
	}

	var nCoins = 2
	var nCoinBi = big.NewInt(int64(nCoins))
	var y = new(big.Int).Div(
		new(big.Int).Mul(D, D),
		new(big.Int).Mul(xj, new(big.Int).Mul(nCoinBi, nCoinBi)),
	)
	var k0i = new(big.Int).Div(
		new(big.Int).Mul(
			new(big.Int).Mul(constant.BONE, nCoinBi),
			xj,
		),
		D,
	)

	if k0i.Cmp(new(big.Int).Sub(new(big.Int).Mul(constant.TenPowInt(16), nCoinBi), constant.One)) <= 0 {
		return nil, ErrUnsafeValuesXi
	}
	if k0i.Cmp(new(big.Int).Add(new(big.Int).Mul(constant.TenPowInt(20), nCoinBi), constant.One)) >= 0 {
		return nil, ErrUnsafeValuesXi
	}

	var convergenceLimit = new(big.Int).Div(xj, constant.TenPowInt(14))
	var temp = new(big.Int).Div(D, constant.TenPowInt(14))
	if temp.Cmp(convergenceLimit) > 0 {
		convergenceLimit = temp
	}
	if big.NewInt(100).Cmp(convergenceLimit) > 0 {
		convergenceLimit = big.NewInt(100)
	}

	for j := 0; j < 255; j += 1 {
		var K0 = new(big.Int).Div(new(big.Int).Mul(new(big.Int).Mul(k0i, y), nCoinBi), D)

		var g1k0 = new(big.Int).Add(mGamma, constant.BONE)
		if g1k0.Cmp(K0) > 0 {
			g1k0 = new(big.Int).Add(new(big.Int).Sub(g1k0, K0), constant.One)
		} else {
			g1k0 = new(big.Int).Add(new(big.Int).Sub(K0, g1k0), constant.One)
		}

		// D / (A * N**N) * _g1k0**2 / gamma**2
		var mul1 = new(big.Int).Div(
			new(big.Int).Mul(
				new(big.Int).Div(
					new(big.Int).Mul(
						new(big.Int).Div(
							new(big.Int).Mul(constant.BONE, D),
							mGamma,
						),
						g1k0,
					),
					mGamma,
				),
				new(big.Int).Mul(g1k0, AMultiplier),
			),
			mA,
		)
		// 2*K0 / _g1k0
		var mul2 = new(big.Int).Add(
			constant.BONE,
			new(big.Int).Div(
				new(big.Int).Mul(
					new(big.Int).Mul(big.NewInt(2), constant.BONE),
					K0,
				),
				g1k0,
			),
		)

		s := new(big.Int).Add(xj, y)
		var yfprime = new(big.Int).Add(
			new(big.Int).Add(new(big.Int).Mul(constant.BONE, y), new(big.Int).Mul(s, mul2)), mul1,
		)
		yPrev := new(big.Int).Set(y)

		var dyfprime = new(big.Int).Mul(D, mul2)
		if yfprime.Cmp(dyfprime) < 0 {
			y = new(big.Int).Div(yPrev, constant.Two)
			continue
		} else {
			yfprime = new(big.Int).Sub(yfprime, dyfprime)
		}

		if y.Cmp(constant.ZeroBI) == 0 {
			return nil, ErrDenominatorZero
		}

		var fprime = new(big.Int).Div(yfprime, y)
		if fprime.Cmp(constant.ZeroBI) == 0 {
			return nil, ErrDenominatorZero
		}

		var yMinus = new(big.Int).Div(mul1, fprime)
		var yPlus = new(big.Int).Add(
			new(big.Int).Div(
				new(big.Int).Add(yfprime, new(big.Int).Mul(constant.BONE, D)),
				fprime,
			),
			new(big.Int).Div(new(big.Int).Mul(yMinus, constant.BONE), K0),
		)
		yMinus = new(big.Int).Add(yMinus, new(big.Int).Div(new(big.Int).Mul(constant.BONE, s), fprime))
		if yPlus.Cmp(yMinus) < 0 {
			y = new(big.Int).Div(yPrev, constant.Two)
		} else {
			y = new(big.Int).Sub(yPlus, yMinus)
		}

		var diff = constant.ZeroBI
		if y.Cmp(yPrev) > 0 {
			diff = new(big.Int).Sub(y, yPrev)
		} else {
			diff = new(big.Int).Sub(yPrev, y)
		}

		var temp = new(big.Int).Div(y, constant.TenPowInt(14))
		if convergenceLimit.Cmp(temp) > 0 {
			temp = convergenceLimit
		}
		if diff.Cmp(temp) < 0 {
			var frac = new(big.Int).Div(new(big.Int).Mul(y, constant.BONE), D)
			if frac.Cmp(new(big.Int).Sub(constant.TenPowInt(16), constant.One)) <= 0 ||
				frac.Cmp(new(big.Int).Add(constant.TenPowInt(20), constant.One)) >= 0 {
				return nil, ErrUnsafeValueY
			}
			return y, nil
		}
	}
	return nil, ErrDidNotCoverage
}

func halfpow(power *big.Int, precision *big.Int) (*big.Int, error) {
	var intpow = new(big.Int).Div(power, constant.BONE)
	var otherpow = new(big.Int).Sub(power, new(big.Int).Mul(intpow, constant.BONE))
	if intpow.Cmp(big.NewInt(59)) > 0 {
		return constant.ZeroBI, nil
	}
	var result = new(big.Int).Div(constant.BONE, new(big.Int).Exp(constant.Two, intpow, nil))
	if otherpow.Cmp(constant.ZeroBI) == 0 {
		return result, nil
	}
	var term = constant.BONE
	var x = new(big.Int).Mul(constant.Five, constant.TenPowInt(17))
	var S = constant.BONE
	var neg = false
	for i := 1; i < 256; i += 1 {
		var K = new(big.Int).Mul(big.NewInt(int64(i)), constant.BONE)
		var c = new(big.Int).Sub(K, constant.BONE)
		if otherpow.Cmp(c) > 0 {
			c = new(big.Int).Sub(otherpow, c)
			neg = !neg
		} else {
			c = new(big.Int).Sub(c, otherpow)
		}
		term = new(big.Int).Div(new(big.Int).Mul(term, new(big.Int).Div(new(big.Int).Mul(c, x), constant.BONE)), K)
		if neg {
			S = new(big.Int).Sub(S, term)
		} else {
			S = new(big.Int).Add(S, term)
		}
		if term.Cmp(precision) < 0 {
			return new(big.Int).Div(new(big.Int).Mul(result, S), constant.BONE), nil
		}
	}
	return nil, errors.New("did not converge")
}

func (t *PoolSimulator) getPrice(i, mA, mGamma *big.Int, xp []*big.Int, d *big.Int) (*big.Int, error) {
	nCoin := big.NewInt(int64(len(t.Info.Tokens)))
	k0 := new(big.Int).Div(
		new(big.Int).Mul(
			new(big.Int).Div(
				new(big.Int).Mul(
					new(big.Int).Mul(constant.BONE, new(big.Int).Mul(nCoin, nCoin)),
					xp[0],
				),
				d,
			),
			xp[1],
		),
		d,
	)
	if k0.Cmp(constant.BONE) > 0 {
		return nil, ErrK0
	}
	g1k0 := new(big.Int).Sub(new(big.Int).Add(mGamma, constant.BONE), k0)

	k := new(big.Int).Div(
		new(big.Int).Div(
			new(big.Int).Div(
				new(big.Int).Mul(
					new(big.Int).Div(
						new(big.Int).Mul(mA, new(big.Int).Mul(k0, mGamma)),
						g1k0,
					),
					mGamma,
				),
				g1k0,
			),
			AMultiplier,
		),
		new(big.Int).Mul(nCoin, nCoin),
	)
	s := new(big.Int).Add(xp[0], xp[1])
	if d.Cmp(s) > 0 {
		return nil, ErrD
	}

	mul := new(big.Int).Add(
		new(big.Int).Div(
			new(big.Int).Mul(
				new(big.Int).Sub(s, d),
				new(big.Int).Add(mGamma, new(big.Int).Add(constant.BONE, k0)),
			),
			g1k0,
		),
		new(big.Int).Div(
			new(big.Int).Div(
				new(big.Int).Mul(d, k0),
				k,
			),
			new(big.Int).Mul(nCoin, nCoin),
		),
	)
	j := int(new(big.Int).Sub(constants.One, i).Int64())
	dxi := new(big.Int).Add(
		PrecisionPriceScale,
		new(big.Int).Div(new(big.Int).Mul(PrecisionPriceScale, mul), xp[j]),
	)
	dxj := new(big.Int).Add(
		PrecisionPriceScale,
		new(big.Int).Div(new(big.Int).Mul(PrecisionPriceScale, mul), xp[int(i.Int64())]),
	)
	if i.Cmp(constants.Zero) == 0 {
		return new(big.Int).Div(
			new(big.Int).Mul(
				new(big.Int).Div(
					new(big.Int).Mul(PrecisionPriceScale, dxj),
					dxi,
				),
				PrecisionPriceScale,
			),
			t.PriceScale,
		), nil
	} else {
		return new(big.Int).Div(
			new(big.Int).Mul(
				new(big.Int).Div(
					new(big.Int).Mul(PrecisionPriceScale, dxj),
					dxi,
				),
				t.PriceScale,
			),
			PrecisionPriceScale,
		), nil
	}
}

func (t *PoolSimulator) aGamma() (*big.Int, *big.Int) {
	t1 := t.FutureAGammaTime
	mA := new(big.Int).Set(t.FutureA)
	mGamma := new(big.Int).Set(t.FutureGamma)

	var now = time.Now().Unix()
	if now < t1 {
		// handle ramping up and down of A
		t0 := t.InitialAGammaTime

		// Less readable but more compact way of writing and converting to uint256
		// gamma0: uint256 = bitwise_and(AGamma0, 2**128-1)
		// A0: uint256 = shift(AGamma0, -128)
		// A1 = A0 + (A1 - A0) * (block.timestamp - t0) / (t1 - t0)
		// gamma1 = gamma0 + (gamma1 - gamma0) * (block.timestamp - t0) / (t1 - t0)
		t1 -= t0
		t0 = now - t0
		t2 := t1 - t0

		mA = new(big.Int).Div(
			new(big.Int).Add(
				new(big.Int).Mul(t.InitialA, big.NewInt(t2)),
				new(big.Int).Mul(mA, big.NewInt(t0)),
			),
			big.NewInt(t1),
		)
		mGamma = new(big.Int).Div(
			new(big.Int).Add(
				new(big.Int).Mul(t.InitialGamma, big.NewInt(t2)),
				new(big.Int).Mul(mGamma, big.NewInt(t0)),
			),
			big.NewInt(t1),
		)
	}

	return mA, mGamma
}

func (t *PoolSimulator) feeRate(xp []*big.Int) *big.Int {
	nCoinBi := big.NewInt(2)
	feeGamma := new(big.Int).Set(t.FeeGamma)
	f := new(big.Int).Add(xp[0], xp[1])
	f = new(big.Int).Div(
		new(big.Int).Mul(feeGamma, constant.BONE),
		new(big.Int).Sub(
			new(big.Int).Add(feeGamma, constant.BONE),
			new(big.Int).Div(
				new(big.Int).Mul(
					new(big.Int).Div(
						new(big.Int).Mul(
							new(big.Int).Mul(constant.BONE, new(big.Int).Mul(nCoinBi, nCoinBi)),
							xp[0],
						),
						f,
					),
					xp[1],
				),
				f,
			),
		),
	)

	return new(big.Int).Div(
		new(big.Int).Add(
			new(big.Int).Mul(t.MidFee, f),
			new(big.Int).Mul(t.OutFee, new(big.Int).Sub(constant.BONE, f)),
		),
		constant.BONE,
	)
}

// GetDy https://basescan.org/address/0x73c3a78e5ff0d216a50b11d51b262ca839fcfe17#code
func (t *PoolSimulator) GetDy(i int, j int, dx *big.Int) (*big.Int, *big.Int, error) {
	if i+j != 1 {
		return nil, nil, fmt.Errorf("tokenIn and tokenOut are not valid")
	}

	mA, mGamma := t.aGamma()

	xp := []*big.Int{
		new(big.Int).Set(t.Pool.Info.Reserves[0]),
		new(big.Int).Set(t.Pool.Info.Reserves[1]),
	} // xp: uint256[N_COINS] = self.balances

	var err error
	d0 := new(big.Int).Set(t.D)
	if t.FutureAGammaTime > 0 {
		d0, err = newtonD(mA, mGamma, t.standardize(xp[0], xp[1]))
		if err != nil {
			return nil, nil, err
		}
	}

	xp[i] = new(big.Int).Add(xp[i], dx)
	xp = t.standardize(xp[0], xp[1])

	y, err := newtonY(mA, mGamma, xp[i], d0)
	if err != nil {
		return nil, nil, err
	}
	var dy = new(big.Int).Sub(xp[j], y)
	if dy.Cmp(constant.ZeroBI) <= 0 {
		return nil, nil, ErrDySmallerThanZero
	}

	xp[j] = new(big.Int).Sub(xp[j], dy)
	dy = t.unStandardize(new(big.Int).Sub(dy, constant.One), big.NewInt(int64(j)))
	dyFee := new(big.Int).Div(
		new(big.Int).Mul(t.feeRate(xp), dy),
		PrecisionFee,
	)
	dy = new(big.Int).Sub(dy, dyFee)

	return dy, dyFee, nil
}

func (t *PoolSimulator) Exchange(i int, j int, dx *big.Int) (*big.Int, error) {
	if i+j != 1 {
		return nil, ErrIndexOutOfRange
	}
	if dx.Cmp(constant.ZeroBI) <= 0 {
		return nil, errors.New("do not exchange 0 coins")
	}

	var mA, mGamma = t.aGamma()
	var err error
	if t.FutureAGammaTime > 0 {
		t.D, err = newtonD(mA, mGamma, t.standardize(t.Info.Reserves[0], t.Info.Reserves[1]))
		if err != nil {
			return nil, err
		}
		now := time.Now().Unix()
		if now >= t.FutureAGammaTime {
			t.FutureAGammaTime = 1
		}
	}

	xp1 := []*big.Int{
		new(big.Int).Set(t.Pool.Info.Reserves[0]),
		new(big.Int).Set(t.Pool.Info.Reserves[1]),
	} // xp: uint256[N_COINS] = self.balances
	xp1[i] = new(big.Int).Add(xp1[i], dx)
	t.Info.Reserves[0] = new(big.Int).Set(xp1[i])
	xp1 = t.standardize(xp1[0], xp1[1])

	y, err := newtonY(mA, mGamma, xp1[i], t.D)
	if err != nil {
		return nil, err
	}
	dy := new(big.Int).Sub(xp1[j], y)
	if dy.Cmp(constant.ZeroBI) <= 0 {
		return nil, ErrDySmallerThanZero
	}

	xp1[j] = new(big.Int).Sub(xp1[j], dy)
	dy = t.unStandardize(new(big.Int).Sub(dy, constant.One), big.NewInt(int64(j)))
	dyFee := new(big.Int).Div(
		new(big.Int).Mul(t.feeRate(xp1), dy),
		PrecisionFee,
	)
	dy = new(big.Int).Sub(dy, dyFee)
	t.Info.Reserves[j] = new(big.Int).Sub(t.Info.Reserves[j], dy)

	xp1 = t.standardize(t.Info.Reserves[0], t.Info.Reserves[1])
	err = t.tweakPrice(mA, mGamma, xp1, big.NewInt(0))
	if err != nil {
		return nil, err
	}

	return dy, nil
}

func (t *PoolSimulator) tweakPrice(mA *big.Int, mGamma *big.Int, xp []*big.Int, newD *big.Int) error {
	oldPriceScale := new(big.Int).Set(t.PriceScale)
	newPriceOracle := new(big.Int).Set(t.PriceOracle)
	lastPricesTmp := new(big.Int).Set(t.LastPrices)

	lastPricesTimestamp := t.LastPricesTimestamp
	blockTimestamp := time.Now().Unix()
	if lastPricesTimestamp < blockTimestamp {
		maHalfTime := new(big.Int).Set(t.MaHalfTime)
		alpha, _ := halfpow(
			new(big.Int).Div(
				new(big.Int).Mul(big.NewInt(blockTimestamp-lastPricesTimestamp), constant.BONE), maHalfTime,
			),
			constant.TenPowInt(10),
		)
		tmp := new(big.Int).Mul(constant.Two, oldPriceScale)
		if tmp.Cmp(lastPricesTmp) < 0 {
			tmp = lastPricesTmp
		}
		price := new(big.Int).Div(oldPriceScale, constant.Two)
		if price.Cmp(tmp) < 0 {
			price = tmp
		}
		newPriceOracle = new(big.Int).Div(
			new(big.Int).Add(
				new(big.Int).Mul(
					price,
					new(big.Int).Sub(constant.BONE, alpha),
				),
				new(big.Int).Mul(newPriceOracle, alpha),
			),
			constant.BONE,
		)
		t.PriceOracle = newPriceOracle
		t.LastPricesTimestamp = blockTimestamp
	}

	var err error
	if newD.Cmp(constant.ZeroBI) == 0 {
		newD, err = newtonD(mA, mGamma, xp)
		if err != nil {
			return err
		}
	}
	lastPricesTmp, err = t.getPrice(big.NewInt(1), mA, mGamma, xp, newD)
	if err != nil {
		return err
	}
	t.LastPrices = new(big.Int).Set(lastPricesTmp)
	oldVirtualPrice := new(big.Int).Set(t.VirtualPrice)

	// Update profit numbers without price adjustment first
	nCoin := big.NewInt(int64(len(t.Info.Tokens)))
	mXp := make([]*big.Int, len(t.Info.Tokens))
	mXp[0] = new(big.Int).Div(newD, nCoin)
	mXp[1] = new(big.Int).Div(
		new(big.Int).Mul(newD, PrecisionPriceScale),
		new(big.Int).Mul(nCoin, oldPriceScale),
	)
	newXcpProfit := new(big.Int).Set(constant.BONE)
	newVirtualPrice := new(big.Int).Set(constant.BONE)
	lpTotalSupply := new(big.Int).Set(t.LpSupply)

	if oldVirtualPrice.Cmp(constants.Zero) > 0 {
		tmpGeo, err := geometricMean(mXp, true)
		if err != nil {
			return err
		}
		newVirtualPrice = new(big.Int).Div(
			new(big.Int).Mul(constant.BONE, tmpGeo),
			lpTotalSupply,
		)
		newXcpProfit = new(big.Int).Div(
			new(big.Int).Mul(t.XcpProfit, newVirtualPrice),
			oldVirtualPrice,
		)

		tTmp := t.FutureAGammaTime
		if newVirtualPrice.Cmp(oldVirtualPrice) < 0 && tTmp == 0 {
			return ErrLoss
		}
		if tTmp == 1 {
			t.FutureAGammaTime = 0
		}
	}

	t.XcpProfit = newXcpProfit
	norm := new(big.Int).Div(
		new(big.Int).Mul(newPriceOracle, constant.BONE),
		oldPriceScale,
	)
	if norm.Cmp(constant.BONE) > 0 {
		norm = new(big.Int).Sub(norm, constant.BONE)
	} else {
		norm = new(big.Int).Sub(constant.BONE, norm)
	}
	mAdjustmentStep := new(big.Int).Div(norm, constant.Five)
	if mAdjustmentStep.Cmp(t.AdjustmentStep) < 0 {
		mAdjustmentStep = new(big.Int).Set(t.AdjustmentStep)
	}

	tmpBool := new(big.Int).Sub(newVirtualPrice, constant.BONE).Cmp(
		new(big.Int).Add(
			new(big.Int).Div(
				new(big.Int).Mul(
					new(big.Int).Sub(newXcpProfit, constant.BONE),
					t.MinRemainingPostRebalanceRatio,
				),
				PrecisionFee,
			),
			t.AllowedExtraProfit,
		),
	) > 0
	needsAdjustment := t.NotAdjusted
	if !needsAdjustment && tmpBool && norm.Cmp(mAdjustmentStep) > 0 && oldVirtualPrice.Cmp(constants.Zero) > 0 {
		needsAdjustment = true
		t.NotAdjusted = true
	}

	if needsAdjustment {
		if norm.Cmp(mAdjustmentStep) > 0 && oldVirtualPrice.Cmp(constants.Zero) > 0 {
			// We reuse lastPrices_ as pNew
			lastPricesTmp = new(big.Int).Div(
				new(big.Int).Add(
					new(big.Int).Mul(oldPriceScale, new(big.Int).Sub(norm, mAdjustmentStep)),
					new(big.Int).Mul(mAdjustmentStep, newPriceOracle),
				),
				norm,
			)

			// Calculate balances*prices
			mXp[0] = xp[0]
			mXp[1] = new(big.Int).Div(new(big.Int).Mul(xp[1], lastPricesTmp), oldPriceScale)

			// Calculate "extended constant product" invariant xCP and virtual price
			tmpD, err := newtonD(mA, mGamma, mXp)
			if err != nil {
				return err
			}
			mXp[0] = new(big.Int).Div(tmpD, nCoin)
			mXp[1] = new(big.Int).Div(
				new(big.Int).Mul(tmpD, PrecisionPriceScale),
				new(big.Int).Mul(nCoin, lastPricesTmp),
			)

			// We reuse oldVirtualPrice here but it's not old anymore
			tmpGeo, err := geometricMean(mXp, true)
			if err != nil {
				return err
			}
			oldVirtualPrice = new(big.Int).Div(new(big.Int).Mul(constant.BONE, tmpGeo), lpTotalSupply)

			// Proceed if we've got enough profit
			// if (oldVirtualPrice > 10**18) and (oldVirtualPrice - 10**18 > (xcpProfit - 10**18) * minRemainingPostRebalanceRatio):
			if oldVirtualPrice.Cmp(constant.BONE) > 0 &&
				new(big.Int).Sub(oldVirtualPrice, constant.BONE).Cmp(
					new(big.Int).Div(
						new(big.Int).Mul(
							new(big.Int).Sub(newXcpProfit, constant.BONE),
							t.MinRemainingPostRebalanceRatio,
						),
						PrecisionFee,
					),
				) > 0 {
				t.PriceScale = lastPricesTmp
				t.D = tmpD
				t.VirtualPrice = oldVirtualPrice

				return nil
			} else {
				t.NotAdjusted = false
				t.D = newD
				t.VirtualPrice = newVirtualPrice

				return nil
			}
		}
	}

	t.D = newD
	t.VirtualPrice = newVirtualPrice
	if needsAdjustment {
		t.NotAdjusted = false
	}

	return nil
}

func (t *PoolSimulator) standardize(x, y *big.Int) []*big.Int {
	var result = make([]*big.Int, 2)
	result[0] = new(big.Int).Mul(x, t.Precisions[0])
	result[1] = new(big.Int).Div(
		new(big.Int).Mul(y, new(big.Int).Mul(t.Precisions[1], t.PriceScale)),
		PrecisionPriceScale,
	)

	return result
}

func (t *PoolSimulator) unStandardize(standizeAmount, i *big.Int) *big.Int {
	if i.Cmp(constant.ZeroBI) == 0 {
		return new(big.Int).Div(standizeAmount, t.Precisions[0])
	}

	return new(big.Int).Div(
		new(big.Int).Div(
			new(big.Int).Mul(standizeAmount, PrecisionPriceScale),
			t.Precisions[1],
		),
		t.PriceScale,
	)
}
