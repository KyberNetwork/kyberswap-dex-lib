package i256

import (
	"math"
	"math/big"

	"github.com/KyberNetwork/int256"
	"github.com/samber/lo"
	"golang.org/x/exp/constraints"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var (
	Zero = int256.NewInt(0)
	One  = int256.NewInt(1)
	Two  = int256.NewInt(2)
	Four = int256.NewInt(4)

	preTenPow = lo.Map(lo.Range(40), func(n int, _ int) *int256.Int {
		if n < 19 {
			return int256.NewInt(int64(math.Pow10(n)))
		}
		val := new(big.Int).Exp(bignumber.Ten, big.NewInt(int64(n)), nil)
		result, _ := int256.FromBig(val)
		return result
	})
)

func TenPow[T constraints.Integer](n T) *int256.Int {
	if int(n) < len(preTenPow) {
		return preTenPow[n]
	}
	val := new(big.Int).Exp(bignumber.Ten, big.NewInt(int64(n)), nil)
	result, _ := int256.FromBig(val)
	return result
}
