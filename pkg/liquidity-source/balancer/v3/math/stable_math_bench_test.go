package math

import (
	"testing"

	"github.com/holiman/uint256"
)

func BenchmarkStableMath_ComputeBalance(b *testing.B) {
	amp := uint256.NewInt(200000)
	balances := []*uint256.Int{
		uint256.MustFromDecimal("340867122491122140643"),
		uint256.MustFromDecimal("384610409069784884043"),
	}
	invariant := uint256.MustFromDecimal("725470946757739599230")

	b.ReportAllocs()
	for b.Loop() {
		_, err := StableMath.ComputeBalance(amp, balances, invariant, 1)
		if err != nil {
			b.Fatal(err)
		}
	}
}
