package swaplimit

import (
	"math/big"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

// TestInventoryUMatchesBig checks that the uint256 methods give the same balances and errors as the
// big.Int methods, for the sufficient, insufficient and absent-token cases. The caller reads the U values,
// so a mismatch would change how much a pool can swap.
func TestInventoryUMatchesBig(t *testing.T) {
	t.Parallel()
	mk := func() *Inventory {
		return NewInventory("test", map[string]*big.Int{
			"a": big.NewInt(1000),
			"b": big.NewInt(50),
		})
	}
	cases := []struct {
		name      string
		dec, inc  string
		decDelta  int64
		incDelta  int64
		wantError bool
	}{
		{name: "ok", dec: "a", inc: "b", decDelta: 100, incDelta: 7},
		{name: "new token", dec: "a", inc: "c", decDelta: 1, incDelta: 9},
		{name: "not enough", dec: "b", inc: "a", decDelta: 500, incDelta: 1, wantError: true},
		{name: "absent source", dec: "zz", inc: "a", decDelta: 1, incDelta: 1, wantError: true},
	}
	for _, c := range cases {
		big1 := mk()
		uInv := mk()
		wantDec, wantInc, wantErr := big1.UpdateLimit(c.dec, c.inc, big.NewInt(c.decDelta), big.NewInt(c.incDelta))
		gotDec, gotInc, gotErr := uInv.UpdateLimitU(c.dec, c.inc, uint256.NewInt(uint64(c.decDelta)), uint256.NewInt(uint64(c.incDelta)))
		if c.wantError {
			require.Error(t, gotErr, c.name)
			require.ErrorIs(t, gotErr, wantErr, c.name)
			continue
		}
		require.NoError(t, gotErr, c.name)
		require.NoError(t, wantErr, c.name)
		require.Equal(t, wantDec.String(), gotDec.Dec(), c.name+" decreased")
		require.Equal(t, wantInc.String(), gotInc.Dec(), c.name+" increased")
		require.Equal(t, big1.GetLimit(c.dec).String(), uInv.GetLimit(c.dec).String(), c.name+" stored source")
	}
}

// TestInventoryGetLimitURejectsNegative checks that a negative balance is never returned as a wrapped
// uint256 value. It returns nil, so the caller cannot read it as a huge positive amount.
func TestInventoryGetLimitURejectsNegative(t *testing.T) {
	t.Parallel()
	inv := NewInventory("test", map[string]*big.Int{"neg": big.NewInt(-5), "ok": big.NewInt(5)})
	require.Nil(t, inv.GetLimitU("neg"))
	require.Equal(t, uint64(5), inv.GetLimitU("ok").Uint64())
	require.Nil(t, inv.GetLimitU("absent"))
}
