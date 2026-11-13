package liquiditybookv21

// The fixture is a real revert payload of LiquidityBookV21Lens on Avalanche's v2.1
// WAVAX/USDC pair (0xd446eb1660f766d533beceef890df7a69d26f7d1) at block 98195140, from bin 8375900
// under a 400k gas limit, captured with eth_call and decoded independently with `cast abi-decode`
// for the expected values below.

import (
	"bytes"
	"encoding/hex"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	tickspkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3/ticks"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/ephemeral"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

// TestLensDecodesPartialSnapshot: a walk that stopped short reports the state, the bins it read,
// and where to resume.
func TestLensDecodesPartialSnapshot(t *testing.T) {
	var r ephemeral.BinSnapshot
	require.NoError(t, lens.UnpackCall(testutil.HexFixture(t, "testdata/lens/partial.hex"), &r))

	assert.EqualValues(t, 8375914, r.ActiveId.Int64())
	assert.Equal(t, "18125915382005123767480", r.ReserveX.String())
	assert.Equal(t, "18682324250", r.ReserveY.String())
	assert.EqualValues(t, 98195140, r.BlockNumber.Int64())
	assert.EqualValues(t, 8375949, r.NextId.Int64())
	assert.False(t, r.Done)
	require.Len(t, r.Bins, 49)
	first, active, last := r.Bins[0], r.Bins[14], r.Bins[48]
	assert.EqualValues(t, 8375900, first.Id.Int64())
	assert.Zero(t, first.ReserveX.Sign())
	assert.Equal(t, "54553411", first.ReserveY.String())
	assert.EqualValues(t, 8375914, active.Id.Int64())
	assert.Equal(t, "149101905134381802", active.ReserveX.String())
	assert.Equal(t, "1573663", active.ReserveY.String())
	assert.EqualValues(t, 8375948, last.Id.Int64())
	assert.Equal(t, "195674866880250768", last.ReserveX.String())
	assert.Zero(t, last.ReserveY.Sign())
}

// TestLensRejectsWhatIsNotAResult: each payload here is the real one with one thing wrong, or a
// different kind of revert; reading any of them as a result would hand the tracker made-up bins.
func TestLensRejectsWhatIsNotAResult(t *testing.T) {
	good := testutil.HexFixture(t, "testdata/lens/partial.hex")
	with := func(word int, value []byte) []byte {
		b := bytes.Clone(good)
		copy(b[word*32:(word+1)*32], value)
		return b
	}
	word := func(hexWord string) []byte {
		b, _ := hex.DecodeString(strings.Repeat("0", 64-len(hexWord)) + hexWord)
		return b
	}
	reason, _ := hex.DecodeString("08c379a0" +
		"0000000000000000000000000000000000000000000000000000000000000020" +
		"0000000000000000000000000000000000000000000000000000000000000004" +
		"6e6f706500000000000000000000000000000000000000000000000000000000")

	for name, payload := range map[string][]byte{
		"empty":                nil,
		"head only":            good[:224],
		"trailing word":        append(bytes.Clone(good), make([]byte, 32)...),
		"truncated entry":      good[:len(good)-32],
		"Error(string)":        reason,
		"offset elsewhere":     with(6, word("100")),
		"count too large":      with(7, word("32")),
		"bool not 0 or 1":      with(5, word("2")),
		"active id past 24":    with(0, word("1000000")),
		"cursor past 24":       with(4, word("1000000")),
		"bin id past 24":       with(8, word("1000000")),
		"reserve past uint128": with(1, word("100000000000000000000000000000000")),
	} {
		var r ephemeral.BinSnapshot
		assert.Error(t, lens.UnpackCall(payload, &r), name)
	}
}

// TestLensErrors: the lens's own error decodes by name.
func TestLensErrors(t *testing.T) {
	raw, _ := hex.DecodeString("9dac528c")
	got, err := lens.UnpackRevert(raw)
	require.NoError(t, err)
	assert.Equal(t, "UnexpectedPairResponse", got.Name)
}

// TestLensPackCall: the arguments the fixture was captured with encode to the same words.
func TestLensPackCall(t *testing.T) {
	data, err := lens.PackCall(common.HexToAddress("0xd446eb1660f766d533beceef890df7a69d26f7d1"),
		big.NewInt(8375900))
	require.NoError(t, err)
	assert.Equal(t, "000000000000000000000000d446eb1660f766d533beceef890df7a69d26f7d1"+
		"00000000000000000000000000000000000000000000000000000000007fce5c", hex.EncodeToString(data[len(data)-64:]))
}

// TestHasState: the pair factory leaves the Extra empty; a pair the tracker has read keeps its bins
// key even with no liquidity, so it is not read whole again.
func TestHasState(t *testing.T) {
	for extra, want := range map[string]bool{
		``:                                     false,
		`{"lensDue":7}`:                        false,
		`{"activeBinId":8388608,"binStep":25}`: false,
		`{"activeBinId":8388608,"bins":null}`:  true,
		`{"activeBinId":8388608,"bins":[]}`:    true,
		`{"bins":[{"id":1,"reserveX":1,"reserveY":0}]}`: true,
	} {
		assert.Equal(t, want, hasState(entity.Pool{Extra: extra}), extra)
	}
}

// TestResetPoolState: with the lens a reset pair is one without bins, read whole on its next update
// at a block no lower than the one asked; without it the pair is kept, as nothing could read it
// whole.
func TestResetPoolState(t *testing.T) {
	p := entity.Pool{Address: "0xpair",
		Extra: `{"activeBinId":8388608,"binStep":25,"bins":[{"id":8388608,"reserveX":1,"reserveY":2}],"lensDue":120}`}

	lensTracker := NewPoolTracker(&Config{
		UseLens: true}, nil, nil)
	assert.Zero(t, lensTracker.NextUpdateAt(p), "a pair with bins waits for its events")
	got, ok := lensTracker.ResetPoolState(p, 100)
	assert.True(t, ok)
	assert.False(t, hasState(got))
	assert.Contains(t, got.Extra, `"lensDue":120`, "the due block never goes back")
	assert.Contains(t, got.Extra, `"activeBinId":8388608`, "the rest of the Extra is kept")
	assert.NotZero(t, lensTracker.NextUpdateAt(got), "it is due for its lens read")
	got, _ = lensTracker.ResetPoolState(p, 150)
	assert.Contains(t, got.Extra, `"lensDue":150`)

	plain := NewPoolTracker(&Config{}, nil, nil)
	got, ok = plain.ResetPoolState(p, 100)
	assert.False(t, ok)
	assert.Equal(t, p, got)
	assert.Zero(t, plain.NextUpdateAt(entity.Pool{}))
}

// TestUpdatesNeverReadBelowThePair: an update reads at its newest log's block, but never below the
// pair's own, and at the latest for a trigger without logs or a log without a block.
func TestUpdatesNeverReadBelowThePair(t *testing.T) {
	assert.EqualValues(t, 0, tickspkg.ReadBlock(nil, 200))
	assert.EqualValues(t, 0, tickspkg.ReadBlock([]ethtypes.Log{{}}, 200), "the empty trigger log")
	assert.EqualValues(t, 250, tickspkg.ReadBlock([]ethtypes.Log{{BlockNumber: 240}, {BlockNumber: 250}}, 200))
	assert.EqualValues(t, 200, tickspkg.ReadBlock([]ethtypes.Log{{BlockNumber: 150}}, 200), "a late log")
}
