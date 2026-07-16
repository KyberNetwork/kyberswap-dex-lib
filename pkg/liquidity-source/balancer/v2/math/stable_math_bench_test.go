package math

import (
	"testing"

	"github.com/holiman/uint256"
)

func BenchmarkGetTokenBalanceGivenInvariantAndAllOtherBalances(b *testing.B) {
	amp := uint256.NewInt(5000)
	balances := []*uint256.Int{
		uint256.MustFromDecimal("9999991000000000000000"),
		uint256.MustFromDecimal("99999910000000000056"),
		uint256.MustFromDecimal("8897791020011100123456"),
		uint256.MustFromDecimal("13288977911102200123456"),
		uint256.MustFromDecimal("199791011102200123456"),
		uint256.MustFromDecimal("1997200112156340123456"),
	}
	invariant, err := StableMath.CalculateInvariantV1(amp, balances, true)
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()

	for b.Loop() {
		_, err := StableMath.GetTokenBalanceGivenInvariantAndAllOtherBalances(amp, balances, invariant, 2)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCalcOutGivenIn(b *testing.B) {
	amp := uint256.NewInt(5000)
	balances := []*uint256.Int{
		uint256.MustFromDecimal("9999991000000000000000"),
		uint256.MustFromDecimal("99999910000000000056"),
		uint256.MustFromDecimal("8897791020011100123456"),
		uint256.MustFromDecimal("13288977911102200123456"),
		uint256.MustFromDecimal("199791011102200123456"),
		uint256.MustFromDecimal("1997200112156340123456"),
	}
	invariant, err := StableMath.CalculateInvariantV1(amp, balances, true)
	if err != nil {
		b.Fatal(err)
	}
	amountIn := uint256.MustFromDecimal("1000000000000000000")

	b.ReportAllocs()

	for b.Loop() {
		bals := make([]*uint256.Int, len(balances))
		copy(bals, balances)
		_, err := StableMath.CalcOutGivenIn(invariant, amp, amountIn, bals, 0, 2)
		if err != nil {
			b.Fatal(err)
		}
	}
}
