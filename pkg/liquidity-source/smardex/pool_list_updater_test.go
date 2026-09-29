package smardex

import (
	"context"
	"math/big"
	"os"
	"strings"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

// Live discovery test against the KyberSwap RPC gateway, which authenticates
// with an `x-api-key` header -- ethrpc.New/ethclient.Dial can't attach headers,
// so the client is built from rpc.DialOptions + rpc.WithHeader instead.
//
// The key is never hardcoded: export it before running, e.g.
//
//	RPC_API_KEY=<key> go test ./pkg/liquidity-source/smardex/ -run TestPoolListUpdaterTestSuite -v
const (
	rpcURL       = "https://api.kyberswap.com/rpc/ethereum"
	rpcAPIKeyEnv = "RPC_API_KEY"

	// Multicall2 on Ethereum mainnet, matching rpcURL's chain.
	multicallAddress = "0x7eCfBaa8742fDf5756DAC92fbc8b90a19b8815bF"

	// Smardex SmardexFactory on Ethereum mainnet.
	// Override with SMARDEX_FACTORY_ADDRESS when pointing the test at another chain.
	defaultFactoryAddress = "0x41A00e3FbE7F479A99bA6822704d9c5dEB611F22"
	factoryAddressEnv     = "SMARDEX_FACTORY_ADDRESS"

	zeroAddress = "0x0000000000000000000000000000000000000000"
)

type PoolListUpdaterTestSuite struct {
	suite.Suite

	client         *ethrpc.Client
	updater        PoolListUpdater
	factoryAddress string
}

func (ts *PoolListUpdaterTestSuite) SetupTest() {
	apiKey := os.Getenv(rpcAPIKeyEnv)
	if apiKey == "" {
		ts.T().Skipf("%s is not set", rpcAPIKeyEnv)
	}

	rpcOpts := []rpc.ClientOption{rpc.WithHeader("x-api-key", apiKey)}
	rpcConn, err := rpc.DialOptions(context.Background(), rpcURL, rpcOpts...)
	ts.Require().NoError(err)

	rpcClient := ethrpc.NewWithClient(ethclient.NewClient(rpcConn))
	rpcClient.SetMulticallContract(common.HexToAddress(multicallAddress))

	ts.client = rpcClient

	ts.factoryAddress = defaultFactoryAddress
	if addr := os.Getenv(factoryAddressEnv); addr != "" {
		ts.factoryAddress = addr
	}

	config := Config{
		DexID:          "smardex",
		FactoryAddress: ts.factoryAddress,
		PoolPagingSize: 20,
		ChainID:        uint(1),
	}

	ts.updater = PoolListUpdater{
		config:       &config,
		ethrpcClient: ts.client,
	}

}

// allPairsLength reads the factory's current pair count, so expectations are
// derived from live chain state instead of a stale hardcoded total.
func (ts *PoolListUpdaterTestSuite) allPairsLength() int {
	var length *big.Int
	_, err := ts.client.NewRequest().AddCall(&ethrpc.Call{
		ABI:    factoryABI,
		Target: ts.factoryAddress,
		Method: factoryAllPairsLengthMethod,
	}, []any{&length}).Call()
	ts.Require().NoError(err)
	ts.Require().NotNil(length)

	return int(length.Int64())
}

func (ts *PoolListUpdaterTestSuite) TestGetNewPools() {
	totalPairs := ts.allPairsLength()
	ts.Require().Positive(totalPairs)

	pagingSize := ts.updater.config.PoolPagingSize

	testCases := []struct {
		name   string
		offset int
	}{
		{name: "from the start of the factory list", offset: 0},
		{name: "resuming mid list", offset: pagingSize},
		{name: "at the last partial page", offset: max(0, totalPairs-pagingSize/2)},
		{name: "past the end of the factory list", offset: totalPairs + 1},
	}

	for _, tc := range testCases {
		ts.Run(tc.name, func() {
			// A page is capped by both the paging size and what's left in the factory.
			expectedCount := max(0, min(pagingSize, totalPairs-tc.offset))

			metadataBytes, err := json.Marshal(PoolListUpdaterMetadata{Offset: tc.offset})
			ts.Require().NoError(err)

			pools, metadataRes, err := ts.updater.GetNewPools(context.Background(), metadataBytes)
			ts.Require().NoError(err)
			assert.Len(ts.T(), pools, expectedCount)

			for _, p := range pools {
				assert.Equal(ts.T(), strings.ToLower(p.Address), p.Address)
				assert.NotEqual(ts.T(), zeroAddress, p.Address)
				assert.Equal(ts.T(), DexTypeSmardex, p.Type)
				assert.Equal(ts.T(), entity.PoolReserves{reserveZero, reserveZero}, p.Reserves)

				ts.Require().Len(p.Tokens, 2)
				for _, token := range p.Tokens {
					assert.Equal(ts.T(), strings.ToLower(token.Address), token.Address)
					assert.NotEqual(ts.T(), zeroAddress, token.Address)
					assert.True(ts.T(), token.Swappable)
				}
			}

			// The cursor advances by exactly the number of pools handed back, so the
			// next call resumes where this one stopped.
			var savedMetadataRes PoolListUpdaterMetadata
			ts.Require().NoError(json.Unmarshal(metadataRes, &savedMetadataRes))
			assert.Equal(ts.T(), tc.offset+expectedCount, savedMetadataRes.Offset)
		})
	}
}

func TestPoolListUpdaterTestSuite(t *testing.T) {
	t.Parallel()
	if os.Getenv(rpcAPIKeyEnv) == "" {
		t.Skipf("live RPC test: set %s to run", rpcAPIKeyEnv)
	}
	suite.Run(t, new(PoolListUpdaterTestSuite))
}
