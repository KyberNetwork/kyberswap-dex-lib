package thogprop

import (
	"math/rand/v2"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

// referenceSqrtDiscriminantCeil is the original solver: Newton from b + ceil(sqrt(kB))·sqrt(2e54),
// each step dividing the 512-bit discriminant with two MulDivs and two MulMods.
func referenceSqrtDiscriminantCeil(m *arithmetic, b, kBudget uint256.Int) uint256.Int {
	div := func(x uint256.Int) (q, r uint256.Int) {
		denominator := m.mul(u(2), powers[54])
		q = m.add(m.md(b, b, x, false), m.md(denominator, kBudget, x, false))
		var r1, r2 uint256.Int
		r1.MulMod(&b, &b, &x)
		r2.MulMod(&denominator, &kBudget, &x)
		r = m.add(r1, r2)
		if r.Cmp(&x) >= 0 {
			q = m.add(q, u(1))
			r = m.sub(r, x)
		}
		return
	}
	if kBudget.IsZero() {
		return b
	}
	var root uint256.Int
	root.Sqrt(&kBudget)
	if square := m.mul(root, root); square != kBudget {
		root = m.add(root, u(1))
	}
	x := m.add(b, m.mul(root, sqrt2e54))
	for m.err == nil {
		q, _ := div(x)
		next := m.div(m.add(x, q), u(2))
		if next.Cmp(&x) >= 0 {
			break
		}
		x = next
	}
	if q, r := div(x); q != x || !r.IsZero() {
		x = m.add(x, u(1))
	}
	return x
}

func randBits(r *rand.Rand, bits int) uint256.Int {
	var z uint256.Int
	for i := range 4 {
		z[i] = r.Uint64()
	}
	if bits < 256 {
		var mask uint256.Int
		mask.Lsh(uint256.NewInt(1), uint(bits))
		mask.SubUint64(&mask, 1)
		z.And(&z, &mask)
	}
	return z
}

// The faster solver must return exactly the reference root and error on every input
// shape the curve can produce (b ≤ 1e54, kBudget = k·budget with k·inventory < 1e54).
func TestSqrtDiscriminantCeilMatchesReference(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	check := func(b, kBudget uint256.Int) {
		var got, want arithmetic
		g, w := sqrtDiscriminantCeil(&got, b, kBudget), referenceSqrtDiscriminantCeil(&want, b, kBudget)
		require.Equal(t, want.err, got.err, "b=%s kBudget=%s", b.Dec(), kBudget.Dec())
		require.Equal(t, w, g, "b=%s kBudget=%s", b.Dec(), kBudget.Dec())
	}
	edges := []uint256.Int{u(0), u(1), u(2), u(3), powers[18], powers[27], powers[54], *new(uint256.Int).Sub(&powers[54], uint256.NewInt(1))}
	for _, b := range edges {
		for _, k := range append(edges, maxInventory) {
			check(b, k)
		}
	}
	for range 50_000 {
		b := randBits(r, 1+r.IntN(180))
		if b.Gt(&powers[54]) {
			b.Mod(&b, &powers[54])
		}
		check(b, randBits(r, 1+r.IntN(250)))
	}
	// Exact roots: with b = 0 and kBudget = 2t², D = 2e54·2t² = (2e27·t)².
	for range 5_000 {
		t2 := randBits(r, 1+r.IntN(110))
		var k uint256.Int
		k.Mul(&t2, &t2)
		k.Lsh(&k, 1)
		check(u(0), k)
		check(u(1), k)
	}
}

func referenceBits(word *uint256.Int, shift, width uint) (z uint256.Int) {
	var mask uint256.Int
	mask.SetAllOne()
	mask.Rsh(&mask, 256-width)
	z.Rsh(word, shift)
	z.And(&z, &mask)
	return
}

func TestBitsMatchesReference(t *testing.T) {
	r := rand.New(rand.NewPCG(3, 4))
	for range 300 {
		word := randBits(r, 256)
		for width := uint(1); width <= 256; width++ {
			for _, shift := range []uint{0, 1, 63, 64, 100, 127, 128, 192, 255, uint(r.IntN(256))} {
				if shift+width > 256 {
					continue
				}
				require.Equal(t, referenceBits(&word, shift, width), bits(&word, shift, width), "shift=%d width=%d", shift, width)
			}
		}
	}
}
