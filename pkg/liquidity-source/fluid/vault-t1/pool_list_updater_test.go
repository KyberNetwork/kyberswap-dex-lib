package vaultT1

import (
	"context"
	"os"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

func TestPoolListUpdater(t *testing.T) {
	t.Parallel()
	_ = logger.SetLogLevel("debug")

	if os.Getenv("CI") != "" {
		t.Skip()
	}

	var (
		pools            []entity.Pool
		metadataBytes, _ = json.Marshal(map[string]any{})
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
		HasNative:                true,
	})

	expectedPool0 := entity.Pool{
		Address:  "0xeabbfca72f8a8bf14c4ac59e69ecb2eb69f0811c",
		Exchange: "fluid-vault-t1",
		Type:     "fluid-vault-t1",
		Reserves: []string{"0", "0"},
		Tokens: []*entity.PoolToken{
			{
				Address:   "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
				Swappable: true,
			},
			{
				Address:   "",
				Swappable: true,
			},
		},
		StaticExtra: string(staticExtraBytes),
	}

	require.Equal(t, expectedPool0, pools[0])

	staticExtraBytes, _ = json.Marshal(&StaticExtra{
		VaultLiquidationResolver: config.VaultLiquidationResolver,
		HasNative:                false,
	})

	expectedPool21 := entity.Pool{
		Address:  "0x3a0b7c8840d74d39552ef53f586dd8c3d1234c40",
		Exchange: "fluid-vault-t1",
		Type:     "fluid-vault-t1",
		Reserves: []string{"0", "0"},
		Tokens: []*entity.PoolToken{
			{
				Address:   "0xdac17f958d2ee523a2206206994597c13d831ec7",
				Swappable: true,
			},
			{
				Address:   "0x2260fac5e5542a773aa44fbcfedf7c193bc2c599",
				Swappable: true,
			},
		},
		StaticExtra: string(staticExtraBytes),
	}

	require.Equal(t, expectedPool21, pools[21])

	// Log all pools
	// for i, pool := range pools {
	// 	jsonEncoded, _ := json.MarshalIndent(pool, "", "  ")
	// 	t.Logf("Pool %d: %s\n", i, string(jsonEncoded))
	// }

}
