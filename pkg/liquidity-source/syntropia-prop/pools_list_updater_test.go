package syntropiaprop

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

// Every pair trades through one contract, so the pool id is derived from the pair, and a
// pair listed later is the only one emitted on the next run.
func TestBuildPools(t *testing.T) {
	t.Parallel()
	const cbBTC = "0xcbb7c0000ab88b473b1f5afd9ef808440eed33bf"
	u := &PoolsListUpdater{config: &Config{DexID: DexType, Pool: testPool}}
	metadata := Metadata{Seen: map[string]bool{}}
	wethUSDC := TokenPair{common.HexToAddress(testWETH), common.HexToAddress(testUSDC)}
	usdcCbBTC := TokenPair{common.HexToAddress(testUSDC), common.HexToAddress(cbBTC)}

	pools, err := u.buildPools([]TokenPair{wethUSDC}, &metadata)
	require.NoError(t, err)
	require.Len(t, pools, 1)
	require.Equal(t, testPool+"_"+testWETH+"_"+testUSDC, pools[0].Address)
	require.Equal(t, testWETH, pools[0].Tokens[0].Address)
	require.Equal(t, testUSDC, pools[0].Tokens[1].Address)
	require.Equal(t, entity.PoolReserves{"0", "0"}, pools[0].Reserves, "discovery never sets reserves")
	require.Equal(t, `{"pool":"`+testPool+`"}`, pools[0].StaticExtra)

	pools, err = u.buildPools([]TokenPair{wethUSDC, usdcCbBTC}, &metadata)
	require.NoError(t, err)
	require.Len(t, pools, 1)
	require.Equal(t, testPool+"_"+testUSDC+"_"+cbBTC, pools[0].Address)

	pools, err = u.buildPools([]TokenPair{wethUSDC, usdcCbBTC}, &metadata)
	require.NoError(t, err)
	require.Empty(t, pools)
}
