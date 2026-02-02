package carbon

import (
	"math/big"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/goccy/go-json"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	abis "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/carbon/abi"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	bignum "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulatorTestSuite struct {
	suite.Suite

	controller *abis.ControllerCaller

	pools map[string]string
	sims  map[string]*PoolSimulator
}

func (ts *PoolSimulatorTestSuite) SetupSuite() {
	rpcUrl := os.Getenv("ETHEREUM_RPC_ENDPOINT")
	if rpcUrl == "" {
		rpcUrl = "https://eth.drpc.org"
	}

	if client, err := ethclient.Dial(rpcUrl); err == nil {
		ts.controller, _ = abis.NewControllerCaller(common.HexToAddress("0xc537e898cd774e2dcba3b14ea6f34c93d5ea45e1"), client)
	}

	ts.pools = map[string]string{
		"c_USDC_USDT_24289442": `{
			"address": "0x2a687b7e028a51bd952ec5a05a05cebdc91ddbffa75f15beb038167cc5a66eaa",
			"timestamp": 1769074246,
			"reserves": ["27415888", "2234364500"],
			"tokens": [
				{ "address": "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "symbol": "USDC", "decimals": 6, "swappable": true },
				{ "address": "0xdac17f958d2ee523a2206206994597c13d831ec7", "symbol": "USDT", "decimals": 6, "swappable": true }
			],
			"extra": "{\"strategies\":[{\"id\":1020847100762815390390123822295304634371,\"orders\":[{\"y\":\"9608\",\"z\":\"120523080\",\"A\":149261415419,\"B\":281348398405087},{\"y\":\"120671900\",\"z\":\"120681180\",\"A\":149279219898,\"B\":281381958729918}]},{\"id\":1020847100762815390390123822295304634823,\"orders\":[{\"y\":\"32079\",\"z\":\"61964559\",\"A\":839262887691,\"B\":280481030188081},{\"y\":\"79866210\",\"z\":\"79866210\",\"A\":16901673043433,\"B\":264601449757659}]},{\"id\":1020847100762815390390123822295304635604,\"orders\":[{\"y\":\"9908\",\"z\":\"45214826\",\"A\":140779792850,\"B\":281348398405086},{\"y\":\"45242959\",\"z\":\"45242959\",\"A\":140772690331,\"B\":281334204020325}]},{\"id\":1020847100762815390390123822295304635802,\"orders\":[{\"y\":\"89973\",\"z\":\"200799\",\"A\":1393402436374,\"B\":280796803593872},{\"y\":\"110694\",\"z\":\"200755\",\"A\":1393157891298,\"B\":280747523160770}]},{\"id\":1020847100762815390390123822295304636370,\"orders\":[{\"y\":\"179687\",\"z\":\"1010865468\",\"A\":42189950296,\"B\":281390572197775},{\"y\":\"1011981287\",\"z\":\"1012152795\",\"A\":42206825434,\"B\":422226538111202}]},{\"id\":1020847100762815390390123822295304636371,\"orders\":[{\"y\":\"27094633\",\"z\":\"1002262282\",\"A\":52822198354,\"B\":281348398405087},{\"y\":\"976491450\",\"z\":\"1003404912\",\"A\":87849539609,\"B\":281460902609959}]}],\"tradingFeePpm\":10}",
			"staticExtra": "{\"t0\":\"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48\",\"t1\":\"0xdac17f958d2ee523a2206206994597c13d831ec7\",\"c\":\"0xc537e898cd774e2dcba3b14ea6f34c93d5ea45e1\"}",
			"blockNumber": 24289442
		}`,
		"c_USDC_USDT_24290118": `{
			"address": "0x2a687b7e028a51bd952ec5a05a05cebdc91ddbffa75f15beb038167cc5a66eaa",
			"timestamp": 1769082426,
			"reserves": ["19511801", "2242309885"],
			"tokens": [
				{ "address": "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "symbol": "USDC", "decimals": 6, "swappable": true },
				{ "address": "0xdac17f958d2ee523a2206206994597c13d831ec7", "symbol": "USDT", "decimals": 6, "swappable": true }
			],
			"extra": "{\"strategies\":[{\"id\":1020847100762815390390123822295304634371,\"orders\":[{\"y\":\"9608\",\"z\":\"120523080\",\"A\":149261415419,\"B\":281348398405087},{\"y\":\"120671900\",\"z\":\"120681180\",\"A\":149279219898,\"B\":281381958729918}]},{\"id\":1020847100762815390390123822295304634823,\"orders\":[{\"y\":\"32079\",\"z\":\"61964559\",\"A\":839262887691,\"B\":280481030188081},{\"y\":\"79866210\",\"z\":\"79866210\",\"A\":16901673043433,\"B\":264601449757659}]},{\"id\":1020847100762815390390123822295304635604,\"orders\":[{\"y\":\"9908\",\"z\":\"45214826\",\"A\":140779792850,\"B\":281348398405086},{\"y\":\"45242959\",\"z\":\"45242959\",\"A\":140772690331,\"B\":281334204020325}]},{\"id\":1020847100762815390390123822295304635802,\"orders\":[{\"y\":\"89973\",\"z\":\"200799\",\"A\":1393402436374,\"B\":280796803593872},{\"y\":\"110694\",\"z\":\"200755\",\"A\":1393157891298,\"B\":280747523160770}]},{\"id\":1020847100762815390390123822295304636370,\"orders\":[{\"y\":\"7344622\",\"z\":\"1010865468\",\"A\":42189950296,\"B\":281390572197775},{\"y\":\"1004818376\",\"z\":\"1012152795\",\"A\":42206825434,\"B\":422226538111202}]},{\"id\":1020847100762815390390123822295304636371,\"orders\":[{\"y\":\"12025611\",\"z\":\"1002262282\",\"A\":52822198354,\"B\":281348398405087},{\"y\":\"991599746\",\"z\":\"1003404912\",\"A\":87849539609,\"B\":281460902609959}]}],\"tradingFeePpm\":10}",
			"staticExtra": "{\"t0\":\"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48\",\"t1\":\"0xdac17f958d2ee523a2206206994597c13d831ec7\",\"c\":\"0xc537e898cd774e2dcba3b14ea6f34c93d5ea45e1\"}",
			"blockNumber": 24290250
		}`,
		"c_USDC_USDT_24368818": `{
			"address":"0x2a687b7e028a51bd952ec5a05a05cebdc91ddbffa75f15beb038167cc5a66eaa",
			"timestamp":1770031012,
			"reserves":["116863386","1599817364"],
			"tokens":[
				{"address":"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48","symbol":"USDC","decimals":6,"swappable":true},
				{"address":"0xdac17f958d2ee523a2206206994597c13d831ec7","symbol":"USDT","decimals":6,"swappable":true}
			],
			"extra":"{\"strategies\":[{\"id\":1020847100762815390390123822295304634371,\"orders\":[{\"y\":\"1833556\",\"z\":\"120523080\",\"A\":149261415419,\"B\":281348398405087},{\"y\":\"118902651\",\"z\":\"120690051\",\"A\":149279219898,\"B\":281381958729918}]},{\"id\":1020847100762815390390123822295304634823,\"orders\":[{\"y\":\"1817\",\"z\":\"61964559\",\"A\":839262887691,\"B\":280481030188081},{\"y\":\"79896687\",\"z\":\"79896687\",\"A\":16901673043433,\"B\":264601449757659}]},{\"id\":1020847100762815390390123822295304635604,\"orders\":[{\"y\":\"28\",\"z\":\"45214826\",\"A\":140779792850,\"B\":281348398405086},{\"y\":\"45252851\",\"z\":\"45252857\",\"A\":140772690331,\"B\":281334204020325}]},{\"id\":1020847100762815390390123822295304635802,\"orders\":[{\"y\":\"56556\",\"z\":\"200799\",\"A\":1393402436374,\"B\":280796803593872},{\"y\":\"144177\",\"z\":\"200755\",\"A\":1393157891298,\"B\":280747523160770}]},{\"id\":1020847100762815390390123822295304636370,\"orders\":[{\"y\":\"24777513\",\"z\":\"1011713445\",\"A\":70307919111,\"B\":281362454228961},{\"y\":\"989539395\",\"z\":\"1014303000\",\"A\":70343069908,\"B\":422226538111202}]},{\"id\":1020847100762815390390123822295304636491,\"orders\":[{\"y\":\"90193916\",\"z\":\"455851290\",\"A\":231707608326,\"B\":281291138249378},{\"y\":\"366081603\",\"z\":\"456026125\",\"A\":231808026756,\"B\":281413045402586}]}],\"tradingFeePpm\":10}",
			"staticExtra":"{\"t0\":\"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48\",\"t1\":\"0xdac17f958d2ee523a2206206994597c13d831ec7\",\"c\":\"0xC537e898CD774e2dCBa3B14Ea6f34C93d5eA45e1\"}",
			"blockNumber":24368818
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

//nolint:unused
func (ts *PoolSimulatorTestSuite) calcTargetAmount(
	t *testing.T,
	blockNumber uint64,
	tokenIn, tokenOut string,
	actions []TradeAction,
) *big.Int {
	targetAmount, err := ts.controller.CalculateTradeTargetAmount(
		&bind.CallOpts{
			Context:     t.Context(),
			BlockNumber: big.NewInt(int64(blockNumber)),
		},
		common.HexToAddress(tokenIn), common.HexToAddress(tokenOut),
		lo.Map(actions, func(e TradeAction, _ int) abis.TradeAction {
			return abis.TradeAction{
				StrategyId: bignum.NewBig(e.StrategyId),
				Amount:     e.SourceAmount.ToBig(),
			}
		}),
	)
	require.NoError(t, err)

	return targetAmount
}

func (ts *PoolSimulatorTestSuite) TestCalcAmountOut() {
	ts.T().Parallel()

	testCases := []struct {
		pool     string
		tokenIn  string
		tokenOut string
		amountIn string

		matchType             MatchType
		expectedAmountOut     string
		expectedFastAmountOut string
		expectedErr           error
	}{
		{
			pool:              "c_USDC_USDT_24289442",
			tokenIn:           "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			tokenOut:          "0xdac17f958d2ee523a2206206994597c13d831ec7",
			amountIn:          "100000000",
			expectedAmountOut: "100047506",
		},
		{
			pool:              "c_USDC_USDT_24289442",
			tokenIn:           "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			tokenOut:          "0xdac17f958d2ee523a2206206994597c13d831ec7",
			amountIn:          "10000000000",
			expectedAmountOut: "2234342156",
		},
		{
			pool:              "c_USDC_USDT_24289442",
			tokenIn:           "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			tokenOut:          "0xdac17f958d2ee523a2206206994597c13d831ec7",
			amountIn:          "1000000000000000000",
			expectedAmountOut: "2234342156",
		},
		{
			pool:              "c_USDC_USDT_24289442",
			tokenOut:          "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			tokenIn:           "0xdac17f958d2ee523a2206206994597c13d831ec7",
			amountIn:          "1000000000000000000",
			expectedAmountOut: "27415613",
		},
		{
			pool:              "c_USDC_USDT_24289442",
			tokenOut:          "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			tokenIn:           "0xdac17f958d2ee523a2206206994597c13d831ec7",
			amountIn:          "10",
			expectedAmountOut: "8",
		},
		{
			pool:                  "c_USDC_USDT_24289442",
			tokenOut:              "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			tokenIn:               "0xdac17f958d2ee523a2206206994597c13d831ec7",
			amountIn:              "100000",
			expectedAmountOut:     "99939",
			expectedFastAmountOut: "99632",
		},
		{
			pool:              "c_USDC_USDT_24289442",
			tokenOut:          "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			tokenIn:           "0xdac17f958d2ee523a2206206994597c13d831ec7",
			amountIn:          "10000000000",
			expectedAmountOut: "27415613",
		},
		{
			pool:        "c_USDC_USDT_24289442",
			tokenOut:    "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			tokenIn:     "0xdac17f958d2ee523a2206206994597c13d831ec7",
			amountIn:    "1",
			expectedErr: ErrInvalidSwap,
		},
		{
			pool:              "c_USDC_USDT_24290118",
			tokenOut:          "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			tokenIn:           "0xdac17f958d2ee523a2206206994597c13d831ec7",
			amountIn:          "100000",
			expectedAmountOut: "99939",
		},
		{
			pool:                  "c_USDC_USDT_24290118",
			tokenOut:              "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			tokenIn:               "0xdac17f958d2ee523a2206206994597c13d831ec7",
			amountIn:              "10000000",
			expectedAmountOut:     "9993127",
			expectedFastAmountOut: "9993127",
			matchType:             MatchTypeFast,
		},
		{
			pool:              "c_USDC_USDT_24368818",
			tokenOut:          "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			tokenIn:           "0xdac17f958d2ee523a2206206994597c13d831ec7",
			amountIn:          "100000000",
			expectedAmountOut: "99896069",
		},
		{
			pool:              "c_USDC_USDT_24368818",
			tokenOut:          "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			tokenIn:           "0xdac17f958d2ee523a2206206994597c13d831ec7",
			amountIn:          "1000000000000",
			expectedAmountOut: "116862217",
		},
	}

	for _, tc := range testCases {
		ts.T().Run(tc.pool, func(t *testing.T) {
			sim := ts.sims[tc.pool].CloneState().(*PoolSimulator)

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

			swapInfo, ok := res.SwapInfo.(SwapInfo)
			require.True(t, ok)
			require.NotNil(t, swapInfo.TradeActions)
		})
	}
}

func TestPoolSimulatorTestSuite(t *testing.T) {
	t.Parallel()

	suite.Run(t, new(PoolSimulatorTestSuite))
}
