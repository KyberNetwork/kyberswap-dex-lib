package midas

import (
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
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
		"dv-mHYPER":              `{"address":"0xba9fd2850965053ffab368df8aa7ed2486f11024","exchange":"midas","type":"midas","timestamp":1758873956,"reserves":["15000000000000000000000000","80678596996338","98323676234727387617186711","98608250793368571215532080","87177086364248"],"tokens":[{"address":"0x9b5528528656dbc094765e2abb79f293c21191b9","symbol":"mHYPER","decimals":18,"swappable":true},{"address":"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48","symbol":"USDC","decimals":6,"swappable":true},{"address":"0x6b175474e89094c44da98b954eedeac495271d0f","symbol":"DAI","decimals":18,"swappable":true},{"address":"0xdc035d45d973e3ec169d2276ddab16f1e407384f","symbol":"USDS","decimals":18,"swappable":true},{"address":"0xdac17f958d2ee523a2206206994597c13d831ec7","symbol":"USDT","decimals":6,"swappable":true}],"extra":"{\"mToken\":\"0x9b5528528656dbc094765e2abb79f293c21191b9\",\"paymentTokens\":[\"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48\",\"0x6b175474e89094c44da98b954eedeac495271d0f\",\"0xdc035d45d973e3ec169d2276ddab16f1e407384f\",\"0xdac17f958d2ee523a2206206994597c13d831ec7\"],\"paused\":false,\"fnPaused\":false,\"tokensConfig\":[{\"fee\":\"0\",\"allowance\":\"80678596996338000000000000\",\"stable\":true},{\"fee\":\"0\",\"allowance\":\"98323676234727387617186711\",\"stable\":true},{\"fee\":\"0\",\"allowance\":\"98608250793368571215532080\",\"stable\":true},{\"fee\":\"0\",\"allowance\":\"87177086364248000000000000\",\"stable\":true}],\"instantDailyLimit\":\"15000000000000000000000000\",\"dailyLimits\":\"89692050074546866388727\",\"instantFee\":\"0\",\"minAmount\":\"0\",\"mTokenRate\":\"1032063910000000000\",\"tokenRates\":[\"999747170000000000\",\"999649900000000000\",\"999859270000000000\",\"1000173700000000000\"],\"waivedFeeRestriction\":false,\"mTokenDecimals\":18,\"minMTokenAmountForFirstDeposit\":\"0\",\"totalMinted\":\"0\",\"mTokenTotalSupply\":\"178488232801024474583890505\"}","staticExtra":"{\"isDv\":true,\"type\":\"dv\"}"}`,
		"dv-mBTC":                `{"address":"0x10cc8dbca90db7606013d8cd2e77eb024df693bd","exchange":"midas","type":"midas","timestamp":1758873992,"reserves":["150000000000000000000","11991767962"],"tokens":[{"address":"0x007115416ab6c266329a03b09a8aa39ac2ef7d9d","symbol":"mBTC","decimals":18,"swappable":true},{"address":"0x2260fac5e5542a773aa44fbcfedf7c193bc2c599","symbol":"WBTC","decimals":8,"swappable":true}],"extra":"{\"mToken\":\"0x007115416ab6c266329a03b09a8aa39ac2ef7d9d\",\"paymentTokens\":[\"0x2260fac5e5542a773aa44fbcfedf7c193bc2c599\"],\"paused\":false,\"fnPaused\":false,\"tokensConfig\":[{\"fee\":\"0\",\"allowance\":\"119917679620000000000\",\"stable\":true}],\"instantDailyLimit\":\"150000000000000000000\",\"dailyLimits\":\"0\",\"instantFee\":\"0\",\"minAmount\":\"0\",\"mTokenRate\":\"1031269460000000000\",\"tokenRates\":[\"1000250000000000000\"],\"waivedFeeRestriction\":false,\"mTokenDecimals\":18,\"minMTokenAmountForFirstDeposit\":\"0\",\"totalMinted\":\"0\",\"mTokenTotalSupply\":\"12689350324218321024\"}","staticExtra":"{\"isDv\":true,\"type\":\"dv\"}"}`,
		"rv-mBTC":                `{"address":"0x30d9d1e76869516aea980390494aaed45c3efc1a","exchange":"midas","type":"midas","timestamp":1758874004,"reserves":["15000000000000000000","12291619805","0"],"tokens":[{"address":"0x007115416ab6c266329a03b09a8aa39ac2ef7d9d","symbol":"mBTC","decimals":18,"swappable":true},{"address":"0x2260fac5e5542a773aa44fbcfedf7c193bc2c599","symbol":"WBTC","decimals":8,"swappable":true},{"address":"0xcbb7c0000ab88b473b1f5afd9ef808440eed33bf","symbol":"cbBTC","decimals":8,"swappable":true}],"extra":"{\"mToken\":\"0x007115416ab6c266329a03b09a8aa39ac2ef7d9d\",\"paymentTokens\":[\"0x2260fac5e5542a773aa44fbcfedf7c193bc2c599\",\"0xcbb7c0000ab88b473b1f5afd9ef808440eed33bf\"],\"paused\":false,\"fnPaused\":false,\"tokensConfig\":[{\"fee\":\"0\",\"allowance\":\"122916198056572888262\",\"stable\":true},{\"fee\":\"0\",\"allowance\":\"10000000\",\"stable\":true}],\"instantDailyLimit\":\"15000000000000000000\",\"dailyLimits\":\"0\",\"instantFee\":\"7\",\"minAmount\":\"0\",\"mTokenRate\":\"1031269460000000000\",\"tokenRates\":[\"1000250000000000000\",\"1000000000000000000\"],\"waivedFeeRestriction\":false,\"mTokenDecimals\":18,\"tokenBalances\":[\"307402038\",\"0\"]}","staticExtra":"{\"isDv\":false,\"type\":\"rv\"}"}`,
		"rv-ustb-mTBILL":         `{"address":"0x569d7dccbf6923350521ecbc28a555a500c4f0ec","exchange":"midas","type":"midas","timestamp":1758874028,"reserves":["5000000000000000000000000","68458535848678"],"tokens":[{"address":"0xdd629e5241cbc5919847783e6c96b2de4754e438","symbol":"mTBILL","decimals":18,"swappable":true},{"address":"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48","symbol":"USDC","decimals":6,"swappable":true}],"extra":"{\"mToken\":\"0xdd629e5241cbc5919847783e6c96b2de4754e438\",\"paymentTokens\":[\"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48\"],\"paused\":false,\"fnPaused\":false,\"tokensConfig\":[{\"fee\":\"0\",\"allowance\":\"68458535848678250420048886\",\"stable\":true}],\"instantDailyLimit\":\"5000000000000000000000000\",\"dailyLimits\":\"6619226724036878940784\",\"instantFee\":\"7\",\"minAmount\":\"0\",\"mTokenRate\":\"1038507630000000000\",\"tokenRates\":[\"999747170000000000\"],\"waivedFeeRestriction\":false,\"mTokenDecimals\":18,\"tokenBalances\":[\"9\"],\"redemption\":{\"usdc\":\"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48\",\"redemptionFee\":\"0\",\"ustbBalance\":\"1365532675527\",\"chainlinkPrice\":{\"isBadData\":false,\"updatedAt\":\"1758874019\",\"price\":\"10836320\"},\"chainLinkFeedPrecision\":\"1000000\",\"superstateTokenPrecision\":\"1000000\"}}","staticExtra":"{\"isDv\":false,\"type\":\"rvUstb\"}"}`,
		"rv-swapper-ustb-mHYPER": `{"address":"0x6be2f55816efd0d91f52720f096006d63c366e98","exchange":"midas","type":"midas","timestamp":1758872943,"reserves":["5000000000000000000000000","96087162519097"],"tokens":[{"address":"0x9b5528528656dbc094765e2abb79f293c21191b9","symbol":"mHYPER","decimals":18,"swappable":true},{"address":"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48","symbol":"USDC","decimals":6,"swappable":true}],"extra":"{\"mToken\":\"0x9b5528528656dbc094765e2abb79f293c21191b9\",\"paymentTokens\":[\"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48\"],\"paused\":false,\"fnPaused\":false,\"tokensConfig\":[{\"fee\":\"0\",\"allowance\":\"96087162519097693607165549\",\"stable\":true}],\"instantDailyLimit\":\"5000000000000000000000000\",\"dailyLimits\":\"6000000000000000000000\",\"instantFee\":\"50\",\"minAmount\":\"0\",\"mTokenRate\":\"1032063910000000000\",\"tokenRates\":[\"999708270000000000\"],\"waivedFeeRestriction\":false,\"mTokenDecimals\":18,\"tokenBalances\":[\"0\"],\"mToken2Balance\":\"7408230094820243793\",\"swapperVaultType\":\"rvUstb\",\"mTbillRedemptionVault\":{\"mToken\":\"0xdd629e5241cbc5919847783e6c96b2de4754e438\",\"paymentTokens\":[\"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48\"],\"paused\":false,\"fnPaused\":false,\"tokensConfig\":[{\"fee\":\"0\",\"allowance\":\"68458535848678250420048886\",\"stable\":true}],\"instantDailyLimit\":\"5000000000000000000000000\",\"dailyLimits\":\"6619226724036878940784\",\"instantFee\":\"7\",\"minAmount\":\"0\",\"mTokenRate\":\"1038507630000000000\",\"tokenRates\":[\"999708270000000000\"],\"waivedFeeRestriction\":false,\"mTokenDecimals\":18,\"tokenBalances\":[\"9\"],\"redemption\":{\"usdc\":\"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48\",\"redemptionFee\":\"0\",\"ustbBalance\":\"1365532675527\",\"chainlinkPrice\":{\"isBadData\":false,\"updatedAt\":\"1758872939\",\"price\":\"10836304\"},\"chainLinkFeedPrecision\":\"1000000\",\"superstateTokenPrecision\":\"1000000\"}}}","staticExtra":"{\"isDv\":false,\"type\":\"rvSwapper\"}"}`,
		"rv-swapper-rv-mBASIC":   `{"address":"0xf804a646c034749b5484bf7dfe875f6a4f969840","exchange":"midas","type":"midas","timestamp":1758888434,"reserves":["1000000000000000000000000","7218602534375"],"tokens":[{"address":"0x1c2757c1fef1038428b5bef062495ce94bbe92b2","symbol":"mBASIS","decimals":18,"swappable":true},{"address":"0x833589fcd6edb6e08f4c7c32d4f71b54bda02913","symbol":"USDC","decimals":6,"swappable":true}],"extra":"{\"mToken\":\"0x1c2757c1fef1038428b5bef062495ce94bbe92b2\",\"paymentTokens\":[\"0x833589fcd6edb6e08f4c7c32d4f71b54bda02913\"],\"paused\":false,\"fnPaused\":false,\"tokensConfig\":[{\"fee\":\"0\",\"allowance\":\"7218602534375785923243775\",\"stable\":true}],\"instantDailyLimit\":\"1000000000000000000000000\",\"dailyLimits\":\"568063562290720977542\",\"instantFee\":\"50\",\"minAmount\":\"0\",\"mTokenRate\":\"1143344350000000000\",\"tokenRates\":[\"999730000000000000\"],\"waivedFeeRestriction\":false,\"mTokenDecimals\":18,\"tokenBalances\":[\"45173578553\"],\"mToken2Balance\":\"132374207572670032128922\",\"swapperVaultType\":\"rv\",\"mTbillRedemptionVault\":{\"mToken\":\"0xdd629e5241cbc5919847783e6c96b2de4754e438\",\"paymentTokens\":[\"0x833589fcd6edb6e08f4c7c32d4f71b54bda02913\"],\"paused\":false,\"fnPaused\":false,\"tokensConfig\":[{\"fee\":\"0\",\"allowance\":\"5212112473374236394229144\",\"stable\":true}],\"instantDailyLimit\":\"1000000000000000000000000\",\"dailyLimits\":\"558079088702079086895\",\"instantFee\":\"7\",\"minAmount\":\"0\",\"mTokenRate\":\"1038507630000000000\",\"tokenRates\":[\"999730000000000000\"],\"waivedFeeRestriction\":false,\"mTokenDecimals\":18,\"tokenBalances\":[\"129483546770\"]}}","staticExtra":"{\"isDv\":false,\"type\":\"rvSwapper\"}"}`,
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
		{
			name:              "WBTC -> mBTC",
			pool:              "dv-mBTC",
			tokenIn:           "0x2260fac5e5542a773aa44fbcfedf7c193bc2c599",
			tokenOut:          "0x007115416ab6c266329a03b09a8aa39ac2ef7d9d",
			amountIn:          "10000000000",
			expectedAmountOut: "96967867156659521363",
		},
		{
			name:              "WBTC -> mBTC",
			pool:              "dv-mBTC",
			tokenIn:           "0x2260fac5e5542a773aa44fbcfedf7c193bc2c599",
			tokenOut:          "0x007115416ab6c266329a03b09a8aa39ac2ef7d9d",
			amountIn:          "600000000",
			expectedAmountOut: "5818072029399571281",
		},
		{
			name:              "mBTC -> WBTC",
			pool:              "rv-mBTC",
			tokenIn:           "0x007115416ab6c266329a03b09a8aa39ac2ef7d9d",
			tokenOut:          "0x2260fac5e5542a773aa44fbcfedf7c193bc2c599",
			amountIn:          "60000000000000000",
			expectedAmountOut: "6183285",
		},
		{
			name:              "mTBILL -> USDC",
			pool:              "rv-ustb-mTBILL",
			tokenIn:           "0xdd629e5241cbc5919847783e6c96b2de4754e438",
			tokenOut:          "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			amountIn:          "11000000000000000000",
			expectedAmountOut: "11415587",
		},
		{
			name:              "mTBILL -> USDC",
			pool:              "rv-ustb-mTBILL",
			tokenIn:           "0xdd629e5241cbc5919847783e6c96b2de4754e438",
			tokenOut:          "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			amountIn:          "99000000000000000000000",
			expectedAmountOut: "102740286791",
		},
		{
			name:              "mHYPER -> USDC",
			pool:              "rv-swapper-ustb-mHYPER",
			tokenIn:           "0x9b5528528656dbc094765e2abb79f293c21191b9",
			tokenOut:          "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			amountIn:          "1000000000000000",
			expectedAmountOut: "1026",
		},
		{
			name:              "mBASIC -> USDC",
			pool:              "rv-swapper-rv-mBASIC",
			tokenIn:           "0x1c2757c1fef1038428b5bef062495ce94bbe92b2",
			tokenOut:          "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913",
			amountIn:          "1000000000000000000",
			expectedAmountOut: "1137627",
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

func TestCalcAmountOutShouldNotPanic(t *testing.T) {
	poolEntityStr := "{\"address\":\"0x71efa7af1686c5c04aa34a120a91cb4262679c44\",\"exchange\":\"midas\",\"type\":\"midas\",\"timestamp\":1769076893,\"reserves\":[\"200000000000000000000000000\",\"974403940643546\",\"815006435106971\",\"983391413640828\"],\"tokens\":[{\"address\":\"0x2fe058ccf29f123f9dd2aec0418aa66a877d8e50\",\"symbol\":\"msyrupUSDp\",\"decimals\":18,\"swappable\":true},{\"address\":\"0x356b8d89c1e1239cbbb9de4815c39a1474d5ba7d\",\"symbol\":\"syrupUSDT\",\"decimals\":6,\"swappable\":true},{\"address\":\"0xdac17f958d2ee523a2206206994597c13d831ec7\",\"symbol\":\"USDT\",\"decimals\":6,\"swappable\":true},{\"address\":\"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48\",\"symbol\":\"USDC\",\"decimals\":6,\"swappable\":true}],\"extra\":\"{\\\"mToken\\\":\\\"0x2fe058ccf29f123f9dd2aec0418aa66a877d8e50\\\",\\\"paymentTokens\\\":[\\\"0x356b8d89c1e1239cbbb9de4815c39a1474d5ba7d\\\",\\\"0xdac17f958d2ee523a2206206994597c13d831ec7\\\",\\\"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48\\\"],\\\"paused\\\":false,\\\"fnPaused\\\":false,\\\"tokensConfig\\\":[{\\\"fee\\\":\\\"0\\\",\\\"allowance\\\":\\\"974403940643546000000000000\\\",\\\"stable\\\":false},{\\\"fee\\\":\\\"0\\\",\\\"allowance\\\":\\\"815006435106971269000000000\\\",\\\"stable\\\":true},{\\\"fee\\\":\\\"0\\\",\\\"allowance\\\":\\\"983391413640828034137925831\\\",\\\"stable\\\":true}],\\\"instantDailyLimit\\\":\\\"200000000000000000000000000\\\",\\\"dailyLimits\\\":\\\"0\\\",\\\"instantFee\\\":\\\"50\\\",\\\"minAmount\\\":\\\"0\\\",\\\"mTokenRate\\\":\\\"1029712680000000000\\\",\\\"tokenRates\\\":[\\\"1113008000000000000\\\",\\\"998918290000000000\\\",\\\"999763980000000000\\\"],\\\"waivedFeeRestriction\\\":false,\\\"mTokenDecimals\\\":18,\\\"tokenBalances\\\":[\\\"1000000\\\",\\\"93541\\\",\\\"277886\\\"],\\\"mToken2Balance\\\":\\\"8172049868700887549284037\\\",\\\"swapperVaultType\\\":\\\"rvUstb\\\",\\\"mTbillRedemptionVault\\\":{\\\"mToken\\\":\\\"0xdd629e5241cbc5919847783e6c96b2de4754e438\\\",\\\"paymentTokens\\\":[\\\"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48\\\"],\\\"paused\\\":false,\\\"fnPaused\\\":false,\\\"tokensConfig\\\":[{\\\"fee\\\":\\\"0\\\",\\\"allowance\\\":\\\"0\\\",\\\"stable\\\":false},{\\\"fee\\\":\\\"0\\\",\\\"allowance\\\":\\\"0\\\",\\\"stable\\\":false},{\\\"fee\\\":\\\"0\\\",\\\"allowance\\\":\\\"42882504472679348375436984\\\",\\\"stable\\\":true}],\\\"instantDailyLimit\\\":\\\"10000000000000000000000000\\\",\\\"dailyLimits\\\":\\\"241400755984608339229\\\",\\\"instantFee\\\":\\\"7\\\",\\\"minAmount\\\":\\\"0\\\",\\\"mTokenRate\\\":\\\"1049570220000000000\\\",\\\"tokenRates\\\":[null,null,\\\"999763980000000000\\\"],\\\"waivedFeeRestriction\\\":false,\\\"mTokenDecimals\\\":18,\\\"tokenBalances\\\":[\\\"0\\\",\\\"0\\\",\\\"7\\\"],\\\"redemption\\\":{\\\"usdc\\\":\\\"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48\\\",\\\"redemptionFee\\\":\\\"0\\\",\\\"ustbBalance\\\":\\\"3019539914382\\\",\\\"chainlinkPrice\\\":{\\\"isBadData\\\":false,\\\"updatedAt\\\":\\\"1769076887\\\",\\\"price\\\":\\\"10966681\\\"},\\\"chainLinkFeedPrecision\\\":\\\"1000000\\\",\\\"superstateTokenPrecision\\\":\\\"1000000\\\"}}}\",\"staticExtra\":\"{\\\"isDv\\\":false,\\\"type\\\":\\\"rvSwapper\\\"}\"}"

	var entity entity.Pool
	err := json.Unmarshal([]byte(poolEntityStr), &entity)
	assert.NoError(t, err)

	sim, err := NewPoolSimulator(entity)
	assert.NoError(t, err)

	_, err = sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  "0x2fe058ccf29f123f9dd2aec0418aa66a877d8e50",
			Amount: bignum.NewBig("4859754219688795000000"),
		},
		TokenOut: "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
	})
	assert.Error(t, err)

	defer func() {
		r := recover()
		assert.Nil(t, r, "The code did panic")
	}()
}
