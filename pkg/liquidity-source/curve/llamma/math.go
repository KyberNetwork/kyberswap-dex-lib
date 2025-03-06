package llamma

import (
	"github.com/KyberNetwork/int256"
	"github.com/holiman/uint256"
)

func MaxUint256(a, b *uint256.Int) *uint256.Int {
	if a.Cmp(b) > 0 {
		return a.Clone()
	}
	return b.Clone()
}

func LnInt(x *uint256.Int) *int256.Int {

	var res uint256.Int
	for i := 0; i < 8; i++ {
		t := new(uint256.Int).Exp(uint256.NewInt(2), uint256.NewInt(uint64(7-i)))
		p := new(uint256.Int).Exp(uint256.NewInt(2), t)
		if x.Cmp(new(uint256.Int).Mul(p, tenPow18)) >= 0 {
			x.Div(x, p)
			res.Add(&res, new(uint256.Int).Mul(t, tenPow18))
		}
	}
	d := tenPow18.Clone()
	for i := 0; i < 59; i++ {
		if x.Cmp(new(uint256.Int).Mul(uint256.NewInt(2), tenPow18)) >= 0 {
			res.Add(&res, d.Clone())
			x.Div(x, uint256.NewInt(2))
		}
		x.Mul(x, x).Div(x, tenPow18)
		d.Div(d, uint256.NewInt(2))
	}
	res.Mul(&res, tenPow18).Div(&res, uint256.NewInt(1442695040888963328))
	i256Res := new(int256.Int)
	_ = i256Res.SetFromDec(res.String())
	return i256Res
}
