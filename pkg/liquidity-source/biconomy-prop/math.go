package biconomyprop

import (
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

var bpsDen = uint256.NewInt(bpsDenominator)

// segmentOut sets out = take * price / 1e18 rounded down, the per-level arithmetic of the venue's
// merge and the executor's fill.
func segmentOut(out, take, price *uint256.Int) {
	big256.MulWadDown(out, take, price)
}

// boardLive mirrors the venue's inclusion test: depth left and not expired (a board is still
// live at block.timestamp == expiresAt).
func boardLive(b *Board, now uint64) bool {
	if b.Remaining == nil || b.Remaining.IsZero() || b.Filled == nil {
		return false
	}
	n := len(b.Sizes)
	if n == 0 || len(b.Prices) != n {
		return false
	}
	return now <= b.ExpiresAt
}

// fillLimit sets limit to the cumulative size a fill may reach on this board: Filled + Remaining,
// which is below the top level when the board's per-block cap binds. Remaining never exceeds the
// top size, so the sum cannot overflow.
func fillLimit(limit *uint256.Int, b *Board) {
	limit.Add(b.Filled, b.Remaining)
	if top := b.Sizes[len(b.Sizes)-1]; limit.Gt(top) {
		limit.Set(top)
	}
}

// afterProtocolFee sets fee = out * feeBps / 10000 and net = out - fee, as the venue's checked
// arithmetic does (it reverts on overflow).
func afterProtocolFee(net, fee, out *uint256.Int, feeBps uint64) error {
	if feeBps == 0 {
		fee.Clear()
	} else {
		var bps uint256.Int
		if _, overflow := fee.MulOverflow(out, bps.SetUint64(feeBps)); overflow {
			return ErrOverflow
		}
		fee.Div(fee, bpsDen)
	}
	net.Sub(out, fee)
	return nil
}
