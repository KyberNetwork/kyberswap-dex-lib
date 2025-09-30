package quantamm

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

var (
	entityPool entity.Pool
	_          = json.Unmarshal([]byte(`{"address":"0x6b61d8680c4f9e560c8306807908553f95c749c5","exchange":"balancer-v3-quantamm","type":"balancer-v3-quantamm","timestamp":1751292261,"reserves":["132011160","2126502393706755897","86035501921"],"tokens":[{"address":"0x2260fac5e5542a773aa44fbcfedf7c193bc2c599","symbol":"WBTC","decimals":8,"swappable":true},{"address":"0x45804880de22913dafe09f4980848ece6ecbaf78","symbol":"PAXG","decimals":18,"swappable":true},{"address":"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48","symbol":"USDC","decimals":6,"swappable":true}],"extra":"{\"hook\":{},\"fee\":\"20000000000000000\",\"aggrFee\":\"500000000000000000\",\"balsE18\":[\"1320111600000000000\",\"2126502393706755897\",\"86035501921000000000000\"],\"decs\":[\"10000000000\",\"1\",\"1000000000000\"],\"rates\":[\"1000000000000000000\",\"1000000000000000000\",\"1000000000000000000\"],\"buffs\":[null,null,null],\"w\":[\"615205323000000000\",\"30053226000000000\",\"354826063000000000\"],\"m\":[\"115792089237316195423570985008687907853269984665640564039457584007595129639936\",\"0\",\"318000000000\",\"0\",\"0\"],\"u\":1751241623,\"i\":1751327723}","staticExtra":"{\"buffs\":[\"\",\"\",\"\"],\"mxTSR\":\"100000000000000000\"}","blockNumber":22817711}`),
		&entityPool)
	poolSim = lo.Must(NewPoolSimulator(pool.FactoryParams{EntityPool: entityPool}))
	tokens  = entityPool.Tokens
)

func TestCalcAmountOut(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                    string
		tokenInIdx, tokenOutIdx int
		amountIn                *big.Int
		expectedAmountOut       *big.Int
		expectedError           assert.ErrorAssertionFunc
	}{
		{
			name:              "0->1 ok",
			tokenInIdx:        0,
			tokenOutIdx:       1,
			amountIn:          big.NewInt(.2e6),
			expectedAmountOut: big.NewInt(60821945406164286),
			expectedError:     assert.NoError,
		},
		{
			name:              "0->2 ok",
			tokenInIdx:        0,
			tokenOutIdx:       2,
			amountIn:          big.NewInt(.2e6),
			expectedAmountOut: big.NewInt(196090530),
			expectedError:     assert.NoError,
		},
		{
			name:              "1->0 ok",
			tokenInIdx:        1,
			tokenOutIdx:       0,
			amountIn:          big.NewInt(.01e18),
			expectedAmountOut: big.NewInt(31028),
			expectedError:     assert.NoError,
		},
		{
			name:              "1->2 ok",
			tokenInIdx:        1,
			tokenOutIdx:       2,
			amountIn:          big.NewInt(.01e18),
			expectedAmountOut: big.NewInt(31099566),
			expectedError:     assert.NoError,
		},
		{
			name:              "2->0 ok",
			tokenInIdx:        2,
			tokenOutIdx:       0,
			amountIn:          big.NewInt(1e8),
			expectedAmountOut: big.NewInt(97678),
			expectedError:     assert.NoError,
		},
		{
			name:              "2->1 ok",
			tokenInIdx:        2,
			tokenOutIdx:       1,
			amountIn:          big.NewInt(1e8),
			expectedAmountOut: big.NewInt(30565620652560064),
			expectedError:     assert.NoError,
		},
		{
			name:          "0->1 max trade size ratio exceeded",
			tokenInIdx:    0,
			tokenOutIdx:   1,
			amountIn:      big.NewInt(1e8),
			expectedError: assert.Error,
		},
		{
			name:          "1->2 max trade size ratio exceeded",
			tokenInIdx:    1,
			tokenOutIdx:   2,
			amountIn:      big.NewInt(1e18),
			expectedError: assert.Error,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountOutResult, error) {
				return poolSim.CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: pool.TokenAmount{
						Token:  tokens[tc.tokenInIdx].Address,
						Amount: tc.amountIn,
					},
					TokenOut: tokens[tc.tokenOutIdx].Address,
				})
			})
			tc.expectedError(t, err)
			if err == nil {
				assert.Equal(t, tc.expectedAmountOut, result.TokenAmountOut.Amount)
			}
		})
	}
}

func TestCalcAmountIn(t *testing.T) {
	t.Parallel()
	testutil.TestCalcAmountIn(t, poolSim)
}
