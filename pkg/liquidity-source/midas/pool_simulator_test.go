package midas

import (
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	bignum "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulatorTestSuite struct {
	suite.Suite

	pools map[string]string
	sims  map[string]*PoolSimulator
}

func (ts *PoolSimulatorTestSuite) SetupSuite() {
	ts.pools = map[string]string{
		"dv-USDC-mHYPER":         `{"address":"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48-0x9b5528528656dbc094765e2abb79f293c21191b9","exchange":"midas","type":"midas","reserves":["15000000000000000000000000","93542965105311"],"tokens":[{"address":"0x9b5528528656dbc094765e2abb79f293c21191b9","symbol":"mHYPER","decimals":18,"swappable":true},{"address":"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48","symbol":"USDC","decimals":6,"swappable":true}],"extra":"{\"tokenConfig\":{\"dataFeed\":\"0x3aac6fd73fa4e16ec683bd4aaf5ec89bb2c0edc2\",\"fee\":\"0\",\"allowance\":\"93542965105311000000000000\",\"stable\":true},\"instantDailyLimit\":\"15000000000000000000000000\",\"dailyLimits\":\"2254236289680381515993793\",\"instantFee\":\"0\",\"minAmount\":\"0\",\"mTokenRate\":\"1032063910000000000\",\"tokenRate\":\"999763290000000000\",\"minMTokenAmountForFirstDeposit\":\"0\",\"totalMinted\":\"0\",\"mTokenTotalSupply\":\"166099531055421689664642174\"}","staticExtra":"{\"isDv\":true,\"vault\":\"0xbA9FD2850965053Ffab368Df8AA7eD2486f11024\",\"type\":\"depositVault\"}"}`,
		"dv-USDS-mHYPER":         `{"address":"0xdc035d45d973e3ec169d2276ddab16f1e407384f-0x9b5528528656dbc094765e2abb79f293c21191b9","exchange":"midas","type":"midas","reserves":["15000000000000000000000000","98608258760059571215532081"],"tokens":[{"address":"0x9b5528528656dbc094765e2abb79f293c21191b9","symbol":"mHYPER","decimals":18,"swappable":true},{"address":"0xdc035d45d973e3ec169d2276ddab16f1e407384f","symbol":"USDS","decimals":18,"swappable":true}],"extra":"{\"tokenConfig\":{\"dataFeed\":\"0x62c81e9a3bc0032cb504a850b1b7172604f15e5e\",\"fee\":\"0\",\"allowance\":\"98608258760059571215532081\",\"stable\":true},\"instantDailyLimit\":\"15000000000000000000000000\",\"dailyLimits\":\"2254236289680381515993793\",\"instantFee\":\"0\",\"minAmount\":\"0\",\"mTokenRate\":\"1032063910000000000\",\"tokenRate\":\"999840160000000000\",\"minMTokenAmountForFirstDeposit\":\"0\",\"totalMinted\":\"0\",\"mTokenTotalSupply\":\"166099531055421689664642174\"}","staticExtra":"{\"isDv\":true,\"vault\":\"0xba9fd2850965053ffab368df8aa7ed2486f11024\",\"type\":\"depositVault\"}"}`,
		"dv-WBTC-mBTC":           `{"address":"0x2260fac5e5542a773aa44fbcfedf7c193bc2c599-0x007115416ab6c266329a03b09a8aa39ac2ef7d9d","exchange":"midas","type":"midas","timestamp":1758762651,"reserves":["150000000000000000000","11991769151"],"tokens":[{"address":"0x007115416ab6c266329a03b09a8aa39ac2ef7d9d","symbol":"mBTC","decimals":18,"swappable":true},{"address":"0x2260fac5e5542a773aa44fbcfedf7c193bc2c599","symbol":"WBTC","decimals":8,"swappable":true}],"extra":"{\"tokenRemoved\":false,\"paused\":false,\"fnPaused\":false,\"tokenConfig\":{\"dataFeed\":\"0x488f2ab54feef6b4431834d34126e103ce573796\",\"fee\":\"0\",\"allowance\":\"119917691510000000000\",\"stable\":true},\"instantDailyLimit\":\"150000000000000000000\",\"dailyLimits\":\"0\",\"instantFee\":\"0\",\"minAmount\":\"0\",\"mTokenRate\":\"1031269460000000000\",\"tokenRate\":\"999924720000000000\",\"waivedFeeRestriction\":false,\"minMTokenAmountForFirstDeposit\":\"0\",\"totalMinted\":\"0\",\"mTokenTotalSupply\":\"12689338794738916098\"}","staticExtra":"{\"isDv\":true,\"vault\":\"0x10cc8dbca90db7606013d8cd2e77eb024df693bd\",\"type\":\"depositVault\"}"}`,
		"rv-mBTC-WBTC":           `{"address":"0x007115416ab6c266329a03b09a8aa39ac2ef7d9d-0x2260fac5e5542a773aa44fbcfedf7c193bc2c599","exchange":"midas","type":"midas","reserves":["15000000000000000000","12291619805"],"tokens":[{"address":"0x007115416ab6c266329a03b09a8aa39ac2ef7d9d","symbol":"mBTC","decimals":18,"swappable":true},{"address":"0x2260fac5e5542a773aa44fbcfedf7c193bc2c599","symbol":"WBTC","decimals":8,"swappable":true}],"extra":"{\"tokenRemoved\":false,\"paused\":false,\"fnPaused\":false,\"tokenConfig\":{\"dataFeed\":\"0x488f2ab54feef6b4431834d34126e103ce573796\",\"fee\":\"0\",\"allowance\":\"122916198056572888262\",\"stable\":true},\"instantDailyLimit\":\"15000000000000000000\",\"dailyLimits\":\"0\",\"instantFee\":\"7\",\"minAmount\":\"0\",\"mTokenRate\":\"1031269460000000000\",\"tokenRate\":\"999924720000000000\",\"waivedFeeRestriction\":false,\"tokenBalance\":\"307400849\"}","staticExtra":"{\"isDv\":false,\"vault\":\"0x30d9d1e76869516aea980390494aaed45c3efc1a\",\"type\":\"redemptionVault\"}"}`,
		"rv-ustb-mTBILL-USDC":    `{"address":"0xdd629e5241cbc5919847783e6c96b2de4754e438-0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48","exchange":"midas","type":"midas","reserves":["5000000000000000000000000","69069197742549"],"tokens":[{"address":"0xdd629e5241cbc5919847783e6c96b2de4754e438","symbol":"mTBILL","decimals":18,"swappable":true},{"address":"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48","symbol":"USDC","decimals":6,"swappable":true}],"extra":"{\"tokenRemoved\":false,\"paused\":false,\"fnPaused\":false,\"tokenConfig\":{\"dataFeed\":\"0x3aac6fd73fa4e16ec683bd4aaf5ec89bb2c0edc2\",\"fee\":\"0\",\"allowance\":\"69069197742549406576584699\",\"stable\":true},\"instantDailyLimit\":\"5000000000000000000000000\",\"dailyLimits\":\"22257190592717158915583\",\"instantFee\":\"7\",\"minAmount\":\"0\",\"mTokenRate\":\"1038410460000000000\",\"tokenRate\":\"999763290000000000\",\"waivedFeeRestriction\":false,\"tokenBalance\":\"4\",\"redemption\":{\"superstateToken\":\"0x0000000000000000000000000000000000000000\",\"usdc\":\"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48\",\"redemptionFee\":\"0\",\"ustbBalance\":\"1421890820037\",\"chainlinkPrice\":{\"isBadData\":false,\"updatedAt\":\"1758760979\",\"price\":\"10834516\"},\"chainLinkFeedPrecision\":\"1000000\",\"superstateTokenPrecision\":\"1000000\"}}","staticExtra":"{\"isDv\":false,\"vault\":\"0x569d7dccbf6923350521ecbc28a555a500c4f0ec\",\"type\":\"redemptionVaultUstb\"}"}`,
		"rv-swapper-mHYPER-USDC": `{"address":"0x9b5528528656dbc094765e2abb79f293c21191b9-0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48","exchange":"midas","type":"midas","reserves":["100000000000000000000000","96501608758990"],"tokens":[{"address":"0x9b5528528656dbc094765e2abb79f293c21191b9","symbol":"mHYPER","decimals":18,"swappable":true},{"address":"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48","symbol":"USDC","decimals":6,"swappable":true}],"extra":"{\"tokenRemoved\":false,\"paused\":false,\"fnPaused\":false,\"tokenConfig\":{\"dataFeed\":\"0x3aac6fd73fa4e16ec683bd4aaf5ec89bb2c0edc2\",\"fee\":\"0\",\"allowance\":\"96501608758990912170893537\",\"stable\":true},\"instantDailyLimit\":\"100000000000000000000000\",\"dailyLimits\":\"98777394102348649608399\",\"instantFee\":\"50\",\"minAmount\":\"0\",\"mTokenRate\":\"1032063910000000000\",\"tokenRate\":\"999763290000000000\",\"waivedFeeRestriction\":false,\"tokenBalance\":\"0\",\"mToken1Balance\":\"98791394102348649608399\",\"mToken2Balance\":\"600402872796471689920331\",\"swapperVaultType\":\"redemptionVaultUstb\",\"mTbillRedemptionVault\":{\"tokenRemoved\":false,\"paused\":false,\"fnPaused\":false,\"tokenConfig\":{\"dataFeed\":\"0x3aac6fd73fa4e16ec683bd4aaf5ec89bb2c0edc2\",\"fee\":\"0\",\"allowance\":\"69093325503972790497333925\",\"stable\":true},\"instantDailyLimit\":\"5000000000000000000000000\",\"dailyLimits\":\"122812728479005885228780\",\"instantFee\":\"7\",\"minAmount\":\"0\",\"mTokenRate\":\"1038410460000000000\",\"tokenRate\":\"999763290000000000\",\"waivedFeeRestriction\":true,\"tokenBalance\":\"0\",\"redemption\":{\"superstateToken\":\"0x0000000000000000000000000000000000000000\",\"usdc\":\"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48\",\"redemptionFee\":\"0\",\"ustbBalance\":\"1424117757224\",\"chainlinkPrice\":{\"isBadData\":false,\"updatedAt\":\"1758756419\",\"price\":\"10834452\"},\"chainLinkFeedPrecision\":\"1000000\",\"superstateTokenPrecision\":\"1000000\"}}}","staticExtra":"{\"isDv\":false,\"vault\":\"0x6be2f55816efd0d91f52720f096006d63c366e98\",\"type\":\"redemptionVaultSwapper\"}"}`,
	}

	ts.sims = map[string]*PoolSimulator{}
	for k, p := range ts.pools {
		var ep entity.Pool
		err := json.Unmarshal([]byte(p), &ep)
		ts.Require().Nil(err)

		sim, err := NewPoolSimulator(ep)
		ts.Require().Nil(err)
		ts.Require().NotNil(sim)

		ts.sims[k] = sim
	}
}

func (ts *PoolSimulatorTestSuite) TestCalcAmountOut() {
	ts.T().Parallel()

	testCases := []struct {
		name     string
		pool     string
		tokenIn  string
		tokenOut string
		amountIn string

		expectedAmountOut string
		expectedError     error
	}{
		{
			name:              "mHYPER deposit USDC",
			pool:              "dv-USDC-mHYPER",
			tokenIn:           "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			tokenOut:          "0x9b5528528656dbc094765e2abb79f293c21191b9",
			amountIn:          "10000000000000",
			expectedAmountOut: "9689322437406032345419383",
		},
		{
			name:          "mHYPER deposit USDC",
			pool:          "dv-USDC-mHYPER",
			tokenIn:       "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			tokenOut:      "0x9b5528528656dbc094765e2abb79f293c21191b9",
			amountIn:      "100000000000000",
			expectedError: ErrMVExceedAllowance,
		},
		{
			name:              "mHYPER deposit USDC",
			pool:              "dv-USDC-mHYPER",
			tokenIn:           "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			tokenOut:          "0x9b5528528656dbc094765e2abb79f293c21191b9",
			amountIn:          "1",
			expectedAmountOut: "968932243740",
		},
		{
			name:              "mHYPER deposit USDC",
			pool:              "dv-USDC-mHYPER",
			tokenIn:           "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			tokenOut:          "0x9b5528528656dbc094765e2abb79f293c21191b9",
			amountIn:          "1",
			expectedAmountOut: "968932243740",
		},
		{
			name:              "mHYPER deposit USDS",
			pool:              "dv-USDS-mHYPER",
			tokenIn:           "0xdc035d45d973e3ec169d2276ddab16f1e407384f",
			tokenOut:          "0x9b5528528656dbc094765e2abb79f293c21191b9",
			amountIn:          "1000000000000000000",
			expectedAmountOut: "968932243740603234",
		},
		{
			name:              "mHYPER redeem USDC",
			pool:              "rv-swapper-mHYPER-USDC",
			tokenIn:           "0x9b5528528656dbc094765e2abb79f293c21191b9",
			tokenOut:          "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			amountIn:          "1000000000000000000000",
			expectedAmountOut: "1026903590",
		},
		{
			name:              "mTBILL redeem USDC",
			pool:              "rv-ustb-mTBILL-USDC",
			tokenIn:           "0xdd629e5241cbc5919847783e6c96b2de4754e438",
			tokenOut:          "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			amountIn:          "1000000000000000000",
			expectedAmountOut: "1037683",
		},
		{
			name:              "mTBILL redeem USDC",
			pool:              "rv-ustb-mTBILL-USDC",
			tokenIn:           "0xdd629e5241cbc5919847783e6c96b2de4754e438",
			tokenOut:          "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			amountIn:          "99999000000000000000000",
			expectedAmountOut: "103767319584",
		},
		{
			name:          "mTBILL redeem USDC",
			pool:          "rv-ustb-mTBILL-USDC",
			tokenIn:       "0xdd629e5241cbc5919847783e6c96b2de4754e438",
			tokenOut:      "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			amountIn:      "9999900000000000000000000000000",
			expectedError: ErrMVExceedLimit,
		},
		{
			name:              "mBTC deposit WBTC",
			pool:              "dv-WBTC-mBTC",
			tokenIn:           "0x2260fac5e5542a773aa44fbcfedf7c193bc2c599",
			tokenOut:          "0x007115416ab6c266329a03b09a8aa39ac2ef7d9d",
			amountIn:          "600000000",
			expectedAmountOut: "5818072029399571281",
		},
		{
			name:              "mBTC redeem WBTC",
			pool:              "rv-mBTC-WBTC",
			tokenIn:           "0x007115416ab6c266329a03b09a8aa39ac2ef7d9d",
			tokenOut:          "0x2260fac5e5542a773aa44fbcfedf7c193bc2c599",
			amountIn:          "1000000000000000000",
			expectedAmountOut: "103054757",
		},
	}

	for _, tc := range testCases {
		ts.T().Run(tc.pool, func(t *testing.T) {
			cloned := ts.sims[tc.pool].CloneState()

			res, err := cloned.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{
					Token:  tc.tokenIn,
					Amount: bignum.NewBig(tc.amountIn),
				},
				TokenOut: tc.tokenOut,
			})
			if tc.expectedError == nil {
				require.NotNil(t, res)
				require.Equal(t, tc.expectedAmountOut, res.TokenAmountOut.Amount.String())
				cloned.UpdateBalance(pool.UpdateBalanceParams{
					TokenAmountIn: pool.TokenAmount{
						Token:  tc.tokenIn,
						Amount: bignum.NewBig(tc.amountIn),
					},
					TokenAmountOut: *res.TokenAmountOut,
					SwapInfo:       res.SwapInfo,
				})
				require.Equal(t, tc.expectedAmountOut, res.TokenAmountOut.Amount.String())
			} else {
				require.ErrorIs(t, tc.expectedError, err)
			}
		})
	}
}

func TestPoolSimulatorTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(PoolSimulatorTestSuite))
}
