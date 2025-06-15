package euler

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

func TestCalcAmountOut(t *testing.T) {
	t.Parallel()
	poolStr := `{"address":"0x69058613588536167ba0aa94f0cc1fe420ef28a8","exchange":"uniswap-v4-euler","type":"uniswap-v4-euler","timestamp":1749734358,"reserves":["836474165989","269725806317064027913"],"tokens":[{"address":"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48","symbol":"USDC","decimals":6,"swappable":true},{"address":"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2","symbol":"WETH","decimals":18,"swappable":true}],"extra":"{\"p\":1,\"v\":[{\"Cash\":\"3557692641414\",\"Debt\":\"0\",\"MaxDeposit\":\"46938844142891\",\"MaxWithdraw\":\"67500000000000\",\"TotalBorrows\":\"24503463215694\",\"EulerAccountAssets\":\"337060655490\"},{\"Cash\":\"4649319513393913032975\",\"Debt\":\"31774878270183832877\",\"MaxDeposit\":\"58923495148231711113630\",\"MaxWithdraw\":\"90000000000000000000000\",\"TotalBorrows\":\"36427185338374375853394\",\"EulerAccountAssets\":\"0\"}]}","staticExtra":"{\"v0\":\"0x797DD80692c3b2dAdabCe8e30C07fDE5307D48a9\",\"v1\":\"0xD8b27CF359b7D15710a5BE299AF6e7Bf904984C2\",\"ea\":\"0x0afBf798467F9b3b97F90d05bf7DF592D89A6CF1\",\"f\":\"500000000000000\",\"pf\":\"0\",\"er0\":\"751024805196\",\"er1\":\"301566016943501539193\",\"px\":\"379218809252938\",\"py\":\"1000000\",\"cx\":\"850000000000000000\",\"cy\":\"850000000000000000\"}","blockNumber":22688739}`

	var pool entity.Pool
	err := json.Unmarshal([]byte(poolStr), &pool)
	require.Nil(t, err)

	s, err := NewPoolSimulator(pool)
	require.Nil(t, err)

	t.Run("swap USDC -> WETH", func(t *testing.T) {
		amountIn, _ := new(big.Int).SetString("1000000", 10)

		tokenAmountIn := poolpkg.TokenAmount{
			Token:  "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			Amount: amountIn,
		}
		tokenOut := "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"

		expectedAmountOut := "365327771994315"

		result, err := testutil.MustConcurrentSafe(t, func() (*poolpkg.CalcAmountOutResult, error) {
			return s.CalcAmountOut(poolpkg.CalcAmountOutParams{
				TokenAmountIn: tokenAmountIn,
				TokenOut:      tokenOut,
			})
		})

		require.Nil(t, err)
		require.Equal(t, expectedAmountOut, result.TokenAmountOut.Amount.String())
	})

	t.Run("swap WETH -> USDC", func(t *testing.T) {
		amountIn, _ := new(big.Int).SetString("10000000000000000000000", 10)

		tokenAmountIn := poolpkg.TokenAmount{
			Token:  "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
			Amount: amountIn,
		}
		tokenOut := "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"

		expectedAmountOut := "833188497022"

		result, err := testutil.MustConcurrentSafe(t, func() (*poolpkg.CalcAmountOutResult, error) {
			return s.CalcAmountOut(poolpkg.CalcAmountOutParams{
				TokenAmountIn: tokenAmountIn,
				TokenOut:      tokenOut,
			})
		})

		require.Nil(t, err)
		require.Equal(t, expectedAmountOut, result.TokenAmountOut.Amount.String())
	})

	t.Run("swap WETH -> USDC : invalid amount out", func(t *testing.T) {
		amountIn, _ := new(big.Int).SetString("1000000", 10)

		tokenAmountIn := poolpkg.TokenAmount{
			Token:  "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
			Amount: amountIn,
		}
		tokenOut := "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"

		result, err := testutil.MustConcurrentSafe(t, func() (*poolpkg.CalcAmountOutResult, error) {
			return s.CalcAmountOut(poolpkg.CalcAmountOutParams{
				TokenAmountIn: tokenAmountIn,
				TokenOut:      tokenOut,
			})
		})

		require.Nil(t, result)
		require.ErrorIs(t, err, ErrInvalidAmountOut)
	})

	t.Run("swap USDC -> WETH : swap limit exceeded", func(t *testing.T) {
		amountIn, _ := new(big.Int).SetString("1000000000000000", 10)

		tokenAmountIn := poolpkg.TokenAmount{
			Token:  "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			Amount: amountIn,
		}
		tokenOut := "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"

		result, err := testutil.MustConcurrentSafe(t, func() (*poolpkg.CalcAmountOutResult, error) {
			return s.CalcAmountOut(poolpkg.CalcAmountOutParams{
				TokenAmountIn: tokenAmountIn,
				TokenOut:      tokenOut,
			})
		})

		require.Nil(t, result)
		require.ErrorIs(t, err, ErrSwapLimitExceeded)
	})
}

func TestCalcAmountIn(t *testing.T) {
	t.Parallel()
	poolStr := `{"address":"0x69058613588536167ba0aa94f0cc1fe420ef28a8","exchange":"uniswap-v4-euler","type":"uniswap-v4-euler","timestamp":1749734358,"reserves":["836474165989","269725806317064027913"],"tokens":[{"address":"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48","symbol":"USDC","decimals":6,"swappable":true},{"address":"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2","symbol":"WETH","decimals":18,"swappable":true}],"extra":"{\"p\":1,\"v\":[{\"Cash\":\"3557692641414\",\"Debt\":\"0\",\"MaxDeposit\":\"46938844142891\",\"MaxWithdraw\":\"67500000000000\",\"TotalBorrows\":\"24503463215694\",\"EulerAccountAssets\":\"337060655490\"},{\"Cash\":\"4649319513393913032975\",\"Debt\":\"31774878270183832877\",\"MaxDeposit\":\"58923495148231711113630\",\"MaxWithdraw\":\"90000000000000000000000\",\"TotalBorrows\":\"36427185338374375853394\",\"EulerAccountAssets\":\"0\"}]}","staticExtra":"{\"v0\":\"0x797DD80692c3b2dAdabCe8e30C07fDE5307D48a9\",\"v1\":\"0xD8b27CF359b7D15710a5BE299AF6e7Bf904984C2\",\"ea\":\"0x0afBf798467F9b3b97F90d05bf7DF592D89A6CF1\",\"f\":\"500000000000000\",\"pf\":\"0\",\"er0\":\"751024805196\",\"er1\":\"301566016943501539193\",\"px\":\"379218809252938\",\"py\":\"1000000\",\"cx\":\"850000000000000000\",\"cy\":\"850000000000000000\"}","blockNumber":22688739}`

	var pool entity.Pool
	err := json.Unmarshal([]byte(poolStr), &pool)
	require.Nil(t, err)

	s, err := NewPoolSimulator(pool)
	require.Nil(t, err)

	t.Run("swap USDC -> WETH", func(t *testing.T) {
		amountOut, _ := new(big.Int).SetString("365327771994315", 10)

		tokenAmountOut := poolpkg.TokenAmount{
			Token:  "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", // WETH
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

		require.Nil(t, err)
		require.Equal(t, expectedAmountIn, result.TokenAmountIn.Amount.String())
	})

	t.Run("swap WETH -> USDC", func(t *testing.T) {
		amountOut, _ := new(big.Int).SetString("833188497022", 10)

		tokenAmountOut := poolpkg.TokenAmount{
			Token:  "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", // USDC
			Amount: amountOut,
		}
		tokenIn := "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2" // WETH

		expectedAmountIn := "9999999995811266764061"

		result, err := testutil.MustConcurrentSafe(t, func() (*poolpkg.CalcAmountInResult, error) {
			return s.CalcAmountIn(poolpkg.CalcAmountInParams{
				TokenAmountOut: tokenAmountOut,
				TokenIn:        tokenIn,
			})
		})

		require.Nil(t, err)
		require.Equal(t, expectedAmountIn, result.TokenAmountIn.Amount.String())
	})

	t.Run("swap WETH -> USDC : swap limit exceeded", func(t *testing.T) {
		amountIn, _ := new(big.Int).SetString("269725806317064027914", 10)

		tokenAmountOut := poolpkg.TokenAmount{
			Token:  "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			Amount: amountIn,
		}
		tokenIn := "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"

		result, err := testutil.MustConcurrentSafe(t, func() (*poolpkg.CalcAmountInResult, error) {
			return s.CalcAmountIn(poolpkg.CalcAmountInParams{
				TokenAmountOut: tokenAmountOut,
				TokenIn:        tokenIn,
			})
		})

		require.Nil(t, result)
		require.ErrorIs(t, err, ErrSwapLimitExceeded)
	})
}
