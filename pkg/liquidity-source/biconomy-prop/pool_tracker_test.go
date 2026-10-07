package biconomyprop

import (
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func bi(dec string) *big.Int {
	v, ok := new(big.Int).SetString(dec, 10)
	if !ok {
		panic("bad decimal " + dec)
	}
	return v
}

// The embedded ABI decodes board()'s outputs into rawBoard by name.
func TestVenueABI_BoardDecodesIntoRawBoard(t *testing.T) {
	method, ok := venueABI.Methods[methodBoard]
	require.True(t, ok)
	require.Len(t, method.Outputs, 5)
	assert.Equal(t, []string{"sizes", "prices", "filled", "remaining", "expiresAt"},
		[]string{method.Outputs[0].Name, method.Outputs[1].Name, method.Outputs[2].Name, method.Outputs[3].Name, method.Outputs[4].Name})

	encoded, err := method.Outputs.Pack(
		[]*big.Int{bi("1000000000000000000"), bi("3000000000000000000")},
		[]*big.Int{bi("2000000000000000000000"), bi("1999000000000000000000")},
		bi("1000000000000000000"),
		bi("2000000000000000000"),
		bi("1700000000"),
	)
	require.NoError(t, err)

	var rb rawBoard
	require.NoError(t, venueABI.UnpackIntoInterface(&rb, methodBoard, encoded))
	require.Len(t, rb.Sizes, 2)
	assert.Equal(t, "3000000000000000000", rb.Sizes[1].String())
	assert.Equal(t, "1999000000000000000000", rb.Prices[1].String())
	assert.Equal(t, "1000000000000000000", rb.Filled.String())
	assert.Equal(t, "2000000000000000000", rb.Remaining.String())
	assert.Equal(t, "1700000000", rb.ExpiresAt.String())

	for _, name := range []string{"levels", "quote", "swap", "swapWithFee", "makers", "getPairs", "isActive", "feeBps"} {
		_, ok := venueABI.Methods[name]
		assert.True(t, ok, "missing method %s", name)
	}
}

func TestToBoard(t *testing.T) {
	exp := farExpiry()
	live := toBoard(rawBoard{
		Sizes:     []*big.Int{bi("10"), bi("30")},
		Prices:    []*big.Int{bi("2000000000000000000"), bi("1000000000000000000")},
		Filled:    bi("10"),
		Remaining: bi("20"),
		ExpiresAt: new(big.Int).SetUint64(exp),
	})
	require.Len(t, live.Sizes, 2)
	require.Len(t, live.Prices, 2)
	assert.Equal(t, "10", live.Filled.Dec())
	assert.Equal(t, "20", live.Remaining.Dec())
	assert.Equal(t, exp, live.ExpiresAt)
	assert.True(t, boardLive(&live, uint64(time.Now().Unix())))

	// A dark board comes back with empty arrays, zero meter, zero remaining
	// and zero expiry.
	dark := toBoard(rawBoard{
		Sizes:     []*big.Int{},
		Prices:    []*big.Int{},
		Filled:    bi("0"),
		Remaining: bi("0"),
		ExpiresAt: bi("0"),
	})
	assert.Empty(t, dark.Sizes)
	assert.True(t, dark.Remaining.IsZero())
	assert.Equal(t, uint64(0), dark.ExpiresAt)
	assert.False(t, boardLive(&dark, uint64(time.Now().Unix())))
}

func TestBoardLive(t *testing.T) {
	now := uint64(time.Now().Unix())
	sizes := []string{"10", "30"}
	prices := []string{"2000000000000000000", "1000000000000000000"}

	live := liveBoard(sizes, prices, "0", now+60)
	assert.True(t, boardLive(&live, now))

	atExpiry := liveBoard(sizes, prices, "0", now)
	assert.True(t, boardLive(&atExpiry, now), "block.timestamp == expiresAt is still live onchain")

	expired := liveBoard(sizes, prices, "0", now-1)
	assert.False(t, boardLive(&expired, now))

	exhausted := liveBoard(sizes, prices, "30", now+60)
	assert.True(t, exhausted.Remaining.IsZero())
	assert.False(t, boardLive(&exhausted, now))

	// remaining == 0 wins regardless of levels or expiry: the venue reports
	// it for dark, expired and exhausted boards alike.
	zeroed := liveBoard(sizes, prices, "0", now+60)
	zeroed.Remaining.Clear()
	assert.False(t, boardLive(&zeroed, now))

	dark := darkBoard()
	assert.False(t, boardLive(&dark, now))

	var zero Board
	assert.False(t, boardLive(&zero, now), "nil Filled/Remaining must not panic")

	malformed := liveBoard(sizes, prices, "0", now+60)
	malformed.Prices = malformed.Prices[:1]
	assert.False(t, boardLive(&malformed, now))
}

// Reserves sum every live board's output up to its limit.
func TestDeliverable(t *testing.T) {
	now := uint64(time.Now().Unix())
	members := []MemberExtra{
		{ // 10 @ 2.0 + 20 @ 1.0 = 40, minus 10 already filled at 2.0 -> 20
			Dir0: liveBoard([]string{"10", "30"}, []string{"2000000000000000000", "1000000000000000000"}, "10", now+60),
			Dir1: liveBoard([]string{"7"}, []string{"1500000000000000000"}, "0", now+60), // floor(7*1.5) = 10
		},
		{ // expired: nothing
			Dir0: liveBoard([]string{"100"}, []string{"1000000000000000000"}, "0", now-1),
			Dir1: darkBoard(),
		},
		{ // exhausted: nothing
			Dir0: liveBoard([]string{"100"}, []string{"1000000000000000000"}, "100", now+60),
			Dir1: darkBoard(),
		},
	}
	assert.Equal(t, "20", deliverable(members, 0, now).Dec())
	assert.Equal(t, "10", deliverable(members, 1, now).Dec())
	assert.Equal(t, "0", deliverable(nil, 0, now).Dec())
}
