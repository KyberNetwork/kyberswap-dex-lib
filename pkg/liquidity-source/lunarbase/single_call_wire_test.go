package lunarbase_test

import (
	"testing"

	"github.com/KyberNetwork/msgpack/v5"
	"github.com/stretchr/testify/require"

	lb "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/lunarbase"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

func TestSingleCallWireUsesV3OnlyForCompleteNoHashSnapshots(t *testing.T) {
	for _, complete := range []bool{false, true} {
		extra := wireExtra()
		extra.SnapshotComplete = complete
		if complete {
			extra.BlockHash = ""
			extra.MaxPunishmentX24 = 0
		}
		original := wireSimulator(t, extra)
		var fields map[string]msgpack.RawMessage
		require.NoError(t, wireReadRaw(wireRaw(t, original), &fields))
		var version uint64
		require.NoError(t, wireReadRaw(fields["_lunarbaseWire"], &version))
		if complete {
			require.Equal(t, uint64(3), version)
			require.Contains(t, fields, "SnapshotComplete")
		} else {
			require.Equal(t, uint64(2), version)
			require.NotContains(t, fields, "SnapshotComplete")
		}
		decoded := wireMapRoundTrip(t, map[string]pool.IPoolSimulator{"pool": original})["pool"].(*lb.PoolSimulator)
		require.Equal(t, original.Extra, decoded.Extra)
		want, err := original.CalcAmountOut(wireQuoteParams())
		require.NoError(t, err)
		got, err := decoded.CalcAmountOut(wireQuoteParams())
		require.NoError(t, err)
		require.Equal(t, want.TokenAmountOut, got.TokenAmountOut)
		require.Equal(t, want.Fee, got.Fee)
	}
}

func TestSingleCallWirePreservesKnownZeroFreshness(t *testing.T) {
	extra := wireExtra()
	extra.BlockHash = ""
	extra.SnapshotComplete = true
	extra.LatestUpdateBlock = 0
	sim := wireSimulator(t, extra)
	decoded := wireMapRoundTrip(t, map[string]pool.IPoolSimulator{"pool": sim})["pool"]
	_, err := decoded.CalcAmountOut(wireQuoteParams())
	require.ErrorIs(t, err, lb.ErrStalePool)

	var fields map[string]msgpack.RawMessage
	require.NoError(t, wireReadRaw(wireRaw(t, sim), &fields))
	delete(fields, "SnapshotComplete")
	target := wireSimulator(t, wireExtra())
	before := wireRaw(t, target)
	require.Error(t, wireReadRaw(wireRaw(t, fields), target))
	require.Equal(t, before, wireRaw(t, target))
}
