package eulerswap

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

func TestCalcAmountOut(t *testing.T) {
	poolStr := `{"address":"0x2bfed8dbeb8e6226a15300ac77ee9130e52410fe","exchange":"euler-swap","type":"euler-swap","timestamp":1743652017,"reserves":["1987079202","2012923319"],"tokens":[{"address":"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48","symbol":"USDC","decimals":6,"swappable":true},{"address":"0xdac17f958d2ee523a2206206994597c13d831ec7","symbol":"USDT","decimals":6,"swappable":true}],"extra":"{\"p\":1,\"v\":[{\"Cash\":\"357461819\",\"Debt\":\"0\",\"MaxDeposit\":\"591856746\",\"MaxWithdraw\":\"900000000\",\"TotalBorrows\":\"50681434\",\"EulerAccountAssets\":\"208103422\"},{\"Cash\":\"360947656\",\"Debt\":\"0\",\"MaxDeposit\":\"608026617\",\"MaxWithdraw\":\"900000000\",\"TotalBorrows\":\"31025726\",\"EulerAccountAssets\":\"191956291\"}]}","staticExtra":"{\"v0\":\"0xa66957e58b60d6b92b850c8773a9ff9b0Ba96A65\",\"v1\":\"0x4212E01c7C8e1c21DEa6030C74aE2084f5337BD1\",\"ea\":\"0x603765f9e9B8E3CBACbdf7A6e963B7EAD77AE86f\",\"fm\":\"1000000000000000000\",\"er0\":\"2000000000\",\"er1\":\"2000000000\",\"px\":\"1000000000000000000\",\"py\":\"1000000000000000000\",\"cx\":\"970000000000000000\",\"cy\":\"970000000000000000\",\"p\":false}","blockNumber":22185795}`

	var pool entity.Pool
	err := json.Unmarshal([]byte(poolStr), &pool)
	assert.Nil(t, err)

	s, err := NewPoolSimulator(pool)
	assert.Nil(t, err)

	t.Run("swap USDC -> USDT", func(t *testing.T) {
		amountIn, _ := new(big.Int).SetString("1000000", 10)

		tokenAmountIn := poolpkg.TokenAmount{
			Token:  "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			Amount: amountIn,
		}
		tokenOut := "0xdac17f958d2ee523a2206206994597c13d831ec7"

		expectedAmountOut := "1000376"

		result, err := testutil.MustConcurrentSafe(t, func() (*poolpkg.CalcAmountOutResult, error) {
			return s.CalcAmountOut(poolpkg.CalcAmountOutParams{
				TokenAmountIn: tokenAmountIn,
				TokenOut:      tokenOut,
			})
		})

		assert.Nil(t, err)
		assert.Equal(t, expectedAmountOut, result.TokenAmountOut.Amount.String())
	})

	t.Run("swap USDT -> USDC", func(t *testing.T) {
		amountIn, _ := new(big.Int).SetString("1000000000", 10)

		tokenAmountIn := poolpkg.TokenAmount{
			Token:  "0xdac17f958d2ee523a2206206994597c13d831ec7",
			Amount: amountIn,
		}
		tokenOut := "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"

		result, err := testutil.MustConcurrentSafe(t, func() (*poolpkg.CalcAmountOutResult, error) {
			return s.CalcAmountOut(poolpkg.CalcAmountOutParams{
				TokenAmountIn: tokenAmountIn,
				TokenOut:      tokenOut,
			})
		})

		assert.Nil(t, result)
		assert.ErrorIs(t, err, ErrSwapLimitExceeded)
	})

	t.Run("swap USDT -> USDC with large amount", func(t *testing.T) {
		amountIn, _ := new(big.Int).SetString("1000000", 10)

		tokenAmountIn := poolpkg.TokenAmount{
			Token:  "0xdac17f958d2ee523a2206206994597c13d831ec7",
			Amount: amountIn,
		}
		tokenOut := "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"

		expectedAmountOut := "999593"

		result, err := testutil.MustConcurrentSafe(t, func() (*poolpkg.CalcAmountOutResult, error) {
			return s.CalcAmountOut(poolpkg.CalcAmountOutParams{
				TokenAmountIn: tokenAmountIn,
				TokenOut:      tokenOut,
			})
		})

		assert.Nil(t, err)
		assert.Equal(t, expectedAmountOut, result.TokenAmountOut.Amount.String())
	})
}

func TestCalcAmountIn(t *testing.T) {
	poolStr := `{"address":"0x2bfed8dbeb8e6226a15300ac77ee9130e52410fe","exchange":"euler-swap","type":"euler-swap","timestamp":1743652017,"reserves":["1987079202","2012923319"],"tokens":[{"address":"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48","symbol":"USDC","decimals":6,"swappable":true},{"address":"0xdac17f958d2ee523a2206206994597c13d831ec7","symbol":"USDT","decimals":6,"swappable":true}],"extra":"{\"p\":1,\"v\":[{\"Cash\":\"357461819\",\"Debt\":\"0\",\"MaxDeposit\":\"591856746\",\"MaxWithdraw\":\"900000000\",\"TotalBorrows\":\"50681434\",\"EulerAccountAssets\":\"208103422\"},{\"Cash\":\"360947656\",\"Debt\":\"0\",\"MaxDeposit\":\"608026617\",\"MaxWithdraw\":\"900000000\",\"TotalBorrows\":\"31025726\",\"EulerAccountAssets\":\"191956291\"}]}","staticExtra":"{\"v0\":\"0xa66957e58b60d6b92b850c8773a9ff9b0Ba96A65\",\"v1\":\"0x4212E01c7C8e1c21DEa6030C74aE2084f5337BD1\",\"ea\":\"0x603765f9e9B8E3CBACbdf7A6e963B7EAD77AE86f\",\"fm\":\"1000000000000000000\",\"er0\":\"2000000000\",\"er1\":\"2000000000\",\"px\":\"1000000000000000000\",\"py\":\"1000000000000000000\",\"cx\":\"970000000000000000\",\"cy\":\"970000000000000000\",\"p\":false}","blockNumber":22185795}`

	var pool entity.Pool
	err := json.Unmarshal([]byte(poolStr), &pool)
	assert.Nil(t, err)

	s, err := NewPoolSimulator(pool)
	assert.Nil(t, err)

	t.Run("swap USDT -> USDC", func(t *testing.T) {
		amountOut, _ := new(big.Int).SetString("999593", 10)

		tokenAmountOut := poolpkg.TokenAmount{
			Token:  "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", // USDC
			Amount: amountOut,
		}
		tokenIn := "0xdac17f958d2ee523a2206206994597c13d831ec7" // USDT

		expectedAmountIn := "1000000"

		result, err := testutil.MustConcurrentSafe(t, func() (*poolpkg.CalcAmountInResult, error) {
			return s.CalcAmountIn(poolpkg.CalcAmountInParams{
				TokenAmountOut: tokenAmountOut,
				TokenIn:        tokenIn,
			})
		})

		assert.Nil(t, err)
		assert.Equal(t, expectedAmountIn, result.TokenAmountIn.Amount.String())
	})

	t.Run("swap USDT -> USDC with large amount", func(t *testing.T) {
		amountIn, _ := new(big.Int).SetString("1000000000", 10)

		tokenAmountIn := poolpkg.TokenAmount{
			Token:  "0xdac17f958d2ee523a2206206994597c13d831ec7",
			Amount: amountIn,
		}
		tokenOut := "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"

		result, err := testutil.MustConcurrentSafe(t, func() (*poolpkg.CalcAmountOutResult, error) {
			return s.CalcAmountOut(poolpkg.CalcAmountOutParams{
				TokenAmountIn: tokenAmountIn,
				TokenOut:      tokenOut,
			})
		})

		assert.Nil(t, result)
		assert.ErrorIs(t, err, ErrSwapLimitExceeded)
	})

	t.Run("swap USDC -> USDT", func(t *testing.T) {
		amountOut, _ := new(big.Int).SetString("1000376", 10)

		tokenAmountOut := poolpkg.TokenAmount{
			Token:  "0xdac17f958d2ee523a2206206994597c13d831ec7", // USDT
			Amount: amountOut,
		}
		tokenIn := "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48" // USDC

		expectedAmountIn := "1000000"

		result, err := testutil.MustConcurrentSafe(t, func() (*poolpkg.CalcAmountInResult, error) {
			return s.CalcAmountIn(poolpkg.CalcAmountInParams{
				TokenAmountOut: tokenAmountOut,
				TokenIn:        tokenIn,
			})
		})

		assert.Nil(t, err)
		assert.Equal(t, expectedAmountIn, result.TokenAmountIn.Amount.String())
	})
}
