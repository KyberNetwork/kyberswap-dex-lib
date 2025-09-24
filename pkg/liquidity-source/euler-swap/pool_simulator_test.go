package eulerswap

import (
	"log"
	"math/big"
	"os"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

func TestCalcAmountOut(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("Skipping testing in CI environment")
	}

	t.Parallel()
	poolStr := `{"address":"0x69058613588536167ba0aa94f0cc1fe420ef28a8","exchange":"euler-swap","type":"euler-swap","timestamp":1749734358,"reserves":["836474165989","269725806317064027913"],"tokens":[{"address":"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48","symbol":"USDC","decimals":6,"swappable":true},{"address":"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2","symbol":"WETH","decimals":18,"swappable":true}],"extra":"{\"p\":1,\"v\":[{\"c\":\"3557692641414\",\"d\":\"0\",\"mD\":\"46938844142891\",\"mW\":\"67500000000000\",\"tB\":\"24503463215694\",\"eAA\":\"337060655490\",\"dP\":\"999643500000\",\"vP\":[\"999643500000\",\"999728620000\"],\"vVP\":[\"999643500000\",\"999728620000\"],\"ltv\":[0,0],\"vLtv\":[0,9000]},{\"c\":\"4649319513393913032975\",\"d\":\"31774878270183832877\",\"mD\":\"58923495148231711113630\",\"mW\":\"90000000000000000000000\",\"tB\":\"36427185338374375853394\",\"eAA\":\"0\",\"dP\":\"999728620000\",\"vP\":[\"999643500000\",\"999728620000\"],\"vVP\":[\"999643500000\",\"999728620000\"],\"ltv\":[0,0],\"vLtv\":[9000,0]}],\"cV\":\"0x39de0f00189306062d79edec6dca5bb6bfd108f9\",\"c\":[\"832634226392\",\"0\"]}","staticExtra":"{\"v0\":\"0x797DD80692c3b2dAdabCe8e30C07fDE5307D48a9\",\"v1\":\"0xD8b27CF359b7D15710a5BE299AF6e7Bf904984C2\",\"ea\":\"0x0afBf798467F9b3b97F90d05bf7DF592D89A6CF1\",\"f\":\"500000000000000\",\"pf\":\"0\",\"er0\":\"751024805196\",\"er1\":\"301566016943501539193\",\"px\":\"379218809252938\",\"py\":\"1000000\",\"cx\":\"850000000000000000\",\"cy\":\"850000000000000000\"}","blockNumber":22688739}`

	var pool entity.Pool
	err := json.Unmarshal([]byte(poolStr), &pool)
	require.Nil(t, err)

	s, err := NewPoolSimulator(pool)
	require.Nil(t, err)

	t.Run("swap USDC -> WETH", func(t *testing.T) {
		amountIn, _ := new(big.Int).SetString("1000", 10)

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
	if os.Getenv("CI") != "" {
		t.Skip("Skipping testing in CI environment")
	}

	t.Parallel()
	poolStr := `{"address":"0x69058613588536167ba0aa94f0cc1fe420ef28a8","exchange":"euler-swap","type":"euler-swap","timestamp":1749734358,"reserves":["836474165989","269725806317064027913"],"tokens":[{"address":"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48","symbol":"USDC","decimals":6,"swappable":true},{"address":"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2","symbol":"WETH","decimals":18,"swappable":true}],"extra":"{\"p\":1,\"v\":[{\"c\":\"993557692641414\",\"d\":\"0\",\"mD\":\"9946938844142891\",\"mW\":\"675000000000000\",\"tB\":\"24503463215694\",\"eAA\":\"337060655490\",\"dP\":\"999643500000\",\"vVP\":[\"999643500000\",\"999728620000\"],\"vLtv\":[0,9000]},{\"c\":\"94649319513393913032975\",\"d\":\"774878270183832877\",\"mD\":\"58923495148231711113630\",\"mW\":\"90000000000000000000000\",\"tB\":\"36427185338374375853394\",\"eAA\":\"0\",\"dP\":\"999728620000\",\"vVP\":[\"999643500000\",\"999728620000\"],\"vLtv\":[9000,0]}]}","staticExtra":"{\"v0\":\"0x797DD80692c3b2dAdabCe8e30C07fDE5307D48a9\",\"v1\":\"0xD8b27CF359b7D15710a5BE299AF6e7Bf904984C2\",\"ea\":\"0x0afBf798467F9b3b97F90d05bf7DF592D89A6CF1\",\"f\":\"500000000000000\",\"pf\":\"0\",\"er0\":\"751024805196\",\"er1\":\"301566016943501539193\",\"px\":\"379218809252938\",\"py\":\"1000000\",\"cx\":\"850000000000000000\",\"cy\":\"850000000000000000\"}","blockNumber":22688739}`

	var pool entity.Pool
	err := json.Unmarshal([]byte(poolStr), &pool)
	require.Nil(t, err)

	s, err := NewPoolSimulator(pool)
	require.Nil(t, err)

	testutil.TestCalcAmountIn(t, s)
}

func TestSwapEdgeCases(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("Skipping testing in CI environment")
	}

	t.Parallel()

	poolStr := `{"address":"0x98e48d708f52d29f0f09be157f597d062747e8a8","exchange":"uniswap-v4-euler","type":"euler-swap","timestamp":1752145833,"reserves":["10392721374273","52156542521336"],"tokens":[{"address":"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48","symbol":"USDC","decimals":6,"swappable":true},{"address":"0xdac17f958d2ee523a2206206994597c13d831ec7","symbol":"USDT","decimals":6,"swappable":true}],"extra":"{\"p\":1,\"v\":[{\"c\":\"17271279289973\",\"d\":\"19814269629134\",\"mD\":\"22900683055346\",\"mW\":\"67500000000000\",\"tB\":\"34828037654680\",\"eAA\":\"0\"},{\"c\":\"5674807873177\",\"d\":\"0\",\"mD\":\"18864221709050\",\"mW\":\"45000000000000\",\"tB\":\"25460970417772\",\"eAA\":\"21967163791256\"}]}","staticExtra":"{\"v0\":\"0x797DD80692c3b2dAdabCe8e30C07fDE5307D48a9\",\"v1\":\"0x313603FA690301b0CaeEf8069c065862f9162162\",\"ea\":\"0x0Afbf798467F9b3b97F90d05bF7df592D89A6cF6\",\"f\":\"5000000000000\",\"pf\":\"0\",\"er0\":\"32380768989027\",\"er1\":\"30176535964462\",\"px\":\"1000000\",\"py\":\"1000387\",\"cx\":\"999990000000000000\",\"cy\":\"999999000000000000\",\"pfr\":\"0x0000000000000000000000000000000000000000\",\"evc\":\"0x0C9a3dd6b8F28529d72d7f9cE918D493519EE383\"}","blockNumber":22888393}`

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

	poolStr := `{"address":"0xe934cc9c5c49bbff0f8905c5bcfa65ce5e6de8a8","exchange":"uniswap-v4-euler","type":"euler-swap","timestamp":1758512001,"reserves":["3058437103820","677611878674"],"tokens":[{"address":"0x00000000efe302beaa2b3e6e1b18d08d69a9012a","symbol":"AUSD","decimals":6,"swappable":true},{"address":"0xb97ef9ef8734c71904d8002f8b6bc66dd9c48a6e","symbol":"USDC","decimals":6,"swappable":true}],"extra":"{\"p\":1,\"v\":[{\"c\":\"4812119143924\",\"d\":\"0\",\"mD\":\"52409500588666\",\"mW\":\"115792089237316195423570985008687907853269984665640564039457584007913129639935\",\"tB\":\"17778380267409\",\"eAA\":\"832634226392\",\"dP\":\"999643500000\",\"vP\":[\"999643500000\",\"999728620000\"],\"vVP\":[\"999643500000\",\"999728620000\"],\"ltv\":[0,0],\"vLtv\":[0,9000]},{\"c\":\"3989390539783\",\"d\":\"331244946983\",\"mD\":\"49793730393690\",\"mW\":\"115792089237316195423570985008687907853269984665640564039457584007913129639935\",\"tB\":\"46216879066526\",\"eAA\":\"0\",\"dP\":\"999728620000\",\"vP\":[\"999643500000\",\"999728620000\"],\"vVP\":[\"999643500000\",\"999728620000\"],\"ltv\":[0,0],\"vLtv\":[9000,0]},{\"d\":\"331244946983\",\"mW\":\"115792089237316195423570985008687907853269984665640564039457584007913129639935\",\"dP\":\"999728620000\",\"vP\":[\"999643500000\",\"999728620000\"],\"vVP\":[\"999643500000\",\"999728620000\"],\"ltv\":[0,0],\"vLtv\":[9000,0]}],\"cV\":\"0x39de0f00189306062d79edec6dca5bb6bfd108f9\",\"c\":[\"832634226392\",\"0\"]}","staticExtra":"{\"v0\":\"0x2137568666f12fc5A026f5430Ae7194F1C1362aB\",\"v1\":\"0x39dE0f00189306062D79eDEC6DcA5bb6bFd108f9\",\"ea\":\"0xA925fE59719e1253751fad11Ee4C73BfEE1b9B72\",\"f\":\"3000000000000\",\"pf\":\"0\",\"er0\":\"2656931030680\",\"er1\":\"1079191996534\",\"px\":\"1000193\",\"py\":\"1000000\",\"cx\":\"999985520758315300\",\"cy\":\"999985520758315300\",\"pfr\":\"0x0000000000000000000000000000000000000000\",\"evc\":\"0xddcbe30A761Edd2e19bba930A977475265F36Fa1\"}","blockNumber":69096336}`

	var pool entity.Pool
	err := json.Unmarshal([]byte(poolStr), &pool)
	require.Nil(t, err)

	s, err := NewPoolSimulator(pool)
	require.Nil(t, err)

	// Forward swap: AUSD -> USDC
	amountIn, _ := new(big.Int).SetString("1000000000", 10)
	tokenAmountIn := poolpkg.TokenAmount{
		Token:  "0x00000000efe302beaa2b3e6e1b18d08d69a9012a",
		Amount: amountIn,
	}
	tokenOut := "0xb97ef9ef8734c71904d8002f8b6bc66dd9c48a6e"

	forwardResult, err := testutil.MustConcurrentSafe(t, func() (*poolpkg.CalcAmountOutResult, error) {
		return s.CalcAmountOut(poolpkg.CalcAmountOutParams{
			TokenAmountIn: tokenAmountIn,
			TokenOut:      tokenOut,
		})
	})

	require.Nil(t, err)
	require.NotNil(t, forwardResult)

	s.UpdateBalance(poolpkg.UpdateBalanceParams{
		SwapInfo:       forwardResult.SwapInfo,
		TokenAmountIn:  tokenAmountIn,
		TokenAmountOut: *forwardResult.TokenAmountOut,
	})

	// Reverse swap: USDC -> AUSD
	reverseTokenAmountIn := poolpkg.TokenAmount{
		Token:  "0xb97ef9ef8734c71904d8002f8b6bc66dd9c48a6e",
		Amount: forwardResult.TokenAmountOut.Amount,
	}
	reverseTokenOut := "0x00000000efe302beaa2b3e6e1b18d08d69a9012a"

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

func TestMergeSwaps(t *testing.T) {
	t.Parallel()

	poolStr := `{"address":"0xe934cc9c5c49bbff0f8905c5bcfa65ce5e6de8a8","exchange":"uniswap-v4-euler","type":"euler-swap","timestamp":1758512001,"reserves":["3058437103820","677611878674"],"tokens":[{"address":"0x00000000efe302beaa2b3e6e1b18d08d69a9012a","symbol":"AUSD","decimals":6,"swappable":true},{"address":"0xb97ef9ef8734c71904d8002f8b6bc66dd9c48a6e","symbol":"USDC","decimals":6,"swappable":true}],"extra":"{\"p\":1,\"v\":[{\"c\":\"4812119143924\",\"d\":\"0\",\"mD\":\"52409500588666\",\"mW\":\"115792089237316195423570985008687907853269984665640564039457584007913129639935\",\"tB\":\"17778380267409\",\"eAA\":\"832634226392\",\"dP\":\"999643500000\",\"vP\":[\"999643500000\",\"999728620000\"],\"vVP\":[\"999643500000\",\"999728620000\"],\"ltv\":[0,0],\"vLtv\":[0,9000]},{\"c\":\"3989390539783\",\"d\":\"331244946983\",\"mD\":\"49793730393690\",\"mW\":\"115792089237316195423570985008687907853269984665640564039457584007913129639935\",\"tB\":\"46216879066526\",\"eAA\":\"0\",\"dP\":\"999728620000\",\"vP\":[\"999643500000\",\"999728620000\"],\"vVP\":[\"999643500000\",\"999728620000\"],\"ltv\":[0,0],\"vLtv\":[9000,0]},{\"d\":\"331244946983\",\"mW\":\"115792089237316195423570985008687907853269984665640564039457584007913129639935\",\"dP\":\"999728620000\",\"vP\":[\"999643500000\",\"999728620000\"],\"vVP\":[\"999643500000\",\"999728620000\"],\"ltv\":[0,0],\"vLtv\":[9000,0]}],\"cV\":\"0x39de0f00189306062d79edec6dca5bb6bfd108f9\",\"c\":[\"832634226392\",\"0\"]}","staticExtra":"{\"v0\":\"0x2137568666f12fc5A026f5430Ae7194F1C1362aB\",\"v1\":\"0x39dE0f00189306062D79eDEC6DcA5bb6bFd108f9\",\"ea\":\"0xA925fE59719e1253751fad11Ee4C73BfEE1b9B72\",\"f\":\"3000000000000\",\"pf\":\"0\",\"er0\":\"2656931030680\",\"er1\":\"1079191996534\",\"px\":\"1000193\",\"py\":\"1000000\",\"cx\":\"999985520758315300\",\"cy\":\"999985520758315300\",\"pfr\":\"0x0000000000000000000000000000000000000000\",\"evc\":\"0xddcbe30A761Edd2e19bba930A977475265F36Fa1\"}","blockNumber":69096336}`

	var pool entity.Pool
	err := json.Unmarshal([]byte(poolStr), &pool)
	require.Nil(t, err)

	testCases := []struct {
		name     string
		amountIn string
	}{
		{
			name:     "ok",
			amountIn: "20000000000000",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Single swap
			singlePool, err := NewPoolSimulator(pool)
			require.Nil(t, err)

			amountIn, _ := new(big.Int).SetString(tc.amountIn, 10)
			tokenAmountIn := poolpkg.TokenAmount{
				Token:  "0xb97ef9ef8734c71904d8002f8b6bc66dd9c48a6e",
				Amount: amountIn,
			}
			tokenOut := "0x00000000efe302beaa2b3e6e1b18d08d69a9012a"

			singleResult, singleErr := testutil.MustConcurrentSafe(t, func() (*poolpkg.CalcAmountOutResult, error) {
				return singlePool.CalcAmountOut(poolpkg.CalcAmountOutParams{
					TokenAmountIn: tokenAmountIn,
					TokenOut:      tokenOut,
				})
			})

			// Chunked swaps (20 chunks)
			chunkedPool, err := NewPoolSimulator(pool)
			require.Nil(t, err)

			chunkAmount := new(big.Int).Div(amountIn, big.NewInt(20))
			var totalAmountOut *big.Int
			var chunkedErr error

			for i := 0; i < 20; i++ {
				chunkTokenAmountIn := poolpkg.TokenAmount{
					Token:  "0xb97ef9ef8734c71904d8002f8b6bc66dd9c48a6e",
					Amount: chunkAmount,
				}

				chunkResult, err := testutil.MustConcurrentSafe(t, func() (*poolpkg.CalcAmountOutResult, error) {
					return chunkedPool.CalcAmountOut(poolpkg.CalcAmountOutParams{
						TokenAmountIn: chunkTokenAmountIn,
						TokenOut:      tokenOut,
					})
				})

				if err != nil {
					log.Println(i)
					chunkedErr = err
					break
				}

				chunkedPool.UpdateBalance(poolpkg.UpdateBalanceParams{
					SwapInfo:       chunkResult.SwapInfo,
					TokenAmountIn:  chunkTokenAmountIn,
					TokenAmountOut: *chunkResult.TokenAmountOut,
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
