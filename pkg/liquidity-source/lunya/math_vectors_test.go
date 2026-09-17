package lunya

import (
	"bufio"
	"math/big"
	"os"
	"strings"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

// testdata/stableswap_vectors.csv holds inputs and results of StableSwapMath.sol itself - getD,
// getOtherReserve, sqrtPriceFromReserves and movePriceTowardsTarget, reverts included - run through a
// Foundry harness over random pools, amplifications, fees, amounts and price limits.
func TestMathMatchesStableSwapMathVectors(t *testing.T) {
	t.Parallel()

	file, err := os.Open("testdata/stableswap_vectors.csv")
	require.NoError(t, err)
	defer func() { _ = file.Close() }()

	u := func(s string) *uint256.Int { return uint256.MustFromDecimal(s) }
	signed := func(s string) *uint256.Int {
		b, ok := new(big.Int).SetString(s, 10)
		require.True(t, ok, s)
		v := uint256.MustFromBig(new(big.Int).Abs(b))
		if b.Sign() < 0 {
			v.Neg(v)
		}
		return v
	}

	counts := map[string]int{}
	failures := 0
	fail := func(line int, format string, args ...any) {
		failures++
		if failures <= 20 {
			t.Errorf("line %d: "+format, append([]any{line}, args...)...)
		}
	}
	check := func(line int, ok string, err error, got []uint256.Int, want []string) {
		if ok == "0" {
			if err == nil {
				fail(line, "Solidity reverted, Go returned %v", got)
			}
			return
		}
		if err != nil {
			fail(line, "Solidity returned %v, Go failed: %v", want, err)
			return
		}
		for i := range want {
			if got[i].Dec() != want[i] {
				fail(line, "output %d: Solidity %s, Go %s", i, want[i], got[i].Dec())
			}
		}
	}

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1<<20), 1<<20)
	for line := 1; scanner.Scan(); line++ {
		text := scanner.Text()
		if text == "" || strings.HasPrefix(text, "#") {
			continue
		}
		f := strings.Split(text, ",")
		counts[f[0]]++

		switch f[0] {
		case "D":
			d, err := getD(u(f[1]), u(f[2]), u(f[3]))
			check(line, f[4], err, []uint256.Int{d}, f[5:])
		case "Y":
			other, err := getOtherReserve(u(f[1]), u(f[2]), u(f[3]), u(f[2]))
			check(line, f[4], err, []uint256.Int{other}, f[5:])
		case "P":
			price, err := sqrtPriceFromReserves(u(f[1]), u(f[2]))
			check(line, f[3], err, []uint256.Int{price}, f[4:])
		case "M":
			fee := u(f[6]).Uint64()
			price, input, output, feeAmount, err := movePriceTowardsTarget(f[1] == "1", u(f[2]), u(f[3]), u(f[4]),
				signed(f[5]), fee, u(f[7]))
			check(line, f[8], err, []uint256.Int{price, input, output, feeAmount}, f[9:])
		default:
			t.Fatalf("line %d: unknown kind %q", line, f[0])
		}
	}
	require.NoError(t, scanner.Err())
	require.NotZero(t, counts["M"])

	t.Logf("vectors: %v, failures: %d", counts, failures)
}
