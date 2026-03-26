package mooniswap

import (
	"context"
	"math/big"
	"os"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type IntegrationTestSuite struct {
	suite.Suite
	client  *ethrpc.Client
	tracker *PoolTracker
	updater *PoolsListUpdater
}

func (ts *IntegrationTestSuite) SetupTest() {
	rpcURL := os.Getenv("ETH_RPC_URL")
	if rpcURL == "" {
		rpcURL = "https://eth.drpc.org"
	}

	client := ethrpc.New(rpcURL)
	client.SetMulticallContract(common.HexToAddress("0x5BA1e12693Dc8F9c48aAD8770482f4739bEeD696"))

	ts.client = client

	cfg := &Config{
		DexID:          string(DexType),
		FactoryAddress: "0xbaf9a5d4b0052359326a6cdab54babaa3a3a9643",
	}

	ts.updater = NewPoolsListUpdater(cfg, client)

	tracker, err := NewPoolTracker(cfg, client)
	require.NoError(ts.T(), err)
	ts.tracker = tracker
}

func (ts *IntegrationTestSuite) TestGetNewPools() {
	pools, _, err := ts.updater.GetNewPools(context.Background(), nil)
	require.NoError(ts.T(), err)
	require.True(ts.T(), len(pools) > 0, "should discover at least 1 pool")

	ts.T().Logf("Discovered %d pools", len(pools))
	for i, p := range pools {
		if i < 10 {
			ts.T().Logf("  %s: %s / %s", p.Address, p.Tokens[0].Address, p.Tokens[1].Address)
		}
	}
}

func (ts *IntegrationTestSuite) TestGetNewPoolState() {
	ethUsdtPool := entity.Pool{
		Address: "0xbba17b81ab4193455be10741512d0e71520f43cb",
		Tokens: []*entity.PoolToken{
			{Address: "0x0000000000000000000000000000000000000000", Swappable: true},
			{Address: "0xdac17f958d2ee523a2206206994597c13d831ec7", Swappable: true},
		},
		Reserves: []string{"0", "0"},
		Extra:    `{"fee":"0","slpFee":"0","bA0":"0","bA1":"0","bR0":"0","bR1":"0"}`,
	}

	updated, err := ts.tracker.GetNewPoolState(context.Background(), ethUsdtPool, pool.GetNewPoolStateParams{})
	require.NoError(ts.T(), err)
	require.True(ts.T(), updated.BlockNumber > 0)

	var extra Extra
	err = json.Unmarshal([]byte(updated.Extra), &extra)
	require.NoError(ts.T(), err)

	ts.T().Logf("Block: %d", updated.BlockNumber)
	ts.T().Logf("Fee: %s", extra.Fee)
	ts.T().Logf("SlippageFee: %s", extra.SlippageFee)
	ts.T().Logf("BalAdd0: %s", extra.BalAdd0)
	ts.T().Logf("BalAdd1: %s", extra.BalAdd1)
	ts.T().Logf("BalRem0: %s", extra.BalRem0)
	ts.T().Logf("BalRem1: %s", extra.BalRem1)
	ts.T().Logf("Reserves: %v", updated.Reserves)

	require.NotEqual(ts.T(), "0", extra.Fee)
}

func (ts *IntegrationTestSuite) TestCalcAmountOut_CompareWithQuoter() {
	ethUsdtPool := entity.Pool{
		Address: "0xbba17b81ab4193455be10741512d0e71520f43cb",
		Tokens: []*entity.PoolToken{
			{Address: "0x0000000000000000000000000000000000000000", Swappable: true},
			{Address: "0xdac17f958d2ee523a2206206994597c13d831ec7", Swappable: true},
		},
		Reserves: []string{"0", "0"},
		Extra:    `{"fee":"0","slpFee":"0","bA0":"0","bA1":"0","bR0":"0","bR1":"0"}`,
	}

	updated, err := ts.tracker.GetNewPoolState(context.Background(), ethUsdtPool, pool.GetNewPoolStateParams{})
	require.NoError(ts.T(), err)

	sim, err := NewPoolSimulator(updated)
	require.NoError(ts.T(), err)

	tests := []struct {
		name     string
		tokenIn  string
		tokenOut string
		amountIn *big.Int
	}{
		{"1 ETH -> USDT", "0x0000000000000000000000000000000000000000", "0xdac17f958d2ee523a2206206994597c13d831ec7", big.NewInt(1e18)},
		{"1000 USDT -> ETH", "0xdac17f958d2ee523a2206206994597c13d831ec7", "0x0000000000000000000000000000000000000000", big.NewInt(1000000000)},
		{"0.01 ETH -> USDT", "0x0000000000000000000000000000000000000000", "0xdac17f958d2ee523a2206206994597c13d831ec7", big.NewInt(1e16)},
	}

	for _, tc := range tests {
		ts.T().Run(tc.name, func(t *testing.T) {
			localResult, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{Token: tc.tokenIn, Amount: tc.amountIn},
				TokenOut:      tc.tokenOut,
			})
			require.NoError(t, err)

			var onchainAmountOut *big.Int
			req := ts.client.NewRequest().SetContext(context.Background())
			req.AddCall(&ethrpc.Call{
				ABI:    poolABI,
				Target: updated.Address,
				Method: poolMethodGetReturn,
				Params: []any{
					common.HexToAddress(tc.tokenIn),
					common.HexToAddress(tc.tokenOut),
					tc.amountIn,
				},
			}, []any{&onchainAmountOut})

			_, err = req.Call()
			require.NoError(t, err)

			diff := new(big.Int).Abs(new(big.Int).Sub(localResult.TokenAmountOut.Amount, onchainAmountOut))
			t.Logf("local=%s onchain=%s diff=%s", localResult.TokenAmountOut.Amount, onchainAmountOut, diff)

			// Allow up to 1 wei difference due to potential block-boundary timing
			require.True(t, diff.Cmp(big.NewInt(1)) <= 0,
				"local=%s onchain=%s diff=%s", localResult.TokenAmountOut.Amount, onchainAmountOut, diff)
		})
	}
}

func TestIntegration(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("skipping integration test in CI")
	}
	suite.Run(t, new(IntegrationTestSuite))
}
