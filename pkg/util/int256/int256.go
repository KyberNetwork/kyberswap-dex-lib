package i256

import (
	"math/big"

	"github.com/KyberNetwork/int256"
	"github.com/samber/lo"
)

var (
	Zero = int256.NewInt(0)
	One  = int256.NewInt(1)
	Two  = int256.NewInt(2)
	Four = int256.NewInt(4)

	preTenPow = initTenPowCache()
)

func initTenPowCache() []*int256.Int {
	cache := make([]*int256.Int, 40)
	for i := 0; i < 40; i++ {
		val := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(i)), nil)
		cache[i], _ = int256.FromBig(val)
	}
	return cache
}

func TenPow(n uint64) *int256.Int {
	if n < uint64(len(preTenPow)) {
		return preTenPow[n]
	}
	val := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(n)), nil)
	result, _ := int256.FromBig(val)
	return result
}

func MustFromBigs[S ~[]*big.Int](bigs S) []*int256.Int {
	return lo.Map(bigs, func(b *big.Int, _ int) *int256.Int {
		i, _ := int256.FromBig(b)
		return i
	})
}

// Min returns the smaller of a or b.
func Min(a, b *int256.Int) *int256.Int {
	if a.Cmp(b) < 0 {
		return new(int256.Int).Set(a)
	}
	return new(int256.Int).Set(b)
}

// Max returns the larger of a or b.
func Max(a, b *int256.Int) *int256.Int {
	if a.Cmp(b) > 0 {
		return new(int256.Int).Set(a)
	}
	return new(int256.Int).Set(b)
}
