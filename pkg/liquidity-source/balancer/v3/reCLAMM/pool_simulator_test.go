package reclamm

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
	// In range so pool does not re-adjust
	entityPoolInRange entity.Pool
	_                 = json.Unmarshal([]byte(`{"address":"0x12c2de9522f377b86828f6af01f58c046f814d3c","exchange":"balancer-v3-reclamm","type":"balancer-v3-reclamm","timestamp":1752054103,"reserves":["3231573612000000000000","6289473995000000000000"],"tokens":[{"address":"0x60a3E35Cc302bFA44Cb288Bc5a4F316Fdb1adb42","symbol":"EURC","decimals":6,"swappable":true},{"address":"0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913","symbol":"USDC","decimals":6,"swappable":true}],"extra":"{\"hook\":{},\"fee\":\"250000000000000\",\"aggrFee\":\"500000000000000000\",\"balsE18\":[\"3231573612000000000000\",\"6289473995000000000000\"],\"decs\":[\"1000000000000\",\"1000000000000\"],\"rates\":[\"1000000000000000000\",\"1000000000000000000\"],\"buffs\":[null,null],\"lastVirtualBalances\":[\"362594117476852465718411\",\"422163369011063269890448\"],\"centerednessMargin\":500000000000000000,\"dailyPriceShiftBase\":999999197747274347,\"startFourthRootPriceRatio\":1011900417200324692,\"endFourthRootPriceRatio\":1011900417200324692,\"priceRatioUpdateStartTime\":1751988959,\"priceRatioUpdateEndTime\":1751988959,\"lastTimestamp\":1752054083,\"currentTimestamp\":1752054103}","staticExtra":"{\"buffs\":[\"\",\"\"],\"hook\": \"0x9d1fcf346ea1b073de4d5834e25572cc6ad71f4d\",\"hookT\": \"RECLAMM\"}","blockNumber":32632378}`),
		&entityPoolInRange)

	// Out of range so pool will re-center
	entityPoolOutOfRange entity.Pool
	_                    = json.Unmarshal([]byte(`{"address":"0x3De1E230193F39DF0c122A345a928d909034c1a1","exchange":"balancer-v3-reclamm","type":"balancer-v3-reclamm","timestamp":1752072827,"reserves":["119997500000000","116125000000000000"],"tokens":[{"address":"0x4200000000000000000000000000000000000006","symbol":"WETH","decimals":18,"swappable":true},{"address":"0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913","symbol":"USDC","decimals":6,"swappable":true}],"extra":"{\"hook\":{},\"fee\":\"250000000000000\",\"aggrFee\":\"500000000000000000\",\"balsE18\":[\"119997500000000\",\"116125000000000000\"],\"decs\":[\"1\",\"1000000000000\"],\"rates\":[\"1000000000000000000\",\"1000000000000000000\"],\"buffs\":[null,null],\"lastVirtualBalances\":[\"2088831129392021\",\"5414869164940852258\"],\"centerednessMargin\":500000000000000000,\"dailyPriceShiftBase\":999999197747274347,\"startFourthRootPriceRatio\":1039289877625411769,\"endFourthRootPriceRatio\":1039289877625411769,\"priceRatioUpdateStartTime\":1752072483,\"priceRatioUpdateEndTime\":1752072483,\"lastTimestamp\":1752072827,\"currentTimestamp\":1752072837}","staticExtra":"{\"buffs\":[\"\",\"\"],\"hook\": \"0x9d1fcf346ea1b073de4d5834e25572cc6ad71f4d\",\"hookT\": \"RECLAMM\"}","blockNumber":32641745}`),
		&entityPoolOutOfRange)

	poolSimInRange    = lo.Must(NewPoolSimulator(entityPoolInRange))
	poolSimOutOfRange = lo.Must(NewPoolSimulator(entityPoolOutOfRange))
)

func TestCalcAmountOutInRange(t *testing.T) {
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
			amountIn:          big.NewInt(20000000),
			expectedAmountOut: big.NewInt(23416743),
			expectedError:     assert.NoError,
		},
		{
			name:              "1->0 ok",
			tokenInIdx:        1,
			tokenOutIdx:       0,
			amountIn:          big.NewInt(100000),
			expectedAmountOut: big.NewInt(85361),
			expectedError:     assert.NoError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountOutResult, error) {
				return poolSimInRange.CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: pool.TokenAmount{
						Token:  entityPoolInRange.Tokens[tc.tokenInIdx].Address,
						Amount: tc.amountIn,
					},
					TokenOut: entityPoolInRange.Tokens[tc.tokenOutIdx].Address,
				})
			})
			tc.expectedError(t, err)
			if err == nil {
				assert.Equal(t, tc.expectedAmountOut, result.TokenAmountOut.Amount)
			}
		})
	}
}

func TestCalcAmountOutOutOfRange(t *testing.T) {
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
			amountIn:          big.NewInt(2000000000000),
			expectedAmountOut: big.NewInt(5002),
			expectedError:     assert.NoError,
		},
		{
			name:              "1->0 ok",
			tokenInIdx:        1,
			tokenOutIdx:       0,
			amountIn:          big.NewInt(100000),
			expectedAmountOut: big.NewInt(39217048938964),
			expectedError:     assert.NoError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountOutResult, error) {
				return poolSimOutOfRange.CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: pool.TokenAmount{
						Token:  entityPoolOutOfRange.Tokens[tc.tokenInIdx].Address,
						Amount: tc.amountIn,
					},
					TokenOut: entityPoolOutOfRange.Tokens[tc.tokenOutIdx].Address,
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
	testutil.TestCalcAmountIn(t, poolSimInRange)
}
