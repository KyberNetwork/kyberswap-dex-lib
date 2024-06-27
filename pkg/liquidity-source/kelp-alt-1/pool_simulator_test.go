package rsethalt1

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
	"github.com/stretchr/testify/assert"
)

func TestPoolSimulator_CalcAmountOut(t *testing.T) {
	t.Run("1. support wsteth", func(t *testing.T) {
		p := `{"address":"0x036676389e48133b63a802f8635ad39e752d375d","exchange":"kelp-rseth","type":"kelp-rseth-alt-1","timestamp":1719383899,"reserves":["10000000000000000000","10000000000000000000","10000000000000000000"],"tokens":[{"address":"0x4186bfc76e2e237523cbc30fd220fe055156b41f","decimals":18,"swappable":true},{"address":"0x82af49447d8a07e3bd95bd0d56f35241523fbab1","decimals":18,"swappable":true},{"address":"0x5979d7b546e38e414f7e9822514be443a4800529","decimals":18,"swappable":true}],"extra":"{\"priceByAsset\":{\"0x4186bfc76e2e237523cbc30fd220fe055156b41f\":1014725159825182726,\"0x5979d7b546e38e414f7e9822514be443a4800529\":1170845514514030655,\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\":null},\"feeBps\":10}"}`
		var pool entity.Pool
		err := json.Unmarshal([]byte(p), &pool)
		assert.Nil(t, err)

		testcases := []struct {
			expectedAmountOut string
			tokenIn           string
			amountIn          string
			tokenOut          string
			expectedError     error
		}{
			{
				tokenIn:           "0x5979d7b546e38e414f7e9822514be443a4800529", //wsteth
				amountIn:          "1000000000000000000",
				tokenOut:          "0x4186bfc76e2e237523cbc30fd220fe055156b41f",
				expectedAmountOut: "1152700963087412422",
				expectedError:     nil,
			},
			{
				tokenIn:           "0x82af49447d8a07e3bd95bd0d56f35241523fbab1", //weth
				amountIn:          "1000000000000000000",
				tokenOut:          "0x4186bfc76e2e237523cbc30fd220fe055156b41f",
				expectedAmountOut: "984503035454554154",
				expectedError:     nil,
			},
			{
				tokenIn:           "0x4186bfc76e2e237523cbc30fd220fe055156b41f", //wsteth
				amountIn:          "1000000000000000000",
				tokenOut:          "0x5979d7b546e38e414f7e9822514be443a4800529",
				expectedAmountOut: "0",
				expectedError:     ErrInvalidTokenOut,
			},
			{
				tokenIn:           "0x4186bfc76e2e237523cbc30fd220fe055156b41f", //weth
				amountIn:          "1000000000000000000",
				tokenOut:          "0x82af49447d8a07e3bd95bd0d56f35241523fbab1",
				expectedAmountOut: "0",
				expectedError:     ErrInvalidTokenOut,
			},
		}

		for _, tt := range testcases {
			simulator, err := NewPoolSimulator(pool)
			assert.Nil(t, err)
			amountIn, _ := new(big.Int).SetString(tt.amountIn, 10)
			result, err := testutil.MustConcurrentSafe[*poolpkg.CalcAmountOutResult](t, func() (any, error) {
				return simulator.CalcAmountOut(poolpkg.CalcAmountOutParams{
					TokenAmountIn: poolpkg.TokenAmount{
						Token:  tt.tokenIn,
						Amount: amountIn,
					},
					TokenOut: tt.tokenOut,
				})
			})

			// assert
			assert.Equal(t, tt.expectedError, err)
			if err == nil {
				assert.Equal(t, tt.expectedAmountOut, result.TokenAmountOut.Amount.String())
			}

		}
	})

	t.Run("1. No-support wsteth", func(t *testing.T) {
		p := `{"address":"0x036676389e48133b63a802f8635ad39e752d375d","exchange":"kelp-rseth","type":"kelp-rseth-alt-1","timestamp":1719383520,"reserves":["10000000000000000000","10000000000000000000"],"tokens":[{"address":"0x4186bfc76e2e237523cbc30fd220fe055156b41f","decimals":18,"swappable":true},{"address":"0xe5d7c2a44ffddf6b295a15c148167daaaf5cf34f","decimals":18,"swappable":true}],"extra":"{\"priceByAsset\":{\"0x4186bfc76e2e237523cbc30fd220fe055156b41f\":1014725159825182726,\"0xe5d7c2a44ffddf6b295a15c148167daaaf5cf34f\":null},\"feeBps\":0}"}`
		var pool entity.Pool
		err := json.Unmarshal([]byte(p), &pool)
		assert.Nil(t, err)

		testcases := []struct {
			expectedAmountOut string
			tokenIn           string
			amountIn          string
			tokenOut          string
			expectedError     error
		}{
			{
				tokenIn:           "0xe5d7c2a44ffddf6b295a15c148167daaaf5cf34f",
				amountIn:          "1000000000000000000",
				tokenOut:          "0x4186bfc76e2e237523cbc30fd220fe055156b41f",
				expectedAmountOut: "985488523978532686",
				expectedError:     nil,
			},
			{
				tokenIn:           "0x4186bfc76e2e237523cbc30fd220fe055156b41f",
				amountIn:          "1000000000000000000",
				tokenOut:          "0xe5d7c2a44ffddf6b295a15c148167daaaf5cf34f",
				expectedAmountOut: "0",
				expectedError:     ErrInvalidTokenOut,
			},
		}

		for _, tt := range testcases {
			simulator, err := NewPoolSimulator(pool)
			assert.Nil(t, err)
			amountIn, _ := new(big.Int).SetString(tt.amountIn, 10)
			result, err := testutil.MustConcurrentSafe[*poolpkg.CalcAmountOutResult](t, func() (any, error) {
				return simulator.CalcAmountOut(poolpkg.CalcAmountOutParams{
					TokenAmountIn: poolpkg.TokenAmount{
						Token:  tt.tokenIn,
						Amount: amountIn,
					},
					TokenOut: tt.tokenOut,
				})
			})
			// assert
			assert.Equal(t, tt.expectedError, err)
			if err == nil {
				assert.Equal(t, tt.expectedAmountOut, result.TokenAmountOut.Amount.String())
			}
		}
	})
}
