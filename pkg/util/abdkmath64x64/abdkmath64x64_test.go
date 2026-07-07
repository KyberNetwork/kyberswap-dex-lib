package abdkmath64x64

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/KyberNetwork/int256"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

// loadVectors reads the golden vectors emitted by ../lmsr-amm/script/GenAbdkVectors.s.sol.
// Every value is a decimal STRING: int256/uint256 exceed the JSON native-number safe range,
// so string form is the only representation that survives a round-trip without float
// precision loss. Decoding straight into []string also fails loudly if the generator ever
// regresses to emitting bare JSON numbers.
func loadVectors(t *testing.T) map[string][]string {
	t.Helper()
	b, err := os.ReadFile("testdata/abdk_vectors.json")
	require.NoError(t, err)

	var out map[string][]string
	require.NoError(t, json.Unmarshal(b, &out))
	return out
}

func iVec(t *testing.T, ss []string) []int256.Int {
	t.Helper()
	out := make([]int256.Int, len(ss))
	for i, s := range ss {
		out[i] = *int256.MustFromDec(s)
	}
	return out
}

func uVec(t *testing.T, ss []string) []uint256.Int {
	t.Helper()
	out := make([]uint256.Int, len(ss))
	for i, s := range ss {
		v, err := uint256.FromDecimal(s)
		require.NoErrorf(t, err, "bad uint decimal %q", s)
		out[i] = *v
	}
	return out
}

// TestGoldenVectors verifies every ported op reproduces the Solidity ABDKMath64x64 result
// to the wei across the generated value matrix.
func TestGoldenVectors(t *testing.T) {
	v := loadVectors(t)

	type binOp struct {
		name    string
		xk, yk  string
		outk    string
		compute func(x, y *int256.Int) (int256.Int, error)
	}
	binOps := []binOp{
		{"add", "add_x", "add_y", "add_out", Add},
		{"sub", "sub_x", "sub_y", "sub_out", Sub},
		{"mul", "mul_x", "mul_y", "mul_out", Mul},
		{"div", "div_x", "div_y", "div_out", Div},
	}
	for _, op := range binOps {
		t.Run(op.name, func(t *testing.T) {
			xs, ys, outs := iVec(t, v[op.xk]), iVec(t, v[op.yk]), iVec(t, v[op.outk])
			require.Equal(t, len(xs), len(outs))
			for i := range xs {
				got, err := op.compute(&xs[i], &ys[i])
				require.NoErrorf(t, err, "%s[%d] x=%s y=%s", op.name, i, xs[i].Dec(), ys[i].Dec())
				require.Truef(t, got.Eq(&outs[i]),
					"%s[%d] x=%s y=%s: got %s want %s",
					op.name, i, xs[i].Dec(), ys[i].Dec(), got.Dec(), outs[i].Dec())
			}
		})
	}

	type unOp struct {
		name     string
		xk, outk string
		compute  func(x *int256.Int) (int256.Int, error)
	}
	unOps := []unOp{
		{"neg", "neg_x", "neg_out", Neg},
		{"exp", "exp_x", "exp_out", Exp},
		{"exp2", "exp2_x", "exp2_out", Exp2},
		{"ln", "ln_x", "ln_out", Ln},
		{"log2", "log2_x", "log2_out", Log2},
	}
	for _, op := range unOps {
		t.Run(op.name, func(t *testing.T) {
			xs, outs := iVec(t, v[op.xk]), iVec(t, v[op.outk])
			require.Equal(t, len(xs), len(outs))
			for i := range xs {
				got, err := op.compute(&xs[i])
				require.NoErrorf(t, err, "%s[%d] x=%s", op.name, i, xs[i].Dec())
				require.Truef(t, got.Eq(&outs[i]),
					"%s[%d] x=%s: got %s want %s",
					op.name, i, xs[i].Dec(), got.Dec(), outs[i].Dec())
			}
		})
	}

	t.Run("divu", func(t *testing.T) {
		xs, ys, outs := uVec(t, v["divu_x"]), uVec(t, v["divu_y"]), iVec(t, v["divu_out"])
		for i := range xs {
			got, err := DivU(&xs[i], &ys[i])
			require.NoErrorf(t, err, "divu[%d] x=%s y=%s", i, xs[i].Dec(), ys[i].Dec())
			require.Truef(t, got.Eq(&outs[i]),
				"divu[%d] x=%s y=%s: got %s want %s",
				i, xs[i].Dec(), ys[i].Dec(), got.Dec(), outs[i].Dec())
		}
	})

	t.Run("mulu", func(t *testing.T) {
		xs, ys, outs := iVec(t, v["mulu_x"]), uVec(t, v["mulu_y"]), uVec(t, v["mulu_out"])
		for i := range xs {
			got, err := MulU(&xs[i], &ys[i])
			require.NoErrorf(t, err, "mulu[%d] x=%s y=%s", i, xs[i].Dec(), ys[i].Dec())
			require.Truef(t, got.Eq(&outs[i]),
				"mulu[%d] x=%s y=%s: got %s want %s",
				i, xs[i].Dec(), ys[i].Dec(), got.Dec(), outs[i].Dec())
		}
	})
}

// TestBounds exercises the require/revert boundaries that the golden vectors deliberately
// avoid, so the simulator can rely on these ops returning errors instead of panicking.
func TestBounds(t *testing.T) {
	zero := int256.NewInt(0)
	one := int256.NewInt(1)

	t.Run("neg_min", func(t *testing.T) {
		_, err := Neg(min64x64)
		require.ErrorIs(t, err, ErrOverflow)
	})

	t.Run("div_by_zero", func(t *testing.T) {
		_, err := Div(ONE, zero)
		require.ErrorIs(t, err, ErrDivByZero)
	})

	t.Run("add_overflow", func(t *testing.T) {
		_, err := Add(max64x64, ONE)
		require.ErrorIs(t, err, ErrOverflow)
	})

	t.Run("sub_overflow", func(t *testing.T) {
		_, err := Sub(min64x64, ONE)
		require.ErrorIs(t, err, ErrOverflow)
	})

	t.Run("divu_by_zero", func(t *testing.T) {
		_, err := DivU(uint256.NewInt(1), uint256.NewInt(0))
		require.ErrorIs(t, err, ErrDivByZero)
	})

	t.Run("mulu_negative", func(t *testing.T) {
		neg := int256.NewInt(-1)
		_, err := MulU(neg, uint256.NewInt(5))
		require.ErrorIs(t, err, ErrNegative)
	})

	t.Run("mulu_zero_y", func(t *testing.T) {
		got, err := MulU(ONE, uint256.NewInt(0))
		require.NoError(t, err)
		require.True(t, got.IsZero())
	})

	t.Run("ln_nonpositive", func(t *testing.T) {
		_, err := Ln(zero)
		require.ErrorIs(t, err, ErrNonPositive)
		_, err = Ln(int256.NewInt(-5))
		require.ErrorIs(t, err, ErrNonPositive)
	})

	t.Run("log2_nonpositive", func(t *testing.T) {
		_, err := Log2(zero)
		require.ErrorIs(t, err, ErrNonPositive)
	})

	t.Run("exp_overflow", func(t *testing.T) {
		// x == 2^70 (== exp2Limit) reverts.
		_, err := Exp(exp2Limit)
		require.ErrorIs(t, err, ErrOverflow)
	})

	t.Run("exp_underflow_zero", func(t *testing.T) {
		// x < -2^70 returns 0.
		var below int256.Int
		below.Sub(negExp2Limit, one)
		got, err := Exp(&below)
		require.NoError(t, err)
		require.True(t, got.IsZero())
	})

	t.Run("exp2_underflow_zero", func(t *testing.T) {
		var below int256.Int
		below.Sub(negExp2Limit, one)
		got, err := Exp2(&below)
		require.NoError(t, err)
		require.True(t, got.IsZero())
	})
}
