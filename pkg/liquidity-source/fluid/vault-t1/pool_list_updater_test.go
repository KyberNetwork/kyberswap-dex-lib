package vaultT1

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

type mockPoolDataStore struct {
	pool *entity.Pool
}

func (d mockPoolDataStore) Get(ctx context.Context, address string) (entity.Pool, error) {
	if d.pool == nil {
		return entity.Pool{}, fmt.Errorf("not found")
	}
	return *d.pool, nil
}

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

	pools, metadataBytes, err = pu.GetNewPools(context.Background(), metadataBytes)
	require.NoError(t, err)
	require.True(t, len(pools) >= 23)

	expectedPool0 := entity.Pool{
		Address:  "0xeAbBfca72F8a8bf14C4ac59e69ECB2eB69F0811C",
		Exchange: "fluid-vault-t1",
		Type:     "fluid-vault-t1",
		Reserves: []string{"0", "0"},
		Tokens: []*entity.PoolToken{
			{
				Address:   "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48",
				Name:      "USD Coin",
				Symbol:    "USDC",
				Decimals:  6,
				Weight:    1,
				Swappable: true,
			},
			{
				Address:  "0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE",
				Name:     "ETH",
				Symbol:   "Ethereum",
				Decimals: 18,
				Weight:   1,
			},
		},
	}

	expectedPool21 := entity.Pool{
		Address:  "0x3A0b7c8840D74D39552EF53F586dD8c3d1234C40",
		Exchange: "fluid-vault-t1",
		Type:     "fluid-vault-t1",
		Reserves: []string{"0", "0"},
		Tokens: []*entity.PoolToken{
			{
				Address:   "0xdAC17F958D2ee523a2206206994597C13D831ec7",
				Name:      "Tether USD",
				Symbol:    "USDT",
				Decimals:  6,
				Weight:    1,
				Swappable: true,
			},
			{
				Address:  "0x2260FAC5E5542a773Aa44fBCfeDf7C193bc2C599",
				Name:     "Wrapped BTC",
				Symbol:   "WBTC",
				Decimals: 8,
				Weight:   1,
			},
		},
	}

	require.Equal(t, expectedPool0, pools[0])
	require.Equal(t, expectedPool21, pools[21])

	// Log all pools
	// for i, pool := range pools {
	// 	jsonEncoded, _ := json.MarshalIndent(pool, "", "  ")
	// 	t.Logf("Pool %d: %s\n", i, string(jsonEncoded))
	// }

}
