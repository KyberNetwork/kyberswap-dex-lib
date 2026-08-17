package equalizer

import (
	"fmt"
	"testing"

	"github.com/holiman/uint256"
)

func TestStableDifferential(t *testing.T) {
	r0, _ := uint256.FromDecimal("262559985336414556295")
	r1, _ := uint256.FromDecimal("223724271")
	fee, _ := uint256.FromDecimal("262500000000000")
	dec18 := uint256.NewInt(1e18)
	dec6 := uint256.NewInt(1e6)

	// 1e18 token0 in -> token1 out
	out1, err := getAmountOut(uint256.NewInt(1e18), r0, r1, dec18, dec6, fee, true)
	fmt.Printf("stable 1e18 t0->t1: offchain=%s onchain=998467 err=%v\n", out1, err)

	// 100e6 token1 in -> token0 out
	out2, err := getAmountOut(uint256.NewInt(100000000), r1, r0, dec6, dec18, fee, true)
	fmt.Printf("stable 100e6 t1->t0: offchain=%s onchain=98554724142238204844 err=%v\n", out2, err)
}