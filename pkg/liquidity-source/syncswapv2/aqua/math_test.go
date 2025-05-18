package syncswapv2aqua

import (
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"

	constant "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

func TestUint256Refactor(t *testing.T) {
	t.Parallel()
	func(D, gamma, _g1k0, ANN, AMultiplier *uint256.Int) {
		var mul1 = new(uint256.Int)
		assert.Equal(t, new(uint256.Int).Div(
			new(uint256.Int).Mul(
				new(uint256.Int).Mul(
					new(uint256.Int).Div(
						new(uint256.Int).Mul(
							new(uint256.Int).Div(new(uint256.Int).Mul(constant.BONE, D), gamma), _g1k0,
						), gamma,
					),
					_g1k0,
				),
				AMultiplier,
			), ANN,
		), mul1.Mul(constant.BONE, D).Div(mul1, gamma).Mul(mul1, _g1k0).Div(mul1, gamma).Mul(mul1, _g1k0).Mul(mul1, AMultiplier).Div(mul1, ANN), "fail")
	}(uint256.NewInt(2), uint256.NewInt(3), uint256.NewInt(4), uint256.NewInt(5), uint256.NewInt(6))

	func(nCoinsBi, K0, _g1k0 *uint256.Int) {
		var mul2 = new(uint256.Int)
		assert.Equal(t, new(uint256.Int).Div(
			new(uint256.Int).Mul(
				new(uint256.Int).Mul(
					new(uint256.Int).Mul(constant.Two, constant.BONE), nCoinsBi,
				), K0,
			), _g1k0,
		), mul2.Mul(constant.Two, constant.BONE).Mul(mul2, nCoinsBi).Mul(mul2, K0).Div(mul2, _g1k0), "fail")
	}(uint256.NewInt(2), uint256.NewInt(3), uint256.NewInt(4))

	func(S, mul2, mul1, nCoinsBi, K0, D *uint256.Int) {
		var negFprime = new(uint256.Int)
		assert.Equal(t, new(uint256.Int).Sub(
			new(uint256.Int).Add(
				new(uint256.Int).Add(S, new(uint256.Int).Div(new(uint256.Int).Mul(S, mul2), constant.BONE)),
				new(uint256.Int).Div(new(uint256.Int).Mul(mul1, nCoinsBi), K0),
			),
			new(uint256.Int).Div(new(uint256.Int).Mul(mul2, D), constant.BONE),
		), negFprime.Mul(S, mul2).Div(negFprime, constant.BONE).Add(S, negFprime).Add(
			new(uint256.Int).Set(negFprime),
			negFprime.Mul(mul1, nCoinsBi).Div(negFprime, K0),
		).Sub(
			new(uint256.Int).Set(negFprime),
			negFprime.Mul(mul2, D).Div(negFprime, constant.BONE),
		), "fail")
	}(uint256.NewInt(2), uint256.NewInt(3), uint256.NewInt(4), uint256.NewInt(5), uint256.NewInt(6), uint256.NewInt(7))

	func(D, negFprime, S *uint256.Int) {
		var DPlus = new(uint256.Int)
		assert.Equal(t, new(uint256.Int).Div(new(uint256.Int).Mul(D, new(uint256.Int).Add(negFprime, S)), negFprime),
			DPlus.Add(negFprime, S).Mul(D, DPlus).Div(DPlus, negFprime), "fail")
	}(uint256.NewInt(2), uint256.NewInt(3), uint256.NewInt(4))

	func(D, negFprime *uint256.Int) {
		var DMinus = new(uint256.Int)
		assert.Equal(t, new(uint256.Int).Div(new(uint256.Int).Mul(D, D), negFprime),
			DMinus.Mul(D, D).Div(DMinus, negFprime), "fail")
	}(uint256.NewInt(2), uint256.NewInt(3))

	func(D, mul1, negFprime, K0, DMinus *uint256.Int) {
		assert.Equal(t, new(uint256.Int).Add(
			DMinus,
			new(uint256.Int).Div(
				new(uint256.Int).Mul(
					new(uint256.Int).Div(
						new(uint256.Int).Mul(D, new(uint256.Int).Div(mul1, negFprime)), constant.BONE,
					), new(uint256.Int).Sub(constant.BONE, K0),
				),
				K0,
			),
		), DMinus.Add(new(
			uint256.Int).Set(DMinus),
			DMinus.Div(mul1, negFprime).Mul(D, DMinus).Div(DMinus, constant.BONE).Mul(DMinus, new(uint256.Int).Sub(constant.BONE, K0)).Div(DMinus, K0),
		), "fail")
	}(uint256.NewInt(2), uint256.NewInt(3), uint256.NewInt(4), uint256.NewInt(5), uint256.NewInt(6))

	func(D, mul1, negFprime, K0, DMinus *uint256.Int) {
		assert.Equal(t, new(uint256.Int).Sub(
			DMinus,
			new(uint256.Int).Div(
				new(uint256.Int).Mul(
					new(uint256.Int).Div(
						new(uint256.Int).Mul(D, new(uint256.Int).Div(mul1, negFprime)), constant.BONE,
					), new(uint256.Int).Sub(K0, constant.BONE),
				),
				K0,
			),
		), DMinus.Add(new(
			uint256.Int).Set(DMinus),
			DMinus.Div(mul1, negFprime).Mul(D, DMinus).Div(DMinus, constant.BONE).Mul(DMinus, new(uint256.Int).Sub(K0, constant.BONE)).Div(DMinus, K0),
		), "fail")
	}(uint256.NewInt(2), uint256.NewInt(3), uint256.NewInt(4), uint256.NewInt(5), uint256.NewInt(6))

	func(K0, _g1k0 *uint256.Int) {
		var mul2 = new(uint256.Int)
		assert.Equal(t, new(uint256.Int).Add(
			new(uint256.Int).Div(
				new(uint256.Int).Mul(
					new(uint256.Int).Mul(constant.Two, constant.BONE), K0,
				), _g1k0,
			), constant.BONE,
		), mul2.Mul(constant.Two, constant.BONE).Mul(mul2, K0).Div(mul2, _g1k0).Add(mul2, constant.BONE), "fail")
	}(uint256.NewInt(2), uint256.NewInt(3))

	func(gamma, ETHER, xp0, xp1, f *uint256.Int) {
		// f = gamma.mul(ETHER).div(
		//     gamma.add(ETHER).sub(ETHER.mul(4).mul(xp0).div(f).mul(xp1).div(f))
		// );
		f1 := new(uint256.Int).Set(f)
		assert.Equal(t, new(uint256.Int).Div(
			new(uint256.Int).Mul(
				gamma,
				constant.BONE,
			),
			new(uint256.Int).Sub(
				new(uint256.Int).Add(
					gamma,
					constant.BONE,
				),
				new(uint256.Int).Div(
					new(uint256.Int).Mul(
						new(uint256.Int).Div(
							new(uint256.Int).Mul(
								new(uint256.Int).Mul(
									constant.BONE,
									constant.Four,
								),
								xp0,
							),
							f,
						),
						xp1,
					),
					f,
				),
			),
		), f.Mul(constant.BONE, constant.Four).Mul(f, xp0).Div(f, f1).Mul(f, xp1).Div(f, f1).Sub(
			new(uint256.Int).Add(gamma, constant.BONE), f,
		).Div(
			new(uint256.Int).Mul(gamma, constant.BONE), f,
		), "fail")
	}(uint256.NewInt(2), uint256.NewInt(3), uint256.NewInt(4), uint256.NewInt(5), uint256.NewInt(6))

	func(minFee, maxFee, f *uint256.Int) {
		var fee = new(uint256.Int)
		assert.Equal(t, new(uint256.Int).Div(
			new(uint256.Int).Add(
				new(uint256.Int).Mul(
					minFee, f,
				),
				new(uint256.Int).Mul(
					maxFee,
					new(uint256.Int).Sub(
						constant.BONE, f,
					),
				),
			),
			constant.BONE,
		), fee.Sub(constant.BONE, f).Mul(maxFee, fee).Add(fee, new(uint256.Int).Mul(minFee, f)).Div(fee, constant.BONE), "fail")
	}(uint256.NewInt(2), uint256.NewInt(3), uint256.NewInt(4))
}
