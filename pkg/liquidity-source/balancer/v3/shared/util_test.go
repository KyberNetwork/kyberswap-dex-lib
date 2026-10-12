package shared

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

// Bush stable pool 0x8CD8...8027 on Robinhood: getPoolData.poolConfigBits, aggregate swap fee 50%.
// Static fee 5e14 and aggregate yield fee 5e17 sit in neighbouring bit fields and must not leak in.
func TestAggregateSwapFeeFromConfigBits(t *testing.T) {
	var r RpcResult
	r.PoolData.PoolConfigBits = common.HexToHash("0x0000000000000000000000000000000000000003d5f1617c0000000601312d01312d00004e200003")
	require.Equal(t, uint256.NewInt(5e17), r.AggregateSwapFee())
}
