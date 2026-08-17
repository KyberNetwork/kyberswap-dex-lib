package equalizer

import (
	"fmt"
	"math/big"
	"testing"
)

func TestFeeRoundtripSweep(t *testing.T) {
	fees := []uint64{
		200000000000000,     // 0.02% stable
		250000000000000,     // 0.025%
		300000000000000,     // 0.03%
		1000000000000000,    // 0.1%
		1250000000000000,    // 0.125%
		2000000000000000,    // 0.2%
		2500000000000000,    // 0.25%
		3000000000000000,    // 0.3%
		5000000000000000,    // 0.5%
		10000000000000000,   // 1%
		12500000000000000,   // 1.25%
		30000000000000000,   // 3%
		999999999999999999,  // ~100%
	}
	for _, f := range fees {
		recovered, _ := new(big.Float).Mul(big.NewFloat(float64(f)/1e18), new(big.Float).SetFloat64(1e18)).Int(nil)
		diff := new(big.Int).Sub(recovered, big.NewInt(int64(f))).Int64()
		fmt.Printf("fee=%d recovered=%s diff=%d\n", f, recovered, diff)
	}
}