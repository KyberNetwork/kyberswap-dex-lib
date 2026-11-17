package swaplimit

import (
	"math/big"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

// A negative or nil legacy balance reads as zero inventory. An oversized one is not a cap, so it saturates to
// MaxUint256 and never wraps into a small value.
func TestToU256BalancesNegativeZeroOversizedUnbounded(t *testing.T) {
	tooBig := new(big.Int).Lsh(big.NewInt(1), 300)
	got := ToU256Balances(map[string]*big.Int{
		"ok":   big.NewInt(5),
		"neg":  big.NewInt(-1),
		"huge": tooBig,
		"nil":  nil,
	})
	require.Equal(t, uint64(5), got["ok"].Uint64())
	require.True(t, got["neg"].IsZero(), "negative must become zero")
	require.True(t, got["huge"].Eq(new(uint256.Int).SetAllOne()), "oversized must saturate, not wrap")
	require.True(t, got["nil"].IsZero(), "nil must become zero")
}
