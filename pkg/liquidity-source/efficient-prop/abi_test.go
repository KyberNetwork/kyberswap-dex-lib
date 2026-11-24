package efficientprop

import (
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func selector(sig string) string {
	return hex.EncodeToString(crypto.Keccak256([]byte(sig))[:4])
}

// The embedded ABIs must stay the contracts' own: a wrong type here makes every probe decode garbage, which looks
// exactly like a listing that never fills.
func TestABISelectors(t *testing.T) {
	for _, tc := range []struct {
		abi    abi.ABI
		method string
		sig    string
	}{
		{doorABI, "quote", "quote(address,address,uint256)"},
		{doorABI, "FRONT", "FRONT()"},
		{doorABI, "LISTING", "LISTING()"},
		{doorABI, "TOKEN0", "TOKEN0()"},
		{doorABI, "TOKEN1", "TOKEN1()"},
		{doorABI, "quoteFillable", "quoteFillable(address,address,uint256)"},
		{frontABI, "doorView", "doorView(uint24)"},
		{frontABI, "clock", "clock()"},
		{frontABI, "quoteFrom", "quoteFrom(address,address,uint256,uint24,uint256,address)"},
		{factoryABI, "FRONT", "FRONT()"},
		{factoryABI, "doorOf", "doorOf(uint24)"},
		{factoryABI, "DOOR_CODEHASH", "DOOR_CODEHASH()"},
	} {
		m, ok := tc.abi.Methods[tc.method]
		require.True(t, ok, tc.sig)
		assert.Equal(t, selector(tc.sig), hex.EncodeToString(m.ID), tc.sig)
	}
	assert.Equal(t, "b6466384", selector("quote(address,address,uint256)"))
}

// doorView's answer is one tuple; ethrpc copies it into the first field of the output struct, field by name.
func TestDoorViewDecodes(t *testing.T) {
	m := frontABI.Methods["doorView"]
	want := DoorView{
		PairWord: big.NewInt(0x100), Token0: common.HexToAddress("0x01"), Token1: common.HexToAddress("0x02"),
		Pool: common.HexToAddress("0x03"), BaseIndex: 0, QuoteIndex: 1, Flags: 1, KeyEpoch: 7, Desk: 3,
		WordIndex: 2, Scopes: 2, FenceBidPpm: 500, FenceAskPpm: 600, TipCap: 0x8001, ConfMul: 10, CreditMode: 0,
		BidFillMs: big.NewInt(11), AskFillMs: big.NewInt(12), Ticket: big.NewInt(13), Word: big.NewInt(14),
		Depth: [4]*big.Int{big.NewInt(1), big.NewInt(3), big.NewInt(2), big.NewInt(4)},
	}
	data, err := m.Outputs.Pack(want)
	require.NoError(t, err)

	var out doorViewOut
	require.NoError(t, frontABI.UnpackIntoInterface(&out, "doorView", data))
	assert.Equal(t, want, out.View)
}

func TestTipCapWei(t *testing.T) {
	assert.Equal(t, uint64(0), tipCapWei(0))
	assert.Equal(t, uint64(1), tipCapWei(0x0001))
	assert.Equal(t, uint64(100_000_000), tipCapWei(0x8001), "0.1 gwei: 1 x 10^8")
	assert.Equal(t, uint64(4095)*1e15, tipCapWei(0xffff))
}

// clock() answers seven words; ethrpc copies them into frontClock field by name.
func TestFrontClockDecodes(t *testing.T) {
	m := frontABI.Methods["clock"]
	data, err := m.Outputs.Pack(big.NewInt(1000), big.NewInt(30_000), big.NewInt(0), big.NewInt(5), big.NewInt(2),
		big.NewInt(53_556_127), big.NewInt(1_793_901_601_000))
	require.NoError(t, err)
	var c frontClock
	require.NoError(t, frontABI.UnpackIntoInterface(&c, "clock", data))
	assert.Equal(t, int64(30_000), c.MaxTtlMs.Int64())
	assert.Equal(t, int64(5), c.AgeSlackMs.Int64())
	assert.Equal(t, int64(2), c.ClockMode.Int64())
}

func TestValidUntil(t *testing.T) {
	// a word stored on a Base fork: maker time 1,795,104,596,900 ms, life 30 s; both sides' depth to 1,797,696,582 s
	word, ok := new(big.Int).SetString("412047767697072188490019452805923422519062484853908627031687169", 10)
	require.True(t, ok)
	depth := new(big.Int).Lsh(big.NewInt(1_797_696_582), 60)
	v := DoorView{PairWord: big.NewInt(1 << 8), Word: word, Depth: [4]*big.Int{depth, new(big.Int), depth, new(big.Int)}}
	both := [2]bool{true, true}

	// the word ends at 1,795,104,626,900 ms: a block of second 1,795,104,625 reads at most ...625,999 ms
	assert.Equal(t, uint64(1_795_104_625), validUntil(v, 30_000, 0, both))
	// the front's cap on a word's life binds below the word's own
	assert.Equal(t, uint64(1_795_104_605), validUntil(v, 10_000, 0, both))
	// an age slack past a second reads a seal-time block later still
	assert.Equal(t, uint64(1_795_104_624), validUntil(v, 30_000, 1_901, both))

	// a side's depth expiring first binds, and only where its direction is quoted
	bid := new(big.Int).Lsh(big.NewInt(1_795_104_610), 60)
	v.Depth[0] = bid
	assert.Equal(t, uint64(1_795_104_609), validUntil(v, 30_000, 0, both))
	assert.Equal(t, uint64(1_795_104_609), validUntil(v, 30_000, 0, [2]bool{true, false}), "WETH in is the bid")
	assert.Equal(t, uint64(1_795_104_625), validUntil(v, 30_000, 0, [2]bool{false, true}))
	// a pair whose base is token 1 sells token 1 into the bid
	v.PairWord = big.NewInt(0)
	assert.Equal(t, uint64(1_795_104_609), validUntil(v, 30_000, 0, [2]bool{false, true}))

	// no word, or a quoted side with no depth expiry: no second is certain
	assert.Equal(t, uint64(0), validUntil(DoorView{Word: new(big.Int)}, 30_000, 0, both))
	v.Depth[2] = new(big.Int)
	assert.Equal(t, uint64(0), validUntil(v, 30_000, 0, both))
}

// The controls word of listing 1 on the Base front, at the slot read on a fork of it.
func TestControlsSlot(t *testing.T) {
	assert.Equal(t, common.HexToHash("0xc8e1c483f59f8925a663796ac1a1d7a52011d2a3656a2af9a7b6948cf2d8cd62"),
		controlsSlot(1))
}
