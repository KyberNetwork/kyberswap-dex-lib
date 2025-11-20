package listaswap

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

func (t *PoolSimulator) getDyWithoutFee(
	updatedBalances []*big.Int,
	i int,
	j int,
	dx *big.Int,
) (*big.Int, error) {
	xp := t._xp(updatedBalances)

	x := new(big.Int).Add(xp[i], new(big.Int).Div(new(big.Int).Mul(dx, t.baseSim.Rates[i]), Precision))
	y, err := t.baseSim.GetY(i, j, x, xp, nil)
	if err != nil {
		return nil, err
	}

	dy := new(big.Int).Sub(new(big.Int).Sub(xp[j], y), bignumber.One)

	return dy, nil
}

func (t *PoolSimulator) _xp(balances []*big.Int) []*big.Int {
	var nTokens = len(t.Info.Tokens)
	result := make([]*big.Int, nTokens)
	for i := 0; i < nTokens; i += 1 {
		result[i] = new(big.Int).Div(new(big.Int).Mul(t.baseSim.Rates[i], balances[i]), Precision)
	}
	return result
}
