package smoothy

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
		"p_24331768": `{
			"address": "0xe5859f4efc09027a9b718781dcb2c6910cac6e91",
			"reserves": ["434976343453","214240024006","26829617672896784070529","23921400807352480328938","17833627961647430344567","33152342565810055129800","39513361218539079180100","1017"],
			"tokens": [{"address":"0xdac17f958d2ee523a2206206994597c13d831ec7","symbol":"USDT","decimals":6,"swappable":true},{"address":"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48","symbol":"USDC","decimals":6,"swappable":true},{"address":"0x6b175474e89094c44da98b954eedeac495271d0f","symbol":"DAI","decimals":18,"swappable":true},{"address":"0x0000000000085d4780b73119b644ae5ecd22b376","symbol":"TUSD","decimals":18,"swappable":true},{"address":"0x57ab1ec28d129707052df4df418d58a2d46d5f51","symbol":"sUSD","decimals":18,"swappable":true},{"address":"0x4fabb145d64652a948d72533023f6e7a623c7c53","symbol":"BUSD","decimals":18,"swappable":true},{"address":"0x8e870d67f660d95d5be530380d0ec0bd388289e1","symbol":"USDP","decimals":18,"swappable":true},{"address":"0x056fd409e1d7a124bd7017459dfea2f387b6d5cd","symbol":"GUSD","decimals":2,"swappable":true}],
			"extra": "{\"sF\":\"400000000000000\",\"aFP\":\"0\",\"tB\":\"790476877772245829053934\",\"tI\":[{\"sW\":\"550000000000000000\",\"hW\":\"1000000000000000000\",\"dM\":12,\"b\":\"434976343453\"},{\"sW\":\"550000000000000000\",\"hW\":\"1000000000000000000\",\"dM\":12,\"b\":\"214240024006\"},{\"sW\":\"550000000000000000\",\"hW\":\"1000000000000000000\",\"dM\":0,\"b\":\"26829617672896784070529\"},{\"sW\":\"30000000000000000\",\"hW\":\"60000000000000000\",\"dM\":0,\"b\":\"23921400807352480328938\"},{\"sW\":\"20000000000000000\",\"hW\":\"40000000000000000\",\"dM\":0,\"b\":\"17833627961647430344567\"},{\"sW\":\"50000000000000000\",\"hW\":\"100000000000000000\",\"dM\":0,\"b\":\"33152342565810055129800\"},{\"sW\":\"50000000000000000\",\"hW\":\"100000000000000000\",\"dM\":0,\"b\":\"39513361218539079180100\"},{\"sW\":\"50000000000000000\",\"hW\":\"100000000000000000\",\"dM\":16,\"b\":\"1017\"}]}",
			"blockNumber": 24331768
		}`,
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
		pool     string
		tokenIn  string
		tokenOut string
		amountIn string

		expectedAmountOut string
		expectedErr       error
	}{
		{
			pool:              "p_24331768",
			tokenIn:           "0xdac17f958d2ee523a2206206994597c13d831ec7",
			tokenOut:          "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			amountIn:          "10000000000",
			expectedAmountOut: "9722135059",
		},
		{
			pool:              "p_24331768",
			tokenIn:           "0xdac17f958d2ee523a2206206994597c13d831ec7",
			tokenOut:          "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			amountIn:          "10000000000000",
			expectedAmountOut: "123703408910",
		},
		{
			pool:              "p_24331768",
			tokenIn:           "0xdac17f958d2ee523a2206206994597c13d831ec7",
			tokenOut:          "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			amountIn:          "10000000000000000",
			expectedAmountOut: "588727205",
		},
		{
			pool:              "p_24331768",
			tokenIn:           "0xdac17f958d2ee523a2206206994597c13d831ec7",
			tokenOut:          "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			amountIn:          "1000000000000000000",
			expectedAmountOut: "8765402",
		},
		{
			pool:        "p_24331768",
			tokenIn:     "0xdac17f958d2ee523a2206206994597c13d831ec7",
			tokenOut:    "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			amountIn:    "10000000000000000000000",
			expectedErr: ErrCannotFindProperResolutionOfFX,
		},
		{
			pool:        "p_24331768",
			tokenOut:    "0xdac17f958d2ee523a2206206994597c13d831ec7",
			tokenIn:     "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			amountIn:    "10000000000000000000000",
			expectedErr: ErrCannotFindProperResolutionOfFX,
		},
		{
			pool:              "p_24331768",
			tokenIn:           "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			tokenOut:          "0xdac17f958d2ee523a2206206994597c13d831ec7",
			amountIn:          "100000000000",
			expectedAmountOut: "99960000000",
		},
		{
			pool:              "p_24331768",
			tokenIn:           "0xdac17f958d2ee523a2206206994597c13d831ec7",
			tokenOut:          "0x57ab1ec28d129707052df4df418d58a2d46d5f51",
			amountIn:          "1000000000",
			expectedAmountOut: "995735783576143089541",
		},
		{
			pool:              "p_24331768",
			tokenIn:           "0xdac17f958d2ee523a2206206994597c13d831ec7",
			tokenOut:          "0x57ab1ec28d129707052df4df418d58a2d46d5f51",
			amountIn:          "1000000000",
			expectedAmountOut: "995735783576143089541",
		},
		{
			pool:              "p_24331768",
			tokenIn:           "0x4fabb145d64652a948d72533023f6e7a623c7c53",
			tokenOut:          "0x57ab1ec28d129707052df4df418d58a2d46d5f51",
			amountIn:          "10000000000",
			expectedAmountOut: "9996000000",
		},
	}

	for _, tc := range testCases {
		ts.T().Run(tc.pool, func(t *testing.T) {
			sim := ts.sims[tc.pool].CloneState()

			res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{
					Token:  tc.tokenIn,
					Amount: bignum.NewBig(tc.amountIn),
				},
				TokenOut: tc.tokenOut,
			})

			if tc.expectedErr != nil {
				require.ErrorContains(t, err, tc.expectedErr.Error())
				return
			}

			require.NotNil(t, res)
			require.Equal(t, tc.expectedAmountOut, res.TokenAmountOut.Amount.String())

			sim.UpdateBalance(pool.UpdateBalanceParams{
				TokenAmountIn: pool.TokenAmount{
					Token:  tc.tokenIn,
					Amount: bignum.NewBig(tc.amountIn),
				},
				TokenAmountOut: pool.TokenAmount{
					Token:  tc.tokenOut,
					Amount: bignum.NewBig(tc.expectedAmountOut),
				},
				SwapInfo: res.SwapInfo,
			})
		})
	}
}

func TestPoolSimulatorTestSuite(t *testing.T) {
	t.Parallel()

	suite.Run(t, new(PoolSimulatorTestSuite))
}
