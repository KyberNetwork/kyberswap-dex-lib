package biconomyprop

import (
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

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

// fillLimit is the cumulative size a fill may reach on this board: Filled + Remaining, which is
// below the top level when the board's per-block cap binds.
func fillLimit(b *Board) *uint256.Int {
	limit := new(uint256.Int).Add(b.Filled, b.Remaining)
	if top := b.Sizes[len(b.Sizes)-1]; limit.Gt(top) {
		limit.Set(top)
	}
	return limit
}

// afterProtocolFee applies the venue's protocol fee: out - out * feeBps / 10000.
func afterProtocolFee(out *uint256.Int, feeBps uint64) (net, fee *uint256.Int) {
	fee = new(uint256.Int)
	if feeBps != 0 {
		big256.MulDivDown(fee, out, uint256.NewInt(feeBps), uint256.NewInt(bpsDenominator))
	}
	return new(uint256.Int).Sub(out, fee), fee
}
