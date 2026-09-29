package lunarbase

import (
	"testing"

	"github.com/holiman/uint256"
)

// A successful legacy getter with K=0 still uses the legacy fee denominator.
// In the punishment model the same max uint24 fee is a 100% sentinel. Mixing
// the models rejects a legal one-wei legacy output at this exact boundary.
func TestZeroConcentrationRetainsLegacyFeeSemantics(t *testing.T) {
	for _, xToY := range []bool{true, false} {
		p := PoolParams{ConcentrationModel: true,
			SqrtPriceX96: new(uint256.Int).Lsh(uint256.NewInt(1), 96),
			ReserveX:     uint256.NewInt(1 << 30), ReserveY: uint256.NewInt(1 << 30),
			FeeAskX24: maxU24, FeeBidX24: maxU24}
		quote := quoteYToX
		if xToY {
			quote = quoteXToY
		}
		a := quote(&p, uint256.NewInt(1<<24))
		if a.AmountOut.Uint64() != 1 || a.Fee.Uint64() != maxU24 {
			t.Fatalf("legacy K=0 direction=%t: got out=%s fee=%s", xToY, a.AmountOut, a.Fee)
		}
		p.ConcentrationModel = false
		b := quote(&p, uint256.NewInt(1<<24))
		if !b.AmountOut.IsZero() || b.Fee.Uint64() != 1<<24 {
			t.Fatalf("punishment max fee direction=%t lost sentinel semantics", xToY)
		}
	}
}
