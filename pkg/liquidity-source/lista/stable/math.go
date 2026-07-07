package stable

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

	// x = xp[i] + dx * rates[i] / PRECISION
	x := bignumber.MulDivDown(new(big.Int), dx, t.baseSim.Rates[i], bignumber.BONE)
	x.Add(xp[i], x)

	y, err := t.baseSim.GetY(i, j, x, xp, nil)
	if err != nil {
		return nil, err
	}

	// dy = (xp[j] - y - 1) * PRECISION / rates[j], converting XP → raw token units (matches get_dy_without_fee on-chain)
	x.Sub(xp[j], y)
	x.Sub(x, bignumber.One)
	return bignumber.MulDivDown(x, x, bignumber.BONE, t.baseSim.Rates[j]), nil
}

func (t *PoolSimulator) _xp(balances []*big.Int) []*big.Int {
	result := make([]*big.Int, len(t.Info.Tokens))
	for i := range t.Info.Tokens {
		result[i] = bignumber.MulDivDown(new(big.Int), t.baseSim.Rates[i], balances[i], bignumber.BONE)
	}
	return result
}
