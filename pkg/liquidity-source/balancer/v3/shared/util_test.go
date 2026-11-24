package shared

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// Bush stable pool 0x8CD8...8027 on Robinhood: getPoolData.poolConfigBits, aggregate swap fee 50%.
// Static fee 5e14 and aggregate yield fee 5e17 sit in neighbouring bit fields and must not leak in.
func TestAggregateSwapFeeFromConfigBits(t *testing.T) {
	var r RpcResult
	r.PoolData.PoolConfigBits = common.HexToHash("0x0000000000000000000000000000000000000003d5f1617c0000000601312d01312d00004e200003")
	require.Equal(t, uint256.NewInt(5e17), r.AggregateSwapFee())
}

func TestFrontierAddresses(t *testing.T) {
	wantVault := common.HexToAddress("0xFF00009C1A83Ec25434522ef355b99e9a70000FF")
	wantRouter := common.HexToAddress("0x7D08bc34e444848983227E266F61146A7e935A94")

	exchanges := []string{
		valueobject.ExchangeFrontierWeighted,
		valueobject.ExchangeFrontierStable,
		valueobject.ExchangeFrontierECLP,
	}
	chains := []valueobject.ChainID{
		valueobject.ChainIDEthereum,
		valueobject.ChainIDBase,
		valueobject.ChainIDMonad,
	}

	for _, exchange := range exchanges {
		for _, chainID := range chains {
			require.Equal(t, wantVault, Vault(chainID, exchange))
			router, ok := BatchRouter(chainID, exchange)
			require.True(t, ok)
			require.Equal(t, wantRouter, router)
		}

		_, ok := BatchRouter(valueobject.ChainIDArbitrumOne, exchange)
		require.False(t, ok)
	}
}
