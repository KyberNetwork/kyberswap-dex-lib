package tricrypto

import (
	"math/big"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

func TestNewtonY(t *testing.T) {
	t.Parallel()
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
	for k := range 3 {
		xp[k] = balances[k]
	}
	xp[i] = new(big.Int).Add(xp[i], dx)
	xp[0] = new(big.Int).Mul(xp[0], precisions[0])

	for k := range 2 {
		xp[k+1] = new(big.Int).Div(new(big.Int).Mul(new(big.Int).Mul(xp[k+1], priceScale[k]), precisions[k+1]), precision)
	}
	u := func(b *big.Int) uint256.Int { return *uint256.MustFromBig(b) }
	xpU := [nCoins]uint256.Int{u(xp[0]), u(xp[1]), u(xp[2])}

	var ret, err = newtonY(u(ann), u(gamma), xpU, u(D), 2)
	assert.Nil(t, err)
	assert.Equal(t, ret.Dec(), "221721964731657747695742644")
	ret, err = newtonY(u(ann), u(gamma), xpU, u(D), 1)
	assert.Nil(t, err)
	assert.Equal(t, ret.Dec(), "217165474602433869159283104")
	ret, err = newtonY(u(ann), u(gamma), xpU, u(D), 0)
	assert.Nil(t, err)
	assert.Equal(t, ret.Dec(), "220406131330584000207811144")
}

// newton_D asserts MIN_A <= ANN <= MAX_A (the contract reverts "unsafe values A").
func TestNewtonDRejectsUnsafeA(t *testing.T) {
	t.Parallel()
	x1 := *uint256.MustFromDecimal("1000000000000000000000000") // 1e24
	x := [nCoins]uint256.Int{x1, x1, x1}
	gamma := *uint256.NewInt(11809167828997)

	D, err := newtonD(*uint256.NewInt(1707629), gamma, x)
	assert.NoError(t, err)
	assert.Equal(t, "3000000000000000000000000", D.Dec())

	for _, ann := range []*uint256.Int{new(uint256.Int).SubUint64(MinA, 1), new(uint256.Int).AddUint64(MaxA, 1)} {
		_, err = newtonD(*ann, gamma, x)
		assert.ErrorIs(t, err, errUnsafeA, ann.Dec())
	}
}
