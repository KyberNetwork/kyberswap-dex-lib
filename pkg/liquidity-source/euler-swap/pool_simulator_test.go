package eulerswap

import (
	"log"
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

var (
	poolSimulators = []string{
		`{"address":"0xe934cc9c5c49bbff0f8905c5bcfa65ce5e6de8a8","exchange":"uniswap-v4-euler","type":"euler-swap","timestamp":1758817768,"reserves":["3058437103820","677611878674"],"tokens":[{"address":"0x00000000efe302beaa2b3e6e1b18d08d69a9012a","symbol":"AUSD","decimals":6,"swappable":true},{"address":"0xb97ef9ef8734c71904d8002f8b6bc66dd9c48a6e","symbol":"USDC","decimals":6,"swappable":true}],"extra":"{\"p\":1,\"v\":[{\"c\":\"3627211859222\",\"d\":\"0\",\"mD\":\"55000119839546\",\"mW\":\"115792089237316195423570985008687907853269984665640564039457584007913129639935\",\"tB\":\"16372668301231\",\"eAA\":\"833077036013\",\"tS\":\"19361774502505\",\"dP\":\"999448260000\",\"vP\":[\"999448260000\",\"999740000000\"],\"vVP\":[\"999448260000\",\"999740000000\"],\"ltv\":[0,0],\"vLtv\":[0,9000]},{\"c\":\"2711184316235\",\"d\":\"331590920916\",\"mD\":\"51803226810771\",\"mW\":\"115792089237316195423570985008687907853269984665640564039457584007913129639935\",\"tB\":\"45485588872993\",\"eAA\":\"0\",\"tS\":\"46437441304445\",\"dP\":\"999740000000\",\"vP\":[\"999448260000\",\"999740000000\"],\"vVP\":[\"999448260000\",\"999740000000\"],\"ltv\":[0,0],\"vLtv\":[9000,0],\"iCE\":true},{\"d\":\"331590920916\",\"mW\":\"115792089237316195423570985008687907853269984665640564039457584007913129639935\",\"dP\":\"999740000000\",\"vP\":[\"999448260000\",\"999740000000\"],\"vVP\":[\"999448260000\",\"999740000000\"],\"ltv\":[0,0],\"vLtv\":[9000,0],\"iCE\":true}],\"cV\":\"0x39de0f00189306062d79edec6dca5bb6bfd108f9\",\"c\":[\"833077036013\",\"0\"]}","staticExtra":"{\"v0\":\"0x2137568666f12fc5A026f5430Ae7194F1C1362aB\",\"v1\":\"0x39dE0f00189306062D79eDEC6DcA5bb6bFd108f9\",\"ea\":\"0xA925fE59719e1253751fad11Ee4C73BfEE1b9B72\",\"f\":\"3000000000000\",\"pf\":\"0\",\"er0\":\"2656931030680\",\"er1\":\"1079191996534\",\"px\":\"1000193\",\"py\":\"1000000\",\"cx\":\"999985520758315300\",\"cy\":\"999985520758315300\",\"pfr\":\"0x0000000000000000000000000000000000000000\",\"evc\":\"0xddcbe30A761Edd2e19bba930A977475265F36Fa1\"}","blockNumber":69283230}`,
		`{"address":"0xe95aae4b43014396badffcd09918291c3d8da8a8","exchange":"uniswap-v4-euler","type":"euler-swap","timestamp":1758817767,"reserves":["641800","1728514"],"tokens":[{"address":"0x00000000efe302beaa2b3e6e1b18d08d69a9012a","symbol":"AUSD","decimals":6,"swappable":true},{"address":"0xb97ef9ef8734c71904d8002f8b6bc66dd9c48a6e","symbol":"USDC","decimals":6,"swappable":true}],"extra":"{\"p\":1,\"v\":[{\"c\":\"3627211859222\",\"d\":\"0\",\"mD\":\"55000119839546\",\"mW\":\"115792089237316195423570985008687907853269984665640564039457584007913129639935\",\"tB\":\"16372668301231\",\"eAA\":\"647070\",\"tS\":\"19361774502505\",\"dP\":\"999448260000\",\"vP\":[\"999740000000\",\"999448260000\"],\"vVP\":[\"999448260000\",\"999740000000\"],\"ltv\":[0,0],\"vLtv\":[0,9000]},{\"c\":\"2711184316235\",\"d\":\"0\",\"mD\":\"51803226810771\",\"mW\":\"115792089237316195423570985008687907853269984665640564039457584007913129639935\",\"tB\":\"45485588872993\",\"eAA\":\"1756531\",\"tS\":\"46437441304445\",\"dP\":\"999740000000\",\"vP\":[\"999740000000\",\"999448260000\"],\"vVP\":[\"999448260000\",\"999740000000\"],\"ltv\":[0,0],\"vLtv\":[9000,0]},null],\"c\":[\"1756531\",\"647070\"]}","staticExtra":"{\"v0\":\"0x2137568666f12fc5A026f5430Ae7194F1C1362aB\",\"v1\":\"0x39dE0f00189306062D79eDEC6DcA5bb6bFd108f9\",\"ea\":\"0x75102A2309cd305c7457a72397d0bCC000C4e044\",\"f\":\"10000000000000\",\"pf\":\"0\",\"er0\":\"0\",\"er1\":\"2370244\",\"px\":\"1000000\",\"py\":\"1000088\",\"cx\":\"999950000000000000\",\"cy\":\"999950000000000000\",\"pfr\":\"0x0000000000000000000000000000000000000000\",\"evc\":\"0xddcbe30A761Edd2e19bba930A977475265F36Fa1\"}","blockNumber":69283230}`,
		`{"address":"0xc96e5efa30d7e9705de171882044622ba0e068a8","exchange":"uniswap-v4-euler","type":"euler-swap","timestamp":1758817767,"reserves":["8566470461","1966979421"],"tokens":[{"address":"0x00000000efe302beaa2b3e6e1b18d08d69a9012a","symbol":"AUSD","decimals":6,"swappable":true},{"address":"0xb97ef9ef8734c71904d8002f8b6bc66dd9c48a6e","symbol":"USDC","decimals":6,"swappable":true}],"extra":"{\"p\":1,\"v\":[{\"c\":\"3627211859222\",\"d\":\"0\",\"mD\":\"55000119839546\",\"mW\":\"115792089237316195423570985008687907853269984665640564039457584007913129639935\",\"tB\":\"16372668301231\",\"eAA\":\"0\",\"tS\":\"19361774502505\",\"dP\":\"999448260000\",\"vP\":[\"999740000000\",\"999448260000\"],\"vVP\":[\"999448260000\",\"999740000000\"],\"ltv\":[0,0],\"vLtv\":[0,9000]},{\"c\":\"2711184316235\",\"d\":\"0\",\"mD\":\"51803226810771\",\"mW\":\"115792089237316195423570985008687907853269984665640564039457584007913129639935\",\"tB\":\"45485588872993\",\"eAA\":\"0\",\"tS\":\"46437441304445\",\"dP\":\"999740000000\",\"vP\":[\"999740000000\",\"999448260000\"],\"vVP\":[\"999448260000\",\"999740000000\"],\"ltv\":[0,0],\"vLtv\":[9000,0]},null],\"c\":[\"0\",\"0\"]}","staticExtra":"{\"v0\":\"0x2137568666f12fc5A026f5430Ae7194F1C1362aB\",\"v1\":\"0x39dE0f00189306062D79eDEC6DcA5bb6bFd108f9\",\"ea\":\"0x541622F9093cFD45390d9354A31614e5BBEFcBd0\",\"f\":\"70000000000000\",\"pf\":\"0\",\"er0\":\"4919694774\",\"er1\":\"5613381207\",\"px\":\"1000000\",\"py\":\"1000084\",\"cx\":\"999990000000000000\",\"cy\":\"999990000000000000\",\"pfr\":\"0x0000000000000000000000000000000000000000\",\"evc\":\"0xddcbe30A761Edd2e19bba930A977475265F36Fa1\"}","blockNumber":69283230}`,
		`{"address":"0xb21a2d348aa234eaadbe54b9f897da55c94128a8","exchange":"euler-swap","type":"euler-swap","timestamp":1758857160,"reserves":["610456622198","734053679371"],"tokens":[{"address":"0x176211869ca2b568f2a7d4ee941e073a821ee1ff","symbol":"USDC","decimals":6,"swappable":true},{"address":"0xaca92e438df0b2401ff60da7e4337b687a2435da","symbol":"mUSD","decimals":6,"swappable":true}],"extra":"{\"p\":1,\"v\":[{\"c\":\"14378841639620\",\"d\":\"28480922420\",\"mD\":\"52961607855958\",\"mW\":\"100000000000000\",\"tB\":\"52659550504421\",\"eAA\":\"0\",\"tS\":\"66795069765034\",\"dP\":\"999754800000\",\"vP\":[\"999754800000\",\"1000011880000\"],\"vVP\":[\"999754800000\",\"1000011880000\"],\"ltv\":[0,0],\"vLtv\":[0,9000],\"iCE\":true},{\"c\":\"10625797631570\",\"d\":\"0\",\"mD\":\"4464340468766\",\"mW\":\"20000000000000\",\"tB\":\"9909861899663\",\"eAA\":\"95883899109\",\"tS\":\"20529987101783\",\"dP\":\"1000011880000\",\"vP\":[\"999754800000\",\"1000011880000\"],\"vVP\":[\"999754800000\",\"1000011880000\"],\"ltv\":[0,0],\"vLtv\":[9000,0]},{\"d\":\"28480922420\",\"mW\":\"115792089237316195423570985008687907853269984665640564039457584007913129639935\",\"dP\":\"999754800000\",\"vP\":[\"999754800000\",\"1000011880000\"],\"vVP\":[\"999754800000\",\"1000011880000\"],\"ltv\":[0,0],\"vLtv\":[0,9000],\"iCE\":true}],\"cV\":\"0xfb6448b96637d90fcf2e4ad2c622a487d0496e6f\",\"c\":[\"0\",\"95883899109\"]}","staticExtra":"{\"v0\":\"0xfB6448B96637d90FcF2E4Ad2c622A487d0496e6f\",\"v1\":\"0xA7ada0D422a8b5FA4A7947F2CB0eE2D32435647d\",\"ea\":\"0x7480908a8cCF339Aa35888fb2Bf056A3482B089b\",\"f\":\"25000000000000\",\"pf\":\"0\",\"er0\":\"754099352747\",\"er1\":\"590412478160\",\"px\":\"1000000\",\"py\":\"1000013\",\"cx\":\"999990000000000000\",\"cy\":\"999990000000000000\",\"pfr\":\"0x0000000000000000000000000000000000000000\",\"evc\":\"0xd8CeCEe9A04eA3d941a959F68fb4486f23271d09\"}","blockNumber":23809742}`,
	}
)

func TestCalcAmountOut(t *testing.T) {
	t.Parallel()
	t.Run("swap USDC -> WETH", func(t *testing.T) {
		amountIn, _ := new(big.Int).SetString("1000", 10)

		tokenAmountIn := poolpkg.TokenAmount{
			Token:  "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			Amount: amountIn,
		}
		tokenOut := "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"

		expectedAmountOut := "365510616640"

		result, err := testutil.MustConcurrentSafe(t, func() (*poolpkg.CalcAmountOutResult, error) {
			// Parse pool data and create simulator
			var pool entity.Pool
			err := json.Unmarshal([]byte(poolSimulators[0]), &pool)
			require.Nil(t, err)

			s, err := NewPoolSimulator(pool)
			require.Nil(t, err)

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
			// Parse pool data and create simulator
			var pool entity.Pool
			err := json.Unmarshal([]byte(poolSimulators[0]), &pool)
			require.Nil(t, err)

			s, err := NewPoolSimulator(pool)
			require.Nil(t, err)

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
			// Parse pool data and create simulator
			var pool entity.Pool
			err := json.Unmarshal([]byte(poolSimulators[0]), &pool)
			require.Nil(t, err)

			s, err := NewPoolSimulator(pool)
			require.Nil(t, err)

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
			// Parse pool data and create simulator
			var pool entity.Pool
			err := json.Unmarshal([]byte(poolSimulators[0]), &pool)
			require.Nil(t, err)

			s, err := NewPoolSimulator(pool)
			require.Nil(t, err)

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

	var pool entity.Pool
	err := json.Unmarshal([]byte(poolSimulators[0]), &pool)
	require.Nil(t, err)

	poolSim, err := NewPoolSimulator(pool)
	require.Nil(t, err)

	testutil.TestCalcAmountIn(t, poolSim)
}

func TestSwapEdgeCases(t *testing.T) {
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

	testCases := []struct {
		name     string
		poolId   int
		amountIn string
	}{
		{
			poolId:   0,
			name:     "AUSD-USDC Pool (Avalanche) - Small reserves",
			amountIn: "1000000000",
		},
		{
			poolId:   1,
			name:     "AUSD-USDC Pool (Avalanche) - Medium reserves",
			amountIn: "1000000000",
		},
		{
			poolId:   2,
			name:     "AUSD-USDC Pool (Avalanche) - Large reserves with debt",
			amountIn: "1000000000",
		},
		{
			poolId:   3,
			name:     "USDC-mUSD Pool (Linea) - Small amount",
			amountIn: "1000000000",
		},
		{
			poolId:   3,
			name:     "USDC-mUSD Pool (Linea) - Large amount",
			amountIn: "1000000000000",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var pool entity.Pool
			err := json.Unmarshal([]byte(poolSimulators[tc.poolId]), &pool)
			require.Nil(t, err, "Failed to parse pool data for %s", tc.name)

			s, err := NewPoolSimulator(pool)
			require.Nil(t, err, "Failed to create simulator for %s", tc.name)

			amountIn, _ := new(big.Int).SetString(tc.amountIn, 10)
			tokenAmountIn := poolpkg.TokenAmount{
				Token:  pool.Tokens[0].Address,
				Amount: amountIn,
			}

			// Forward swap
			forwardResult, err := testutil.MustConcurrentSafe(t, func() (*poolpkg.CalcAmountOutResult, error) {
				return s.CalcAmountOut(poolpkg.CalcAmountOutParams{
					TokenAmountIn: tokenAmountIn,
					TokenOut:      pool.Tokens[1].Address,
				})
			})

			require.Nil(t, err, "Forward swap failed for %s", tc.name)
			require.NotNil(t, forwardResult, "Forward result is nil for %s", tc.name)

			// Reverse swap
			reverseResult, err := testutil.MustConcurrentSafe(t, func() (*poolpkg.CalcAmountOutResult, error) {
				return s.CalcAmountOut(poolpkg.CalcAmountOutParams{
					TokenAmountIn: *forwardResult.TokenAmountOut,
					TokenOut:      pool.Tokens[0].Address,
				})
			})

			require.Nil(t, err, "Reverse swap failed for %s", tc.name)
			require.NotNil(t, reverseResult, "Reverse result is nil for %s", tc.name)

			require.Less(t, reverseResult.TokenAmountOut.Amount.Cmp(amountIn), 0,
				"Reverse swap should return less than original due to fees for %s", tc.name)
			require.Greater(t, reverseResult.TokenAmountOut.Amount.Cmp(big.NewInt(0)), 0,
				"Reverse swap should return positive amount for %s", tc.name)

			t.Logf("Swap reversibility verified for %s: forward=%s, reverse=%s",
				tc.name, forwardResult.TokenAmountOut.Amount.String(), reverseResult.TokenAmountOut.Amount.String())
		})
	}
}

func TestMergeSwaps(t *testing.T) {
	t.Parallel()

	poolStr := `{"address":"0xe934cc9c5c49bbff0f8905c5bcfa65ce5e6de8a8","exchange":"uniswap-v4-euler","type":"euler-swap","timestamp":1758817768,"reserves":["3058437103820","677611878674"],"tokens":[{"address":"0x00000000efe302beaa2b3e6e1b18d08d69a9012a","symbol":"AUSD","decimals":6,"swappable":true},{"address":"0xb97ef9ef8734c71904d8002f8b6bc66dd9c48a6e","symbol":"USDC","decimals":6,"swappable":true}],"extra":"{\"p\":1,\"v\":[{\"c\":\"3627211859222\",\"d\":\"0\",\"mD\":\"55000119839546\",\"mW\":\"115792089237316195423570985008687907853269984665640564039457584007913129639935\",\"tB\":\"16372668301231\",\"eAA\":\"833077036013\",\"tS\":\"19361774502505\",\"dP\":\"999448260000\",\"vP\":[\"999448260000\",\"999740000000\"],\"vVP\":[\"999448260000\",\"999740000000\"],\"ltv\":[0,0],\"vLtv\":[0,9000]},{\"c\":\"2711184316235\",\"d\":\"331590920916\",\"mD\":\"51803226810771\",\"mW\":\"115792089237316195423570985008687907853269984665640564039457584007913129639935\",\"tB\":\"45485588872993\",\"eAA\":\"0\",\"tS\":\"46437441304445\",\"dP\":\"999740000000\",\"vP\":[\"999448260000\",\"999740000000\"],\"vVP\":[\"999448260000\",\"999740000000\"],\"ltv\":[0,0],\"vLtv\":[9000,0],\"iCE\":true},{\"d\":\"331590920916\",\"mW\":\"115792089237316195423570985008687907853269984665640564039457584007913129639935\",\"dP\":\"999740000000\",\"vP\":[\"999448260000\",\"999740000000\"],\"vVP\":[\"999448260000\",\"999740000000\"],\"ltv\":[0,0],\"vLtv\":[9000,0],\"iCE\":true}],\"cV\":\"0x39de0f00189306062d79edec6dca5bb6bfd108f9\",\"c\":[\"833077036013\",\"0\"]}","staticExtra":"{\"v0\":\"0x2137568666f12fc5A026f5430Ae7194F1C1362aB\",\"v1\":\"0x39dE0f00189306062D79eDEC6DcA5bb6bFd108f9\",\"ea\":\"0xA925fE59719e1253751fad11Ee4C73BfEE1b9B72\",\"f\":\"3000000000000\",\"pf\":\"0\",\"er0\":\"2656931030680\",\"er1\":\"1079191996534\",\"px\":\"1000193\",\"py\":\"1000000\",\"cx\":\"999985520758315300\",\"cy\":\"999985520758315300\",\"pfr\":\"0x0000000000000000000000000000000000000000\",\"evc\":\"0xddcbe30A761Edd2e19bba930A977475265F36Fa1\"}","blockNumber":69283230}`

	var pool entity.Pool
	err := json.Unmarshal([]byte(poolStr), &pool)
	require.Nil(t, err)

	testCases := []struct {
		name     string
		amountIn string
		tokenIn  int
		tokenOut int
	}{
		{
			name:     "0->1",
			amountIn: "1000000000",
			tokenIn:  0,
			tokenOut: 1,
		},
		{
			name:     "1->0",
			amountIn: "1000000000",
			tokenIn:  1,
			tokenOut: 0,
		},
		{
			name:     "ok",
			amountIn: "20000000000000",
			tokenIn:  0,
			tokenOut: 1,
		},
		{
			name:     "swap limit exceeds",
			amountIn: "200000000000000",
			tokenIn:  0,
			tokenOut: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Single swap
			singlePool, err := NewPoolSimulator(pool)
			require.Nil(t, err)

			amountIn, _ := new(big.Int).SetString(tc.amountIn, 10)
			tokenAmountIn := poolpkg.TokenAmount{
				Token:  pool.Tokens[tc.tokenIn].Address,
				Amount: amountIn,
			}
			tokenOut := pool.Tokens[tc.tokenOut].Address

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
					Token:  pool.Tokens[tc.tokenIn].Address,
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

func TestMergeSwaps2(t *testing.T) {
	t.Parallel()

	poolStr := `{"address":"0xe95aae4b43014396badffcd09918291c3d8da8a8","exchange":"uniswap-v4-euler","type":"euler-swap","timestamp":1758817767,"reserves":["641800","1728514"],"tokens":[{"address":"0x00000000efe302beaa2b3e6e1b18d08d69a9012a","symbol":"AUSD","decimals":6,"swappable":true},{"address":"0xb97ef9ef8734c71904d8002f8b6bc66dd9c48a6e","symbol":"USDC","decimals":6,"swappable":true}],"extra":"{\"p\":1,\"v\":[{\"c\":\"3627211859222\",\"d\":\"0\",\"mD\":\"55000119839546\",\"mW\":\"115792089237316195423570985008687907853269984665640564039457584007913129639935\",\"tB\":\"16372668301231\",\"eAA\":\"647070\",\"tS\":\"19361774502505\",\"dP\":\"999448260000\",\"vP\":[\"999740000000\",\"999448260000\"],\"vVP\":[\"999448260000\",\"999740000000\"],\"ltv\":[0,0],\"vLtv\":[0,9000]},{\"c\":\"2711184316235\",\"d\":\"0\",\"mD\":\"51803226810771\",\"mW\":\"115792089237316195423570985008687907853269984665640564039457584007913129639935\",\"tB\":\"45485588872993\",\"eAA\":\"1756531\",\"tS\":\"46437441304445\",\"dP\":\"999740000000\",\"vP\":[\"999740000000\",\"999448260000\"],\"vVP\":[\"999448260000\",\"999740000000\"],\"ltv\":[0,0],\"vLtv\":[9000,0]},null],\"c\":[\"1756531\",\"647070\"]}","staticExtra":"{\"v0\":\"0x2137568666f12fc5A026f5430Ae7194F1C1362aB\",\"v1\":\"0x39dE0f00189306062D79eDEC6DcA5bb6bFd108f9\",\"ea\":\"0x75102A2309cd305c7457a72397d0bCC000C4e044\",\"f\":\"10000000000000\",\"pf\":\"0\",\"er0\":\"0\",\"er1\":\"2370244\",\"px\":\"1000000\",\"py\":\"1000088\",\"cx\":\"999950000000000000\",\"cy\":\"999950000000000000\",\"pfr\":\"0x0000000000000000000000000000000000000000\",\"evc\":\"0xddcbe30A761Edd2e19bba930A977475265F36Fa1\"}","blockNumber":69283230}`

	var pool entity.Pool
	err := json.Unmarshal([]byte(poolStr), &pool)
	require.Nil(t, err)

	testCases := []struct {
		name     string
		amountIn string
		tokenIn  int
		tokenOut int
	}{
		{
			name:     "0->1",
			amountIn: "1000000000000",
			tokenIn:  0,
			tokenOut: 1,
		},
		{
			name:     "1->0",
			amountIn: "100000",
			tokenIn:  1,
			tokenOut: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Single swap
			singlePool, err := NewPoolSimulator(pool)
			require.Nil(t, err)

			amountIn, _ := new(big.Int).SetString(tc.amountIn, 10)
			tokenAmountIn := poolpkg.TokenAmount{
				Token:  pool.Tokens[tc.tokenIn].Address,
				Amount: amountIn,
			}
			tokenOut := pool.Tokens[tc.tokenOut].Address

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
					Token:  pool.Tokens[tc.tokenIn].Address,
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
