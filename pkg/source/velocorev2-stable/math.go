package velocorev2stable

import (
	"math/big"

	"github.com/KyberNetwork/blockchain-toolkit/integer"
)

func sqrtRounding(x *big.Int, up bool) *big.Int {
	result := sqrt(x)
	if up && x.Cmp(new(big.Int).Mul(result, result)) > 0 {
		result = new(big.Int).Add(result, integer.One())
	}
	return result
}

func sqrt(x *big.Int) *big.Int {
	if x.Cmp(integer.Zero()) == 0 {
		return integer.Zero()
	}

	result := new(big.Int).Lsh(
		integer.One(),
		log2(x)>>1,
	)

	for i := 0; i < 7; i++ {
		result = new(big.Int).Rsh(
			new(big.Int).Add(
				result,
				new(big.Int).Div(x, result),
			), 1,
		)
	}

	v := new(big.Int).Div(x, result)
	if result.Cmp(v) > 0 {
		result = v
	}

	return result
}

func log2(x *big.Int) uint {
	result := 0
	zero := integer.Zero()
	for i := 7; i >= 0; i-- {
		n := 1 << i
		if new(big.Int).Rsh(x, uint(n)).Cmp(zero) > 0 {
			x = new(big.Int).Rsh(x, uint(n))
			result += n
		}
	}
	return uint(result)
}
