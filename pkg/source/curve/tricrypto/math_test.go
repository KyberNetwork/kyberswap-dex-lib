package tricrypto

import (
	"math/big"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/stretchr/testify/assert"
)

func TestNewtonY(t *testing.T) {
	var precisions = []*big.Int{
		bignumber.NewBig10("1000000000000"),
		bignumber.NewBig10("10000000000"),
		bignumber.NewBig10("1"),
	}
	var precision = bignumber.NewBig10("1000000000000000000")
	var ann = bignumber.NewBig10("1707629")
	var gamma = bignumber.NewBig10("11809167828997")
	var D = bignumber.NewBig10("659307468228931998580648112")
	var priceScale = []*big.Int{
		bignumber.NewBig10("55192676963173615208913"),
		bignumber.NewBig10("3485034192326999988769"),
	}
	var balances = []*big.Int{
		bignumber.NewBig10("220406131330584"),
		bignumber.NewBig10("393490059984"),
		bignumber.NewBig10("63624729793505614488987"),
	}
	var dx = bignumber.NewBig10("12345000000")
	var i = 0

	var xp = make([]*big.Int, 3)
	for k := 0; k < 3; k += 1 {
		xp[k] = balances[k]
	}
	xp[i] = new(big.Int).Add(xp[i], dx)
	xp[0] = new(big.Int).Mul(xp[0], precisions[0])

	for k := 0; k < 2; k += 1 {
		xp[k+1] = new(big.Int).Div(new(big.Int).Mul(new(big.Int).Mul(xp[k+1], priceScale[k]), precisions[k+1]), precision)
	}
	var ret, err = newtonY(ann, gamma, xp, D, 2)
	assert.Nil(t, err)
	assert.Equal(t, ret.String(), "221721964731657747695742644")
	ret, err = newtonY(ann, gamma, xp, D, 1)
	assert.Nil(t, err)
	assert.Equal(t, ret.String(), "217165474602433869159283104")
	ret, err = newtonY(ann, gamma, xp, D, 0)
	assert.Nil(t, err)
	assert.Equal(t, ret.String(), "220406131330584000207811144")
}
