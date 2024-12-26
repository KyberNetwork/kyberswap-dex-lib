package ambient_test

import (
	"context"
	"fmt"
	mutableclient "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql/mutable"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/KyberNetwork/blockchain-toolkit/time/durationjson"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ambient"
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
	if os.Getenv("CI") != "" {
		t.Skip()
	}

	var (
		pools                       []entity.Pool
		firstRunPool, secondRunPool entity.Pool
		metadataBytes, _            = json.Marshal(ambient.PoolListUpdaterMetadata{LastCreateTime: 0})
		err                         error

		config = ambient.Config{
			DexID:                    "ambient",
			SubgraphAPI:              "https://api.studio.thegraph.com/query/47610/croc-mainnet/version/latest",
			SubgraphRequestTimeout:   durationjson.Duration{Duration: time.Second * 10},
			SubgraphLimit:            10,
			PoolIdx:                  big.NewInt(420),
			NativeTokenAddress:       "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
			QueryContractAddress:     "0xCA00926b6190c2C59336E73F02569c356d7B6b56",
			SwapDexContractAddress:   "0xAaAaAAAaA24eEeb8d57D431224f73832bC34f688",
			MulticallContractAddress: multicallAddress,
		}

		graphqlClientCfg = &mutableclient.Config{
			Url:     config.SubgraphAPI,
			Timeout: config.SubgraphRequestTimeout.Duration,
		}
		graphqlClient = mutableclient.New(*graphqlClientCfg)
	)

	{
		t.Logf("first run with limit = 10")

		pu, err := ambient.NewPoolsListUpdater(config, mockPoolDataStore{}, graphqlClient, graphqlClientCfg)
		require.NoError(t, err)
		pools, metadataBytes, err = pu.GetNewPools(context.Background(), metadataBytes)
		require.NoError(t, err)
		t.Logf("%s\n", string(metadataBytes))
		require.Equal(t, 1, len(pools))
		firstRunPool = pools[0]

		jsonEncoded, _ := json.MarshalIndent(firstRunPool, "", "  ")
		fmt.Printf("%s\n", string(jsonEncoded))
	}

	{
		t.Logf("second run with metadata from first run and limit = 1000")

		config.SubgraphLimit = 1000
		pu, err := ambient.NewPoolsListUpdater(config, mockPoolDataStore{pool: &firstRunPool}, graphqlClient, graphqlClientCfg)
		require.NoError(t, err)
		pools, metadataBytes, err = pu.GetNewPools(context.Background(), metadataBytes)
		require.NoError(t, err)
		t.Logf("%s\n", string(metadataBytes))
		require.Equal(t, 1, len(pools))
		secondRunPool = pools[0]

		jsonEncoded, _ := json.MarshalIndent(secondRunPool, "", "  ")
		fmt.Printf("%s\n", string(jsonEncoded))
	}

	require.Lessf(t, len(firstRunPool.Tokens), len(secondRunPool.Tokens),
		"secondRunPool.Tokens must be appended from firstRunPool.Tokens")
	for i, token := range firstRunPool.Tokens {
		require.Equalf(t, token, secondRunPool.Tokens[i],
			"secondRunPool.Tokens must be appended from firstRunPool.Tokens")
	}

	var (
		firstRunExtra, secondRunExtra ambient.Extra
	)
	err = json.Unmarshal([]byte(firstRunPool.Extra), &firstRunExtra)
	require.NoError(t, err)
	err = json.Unmarshal([]byte(secondRunPool.Extra), &secondRunExtra)
	require.NoError(t, err)

	require.Lessf(t, len(firstRunExtra.TokenPairs), len(secondRunExtra.TokenPairs),
		"secondRunPool.Extra.TokenPairs must be extended from firstRunPool.Extra.TokenPairs")
	for pair, info := range firstRunExtra.TokenPairs {
		secondInfo, ok := secondRunExtra.TokenPairs[pair]
		require.Truef(t, ok,
			"secondRunPool.Extra.TokenPairs must be extended from firstRunPool.Extra.TokenPairs")
		require.Equalf(t, info, secondInfo,
			"secondRunPool.Extra.TokenPairs must be extended from firstRunPool.Extra.TokenPairs")
	}

	var (
		tokenAddrs          []common.Address
		tokenAddrsFromPairs []common.Address
	)
	for _, token := range secondRunPool.Tokens {
		tokenAddrs = append(tokenAddrs, common.HexToAddress(token.Address))
	}
	for pair := range secondRunExtra.TokenPairs {
		if pair.Base == ambient.NativeTokenPlaceholderAddress {
			pair.Base = common.HexToAddress(config.NativeTokenAddress)
		}
		tokenAddrsFromPairs = append(tokenAddrsFromPairs, pair.Base, pair.Quote)
	}
	require.Subsetf(t, tokenAddrs, tokenAddrsFromPairs,
		".Tokens[].Address and .Extra.TokenPairs[].{Base,Quote} must be the same set")
	require.Subsetf(t, tokenAddrsFromPairs, tokenAddrs,
		".Tokens[].Address and .Extra.TokenPairs[].{Base,Quote} must be the same set")
}
