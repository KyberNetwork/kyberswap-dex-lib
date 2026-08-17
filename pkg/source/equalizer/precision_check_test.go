package equalizer

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/holiman/uint256"
)

func TestFeePrecision(t *testing.T) {
	// replicate pool_tracker.go: p.SwapFee = float64(realFee.Uint64()) / 1e18
	realFee := uint64(12500000000000000) // 0x2c68af0bb14000 on-chain
	swapFeeFloat64 := float64(realFee) / 1e18
	fmt.Printf("swapFee float64: %.20f\n", swapFeeFloat64)

	// replicate NewPoolSimulator
	swapFeeBigFloat := new(big.Float).Mul(big.NewFloat(swapFeeFloat64), new(big.Float).SetFloat64(1e18))
	swapFeeBig, _ := swapFeeBigFloat.Int(nil)
	fmt.Printf("recovered swapFeeBig: %s (expected %d, diff %d)\n", swapFeeBig, realFee, new(big.Int).Sub(swapFeeBig, big.NewInt(int64(realFee))).Int64())

	// amount out diff caused by 1 wei fee diff on 1e18 in
	amountIn := uint256.NewInt(1e18)
	for _, fee := range []uint64{12500000000000000, 12500000000000001} {
		amt := calAmountAfterFee(amountIn, uint256.NewInt(fee))
		fmt.Printf("fee=%d amountAfterFee=%s\n", fee, amt)
	}
}
