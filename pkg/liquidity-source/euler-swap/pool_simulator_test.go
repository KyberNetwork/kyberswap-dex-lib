package eulerswap

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

var (
	poolList = []string{
		`{"address":"0xe934cc9c5c49bbff0f8905c5bcfa65ce5e6de8a8","exchange":"uniswap-v4-euler","type":"euler-swap","timestamp":1758977005,"reserves":["3612687139261","123358663000"],"tokens":[{"address":"0x00000000efe302beaa2b3e6e1b18d08d69a9012a","symbol":"AUSD","decimals":6,"swappable":true},{"address":"0xb97ef9ef8734c71904d8002f8b6bc66dd9c48a6e","symbol":"USDC","decimals":6,"swappable":true}],"extra":"{\"p\":1,\"v\":[{\"c\":\"2183529047595\",\"d\":\"0\",\"mD\":\"55409954444138\",\"mW\":\"115792089237316195423570985008687907853269984665640564039457584007913129639935\",\"tB\":\"17406516508266\",\"eAA\":\"1387692615399\",\"dP\":\"999208320000\",\"vP\":[\"999208320000\",\"999699190000\"],\"vVP\":[\"999208320000\",\"999699190000\"],\"ltv\":[0,9000],\"vLtv\":[0,9000]},{\"c\":\"4036046080631\",\"d\":\"886742824999\",\"mD\":\"53132571159540\",\"mW\":\"115792089237316195423570985008687907853269984665640564039457584007913129639935\",\"tB\":\"42831382759828\",\"eAA\":\"0\",\"dP\":\"999699190000\",\"vP\":[\"999208320000\",\"999699190000\"],\"vVP\":[\"999208320000\",\"999699190000\"],\"ltv\":[9000,0],\"vLtv\":[9000,0],\"iCE\":true},{\"d\":\"886742824999\",\"mW\":\"115792089237316195423570985008687907853269984665640564039457584007913129639935\",\"dP\":\"999699190000\",\"vP\":[\"999208320000\",\"999699190000\"],\"vVP\":[\"999208320000\",\"999699190000\"],\"ltv\":[9000,0],\"vLtv\":[9000,0],\"iCE\":true}],\"cV\":\"0x39de0f00189306062d79edec6dca5bb6bfd108f9\",\"c\":[\"1387692615399\",\"0\"]}","staticExtra":"{\"v0\":\"0x2137568666f12fc5A026f5430Ae7194F1C1362aB\",\"v1\":\"0x39dE0f00189306062D79eDEC6DcA5bb6bFd108f9\",\"ea\":\"0xA925fE59719e1253751fad11Ee4C73BfEE1b9B72\",\"f\":\"3000000000000\",\"pf\":\"0\",\"er0\":\"2656931030680\",\"er1\":\"1079191996534\",\"px\":\"1000193\",\"py\":\"1000000\",\"cx\":\"999985520758315300\",\"cy\":\"999985520758315300\",\"pfr\":\"0x0000000000000000000000000000000000000000\",\"evc\":\"0xddcbe30A761Edd2e19bba930A977475265F36Fa1\"}","blockNumber":69382196}`,
		`{"address":"0xe95aae4b43014396badffcd09918291c3d8da8a8","exchange":"uniswap-v4-euler","type":"euler-swap","timestamp":1758977004,"reserves":["641800","1728514"],"tokens":[{"address":"0x00000000efe302beaa2b3e6e1b18d08d69a9012a","symbol":"AUSD","decimals":6,"swappable":true},{"address":"0xb97ef9ef8734c71904d8002f8b6bc66dd9c48a6e","symbol":"USDC","decimals":6,"swappable":true}],"extra":"{\"p\":1,\"v\":[{\"c\":\"2183529047595\",\"d\":\"0\",\"mD\":\"55409954444138\",\"mW\":\"115792089237316195423570985008687907853269984665640564039457584007913129639935\",\"tB\":\"17406516508266\",\"eAA\":\"647266\",\"dP\":\"999208320000\",\"vP\":[\"999699190000\",\"999208320000\"],\"vVP\":[\"999208320000\",\"999699190000\"],\"ltv\":[9000,0],\"vLtv\":[0,9000]},{\"c\":\"4036046080631\",\"d\":\"0\",\"mD\":\"53132571159540\",\"mW\":\"115792089237316195423570985008687907853269984665640564039457584007913129639935\",\"tB\":\"42831382759828\",\"eAA\":\"1758193\",\"dP\":\"999699190000\",\"vP\":[\"999699190000\",\"999208320000\"],\"vVP\":[\"999208320000\",\"999699190000\"],\"ltv\":[0,9000],\"vLtv\":[9000,0]},null],\"c\":[\"1758193\",\"647266\"]}","staticExtra":"{\"v0\":\"0x2137568666f12fc5A026f5430Ae7194F1C1362aB\",\"v1\":\"0x39dE0f00189306062D79eDEC6DcA5bb6bFd108f9\",\"ea\":\"0x75102A2309cd305c7457a72397d0bCC000C4e044\",\"f\":\"10000000000000\",\"pf\":\"0\",\"er0\":\"0\",\"er1\":\"2370244\",\"px\":\"1000000\",\"py\":\"1000088\",\"cx\":\"999950000000000000\",\"cy\":\"999950000000000000\",\"pfr\":\"0x0000000000000000000000000000000000000000\",\"evc\":\"0xddcbe30A761Edd2e19bba930A977475265F36Fa1\"}","blockNumber":69382196}`,
		`{"address":"0xc96e5efa30d7e9705de171882044622ba0e068a8","exchange":"uniswap-v4-euler","type":"euler-swap","timestamp":1758977004,"reserves":["8566470461","1966979421"],"tokens":[{"address":"0x00000000efe302beaa2b3e6e1b18d08d69a9012a","symbol":"AUSD","decimals":6,"swappable":true},{"address":"0xb97ef9ef8734c71904d8002f8b6bc66dd9c48a6e","symbol":"USDC","decimals":6,"swappable":true}],"extra":"{\"p\":1,\"v\":[{\"c\":\"2183529047595\",\"d\":\"0\",\"mD\":\"55409954444138\",\"mW\":\"115792089237316195423570985008687907853269984665640564039457584007913129639935\",\"tB\":\"17406516508266\",\"eAA\":\"0\",\"dP\":\"999208320000\",\"vP\":[\"999699190000\",\"999208320000\"],\"vVP\":[\"999208320000\",\"999699190000\"],\"ltv\":[9000,0],\"vLtv\":[0,9000]},{\"c\":\"4036046080631\",\"d\":\"0\",\"mD\":\"53132571159540\",\"mW\":\"115792089237316195423570985008687907853269984665640564039457584007913129639935\",\"tB\":\"42831382759828\",\"eAA\":\"0\",\"dP\":\"999699190000\",\"vP\":[\"999699190000\",\"999208320000\"],\"vVP\":[\"999208320000\",\"999699190000\"],\"ltv\":[0,9000],\"vLtv\":[9000,0]},null],\"c\":[\"0\",\"0\"]}","staticExtra":"{\"v0\":\"0x2137568666f12fc5A026f5430Ae7194F1C1362aB\",\"v1\":\"0x39dE0f00189306062D79eDEC6DcA5bb6bFd108f9\",\"ea\":\"0x541622F9093cFD45390d9354A31614e5BBEFcBd0\",\"f\":\"70000000000000\",\"pf\":\"0\",\"er0\":\"4919694774\",\"er1\":\"5613381207\",\"px\":\"1000000\",\"py\":\"1000084\",\"cx\":\"999990000000000000\",\"cy\":\"999990000000000000\",\"pfr\":\"0x0000000000000000000000000000000000000000\",\"evc\":\"0xddcbe30A761Edd2e19bba930A977475265F36Fa1\"}","blockNumber":69382196}`,
		`{"address":"0xb21a2d348aa234eaadbe54b9f897da55c94128a8","exchange":"euler-swap","type":"euler-swap","timestamp":1758976889,"reserves":["770726182705","573785869166"],"tokens":[{"address":"0x176211869ca2b568f2a7d4ee941e073a821ee1ff","symbol":"USDC","decimals":6,"swappable":true},{"address":"0xaca92e438df0b2401ff60da7e4337b687a2435da","symbol":"mUSD","decimals":6,"swappable":true}],"extra":"{\"p\":1,\"v\":[{\"c\":\"13156282077952\",\"d\":\"0\",\"mD\":\"55029412101035\",\"mW\":\"100000000000000\",\"tB\":\"51814305821012\",\"eAA\":\"131904555000\",\"dP\":\"999675110000\",\"vP\":[\"999675110000\",\"1000007960000\"],\"vVP\":[\"999675110000\",\"1000007960000\"],\"ltv\":[0,9000],\"vLtv\":[0,9000]},{\"c\":\"14696920107493\",\"d\":\"64341224805\",\"mD\":\"6617214231\",\"mW\":\"20000000000000\",\"tB\":\"10296462678275\",\"eAA\":\"0\",\"dP\":\"1000007960000\",\"vP\":[\"999675110000\",\"1000007960000\"],\"vVP\":[\"999675110000\",\"1000007960000\"],\"ltv\":[9000,0],\"vLtv\":[9000,0],\"iCE\":true},{\"d\":\"64341224805\",\"mW\":\"115792089237316195423570985008687907853269984665640564039457584007913129639935\",\"dP\":\"1000007960000\",\"vP\":[\"999675110000\",\"1000007960000\"],\"vVP\":[\"999675110000\",\"1000007960000\"],\"ltv\":[9000,0],\"vLtv\":[9000,0],\"iCE\":true}],\"cV\":\"0xa7ada0d422a8b5fa4a7947f2cb0ee2d32435647d\",\"c\":[\"131904555000\",\"0\"]}","staticExtra":"{\"v0\":\"0xfB6448B96637d90FcF2E4Ad2c622A487d0496e6f\",\"v1\":\"0xA7ada0D422a8b5FA4A7947F2CB0eE2D32435647d\",\"ea\":\"0x7480908a8cCF339Aa35888fb2Bf056A3482B089b\",\"f\":\"25000000000000\",\"pf\":\"0\",\"er0\":\"754099352747\",\"er1\":\"590412478160\",\"px\":\"1000000\",\"py\":\"1000013\",\"cx\":\"999990000000000000\",\"cy\":\"999990000000000000\",\"pfr\":\"0x0000000000000000000000000000000000000000\",\"evc\":\"0xd8CeCEe9A04eA3d941a959F68fb4486f23271d09\"}","blockNumber":23866419}`,
		`{"address":"0x0daa7a2eb668131e1b353aaa4cb2e0cf6b66e8a8","exchange":"uniswap-v4-euler","type":"euler-swap","timestamp":1758976169,"reserves":["2000000000","2000000000"],"tokens":[{"address":"0x078d782b760474a361dda0af3839290b0ef57ad6","symbol":"USDC","decimals":6,"swappable":true},{"address":"0x4200000000000000000000000000000000000006","symbol":"WETH","decimals":18,"swappable":true}],"extra":"{\"p\":1,\"v\":[{\"c\":\"633692842061\",\"d\":\"0\",\"mD\":\"142968037164078\",\"mW\":\"135000000000000\",\"tB\":\"6398269993860\",\"eAA\":\"0\",\"dP\":\"999745440000\",\"vP\":[\"999745440000\",\"3996\"],\"vVP\":[\"999745440000\",\"3996\"],\"ltv\":[0,8400],\"vLtv\":[0,8400]},{\"c\":\"286634179865276490710\",\"d\":\"0\",\"mD\":\"13326014643199570095130\",\"mW\":\"13500000000000000000000\",\"tB\":\"1387351176935153414159\",\"eAA\":\"0\",\"dP\":\"3996\",\"vP\":[\"999745440000\",\"3996\"],\"vVP\":[\"999745440000\",\"3996\"],\"ltv\":[8400,0],\"vLtv\":[8400,0]},null],\"c\":[\"0\",\"0\"]}","staticExtra":"{\"v0\":\"0x6eAe95ee783e4D862867C4e0E4c3f4B95AA682Ba\",\"v1\":\"0x1f3134C3f3f8AdD904B9635acBeFC0eA0D0E1ffC\",\"ea\":\"0x29d5ea019FA72B489C44F15b7E95771b399D37Ef\",\"f\":\"10000000000000000\",\"pf\":\"0\",\"er0\":\"2000000000\",\"er1\":\"2000000000\",\"px\":\"1000000000000000000\",\"py\":\"1000000000000000000\",\"cx\":\"970000000000000000\",\"cy\":\"970000000000000000\",\"pfr\":\"0x0000000000000000000000000000000000000000\",\"evc\":\"0x2A1176964F5D7caE5406B627Bf6166664FE83c60\"}","blockNumber":28227809}`,
		`{"address":"0x4fd5691b59387c8a8857cfb7ff21a8ab80d468a8","exchange":"uniswap-v4-euler","type":"euler-swap","timestamp":1759467837,"reserves":["9104412972181","1308782626335"],"tokens":[{"address":"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48","symbol":"USDC","decimals":6,"swappable":true},{"address":"0xdac17f958d2ee523a2206206994597c13d831ec7","symbol":"USDT","decimals":6,"swappable":true}],"extra":"{\"p\":1,\"v\":[{\"c\":\"1634030603502\",\"d\":\"8699369666729\",\"mD\":\"58558807444541\",\"mW\":\"67500000000000\",\"tB\":\"14807161951956\",\"eAA\":\"0\",\"dP\":\"999725600000\",\"vP\":[\"999725600000\",\"1000400000000\"],\"vVP\":[\"999725600000\",\"1000400000000\"],\"ltv\":[0,9300],\"vLtv\":[0,9300],\"iCE\":true},{\"c\":\"3232808083027\",\"d\":\"0\",\"mD\":\"37076294613446\",\"mW\":\"45000000000000\",\"tB\":\"9690897303526\",\"eAA\":\"10776818245665\",\"dP\":\"1000400000000\",\"vP\":[\"999725600000\",\"1000400000000\"],\"vVP\":[\"999725600000\",\"1000400000000\"],\"ltv\":[9300,0],\"vLtv\":[9300,0]},{\"d\":\"8699369666729\",\"mW\":\"115792089237316195423570985008687907853269984665640564039457584007913129639935\",\"dP\":\"999725600000\",\"vP\":[\"999725600000\",\"1000400000000\"],\"vVP\":[\"999725600000\",\"1000400000000\"],\"ltv\":[0,9300],\"vLtv\":[0,9300],\"iCE\":true}],\"cV\":\"0x797dd80692c3b2dadabce8e30c07fde5307d48a9\",\"c\":[\"0\",\"10776818245665\"]}","staticExtra":"{\"v0\":\"0x797DD80692c3b2dAdabCe8e30C07fDE5307D48a9\",\"v1\":\"0x313603FA690301b0CaeEf8069c065862f9162162\",\"ea\":\"0x0Afbf798467f9b3b97f90d05bF7df592D89A6cF6\",\"f\":\"1000000000000\",\"pf\":\"0\",\"er0\":\"5399367592314\",\"er1\":\"5012600096421\",\"px\":\"1000000\",\"py\":\"1000190\",\"cx\":\"999950000000000000\",\"cy\":\"999950000000000000\",\"pfr\":\"0x0000000000000000000000000000000000000000\",\"evc\":\"0x0C9a3dd6b8F28529d72d7f9cE918D493519EE383\"}","blockNumber":23495040}`,
	}
)

func TestCalcAmountOut(t *testing.T) {
	t.Parallel()
	t.Run("swap USDC -> mUSD", func(t *testing.T) {
		amountIn, _ := new(big.Int).SetString("1000000", 10)

		tokenAmountIn := poolpkg.TokenAmount{
			Token:  "0x176211869ca2b568f2a7d4ee941e073a821ee1ff",
			Amount: amountIn,
		}
		tokenOut := "0xaca92e438df0b2401ff60da7e4337b687a2435da"

		expectedAmountOut := "999960"

		var pool entity.Pool
		err := json.Unmarshal([]byte(poolList[3]), &pool)
		require.Nil(t, err)

		s, err := NewPoolSimulator(pool)
		require.Nil(t, err)

		result, err := testutil.MustConcurrentSafe(t, func() (*poolpkg.CalcAmountOutResult, error) {
			return s.CalcAmountOut(poolpkg.CalcAmountOutParams{
				TokenAmountIn: tokenAmountIn,
				TokenOut:      tokenOut,
			})
		})

		require.Nil(t, err)
		require.Equal(t, expectedAmountOut, result.TokenAmountOut.Amount.String())
	})

	t.Run("swap mUSD -> USDC", func(t *testing.T) {
		amountIn, _ := new(big.Int).SetString("1000000", 10)

		tokenAmountIn := poolpkg.TokenAmount{
			Token:  "0xaca92e438df0b2401ff60da7e4337b687a2435da",
			Amount: amountIn,
		}
		tokenOut := "0x176211869ca2b568f2a7d4ee941e073a821ee1ff"

		expectedAmountOut := "999988"

		var pool entity.Pool
		err := json.Unmarshal([]byte(poolList[3]), &pool)
		require.Nil(t, err)

		s, err := NewPoolSimulator(pool)
		require.Nil(t, err)

		result, err := testutil.MustConcurrentSafe(t, func() (*poolpkg.CalcAmountOutResult, error) {
			return s.CalcAmountOut(poolpkg.CalcAmountOutParams{
				TokenAmountIn: tokenAmountIn,
				TokenOut:      tokenOut,
			})
		})

		require.Nil(t, err)
		require.Equal(t, expectedAmountOut, result.TokenAmountOut.Amount.String())
	})

	t.Run("swap mUSD -> USDC : swap limit exceed", func(t *testing.T) {
		amountIn, _ := new(big.Int).SetString("100000000000000", 10)

		tokenAmountIn := poolpkg.TokenAmount{
			Token:  "0xaca92e438df0b2401ff60da7e4337b687a2435da",
			Amount: amountIn,
		}
		tokenOut := "0x176211869ca2b568f2a7d4ee941e073a821ee1ff"

		var pool entity.Pool
		err := json.Unmarshal([]byte(poolList[3]), &pool)
		require.Nil(t, err)

		s, err := NewPoolSimulator(pool)
		require.Nil(t, err)

		result, err := testutil.MustConcurrentSafe(t, func() (*poolpkg.CalcAmountOutResult, error) {
			return s.CalcAmountOut(poolpkg.CalcAmountOutParams{
				TokenAmountIn: tokenAmountIn,
				TokenOut:      tokenOut,
			})
		})

		require.Nil(t, result)
		require.ErrorIs(t, err, ErrSwapLimitExceeded)
	})

	t.Run("swap USDC -> mUSD : swap limit exceeded", func(t *testing.T) {
		amountIn, _ := new(big.Int).SetString("100000000000000", 10)

		tokenAmountIn := poolpkg.TokenAmount{
			Token:  "0x176211869ca2b568f2a7d4ee941e073a821ee1ff",
			Amount: amountIn,
		}
		tokenOut := "0xaca92e438df0b2401ff60da7e4337b687a2435da"

		var pool entity.Pool
		err := json.Unmarshal([]byte(poolList[3]), &pool)
		require.Nil(t, err)

		s, err := NewPoolSimulator(pool)
		require.Nil(t, err)

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

	var pool entity.Pool
	err := json.Unmarshal([]byte(poolList[1]), &pool)
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

	// Test cases: [poolId, amountIn, description]
	testCases := []struct {
		poolId      int
		amountIn    string
		description string
	}{
		{0, "1000000000", "AUSD-USDC (Small reserves)"},
		{1, "1000000000", "AUSD-USDC (Medium reserves)"},
		{3, "1000000000", "USDC-mUSD (Small amount)"},
		{3, "10000000000", "USDC-mUSD (Large amount)"},
		{5, "1000000", "USDC-USDT (Small amount)"},
		{5, "1000000000", "USDC-USDT (Medium amount)"},
		{5, "10000000000000", "USDC-USDT (Large amount)"},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			var pool entity.Pool
			err := json.Unmarshal([]byte(poolList[tc.poolId]), &pool)
			require.Nil(t, err, "Failed to parse pool data for %s", tc.description)

			s, err := NewPoolSimulator(pool)
			require.Nil(t, err, "Failed to create simulator for %s", tc.description)

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

			require.Nil(t, err, "Forward swap failed for %s", tc.description)
			require.NotNil(t, forwardResult, "Forward result is nil for %s", tc.description)

			// Reverse swap
			reverseResult, err := testutil.MustConcurrentSafe(t, func() (*poolpkg.CalcAmountOutResult, error) {
				return s.CalcAmountOut(poolpkg.CalcAmountOutParams{
					TokenAmountIn: *forwardResult.TokenAmountOut,
					TokenOut:      pool.Tokens[0].Address,
				})
			})

			require.Nil(t, err, "Reverse swap failed for %s", tc.description)
			require.NotNil(t, reverseResult, "Reverse result is nil for %s", tc.description)

			require.Less(t, reverseResult.TokenAmountOut.Amount.Cmp(amountIn), 0,
				"Reverse swap should return less than original due to fees for %s", tc.description)
			require.Greater(t, reverseResult.TokenAmountOut.Amount.Cmp(big.NewInt(0)), 0,
				"Reverse swap should return positive amount for %s", tc.description)

			t.Logf("Swap reversibility verified for %s: forward=%s, reverse=%s",
				tc.description, forwardResult.TokenAmountOut.Amount.String(), reverseResult.TokenAmountOut.Amount.String())
		})
	}
}

func TestMergeSwaps(t *testing.T) {
	// Test cases: [poolId, amountIn, direction]
	testCases := []struct {
		poolId    int
		amountIn  string
		direction string
	}{
		{0, "1000000000", "0->1"},
		{0, "1000000000", "1->0"},
		{1, "200000000000000", "0->1"},
		{1, "200000000000000", "1->0"},
		{2, "200000000000000", "0->1"},
		{2, "200000000000000", "1->0"},
		{3, "200000000000000", "0->1"},
		{3, "200000000000000", "1->0"},
		{4, "25595147392", "1->0"},
	}

	for _, tc := range testCases {
		t.Run(tc.direction, func(t *testing.T) {
			var pool entity.Pool
			err := json.Unmarshal([]byte(poolList[tc.poolId]), &pool)
			require.Nil(t, err)

			tokenIn, tokenOut := 0, 1
			if tc.direction == "1->0" {
				tokenIn, tokenOut = 1, 0
			}

			// Single swap
			singlePool, err := NewPoolSimulator(pool)
			require.Nil(t, err)

			amountIn, _ := new(big.Int).SetString(tc.amountIn, 10)
			tokenAmountIn := poolpkg.TokenAmount{
				Token:  pool.Tokens[tokenIn].Address,
				Amount: amountIn,
			}
			tokenOutAddr := pool.Tokens[tokenOut].Address

			singleResult, singleErr := testutil.MustConcurrentSafe(t, func() (*poolpkg.CalcAmountOutResult, error) {
				return singlePool.CalcAmountOut(poolpkg.CalcAmountOutParams{
					TokenAmountIn: tokenAmountIn,
					TokenOut:      tokenOutAddr,
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
					Token:  pool.Tokens[tokenIn].Address,
					Amount: chunkAmount,
				}

				chunkResult, err := testutil.MustConcurrentSafe(t, func() (*poolpkg.CalcAmountOutResult, error) {
					return chunkedPool.CalcAmountOut(poolpkg.CalcAmountOutParams{
						TokenAmountIn: chunkTokenAmountIn,
						TokenOut:      tokenOutAddr,
					})
				})

				if err != nil {
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
