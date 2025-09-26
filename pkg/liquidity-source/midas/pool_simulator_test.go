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
		"dv-mHYPER": `{"address":"0xba9fd2850965053ffab368df8aa7ed2486f11024","exchange":"midas","type":"midas","timestamp":1758864811,"reserves":["15000000000000000000000000","80678880596338","98323676234727387617186711","98608250793368571215532080","87177086364248"],"tokens":[{"address":"0x9b5528528656dbc094765e2abb79f293c21191b9","symbol":"mHYPER","decimals":18,"swappable":true},{"address":"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48","symbol":"USDC","decimals":6,"swappable":true},{"address":"0x6b175474e89094c44da98b954eedeac495271d0f","symbol":"DAI","decimals":18,"swappable":true},{"address":"0xdc035d45d973e3ec169d2276ddab16f1e407384f","symbol":"USDS","decimals":18,"swappable":true},{"address":"0xdac17f958d2ee523a2206206994597c13d831ec7","symbol":"USDT","decimals":6,"swappable":true}],"extra":"{\"paymentTokens\":[\"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48\",\"0x6b175474e89094c44da98b954eedeac495271d0f\",\"0xdc035d45d973e3ec169d2276ddab16f1e407384f\",\"0xdac17f958d2ee523a2206206994597c13d831ec7\"],\"paused\":false,\"fnPaused\":false,\"tokensConfig\":[{\"fee\":\"0\",\"allowance\":\"80678880596338000000000000\",\"stable\":true},{\"fee\":\"0\",\"allowance\":\"98323676234727387617186711\",\"stable\":true},{\"fee\":\"0\",\"allowance\":\"98608250793368571215532080\",\"stable\":true},{\"fee\":\"0\",\"allowance\":\"87177086364248000000000000\",\"stable\":true}],\"instantDailyLimit\":\"15000000000000000000000000\",\"dailyLimits\":\"89417260890222031311412\",\"instantFee\":\"0\",\"minAmount\":\"0\",\"mTokenRate\":\"1032063910000000000\",\"tokenRates\":[\"999774910000000000\",\"999610950000000000\",\"999859270000000000\",\"1000173700000000000\"],\"waivedFeeRestriction\":false,\"minMTokenAmountForFirstDeposit\":\"0\",\"totalMinted\":\"0\",\"mTokenTotalSupply\":\"178487958011840149748813190\"}","staticExtra":"{\"isDv\":true,\"type\":\"dv\"}"}`,
	}

	ts.sims = map[string]*PoolSimulator{}
	for k, p := range ts.pools {
		var ep entity.Pool
		err := json.Unmarshal([]byte(p), &ep)
		ts.Require().Nil(err)

		sim, err := NewPoolSimulator(ep)
		ts.Require().Nil(err)
		ts.Require().NotNil(sim)

		if sim.isDv {
			ts.Require().Equal(len(sim.Info.Tokens)-1, len(sim.CanSwapTo(sim.Info.Tokens[0])))
			ts.Require().Equal(0, len(sim.CanSwapTo(sim.Info.Tokens[1])))
			ts.Require().Equal(0, len(sim.CanSwapFrom(sim.Info.Tokens[0])))
			ts.Require().Equal(1, len(sim.CanSwapFrom(sim.Info.Tokens[1])))
		} else {
			ts.Require().Equal(0, len(sim.CanSwapTo(sim.Info.Tokens[0])))
			ts.Require().Equal(1, len(sim.CanSwapTo(sim.Info.Tokens[1])))
			ts.Require().Equal(len(sim.Info.Tokens)-1, len(sim.CanSwapFrom(sim.Info.Tokens[0])))
			ts.Require().Equal(0, len(sim.CanSwapFrom(sim.Info.Tokens[1])))
		}

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
			name:              "USDC -> mHYPER",
			pool:              "dv-mHYPER",
			tokenIn:           "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			tokenOut:          "0x9b5528528656dbc094765e2abb79f293c21191b9",
			amountIn:          "10000000000000",
			expectedAmountOut: "9689322437406032345419383",
		},
		{
			name:              "USDS -> mHYPER",
			pool:              "dv-mHYPER",
			tokenIn:           "0xdc035d45d973e3ec169d2276ddab16f1e407384f",
			tokenOut:          "0x9b5528528656dbc094765e2abb79f293c21191b9",
			amountIn:          "10000000000000",
			expectedAmountOut: "9689322437406",
		},
		//{
		//	name:          "mHYPER deposit USDC",
		//	pool:          "dv-USDC-mHYPER",
		//	tokenIn:       "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
		//	tokenOut:      "0x9b5528528656dbc094765e2abb79f293c21191b9",
		//	amountIn:      "100000000000000",
		//	expectedError: ErrMVExceedAllowance,
		//},
		//{
		//	name:              "mHYPER deposit USDC",
		//	pool:              "dv-USDC-mHYPER",
		//	tokenIn:           "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
		//	tokenOut:          "0x9b5528528656dbc094765e2abb79f293c21191b9",
		//	amountIn:          "1",
		//	expectedAmountOut: "968932243740",
		//},
		//{
		//	name:              "mHYPER deposit USDC",
		//	pool:              "dv-USDC-mHYPER",
		//	tokenIn:           "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
		//	tokenOut:          "0x9b5528528656dbc094765e2abb79f293c21191b9",
		//	amountIn:          "1",
		//	expectedAmountOut: "968932243740",
		//},
		//{
		//	name:              "mHYPER deposit USDS",
		//	pool:              "dv-USDS-mHYPER",
		//	tokenIn:           "0xdc035d45d973e3ec169d2276ddab16f1e407384f",
		//	tokenOut:          "0x9b5528528656dbc094765e2abb79f293c21191b9",
		//	amountIn:          "1000000000000000000",
		//	expectedAmountOut: "968932243740603234",
		//},
		//{
		//	name:              "mHYPER redeem USDC",
		//	pool:              "rv-swapper-mHYPER-USDC",
		//	tokenIn:           "0x9b5528528656dbc094765e2abb79f293c21191b9",
		//	tokenOut:          "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
		//	amountIn:          "1000000000000000000000",
		//	expectedAmountOut: "1026903590",
		//},
		//{
		//	name:              "mTBILL redeem USDC",
		//	pool:              "rv-ustb-mTBILL-USDC",
		//	tokenIn:           "0xdd629e5241cbc5919847783e6c96b2de4754e438",
		//	tokenOut:          "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
		//	amountIn:          "1000000000000000000",
		//	expectedAmountOut: "1037683",
		//},
		//{
		//	name:              "mTBILL redeem USDC",
		//	pool:              "rv-ustb-mTBILL-USDC",
		//	tokenIn:           "0xdd629e5241cbc5919847783e6c96b2de4754e438",
		//	tokenOut:          "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
		//	amountIn:          "99999000000000000000000",
		//	expectedAmountOut: "103767319584",
		//},
		//{
		//	name:          "mTBILL redeem USDC",
		//	pool:          "rv-ustb-mTBILL-USDC",
		//	tokenIn:       "0xdd629e5241cbc5919847783e6c96b2de4754e438",
		//	tokenOut:      "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
		//	amountIn:      "9999900000000000000000000000000",
		//	expectedError: ErrMVExceedLimit,
		//},
		//{
		//	name:              "mBTC deposit WBTC",
		//	pool:              "dv-WBTC-mBTC",
		//	tokenIn:           "0x2260fac5e5542a773aa44fbcfedf7c193bc2c599",
		//	tokenOut:          "0x007115416ab6c266329a03b09a8aa39ac2ef7d9d",
		//	amountIn:          "600000000",
		//	expectedAmountOut: "5818072029399571281",
		//},
		//{
		//	name:              "mBTC redeem WBTC",
		//	pool:              "rv-mBTC-WBTC",
		//	tokenIn:           "0x007115416ab6c266329a03b09a8aa39ac2ef7d9d",
		//	tokenOut:          "0x2260fac5e5542a773aa44fbcfedf7c193bc2c599",
		//	amountIn:          "1000000000000000000",
		//	expectedAmountOut: "103054757",
		//},
		//{
		//	name:              "msyrupUSD redeem USDC",
		//	pool:              "rv-swapper-ustb-msyrupUSD-USDC",
		//	tokenIn:           "0x20226607b4fa64228abf3072ce561d6257683464",
		//	tokenOut:          "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
		//	amountIn:          "1000000000000000000",
		//	expectedAmountOut: "1002033",
		//},
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
