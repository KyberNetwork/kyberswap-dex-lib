package balancerv1

import (
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

// TestCloneStateIsIndependent checks that UpdateBalance on a clone does not change the original, for the
// maps and values UpdateBalance writes.
func TestCloneStateIsIndependent(t *testing.T) {
	t.Parallel()
	const tok = "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"
	orig := &PoolSimulator{
		records:           map[string]Record{tok: {Balance: uint256.NewInt(1000)}},
		totalAmountsIn:    map[string]*uint256.Int{tok: uint256.NewInt(10)},
		maxTotalAmountsIn: map[string]*uint256.Int{tok: uint256.NewInt(5000)},
		swapFee:           uint256.NewInt(3000),
	}

	c := orig.CloneState().(*PoolSimulator)
	c.records[tok] = Record{Balance: uint256.NewInt(7)}
	c.totalAmountsIn[tok].SetUint64(99)
	c.maxTotalAmountsIn[tok].SetUint64(98)
	c.swapFee.SetUint64(97)
	c.totalAmountsIn["0xnew"] = uint256.NewInt(1)

	require.Equal(t, uint64(1000), orig.records[tok].Balance.Uint64())
	require.Equal(t, uint64(10), orig.totalAmountsIn[tok].Uint64())
	require.Equal(t, uint64(5000), orig.maxTotalAmountsIn[tok].Uint64())
	require.Equal(t, uint64(3000), orig.swapFee.Uint64())
	require.NotContains(t, orig.totalAmountsIn, "0xnew")
}
