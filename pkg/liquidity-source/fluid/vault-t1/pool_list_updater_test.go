package vaultT1

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
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
			VaultLiquidationResolver: "0x6Cd1E75b524D3CCa4c3320436d6F09e24Dadd613",
		}
	)

	// Setup RPC server
	rpcClient := ethrpc.New("https://ethereum.kyberengineering.io")
	rpcClient.SetMulticallContract(common.HexToAddress("0x5ba1e12693dc8f9c48aad8770482f4739beed696"))

	pu := NewPoolsListUpdater(&config, rpcClient)
	require.NotNil(t, pu)

	pools, _, err = pu.GetNewPools(context.Background(), metadataBytes)
	require.NoError(t, err)
	require.True(t, len(pools) >= 23)

	staticExtraBytes, _ := json.Marshal(&StaticExtra{
		VaultLiquidationResolver: config.VaultLiquidationResolver,
	})

	expectedPool0 := entity.Pool{
		Address:  "0xeAbBfca72F8a8bf14C4ac59e69ECB2eB69F0811C",
		Exchange: "fluid-vault-t1",
		Type:     "fluid-vault-t1",
		Reserves: []string{"0", "0"},
		Tokens: []*entity.PoolToken{
			{
				Address:   "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
				Weight:    1,
				Swappable: true,
			},
			{
				Address:   "0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
				Weight:    1,
				Swappable: true,
			},
		},
		StaticExtra: string(staticExtraBytes),
	}

	expectedPool21 := entity.Pool{
		Address:  "0x3A0b7c8840D74D39552EF53F586dD8c3d1234C40",
		Exchange: "fluid-vault-t1",
		Type:     "fluid-vault-t1",
		Reserves: []string{"0", "0"},
		Tokens: []*entity.PoolToken{
			{
				Address:   "0xdac17f958d2ee523a2206206994597c13d831ec7",
				Weight:    1,
				Swappable: true,
			},
			{
				Address:   "0x2260fac5e5542a773aa44fbcfedf7c193bc2c599",
				Weight:    1,
				Swappable: true,
			},
		},
		StaticExtra: string(staticExtraBytes),
	}

	require.Equal(t, expectedPool0, pools[0])
	require.Equal(t, expectedPool21, pools[21])

	// Log all pools
	// for i, pool := range pools {
	// 	jsonEncoded, _ := json.MarshalIndent(pool, "", "  ")
	// 	t.Logf("Pool %d: %s\n", i, string(jsonEncoded))
	// }

}
