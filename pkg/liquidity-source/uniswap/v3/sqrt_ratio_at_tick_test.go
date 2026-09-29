package uniswapv3

import (
	"crypto/sha256"
	"encoding/hex"
	"math/big"
	"math/rand/v2"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

// tickMathConsts are TickMath.getSqrtRatioAtTick's multipliers for bits 1..19 of |tick|.
var tickMathConsts = []string{
	"fff97272373d413259a46990580e213a", "fff2e50f5f656932ef12357cf3c7fdcc", "ffe5caca7e10e4e61c3624eaa0941cd0",
	"ffcb9843d60f6159c9db58835c926644", "ff973b41fa98c081472e6896dfb254c0", "ff2ea16466c96a3843ec78b326b52861",
	"fe5dee046a99a2a811c461f1969c3053", "fcbe86c7900a88aedcffc83b479aa3a4", "f987a7253ac413176f2b074cf7815e54",
	"f3392b0822b70005940c7a398e4b70f3", "e7159475a2c29b7443b29c7fa6e889d9", "d097f3bdfd2022b8845ad8f792aa5825",
	"a9f746462d870fdf8a65dc1f90e061e5", "70d869a156d2a1b890bb3df62baf32f7", "31be135f97d08fd981231505542fcfa6",
	"9aa508b5b7a84e1c677de54f3e99bc9", "5d6af8dedb81196699c329225ee604", "2216e584f5fa1ea926041bedfe98",
	"48a170391f7dc42444e8fa2",
}

// refSqrtRatioAtTick is TickMath.getSqrtRatioAtTick in big.Int, independent of the uint256 port.
func refSqrtRatioAtTick(tick int) *big.Int {
	abs := tick
	if abs < 0 {
		abs = -abs
	}
	ratio, _ := new(big.Int).SetString("100000000000000000000000000000000", 16)
	if abs&1 != 0 {
		ratio.SetString("fffcb933bd6fad37aa2d162d1a594001", 16)
	}
	var c big.Int
	for i, hexConst := range tickMathConsts {
		if abs&(2<<i) != 0 {
			c.SetString(hexConst, 16)
			ratio.Rsh(ratio.Mul(ratio, &c), 128)
		}
	}
	if tick > 0 {
		maxU256 := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))
		ratio.Div(maxU256, ratio)
	}
	rem := new(big.Int).And(ratio, big.NewInt(1<<32-1))
	ratio.Rsh(ratio, 32)
	if rem.Sign() != 0 {
		ratio.Add(ratio, big.NewInt(1))
	}
	return ratio
}

// TestGetSqrtRatioAtTickMatchesSolidity compares against the big.Int transcription at the range
// ends, every bit boundary of |tick| (where the multiplier set changes), and a dense band and a
// stride across the range. The exhaustive check is the pinned digest below.
func TestGetSqrtRatioAtTickMatchesSolidity(t *testing.T) {
	t.Parallel()

	require.Equal(t, "4295128739", refSqrtRatioAtTick(MinTick).String())
	require.Equal(t, "1461446703485210103287273052203988822378723970342", refSqrtRatioAtTick(MaxTick).String())
	require.Equal(t, "79228162514264337593543950336", refSqrtRatioAtTick(0).String())

	ticks := []int{MinTick, MinTick + 1, MaxTick - 1, MaxTick}
	for k := 0; k < 20; k++ {
		for _, d := range []int{-1, 0, 1} {
			if abs := 1<<k + d; abs <= MaxTick {
				ticks = append(ticks, abs, -abs)
			}
		}
	}
	for tick := -3000; tick <= 3000; tick++ {
		ticks = append(ticks, tick)
	}
	for tick := MinTick; tick <= MaxTick; tick += 97 {
		ticks = append(ticks, tick)
	}

	var got uint256.Int
	for _, tick := range ticks {
		require.NoError(t, GetSqrtRatioAtTick(tick, &got))
		require.Equal(t, refSqrtRatioAtTick(tick).String(), got.Dec(), "tick %d", tick)
	}

	require.ErrorIs(t, GetSqrtRatioAtTick(MinTick-1, &got), errInvalidTick)
	require.ErrorIs(t, GetSqrtRatioAtTick(MaxTick+1, &got), errInvalidTick)
}

// TestGetSqrtRatioAtTickAllTicks pins a SHA-256 over the 20-byte result for every tick in
// [MinTick, MaxTick], computed from the implementation after it was checked against
// refSqrtRatioAtTick on every tick. Any change to any tick's result breaks it.
func TestGetSqrtRatioAtTickAllTicks(t *testing.T) {
	t.Parallel()

	h := sha256.New()
	var got uint256.Int
	for tick := MinTick; tick <= MaxTick; tick++ {
		require.NoError(t, GetSqrtRatioAtTick(tick, &got))
		b := got.Bytes20()
		h.Write(b[:])
	}
	require.Equal(t, "f7906809285d73c1d012b8893c26ccce929372c16bd9b885109ee3f0ff1656db", hex.EncodeToString(h.Sum(nil)))
}

// TestDivMaxUint256 checks the specialized division against big.Int over any divisor in
// [2^64, 2^128), not just those valid ticks produce, including the edges where Knuth's quotient
// estimate is off by two.
func TestDivMaxUint256(t *testing.T) {
	t.Parallel()

	maxU256 := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))
	divisors := [][2]uint64{{1, 0}, {1, 1}, {1<<63 - 1, ^uint64(0)}, {1 << 63, 0}, {1 << 63, 1},
		{^uint64(0), ^uint64(0)}, {^uint64(0), 0}, {1<<63 + 1, ^uint64(0)}}
	rng := rand.New(rand.NewPCG(1, 2))
	for range 200000 {
		hi := rng.Uint64() >> rng.UintN(64)
		divisors = append(divisors, [2]uint64{max(hi, 1), rng.Uint64()})
	}
	var got uint256.Int
	for _, d := range divisors {
		got[0], got[1], got[2], got[3] = divMaxUint256(d[0], d[1])
		div := new(big.Int).Or(new(big.Int).Lsh(new(big.Int).SetUint64(d[0]), 64), new(big.Int).SetUint64(d[1]))
		require.Equal(t, new(big.Int).Div(maxU256, div).String(), got.Dec(), "d = %x:%x", d[0], d[1])
	}
}

// TestDiv3by2 checks one division step against big.Int on arbitrary numerators. Divisors with a
// small top limb and a large low limb make the quotient estimate overshoot by two, which exercises
// the double add-back.
func TestDiv3by2(t *testing.T) {
	t.Parallel()

	rng := rand.New(rand.NewPCG(3, 4))
	limbs := func(ws ...uint64) *big.Int {
		x := new(big.Int)
		for _, w := range ws {
			x.Lsh(x, 64).Or(x, new(big.Int).SetUint64(w))
		}
		return x
	}
	for range 200000 {
		d1 := 1<<63 | rng.Uint64()>>rng.UintN(64)&(1<<63-1)
		d0 := ^uint64(0) - rng.Uint64()>>rng.UintN(64)
		u2 := d1 - 1 - rng.Uint64()>>rng.UintN(64)%d1
		u1, u0 := rng.Uint64(), rng.Uint64()

		q, r1, r0 := div3by2(u2, u1, u0, d1, d0)
		wantQ, wantR := new(big.Int).QuoRem(limbs(u2, u1, u0), limbs(d1, d0), new(big.Int))
		require.Equal(t, wantQ.String(), new(big.Int).SetUint64(q).String(), "u=%x:%x:%x d=%x:%x", u2, u1, u0, d1, d0)
		require.Equal(t, wantR.String(), limbs(r1, r0).String(), "u=%x:%x:%x d=%x:%x", u2, u1, u0, d1, d0)
	}
}
