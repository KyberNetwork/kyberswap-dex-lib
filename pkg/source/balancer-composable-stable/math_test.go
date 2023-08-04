package balancercomposablestable

import (
	"math/big"
	"testing"
)

func TestCalculateInvariant(t *testing.T) {
	amp := big.NewInt(5000000)
	b1, _ := new(big.Int).SetString("1317130394069039114846", 10)

	balances := []*big.Int{
		b1,
		big.NewInt(14000000000000),
	}
	t.Log(CalculateInvariant(amp, balances, false))

}
