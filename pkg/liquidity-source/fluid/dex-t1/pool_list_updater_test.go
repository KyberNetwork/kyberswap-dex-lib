package dexT1

import (
	"context"
	"encoding/json"
	"math/big"
	"os"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestPoolListUpdater(t *testing.T) {
	_ = logger.SetLogLevel("debug")

	if os.Getenv("CI") != "" {
		t.Skip()
	}

	var (
		pools            []entity.Pool
		metadataBytes, _ = json.Marshal(map[string]interface{}{})
		err              error

		config = Config{
			ChainID: valueobject.ChainIDEthereum,
		}
	)

	// Setup RPC server
	rpcClient := ethrpc.New("https://ethereum.kyberengineering.io")
	rpcClient.SetMulticallContract(common.HexToAddress("0x5ba1e12693dc8f9c48aad8770482f4739beed696"))

	pu := NewPoolsListUpdater(&config, rpcClient)
	require.NotNil(t, pu)

	pools, _, err = pu.GetNewPools(context.Background(), metadataBytes)
	require.NoError(t, err)
	require.True(t, len(pools) >= 1)

	expectedPool0 := entity.Pool{
		Address:  "0x6d83f60eEac0e50A1250760151E81Db2a278e03a",
		Exchange: "fluid-dex-t1",
		Type:     "fluid-dex-t1",
		Reserves: pools[0].Reserves,
		Tokens: []*entity.PoolToken{
			{
				Address:   "0x7f39C581F595B53c5cb19bD0b3f8dA6c935E2Ca0",
				Weight:    1,
				Swappable: true,
			},
			{
				Address:   "0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE",
				Weight:    1,
				Swappable: true,
			},
		},

		Extra: pools[0].Extra,
	}

	require.Equal(t, expectedPool0, pools[0])

	var extra PoolExtra
	err = json.Unmarshal([]byte(pools[0].Extra), &extra)
	require.NoError(t, err)

	reserve, ok := new(big.Int).SetString(pools[0].Reserves[0], 10)
	require.True(t, ok)
	require.True(t, reserve.Cmp(big.NewInt(0)) > 0)
	reserve, ok = new(big.Int).SetString(pools[0].Reserves[1], 10)
	require.True(t, ok)
	require.True(t, reserve.Cmp(big.NewInt(0)) > 0)

	require.True(t, extra.CollateralReserves.Token0RealReserves.Cmp(big.NewInt(0)) > 0)
	require.True(t, extra.CollateralReserves.Token1RealReserves.Cmp(big.NewInt(0)) > 0)
	require.True(t, extra.CollateralReserves.Token0ImaginaryReserves.Cmp(big.NewInt(0)) > 0)
	require.True(t, extra.CollateralReserves.Token1ImaginaryReserves.Cmp(big.NewInt(0)) > 0)
	require.True(t, extra.DebtReserves.Token0Debt.Cmp(big.NewInt(0)) > 0)
	require.True(t, extra.DebtReserves.Token1Debt.Cmp(big.NewInt(0)) > 0)
	require.True(t, extra.DebtReserves.Token0RealReserves.Cmp(big.NewInt(0)) > 0)
	require.True(t, extra.DebtReserves.Token1RealReserves.Cmp(big.NewInt(0)) > 0)
	require.True(t, extra.DebtReserves.Token0ImaginaryReserves.Cmp(big.NewInt(0)) > 0)
	require.True(t, extra.DebtReserves.Token1ImaginaryReserves.Cmp(big.NewInt(0)) > 0)

	// Log all pools
	// for i, pool := range pools {
	// 	jsonEncoded, _ := json.MarshalIndent(pool, "", "  ")
	// 	t.Logf("Pool %d: %s\n", i, string(jsonEncoded))
	// }

}
