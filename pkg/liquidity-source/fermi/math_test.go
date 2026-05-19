package fermi

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func bi(s string) *big.Int { v, _ := new(big.Int).SetString(s, 10); return v }

// TestEvalCubic_Identity: y = c0*1e18 at t=0; y = (c0+c1+c2+c3)*1e18 at t=1e18.
func TestEvalCubic_Identity(t *testing.T) {
	c0, c1, c2, c3 := big.NewInt(7), big.NewInt(11), big.NewInt(13), big.NewInt(17)

	at0 := evalCubic(new(big.Int), c0, c1, c2, c3)
	want0 := new(big.Int).Mul(c0, oneE18)
	if at0.Cmp(want0) != 0 {
		t.Fatalf("t=0: got %s want %s", at0, want0)
	}

	at1 := evalCubic(oneE18, c0, c1, c2, c3)
	wantSum := new(big.Int).Add(c0, c1)
	wantSum.Add(wantSum, c2)
	wantSum.Add(wantSum, c3)
	want1 := new(big.Int).Mul(wantSum, oneE18)
	if at1.Cmp(want1) != 0 {
		t.Fatalf("t=1e18: got %s want %s", at1, want1)
	}
}

// TestEvalCubic_PureLinear: c0=c2=c3=0 → y = c1*t.
func TestEvalCubic_PureLinear(t *testing.T) {
	c1 := big.NewInt(42)
	half := new(big.Int).Quo(oneE18, big.NewInt(2))
	y := evalCubic(half, new(big.Int), c1, new(big.Int), new(big.Int))
	want := new(big.Int).Mul(c1, half)
	if y.Cmp(want) != 0 {
		t.Fatalf("linear at t=0.5: got %s want %s", y, want)
	}
}

// TestEvalCubic_NegativeCoefs: signed arithmetic survives.
func TestEvalCubic_NegativeCoefs(t *testing.T) {
	c0 := big.NewInt(-3)
	y := evalCubic(new(big.Int), c0, new(big.Int), new(big.Int), new(big.Int))
	want := new(big.Int).Mul(c0, oneE18)
	if y.Cmp(want) != 0 {
		t.Fatalf("negative c0: got %s want %s", y, want)
	}
}

// TestEvalSpline_Bracketing: two-knot spline; each region returns the correct
// cubic, and an out-of-range input fails.
func TestEvalSpline_Bracketing(t *testing.T) {
	knots := []Knot{
		{XLo: "0", XHi: "1000000000000000000", C0: "0", C1: "2", C2: "0", C3: "0"},
		{XLo: "1000000000000000001", XHi: "2000000000000000000", C0: "5", C1: "0", C2: "0", C3: "0"},
	}

	// knot 0 at x=0.5e18 → t=0.5e18, y = 2*0.5e18 = 1e18
	y, err := evalSpline(knots, bi("500000000000000000"))
	require.NoError(t, err)
	assert.Equal(t, bi("1000000000000000000"), y)

	// knot 1 at x=1.5e18 → constant c0=5, y = 5e18
	y, err = evalSpline(knots, bi("1500000000000000000"))
	require.NoError(t, err)
	assert.Equal(t, bi("5000000000000000000"), y)

	// out of range above
	_, err = evalSpline(knots, bi("3000000000000000000"))
	assert.ErrorIs(t, err, ErrKnotOutOfRange)
}

func TestEvalSpline_Empty(t *testing.T) {
	_, err := evalSpline(nil, bi("1"))
	assert.ErrorIs(t, err, ErrEmptySpline)
}

func TestPriceFactor_Identity(t *testing.T) {
	price := bi("1234500000000000000000")
	got := priceFactor(price, new(big.Int))
	assert.Equal(t, 0, price.Cmp(got))
}

func TestPriceFactor_PositiveAdj_LowersOutput(t *testing.T) {
	price := bi("1000000000000000000000")
	adj := new(big.Int).Mul(oneE18, big.NewInt(100)) // 100 bps × 1e18
	got := priceFactor(price, adj)
	assert.Less(t, got.Cmp(price), 0, "positive adj should lower price")
}

func TestPriceFactor_DenomCollapse(t *testing.T) {
	negAdj := new(big.Int).Neg(bpsTimes)
	got := priceFactor(big.NewInt(1), negAdj)
	assert.Nil(t, got, "collapsed denom must return nil")
}

// --- decode helpers (pairKey, pairBaseSlot, decodeMidPrice, slotOffset) ---

func TestPairKeyForTokens_BothDirections(t *testing.T) {
	weth := common.HexToAddress("0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2")
	usdc := common.HexToAddress("0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48")

	fwd, rev := pairKeyForTokens(weth, usdc)
	assert.NotEqual(t, fwd, rev)

	fwd2, rev2 := pairKeyForTokens(usdc, weth)
	assert.Equal(t, fwd, rev2)
	assert.Equal(t, rev, fwd2)
}

func TestPairBaseSlot_DeterministicAndNonZero(t *testing.T) {
	weth := common.HexToAddress("0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2")
	usdc := common.HexToAddress("0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48")
	fwd, _ := pairKeyForTokens(weth, usdc)

	assert.Equal(t, pairBaseSlot(fwd), pairBaseSlot(fwd))
	assert.NotEqual(t, common.Hash{}, pairBaseSlot(fwd))
}

func TestDecodeMidPrice_Uint256(t *testing.T) {
	expected := big.NewInt(225_323_000_000)
	var w common.Hash
	expected.FillBytes(w[:])
	assert.Equal(t, 0, expected.Cmp(decodeMidPrice(w)))
}

func TestSlotOffset_Increments(t *testing.T) {
	base := common.HexToHash("0x0a")
	require.Equal(t, common.HexToHash("0x0d"), slotOffset(base, 3))
}
