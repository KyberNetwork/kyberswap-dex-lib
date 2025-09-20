package eulerswap

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

func TestCalcAmountOut(t *testing.T) {
	t.Parallel()
	poolStr := `{"address":"0x69058613588536167ba0aa94f0cc1fe420ef28a8","exchange":"euler-swap","type":"euler-swap","timestamp":1749734358,"reserves":["836474165989","269725806317064027913"],"tokens":[{"address":"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48","symbol":"USDC","decimals":6,"swappable":true},{"address":"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2","symbol":"WETH","decimals":18,"swappable":true}],"extra":"{\"p\":1,\"v\":[{\"Cash\":\"3557692641414\",\"Debt\":\"0\",\"MaxDeposit\":\"46938844142891\",\"MaxWithdraw\":\"67500000000000\",\"TotalBorrows\":\"24503463215694\",\"EulerAccountAssets\":\"337060655490\",\"CanBorrow\":true},{\"Cash\":\"4649319513393913032975\",\"Debt\":\"31774878270183832877\",\"MaxDeposit\":\"58923495148231711113630\",\"MaxWithdraw\":\"90000000000000000000000\",\"TotalBorrows\":\"36427185338374375853394\",\"EulerAccountAssets\":\"0\",\"CanBorrow\":true}]}","staticExtra":"{\"v0\":\"0x797DD80692c3b2dAdabCe8e30C07fDE5307D48a9\",\"v1\":\"0xD8b27CF359b7D15710a5BE299AF6e7Bf904984C2\",\"ea\":\"0x0afBf798467F9b3b97F90d05bf7DF592D89A6CF1\",\"f\":\"500000000000000\",\"pf\":\"0\",\"er0\":\"751024805196\",\"er1\":\"301566016943501539193\",\"px\":\"379218809252938\",\"py\":\"1000000\",\"cx\":\"850000000000000000\",\"cy\":\"850000000000000000\"}","blockNumber":22688739}`

	var pool entity.Pool
	err := json.Unmarshal([]byte(poolStr), &pool)
	require.Nil(t, err)

	s, err := NewPoolSimulator(pool)
	require.Nil(t, err)

	// adhocs due to changing data structure
	for i := range s.vaults {
		s.vaults[i].LTV = uint256.NewInt(0)
		s.vaults[i].AssetPrice = uint256.NewInt(0)
		s.vaults[i].SharePrice = uint256.NewInt(0)
		s.vaults[i].TotalAssets = uint256.NewInt(0)
		s.vaults[i].TotalSupply = uint256.NewInt(0)
	}
	s.liabilityValue = uint256.NewInt(0)
	s.collateralValue = uint256.NewInt(0)

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
	poolStr := `{"address":"0x69058613588536167ba0aa94f0cc1fe420ef28a8","exchange":"euler-swap","type":"euler-swap","timestamp":1749734358,"reserves":["836474165989","269725806317064027913"],"tokens":[{"address":"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48","symbol":"USDC","decimals":6,"swappable":true},{"address":"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2","symbol":"WETH","decimals":18,"swappable":true}],"extra":"{\"p\":1,\"v\":[{\"Cash\":\"3557692641414\",\"Debt\":\"0\",\"MaxDeposit\":\"46938844142891\",\"MaxWithdraw\":\"67500000000000\",\"TotalBorrows\":\"24503463215694\",\"EulerAccountAssets\":\"337060655490\",\"CanBorrow\":true},{\"Cash\":\"4649319513393913032975\",\"Debt\":\"31774878270183832877\",\"MaxDeposit\":\"58923495148231711113630\",\"MaxWithdraw\":\"90000000000000000000000\",\"TotalBorrows\":\"36427185338374375853394\",\"EulerAccountAssets\":\"0\",\"CanBorrow\":true}]}","staticExtra":"{\"v0\":\"0x797DD80692c3b2dAdabCe8e30C07fDE5307D48a9\",\"v1\":\"0xD8b27CF359b7D15710a5BE299AF6e7Bf904984C2\",\"ea\":\"0x0afBf798467F9b3b97F90d05bf7DF592D89A6CF1\",\"f\":\"500000000000000\",\"pf\":\"0\",\"er0\":\"751024805196\",\"er1\":\"301566016943501539193\",\"px\":\"379218809252938\",\"py\":\"1000000\",\"cx\":\"850000000000000000\",\"cy\":\"850000000000000000\"}","blockNumber":22688739}`

	var pool entity.Pool
	err := json.Unmarshal([]byte(poolStr), &pool)
	require.Nil(t, err)

	s, err := NewPoolSimulator(pool)
	require.Nil(t, err)

	// adhocs due to changing data structure
	for i := range s.vaults {
		s.vaults[i].LTV = uint256.NewInt(0)
		s.vaults[i].AssetPrice = uint256.NewInt(0)
		s.vaults[i].SharePrice = uint256.NewInt(0)
		s.vaults[i].TotalAssets = uint256.NewInt(0)
		s.vaults[i].TotalSupply = uint256.NewInt(0)
	}
	s.liabilityValue = uint256.NewInt(0)
	s.collateralValue = uint256.NewInt(0)

	testutil.TestCalcAmountIn(t, s)
}

func TestMergeSwaps(t *testing.T) {
	t.Parallel()

	poolStr := `{"address":"0x98e48d708f52d29f0f09be157f597d062747e8a8","exchange":"uniswap-v4-euler","type":"euler-swap","timestamp":1752145833,"reserves":["10392721374273","52156542521336"],"tokens":[{"address":"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48","symbol":"USDC","decimals":6,"swappable":true},{"address":"0xdac17f958d2ee523a2206206994597c13d831ec7","symbol":"USDT","decimals":6,"swappable":true}],"extra":"{\"p\":1,\"v\":[{\"Cash\":\"17271279289973\",\"Debt\":\"19814269629134\",\"MaxDeposit\":\"22900683055346\",\"MaxWithdraw\":\"67500000000000\",\"TotalBorrows\":\"34828037654680\",\"EulerAccountAssets\":\"0\",\"CanBorrow\":true},{\"Cash\":\"5674807873177\",\"Debt\":\"0\",\"MaxDeposit\":\"18864221709050\",\"MaxWithdraw\":\"45000000000000\",\"TotalBorrows\":\"25460970417772\",\"EulerAccountAssets\":\"21967163791256\",\"CanBorrow\":false}]}","staticExtra":"{\"v0\":\"0x797DD80692c3b2dAdabCe8e30C07fDE5307D48a9\",\"v1\":\"0x313603FA690301b0CaeEf8069c065862f9162162\",\"ea\":\"0x0Afbf798467F9b3b97F90d05bF7df592D89A6cF6\",\"f\":\"5000000000000\",\"pf\":\"0\",\"er0\":\"32380768989027\",\"er1\":\"30176535964462\",\"px\":\"1000000\",\"py\":\"1000387\",\"cx\":\"999990000000000000\",\"cy\":\"999999000000000000\",\"pfr\":\"0x0000000000000000000000000000000000000000\",\"evc\":\"0x0C9a3dd6b8F28529d72d7f9cE918D493519EE383\"}","blockNumber":22888393}`

	var pool entity.Pool
	err := json.Unmarshal([]byte(poolStr), &pool)
	require.Nil(t, err)

	testCases := []struct {
		name     string
		amountIn string
	}{
		{
			name:     "Small amount",
			amountIn: "1000000000", // 1000 USDC
		},
		{
			name:     "Medium amount",
			amountIn: "5000000000000", // 5M USDC
		},
		{
			name:     "Large amount",
			amountIn: "10000000000000", // 10M USDC
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Single swap
			singlePool, err := NewPoolSimulator(pool)
			require.Nil(t, err)

			// adhocs due to changing data structure
			for i := range singlePool.vaults {
				singlePool.vaults[i].LTV = uint256.NewInt(0)
				singlePool.vaults[i].AssetPrice = uint256.NewInt(0)
				singlePool.vaults[i].SharePrice = uint256.NewInt(0)
				singlePool.vaults[i].TotalAssets = uint256.NewInt(0)
				singlePool.vaults[i].TotalSupply = uint256.NewInt(0)
			}
			singlePool.liabilityValue = uint256.NewInt(0)
			singlePool.collateralValue = uint256.NewInt(0)

			amountIn, _ := new(big.Int).SetString(tc.amountIn, 10)
			tokenAmountIn := poolpkg.TokenAmount{
				Token:  "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
				Amount: amountIn,
			}
			tokenOut := "0xdac17f958d2ee523a2206206994597c13d831ec7"

			singleResult, singleErr := testutil.MustConcurrentSafe(t, func() (*poolpkg.CalcAmountOutResult, error) {
				return singlePool.CalcAmountOut(poolpkg.CalcAmountOutParams{
					TokenAmountIn: tokenAmountIn,
					TokenOut:      tokenOut,
				})
			})

			// Chunked swaps (20 chunks)
			chunkedPool, err := NewPoolSimulator(pool)
			require.Nil(t, err)

			// adhocs due to changing data structure
			for i := range chunkedPool.vaults {
				chunkedPool.vaults[i].LTV = uint256.NewInt(0)
				chunkedPool.vaults[i].AssetPrice = uint256.NewInt(0)
				chunkedPool.vaults[i].SharePrice = uint256.NewInt(0)
				chunkedPool.vaults[i].TotalAssets = uint256.NewInt(0)
				chunkedPool.vaults[i].TotalSupply = uint256.NewInt(0)
			}
			chunkedPool.liabilityValue = uint256.NewInt(0)
			chunkedPool.collateralValue = uint256.NewInt(0)

			chunkAmount := new(big.Int).Div(amountIn, big.NewInt(20))
			var totalAmountOut *big.Int
			var chunkedErr error

			for i := 0; i < 20; i++ {
				chunkTokenAmountIn := poolpkg.TokenAmount{
					Token:  "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
					Amount: chunkAmount,
				}

				chunkResult, err := testutil.MustConcurrentSafe(t, func() (*poolpkg.CalcAmountOutResult, error) {
					return chunkedPool.CalcAmountOut(poolpkg.CalcAmountOutParams{
						TokenAmountIn: chunkTokenAmountIn,
						TokenOut:      tokenOut,
					})
				})

				if err != nil {
					chunkedErr = err
					break
				}

				chunkedPool.UpdateBalance(poolpkg.UpdateBalanceParams{
					SwapInfo: chunkResult.SwapInfo,
				})

				if totalAmountOut == nil {
					totalAmountOut = chunkResult.TokenAmountOut.Amount
				} else {
					totalAmountOut.Add(totalAmountOut, chunkResult.TokenAmountOut.Amount)
				}
			}

			if singleErr != nil {
				require.NotNil(t, chunkedErr, "Single swap failed but chunked swap succeeded")
				t.Logf("Both processes failed as expected: %v", singleErr)
			} else {
				require.Nil(t, chunkedErr, "Single swap succeeded but chunked swap failed")
				require.NotNil(t, totalAmountOut, "Chunked swap should have produced output")

				diff := new(big.Int).Sub(singleResult.TokenAmountOut.Amount, totalAmountOut)
				diff.Abs(diff)

				// Allow 1% difference due to rounding in chunked calculations
				maxDiff := new(big.Int).Div(singleResult.TokenAmountOut.Amount, big.NewInt(100))
				require.LessOrEqual(t, diff.Cmp(maxDiff), 0,
					"Results differ too much. Single: %s, Chunked: %s",
					singleResult.TokenAmountOut.Amount.String(),
					totalAmountOut.String())

				t.Logf("Both processes succeeded. Single: %s, Chunked: %s",
					singleResult.TokenAmountOut.Amount.String(),
					totalAmountOut.String())
			}
		})
	}
}

func TestSwapEdgeCases(t *testing.T) {
	t.Parallel()

	poolStr := `{"address":"0x98e48d708f52d29f0f09be157f597d062747e8a8","exchange":"uniswap-v4-euler","type":"euler-swap","timestamp":1752145833,"reserves":["10392721374273","52156542521336"],"tokens":[{"address":"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48","symbol":"USDC","decimals":6,"swappable":true},{"address":"0xdac17f958d2ee523a2206206994597c13d831ec7","symbol":"USDT","decimals":6,"swappable":true}],"extra":"{\"p\":1,\"v\":[{\"Cash\":\"17271279289973\",\"Debt\":\"19814269629134\",\"MaxDeposit\":\"22900683055346\",\"MaxWithdraw\":\"67500000000000\",\"TotalBorrows\":\"34828037654680\",\"EulerAccountAssets\":\"0\",\"CanBorrow\":true},{\"Cash\":\"5674807873177\",\"Debt\":\"0\",\"MaxDeposit\":\"18864221709050\",\"MaxWithdraw\":\"45000000000000\",\"TotalBorrows\":\"25460970417772\",\"EulerAccountAssets\":\"21967163791256\",\"CanBorrow\":false}]}","staticExtra":"{\"v0\":\"0x797DD80692c3b2dAdabCe8e30C07fDE5307D48a9\",\"v1\":\"0x313603FA690301b0CaeEf8069c065862f9162162\",\"ea\":\"0x0Afbf798467F9b3b97F90d05bF7df592D89A6cF6\",\"f\":\"5000000000000\",\"pf\":\"0\",\"er0\":\"32380768989027\",\"er1\":\"30176535964462\",\"px\":\"1000000\",\"py\":\"1000387\",\"cx\":\"999990000000000000\",\"cy\":\"999999000000000000\",\"pfr\":\"0x0000000000000000000000000000000000000000\",\"evc\":\"0x0C9a3dd6b8F28529d72d7f9cE918D493519EE383\"}","blockNumber":22888393}`

	var pool entity.Pool
	err := json.Unmarshal([]byte(poolStr), &pool)
	require.Nil(t, err)

	testCases := []struct {
		name        string
		amountIn    string
		description string
		expectError bool
	}{
		{
			name:        "Exact amount at MaxDeposit limit",
			amountIn:    "22900683055346", // Exactly MaxDeposit
			description: "Should fail when amount equals MaxDeposit (due to fees)",
			expectError: true,
		},
		{
			name:        "Amount exceeds MaxDeposit by 1",
			amountIn:    "22900683055347", // MaxDeposit + 1
			description: "Should fail when amount exceeds MaxDeposit",
			expectError: true,
		},
		{
			name:        "Amount exceeds MaxWithdraw",
			amountIn:    "45000000000001", // MaxWithdraw + 1
			description: "Should fail when output amount exceeds MaxWithdraw",
			expectError: true,
		},
		{
			name:        "Amount exceeds Cash balance",
			amountIn:    "10000000000000", // Very large amount that will exceed cash
			description: "Should fail when output amount exceeds Cash",
			expectError: true,
		},
		{
			name:        "Amount exceeds reserves",
			amountIn:    "52156542521337", // Reserve1 + 1
			description: "Should fail when output amount exceeds reserves",
			expectError: true,
		},
		{
			name:        "Very small amount (1 wei)",
			amountIn:    "1",
			description: "Should handle very small amounts",
			expectError: false,
		},
		{
			name:        "Amount causing curve violation",
			amountIn:    "50000000000000", // Very large amount
			description: "Should fail due to curve violation",
			expectError: true,
		},
		{
			name:        "Amount at borrow limit",
			amountIn:    "21967163791256", // Exactly EulerAccountAssets
			description: "Should fail when using exact available balance (due to fees)",
			expectError: true,
		},
		{
			name:        "Amount exceeds borrow limit",
			amountIn:    "21967163791257", // EulerAccountAssets + 1
			description: "Should fail when exceeding borrow limit",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			poolSim, err := NewPoolSimulator(pool)
			require.Nil(t, err)

			// adhocs due to changing data structure
			for i := range poolSim.vaults {
				poolSim.vaults[i].LTV = uint256.NewInt(0)
				poolSim.vaults[i].AssetPrice = uint256.NewInt(0)
				poolSim.vaults[i].SharePrice = uint256.NewInt(0)
				poolSim.vaults[i].TotalAssets = uint256.NewInt(0)
				poolSim.vaults[i].TotalSupply = uint256.NewInt(0)
			}
			poolSim.liabilityValue = uint256.NewInt(0)
			poolSim.collateralValue = uint256.NewInt(0)

			amountIn, _ := new(big.Int).SetString(tc.amountIn, 10)
			tokenAmountIn := poolpkg.TokenAmount{
				Token:  "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
				Amount: amountIn,
			}
			tokenOut := "0xdac17f958d2ee523a2206206994597c13d831ec7"

			result, err := testutil.MustConcurrentSafe(t, func() (*poolpkg.CalcAmountOutResult, error) {
				return poolSim.CalcAmountOut(poolpkg.CalcAmountOutParams{
					TokenAmountIn: tokenAmountIn,
					TokenOut:      tokenOut,
				})
			})

			if tc.expectError {
				require.NotNil(t, err, "Expected error for: %s", tc.description)
				t.Logf("Expected error occurred: %v", err)
			} else {
				require.Nil(t, err, "Expected success for: %s", tc.description)
				require.NotNil(t, result, "Expected result for: %s", tc.description)
				require.Greater(t, result.TokenAmountOut.Amount.Cmp(big.NewInt(0)), 0, "Expected positive output amount")
				t.Logf("Success: input=%s, output=%s", tc.amountIn, result.TokenAmountOut.Amount.String())
			}
		})
	}
}

func TestReverseSwap(t *testing.T) {
	t.Parallel()

	poolStr := `{"address":"0x98e48d708f52d29f0f09be157f597d062747e8a8","exchange":"uniswap-v4-euler","type":"euler-swap","timestamp":1752145833,"reserves":["10392721374273","52156542521336"],"tokens":[{"address":"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48","symbol":"USDC","decimals":6,"swappable":true},{"address":"0xdac17f958d2ee523a2206206994597c13d831ec7","symbol":"USDT","decimals":6,"swappable":true}],"extra":"{\"p\":1,\"v\":[{\"Cash\":\"17271279289973\",\"Debt\":\"19814269629134\",\"MaxDeposit\":\"22900683055346\",\"MaxWithdraw\":\"67500000000000\",\"TotalBorrows\":\"34828037654680\",\"EulerAccountAssets\":\"0\",\"CanBorrow\":true},{\"Cash\":\"5674807873177\",\"Debt\":\"0\",\"MaxDeposit\":\"18864221709050\",\"MaxWithdraw\":\"45000000000000\",\"TotalBorrows\":\"25460970417772\",\"EulerAccountAssets\":\"21967163791256\",\"CanBorrow\":false}]}","staticExtra":"{\"v0\":\"0x797DD80692c3b2dAdabCe8e30C07fDE5307D48a9\",\"v1\":\"0x313603FA690301b0CaeEf8069c065862f9162162\",\"ea\":\"0x0Afbf798467F9b3b97F90d05bF7df592D89A6cF6\",\"f\":\"5000000000000\",\"pf\":\"0\",\"er0\":\"32380768989027\",\"er1\":\"30176535964462\",\"px\":\"1000000\",\"py\":\"1000387\",\"cx\":\"999990000000000000\",\"cy\":\"999999000000000000\",\"pfr\":\"0x0000000000000000000000000000000000000000\",\"evc\":\"0x0C9a3dd6b8F28529d72d7f9cE918D493519EE383\"}","blockNumber":22888393}`

	var pool entity.Pool
	err := json.Unmarshal([]byte(poolStr), &pool)
	require.Nil(t, err)

	s, err := NewPoolSimulator(pool)
	require.Nil(t, err)

	// adhocs due to changing data structure
	for i := range s.vaults {
		s.vaults[i].LTV = uint256.NewInt(0)
		s.vaults[i].AssetPrice = uint256.NewInt(0)
		s.vaults[i].SharePrice = uint256.NewInt(0)
		s.vaults[i].TotalAssets = uint256.NewInt(0)
		s.vaults[i].TotalSupply = uint256.NewInt(0)
	}
	s.liabilityValue = uint256.NewInt(0)
	s.collateralValue = uint256.NewInt(0)

	// Forward swap: USDC -> USDT
	amountIn, _ := new(big.Int).SetString("1000000000", 10) // 1000 USDC
	tokenAmountIn := poolpkg.TokenAmount{
		Token:  "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
		Amount: amountIn,
	}
	tokenOut := "0xdac17f958d2ee523a2206206994597c13d831ec7"

	forwardResult, err := testutil.MustConcurrentSafe(t, func() (*poolpkg.CalcAmountOutResult, error) {
		return s.CalcAmountOut(poolpkg.CalcAmountOutParams{
			TokenAmountIn: tokenAmountIn,
			TokenOut:      tokenOut,
		})
	})

	require.Nil(t, err)
	require.NotNil(t, forwardResult)

	s.UpdateBalance(poolpkg.UpdateBalanceParams{
		SwapInfo: forwardResult.SwapInfo,
	})

	// Reverse swap: USDT -> USDC
	reverseTokenAmountIn := poolpkg.TokenAmount{
		Token:  "0xdac17f958d2ee523a2206206994597c13d831ec7",
		Amount: forwardResult.TokenAmountOut.Amount,
	}
	reverseTokenOut := "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"

	reverseResult, err := testutil.MustConcurrentSafe(t, func() (*poolpkg.CalcAmountOutResult, error) {
		return s.CalcAmountOut(poolpkg.CalcAmountOutParams{
			TokenAmountIn: reverseTokenAmountIn,
			TokenOut:      reverseTokenOut,
		})
	})

	require.Nil(t, err)
	require.NotNil(t, reverseResult)

	require.Less(t, reverseResult.TokenAmountOut.Amount.Cmp(amountIn), 0, "Reverse swap should return less than original due to fees")
	require.Greater(t, reverseResult.TokenAmountOut.Amount.Cmp(big.NewInt(0)), 0, "Reverse swap should return positive amount")

	t.Logf("Swap reversibility verified: forward=%s, reverse=%s", forwardResult.TokenAmountOut.Amount.String(), reverseResult.TokenAmountOut.Amount.String())
}

func TestSwapCanBorrow(t *testing.T) {
	t.Parallel()

	poolStr := `{"address":"0xec078526b7a841c3f8fcd13ecc8efc0f1e25a8a8","exchange":"uniswap-v4-euler","type":"euler-swap","timestamp":1753191616,"reserves":["68185365","9189936696438807"],"tokens":[{"address":"0x078d782b760474a361dda0af3839290b0ef57ad6","symbol":"USDC","decimals":6,"swappable":true},{"address":"0x4200000000000000000000000000000000000006","symbol":"WETH","decimals":18,"swappable":true}],"extra":"{\"p\":1,\"v\":[{\"Cash\":\"7271394737276\",\"Debt\":\"0\",\"MaxDeposit\":\"101559795392750\",\"MaxWithdraw\":\"135000000000000\",\"TotalBorrows\":\"41168809869973\",\"EulerAccountAssets\":\"48786\",\"CollateralValue\":\"0\",\"LiabilityValue\":\"0\",\"AssetPrice\":\"999929680000\",\"SharePrice\":\"1009022921518\",\"TotalAssets\":\"48440204607249\",\"TotalSupply\":\"48003664977701\",\"LTV\":\"8400\"},{\"Cash\":\"1091578060036730433752\",\"Debt\":\"0\",\"MaxDeposit\":\"14202793488426118406190\",\"MaxWithdraw\":\"18000000000000000000000\",\"TotalBorrows\":\"4705628451537151160057\",\"EulerAccountAssets\":\"0\",\"CollateralValue\":\"40977358269523200\",\"LiabilityValue\":\"0\",\"AssetPrice\":\"3682\",\"SharePrice\":\"3702\",\"TotalAssets\":\"5797206511573881593809\",\"TotalSupply\":\"5766033809403791449234\",\"LTV\":\"8400\"}]}","staticExtra":"{\"v0\":\"0x6eAe95ee783e4D862867C4e0E4c3f4B95AA682Ba\",\"v1\":\"0x1f3134C3f3f8AdD904B9635acBeFC0eA0D0E1ffC\",\"ea\":\"0x3e4Ab57CD1Aef2DE5D2d3b889d6f2Fc82b5Dc733\",\"f\":\"1000000000000000\",\"pf\":\"0\",\"er0\":\"43361827\",\"er1\":\"18427910303646597\",\"px\":\"1000000000000000000\",\"py\":\"2441675542\",\"cx\":\"900000000000000000\",\"cy\":\"900000000000000000\",\"pfr\":\"0x0000000000000000000000000000000000000000\",\"evc\":\"0x2A1176964F5D7caE5406B627Bf6166664FE83c60\"}","blockNumber":22443256}`

	var pool entity.Pool
	err := json.Unmarshal([]byte(poolStr), &pool)
	require.Nil(t, err)

	t.Run("Both vaults with zero balance and canBorrow=false should fail", func(t *testing.T) {
		poolSim, err := NewPoolSimulator(pool)
		require.Nil(t, err)

		poolSim.collateralValue = uint256.NewInt(0)
		poolSim.liabilityValue = uint256.NewInt(0)

		amountIn, _ := new(big.Int).SetString("1000000", 10) // 1 USDC
		tokenAmountIn := poolpkg.TokenAmount{
			Token:  "0x078d782b760474a361dda0af3839290b0ef57ad6",
			Amount: amountIn,
		}
		tokenOut := "0x4200000000000000000000000000000000000006"

		result, err := testutil.MustConcurrentSafe(t, func() (*poolpkg.CalcAmountOutResult, error) {
			return poolSim.CalcAmountOut(poolpkg.CalcAmountOutParams{
				TokenAmountIn: tokenAmountIn,
				TokenOut:      tokenOut,
			})
		})

		require.Nil(t, result)
		require.NotNil(t, err, "Should fail when collateralValue < liabilityValue")
	})

	var pool2 entity.Pool
	err = json.Unmarshal([]byte(poolStr), &pool2)
	require.Nil(t, err)

	t.Run("From vault with zero balance and canBorrow=false to vault with canBorrow=true should succeed", func(t *testing.T) {
		poolSim, err := NewPoolSimulator(pool2)
		require.Nil(t, err)

		poolSim.collateralValue = uint256.NewInt(0)
		poolSim.liabilityValue = uint256.NewInt(0)

		amountIn, _ := new(big.Int).SetString("1000000000", 10) // 1000 USDC
		tokenAmountIn := poolpkg.TokenAmount{
			Token:  "0x4200000000000000000000000000000000000006",
			Amount: amountIn,
		}
		tokenOut := "0x078d782b760474a361dda0af3839290b0ef57ad6"

		result, err := testutil.MustConcurrentSafe(t, func() (*poolpkg.CalcAmountOutResult, error) {
			return poolSim.CalcAmountOut(poolpkg.CalcAmountOutParams{
				TokenAmountIn: tokenAmountIn,
				TokenOut:      tokenOut,
			})
		})

		require.Nil(t, err)
		require.NotNil(t, result)
		require.Greater(t, result.TokenAmountOut.Amount.Sign(), 0)
		t.Logf("input=%s, output=%s", amountIn.String(), result.TokenAmountOut.Amount.String())
	})
}
