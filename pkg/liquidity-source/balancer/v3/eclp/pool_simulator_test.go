package eclp

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer/v3/base"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer/v3/math"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer/v3/shared"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer/v3/vault"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

func newPool(poolStr string) *base.PoolSimulator {
	var poolE entity.Pool
	_ = json.Unmarshal([]byte(poolStr), &poolE)
	poolSim, _ := NewPoolSimulator(poolE)
	return poolSim
}

func pool1() *base.PoolSimulator {
	return newPool(`{"address":"0x698a72bc8bc2eeb0f0f9a77cef0a2859399dc469","exchange":"balancer-v3-eclp","type":"balancer-v3-eclp","timestamp":1757384774,"reserves":["486371653149689446958530","149785277730"],"tokens":[{"address":"0x6440f144b7e50d6a8439336510312d2f54beb01d","symbol":"BOLD","decimals":18,"swappable":true},{"address":"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48","symbol":"USDC","decimals":6,"swappable":true}],"extra":"{\"hook\":{},\"fee\":\"100000000000000\",\"aggrFee\":\"500000000000000000\",\"balsE18\":[\"486371653149689446958530\",\"171294919510944911667542\"],\"decs\":[\"1\",\"1000000000000\"],\"rates\":[\"1000000000000000000\",\"1143603177207560876\"],\"buffs\":[null,{\"dRate\":[\"874429\",\"874429190063\",\"874429190063803670\",\"874429190063803670814306\",\"874429190063803670814306611749\"],\"rRate\":[\"1143603\",\"1143603177207\",\"1143603177207560876\",\"1143603177207560876758075\",\"1143603177207560876758075329000\"]}],\"eclp\":{\"p\":{\"a\":\"988000000000000000\",\"b\":\"1050000000000000000\",\"c\":\"707283579973402312\",\"s\":\"706929938183415611\",\"l\":\"400000000000000000000\"},\"d\":{\"tA\":{\"x\":\"-91797936273223331128703595770556872759\",\"y\":\"39662815028401957930226687279861600182\"},\"tB\":{\"x\":\"99489241886312064726218464511911285812\",\"y\":\"10094094753215317810314727669349055050\"},\"u\":\"95643577118339844311670045560972199867\",\"v\":\"24885848918922220631984450912496399132\",\"w\":\"-14784358288624049362599522643509652296\",\"z\":\"3893486556531390423310667555251324042\",\"DSq\":\"100000000000000000108254687936544866500\"}}}","staticExtra":"{\"buffs\":[\"\",\"0xd4fa2d31b7968e448877f69a96de69f5de8cd23e\"]}","blockNumber":23322539}`)
}

func pool2() *base.PoolSimulator {
	return newPool(`{"address":"0x9481d2483d198913281986d36f51dcfb8c051086","exchange":"balancer-v3-eclp","type":"balancer-v3-eclp","timestamp":1757384774,"reserves":["76107112806008791524046","87131314055880678322215"],"tokens":[{"address":"0x66a1e37c9b0eaddca17d3662d6c05f4decf3e110","symbol":"USR","decimals":18,"swappable":true},{"address":"0x4956b52ae2ff65d74ca2d61207523288e4528f96","symbol":"RLP","decimals":18,"swappable":true}],"extra":"{\"hook\":{},\"fee\":\"500000000000000\",\"aggrFee\":\"500000000000000000\",\"balsE18\":[\"84266153284586977816880\",\"107696592241375625832870\"],\"decs\":[\"1\",\"1\"],\"rates\":[\"1107204703709822186\",\"1236026260000000000\"],\"buffs\":[{\"dRate\":[\"903175\",\"903175353798\",\"903175353798064652\",\"903175353798064652493580\",\"903175353798064652493580767627\"],\"rRate\":[\"1107204\",\"1107204703709\",\"1107204703709822186\",\"1107204703709822186538966\",\"1107204703709822186538966260010\"]},null],\"eclp\":{\"p\":{\"a\":\"993048659384309831\",\"b\":\"1020408163265306122\",\"c\":\"706929938183415611\",\"s\":\"707283579973402312\",\"l\":\"500000000000000000000\"},\"d\":{\"tA\":{\"x\":\"-88171788866808888175974353053889462663\",\"y\":\"47177702869330933649503670940336746128\"},\"tB\":{\"x\":\"98000611417254585268303732698148417703\",\"y\":\"19896737467340474486500758816991514700\"},\"u\":\"93086188500437377088380085951534510449\",\"v\":\"33530398221925082107251188866405422599\",\"w\":\"-13640480995082148670537973916655974265\",\"z\":\"4867856539378270148942739543022400736\",\"DSq\":\"100000000000000000108254687936544866500\"}}}","staticExtra":"{\"buffs\":[\"0x1202f5c7b4b9e47a1a484e8b270be34dbbc75055\",\"\"]}","blockNumber":23322539}`)
}

func TestCalcAmountOut(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		name              string
		pool              *base.PoolSimulator
		tokenAmountIn     pool.TokenAmount
		tokenOut          string
		expectedAmountOut string
		expectedError     error
	}{
		{
			name: "1. 0->1 ok",
			pool: pool1(),
			tokenAmountIn: pool.TokenAmount{
				Token:  "0x6440f144b7e50d6a8439336510312d2f54beb01d",
				Amount: big.NewInt(1e18),
			},
			tokenOut:          "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			expectedAmountOut: "997094",
			expectedError:     nil,
		},
		{
			name: "1. 1->0 ok",
			pool: pool1(),
			tokenAmountIn: pool.TokenAmount{
				Token:  "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
				Amount: big.NewInt(1e6),
			},
			tokenOut:          "0x6440f144b7e50d6a8439336510312d2f54beb01d",
			expectedAmountOut: "1002711877784976448",
			expectedError:     nil,
		},
		{
			name: "1. 0->1 AmountIn is too small",
			pool: pool1(),
			tokenAmountIn: pool.TokenAmount{
				Token:  "0x6440f144b7e50d6a8439336510312d2f54beb01d",
				Amount: big.NewInt(1e5),
			},
			tokenOut:          "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			expectedAmountOut: "0",
			expectedError:     vault.ErrAmountInTooSmall,
		},
		{
			name: "1. 0->1 AmountIn is too small for buffering",
			pool: pool1(),
			tokenAmountIn: pool.TokenAmount{
				Token:  "0x6440f144b7e50d6a8439336510312d2f54beb01d",
				Amount: big.NewInt(1e10),
			},
			tokenOut:          "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			expectedAmountOut: "0",
			expectedError:     shared.ErrWrapAmountTooSmall,
		},
		{
			name: "1. 0->1 ErrAssetBoundsExceeded",
			pool: pool1(),
			tokenAmountIn: pool.TokenAmount{
				Token:  "0x6440f144b7e50d6a8439336510312d2f54beb01d",
				Amount: bignumber.NewBig("1000000000000000000000000"), // 1e24
			},
			tokenOut:          "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			expectedAmountOut: "0",
			expectedError:     math.ErrAssetBoundsExceeded,
		},
		{
			name: "2. 0->1 ok",
			pool: pool2(),
			tokenAmountIn: pool.TokenAmount{
				Token:  "0x66a1e37c9b0eaddca17d3662d6c05f4decf3e110",
				Amount: big.NewInt(1e18),
			},
			tokenOut:          "0x4956b52ae2ff65d74ca2d61207523288e4528f96",
			expectedAmountOut: "809576077144881298",
			expectedError:     nil,
		},
		{
			name: "2. 1->0 ok",
			pool: pool2(),
			tokenAmountIn: pool.TokenAmount{
				Token:  "0x4956b52ae2ff65d74ca2d61207523288e4528f96",
				Amount: big.NewInt(1e18),
			},
			tokenOut:          "0x66a1e37c9b0eaddca17d3662d6c05f4decf3e110",
			expectedAmountOut: "1233979403784056964",
			expectedError:     nil,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountOutResult, error) {
				return tc.pool.CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: tc.tokenAmountIn,
					TokenOut:      tc.tokenOut,
				})
			})

			assert.Equal(t, tc.expectedError, err)
			if err == nil {
				assert.Equal(t, tc.expectedAmountOut, result.TokenAmountOut.Amount.String())
			}
		})
	}
}

func TestPoolSimulator_CalcAmountIn(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		name             string
		pool             *base.PoolSimulator
		tokenAmountOut   pool.TokenAmount
		tokenIn          string
		expectedAmountIn string
		expectedError    error
	}{
		{
			name: "1. 0->1",
			pool: pool1(),
			tokenAmountOut: pool.TokenAmount{
				Token:  "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
				Amount: big.NewInt(997094),
			},
			tokenIn:          "0x6440f144b7e50d6a8439336510312d2f54beb01d",
			expectedAmountIn: "999996963127157068",
			expectedError:    nil,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountInResult, error) {
				return tc.pool.CalcAmountIn(pool.CalcAmountInParams{
					TokenAmountOut: tc.tokenAmountOut,
					TokenIn:        tc.tokenIn,
				})
			})

			assert.Equal(t, tc.expectedError, err)
			if err == nil {
				assert.Equal(t, tc.expectedAmountIn, result.TokenAmountIn.Amount.String())
			}
		})
	}
}
