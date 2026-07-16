package libv2

import (
	"math/big"
	"testing"

	"github.com/holiman/uint256"
)

func BenchmarkSafeSqrt(b *testing.B) {
	x, _ := uint256.FromBig(new(big.Int).Exp(big.NewInt(10), big.NewInt(36), nil))

	b.ReportAllocs()

	for b.Loop() {
		SafeSqrt(x)
	}
}
