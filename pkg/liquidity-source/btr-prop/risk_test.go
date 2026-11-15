package btrprop

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

// word packs a MarkWordLib leg word. mark is in 1e18 WAD; its lane float keeps 25 bits.
func word(obs, sigma, ttl uint64, mark uint64, halted, internal bool, depegCode uint64) *big.Int {
	m := new(big.Int).Lsh(new(big.Int).SetUint64(mark), 16)
	e := uint(0)
	if bl := m.BitLen(); bl > 25 {
		e = uint(bl - 25)
	}
	f := new(big.Int).Rsh(m, e)
	f.Or(f, new(big.Int).Lsh(big.NewInt(int64(e)), 25))
	w := new(big.Int).Set(f)
	put := func(v uint64, shift uint) { w.Or(w, new(big.Int).Lsh(new(big.Int).SetUint64(v), shift)) }
	put(obs, 32)
	put(sigma, 64)
	put(ttl, 107)
	if halted {
		put(1, 134)
	}
	if internal {
		put(1, 232)
	}
	put(depegCode, 234)
	return w
}

const wad = 1_000_000_000_000_000_000

// The live WMON leg word of 2026-11-15 (getLegMark on Monad) decodes to its feed fields.
func TestDecodeMarkLiveWord(t *testing.T) {
	w, _ := new(big.Int).SetString("627424820714972079307831745652209381336", 10)
	l, ok := decodeMark(w)
	require.True(t, ok)
	require.Equal(t, uint64(1794759928), l.obs)
	require.Equal(t, uint64(11264), l.sigma)
	require.Equal(t, uint64(180), l.ttl)
	require.False(t, l.halted || l.internal)
	require.Equal(t, "15426126662860800", l.mark.Dec())
	_, ok = decodeMark(new(big.Int))
	require.False(t, ok, "an empty word is an unreadable leg")
}

func TestTradableAt(t *testing.T) {
	const obs, ttl = 1000, 60
	for _, c := range []struct {
		name     string
		word     *big.Int
		base     bool
		at       uint64
		tradable bool
	}{
		{"fresh", word(obs, 100, ttl, wad, false, false, 0), false, obs + 5, true},
		{"ttl edge", word(obs, 100, ttl, wad, false, false, 0), false, obs + ttl, true},
		{"expired inside the window", word(obs, 100, ttl, wad, false, false, 0), false, obs + ttl + 1, false},
		{"halted feed", word(obs, 100, ttl, wad, true, false, 0), false, obs, false},
		{"dead mark", word(obs, 100, ttl, 0, false, false, 0), false, obs, false},
		{"no depeg code, any mark", word(obs, 100, ttl, wad/2, false, false, 0), false, obs, true},
		{"inside band (code 1 = 2.5%)", word(obs, 100, ttl, wad*98/100, false, false, 1), false, obs, true},
		{"above band", word(obs, 100, ttl, wad*103/100, false, false, 1), false, obs, false},
		{"below band", word(obs, 100, ttl, wad*97/100, false, false, 1), false, obs, false},
		{"wider band holds it", word(obs, 100, ttl, wad*103/100, false, false, 2), false, obs, true},
		{"INTERNAL without code reads code 1", word(obs, 100, ttl, wad*97/100, false, true, 0), false, obs, false},
		{"base is exempt from the breaker", word(obs, 100, ttl, wad*90/100, false, false, 1), true, obs, true},
		{"base still gated on TTL", word(obs, 100, ttl, wad, false, false, 0), true, obs + ttl + 1, false},
	} {
		l, ok := decodeMark(c.word)
		require.True(t, ok, c.name)
		l.base = c.base
		require.Equal(t, c.tradable, l.tradableAt(c.at), c.name)
	}
}

func TestLiquidRoom(t *testing.T) {
	a := func(minLiq int64) *rpcAsset { return &rpcAsset{MinLiquidity: big.NewInt(minLiq)} }
	require.Equal(t, "90", liquidRoom(big.NewInt(100), a(10)).Dec())
	require.True(t, liquidRoom(big.NewInt(10), a(10)).IsZero(), "liquid at the floor leaves no room")
	require.True(t, liquidRoom(big.NewInt(5), a(10)).IsZero())
	require.Nil(t, liquidRoom(nil, a(10)), "an unreadable liquid read is unknown, not zero")
	require.Nil(t, liquidRoom(big.NewInt(5), nil))
}

func TestCutAbove(t *testing.T) {
	p := pts(10, 9, 20, 18, 30, 27)
	require.Len(t, cutAbove(p, uint256.NewInt(18)), 2)
	require.Empty(t, cutAbove(p, new(uint256.Int)))
	require.Len(t, cutAbove(p, nil), 3)
}

// refine refuses a path through a depegged, expired or unreadable leg, including an interior
// hop, and cuts the ladder at the output leg's settle-time room.
func TestGuardRefine(t *testing.T) {
	a, b, hub := common.HexToAddress("0x01"), common.HexToAddress("0x02"), common.HexToAddress("0x03")
	leg := func(mark uint64, code uint64) *markLeg {
		l, _ := decodeMark(word(1000, 100, 60, mark, false, false, code))
		return l
	}
	ladder := pts(10, 9, 20, 18, 30, 27)
	mk := func(hubLeg *markLeg, room *uint256.Int) *guard {
		return &guard{
			tokens: []common.Address{a, b, hub},
			legs:   []*markLeg{leg(wad, 1), leg(wad, 1), hubLeg},
			rooms:  []*uint256.Int{nil, room, nil},
			now:    1000, window: 10,
		}
	}
	hops := []common.Address{a, hub, b}
	require.Len(t, mk(leg(wad, 1), nil).refine(0, 1, hops, ladder), 3)
	require.Nil(t, mk(leg(wad*90/100, 1), nil).refine(0, 1, hops, ladder), "depegged interior hop")
	require.Len(t, mk(leg(wad*90/100, 1), nil).refine(0, 1, nil, ladder), 3, "without a hop list only the endpoints are checked")
	require.Nil(t, mk(nil, nil).refine(0, 1, hops, ladder), "unreadable interior word")
	g := mk(leg(wad, 1), nil)
	g.window = 100
	require.Nil(t, g.refine(0, 1, hops, ladder), "TTL expires inside the window")
	require.Len(t, mk(leg(wad, 1), uint256.NewInt(20)).refine(0, 1, hops, ladder), 2, "liquid floor cuts the top")
}
