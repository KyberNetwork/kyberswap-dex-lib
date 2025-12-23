package gateway

import (
	"math/big"
	"strings"
	"testing"

	"github.com/goccy/go-json"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

func getPoolSim(isPaused bool) *PoolSimulator {
	extraStr := `{"address":"0x3f04b65ddbd87f9ce0a2e7eb24d80e7fb87625b5","exchange":"infinifi","type":"infinifi-gateway","timestamp":1766298309,"reserves":["1000000000000000000000000","104131454683989908935933065","99563982997168197725467844","15652134235785515737864774","319965469459800257231810","1000000000000000000","3166349483701620253919558","1000000000000000000","11380467428921897117173","1000000000000000000","3581390415730186225765584","1000000000000000000","1000000000000000000","1000000000000000000","1000000000000000000","963294344644093134158921"],"tokens":[{"address":"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48","symbol":"USDC","decimals":6,"swappable":true},{"address":"0x48f9e38f3070ad8945dfeae3fa70987722e3d89c","symbol":"iUSD","decimals":18,"swappable":true},{"address":"0xdbdc1ef57537e34680b898e1febd3d68c7389bcb","symbol":"siUSD","decimals":18,"swappable":true},{"address":"0x12b004719fb632f1e7c010c6f5d6009fb4258442","symbol":"liUSD-1w","decimals":18,"swappable":true},{"address":"0xf1839becaf586814d022f16cdb3504ff8d8ff361","symbol":"liUSD-2w","decimals":18,"swappable":true},{"address":"0xed2a360ffdc1ed4f8df0bd776a1ffbbe06444a0a","symbol":"liUSD-3w","decimals":18,"swappable":true},{"address":"0x66bcf6151d5558afb47c38b20663589843156078","symbol":"liUSD-4w","decimals":18,"swappable":true},{"address":"0xf0c4a78febf4062aed39a02be8a4c72e9857d7d1","symbol":"liUSD-5w","decimals":18,"swappable":true},{"address":"0xb06cc4548febff3d66a680f9c516381c79bc9707","symbol":"liUSD-6w","decimals":18,"swappable":true},{"address":"0x3a744a6b57984eb62aeb36eb6501d268372cf8bb","symbol":"liUSD-7w","decimals":18,"swappable":true},{"address":"0xf68b95b7e851170c0e5123a3249dd1ca46215085","symbol":"liUSD-8w","decimals":18,"swappable":true},{"address":"0xbb5ca732fafed8870f9c0e8406ad707939c912e1","symbol":"liUSD-9w","decimals":18,"swappable":true},{"address":"0xd15fbf48c6dddadc9ef0693b060d80af51cc26d5","symbol":"liUSD-10w","decimals":18,"swappable":true},{"address":"0xed030a37ec6eb308a416dc64dd4b649a2bbe4fcd","symbol":"liUSD-11w","decimals":18,"swappable":true},{"address":"0x3d360ab96b942c1251ab061178f731efebc2d644","symbol":"liUSD-12w","decimals":18,"swappable":true},{"address":"0xbd3f9814eb946e617f1d774a6762cdbec0bf087a","symbol":"liUSD-13w","decimals":18,"swappable":true}],"extra":"{\"isPaused\":false,\"iusdSupply\":1000000000000000000000000,\"siusdTotalAssets\":1000000000000000000000000,\"siusdSupply\":500000000000000000000000,\"liusdBuckets\":[{\"index\":1,\"totalSupply\":1000000000000000000000000,\"bucketData\":{\"shareToken\":\"0x12b004719fb632f1e7c010c6f5d6009fb4258442\",\"totalReceiptTokens\":1000000000000000000000000,\"multiplier\":1263000000000000000}},{\"index\":2,\"totalSupply\":800000000000000000000000,\"bucketData\":{\"shareToken\":\"0xf1839becaf586814d022f16cdb3504ff8d8ff361\",\"totalReceiptTokens\":1000000000000000000000000,\"multiplier\":1310000000000000000}},{\"index\":3,\"totalSupply\":1000000000000000000,\"bucketData\":{\"shareToken\":\"0xed2a360ffdc1ed4f8df0bd776a1ffbbe06444a0a\",\"totalReceiptTokens\":1153620771798657576,\"multiplier\":1310000000000000000}},{\"index\":4,\"totalSupply\":3166349483701620253919558,\"bucketData\":{\"shareToken\":\"0x66bcf6151d5558afb47c38b20663589843156078\",\"totalReceiptTokens\":3660547616921391472398769,\"multiplier\":1358000000000000000}},{\"index\":5,\"totalSupply\":1000000000000000000,\"bucketData\":{\"shareToken\":\"0xf0c4a78febf4062aed39a02be8a4c72e9857d7d1\",\"totalReceiptTokens\":1159647234013212202,\"multiplier\":1358000000000000000}},{\"index\":6,\"totalSupply\":11380467428921897117173,\"bucketData\":{\"shareToken\":\"0xb06cc4548febff3d66a680f9c516381c79bc9707\",\"totalReceiptTokens\":13213930712512969941799,\"multiplier\":1386000000000000000}},{\"index\":7,\"totalSupply\":1000000000000000000,\"bucketData\":{\"shareToken\":\"0x3a744a6b57984eb62aeb36eb6501d268372cf8bb\",\"totalReceiptTokens\":1163095237516620378,\"multiplier\":1386000000000000000}},{\"index\":8,\"totalSupply\":3581390415730186225765584,\"bucketData\":{\"shareToken\":\"0xf68b95b7e851170c0e5123a3249dd1ca46215085\",\"totalReceiptTokens\":4169246714536778490165271,\"multiplier\":1406000000000000000}},{\"index\":9,\"totalSupply\":1000000000000000000,\"bucketData\":{\"shareToken\":\"0xbb5ca732fafed8870f9c0e8406ad707939c912e1\",\"totalReceiptTokens\":1165736580133285184,\"multiplier\":1406000000000000000}},{\"index\":10,\"totalSupply\":1000000000000000000,\"bucketData\":{\"shareToken\":\"0xd15fbf48c6dddadc9ef0693b060d80af51cc26d5\",\"totalReceiptTokens\":1165736580133285184,\"multiplier\":1406000000000000000}},{\"index\":11,\"totalSupply\":1000000000000000000,\"bucketData\":{\"shareToken\":\"0xed030a37ec6eb308a416dc64dd4b649a2bbe4fcd\",\"totalReceiptTokens\":1165736580133285184,\"multiplier\":1406000000000000000}},{\"index\":12,\"totalSupply\":1000000000000000000,\"bucketData\":{\"shareToken\":\"0x3d360ab96b942c1251ab061178f731efebc2d644\",\"totalReceiptTokens\":1165736580133285184,\"multiplier\":1406000000000000000}},{\"index\":13,\"totalSupply\":963294344644093134158921,\"bucketData\":{\"shareToken\":\"0xbd3f9814eb946e617f1d774a6762cdbec0bf087a\",\"totalReceiptTokens\":1126949876405305527769314,\"multiplier\":1440000000000000000}}]}","blockNumber":24059168}`
	if isPaused {
		extraStr = strings.Replace(extraStr, `\"isPaused\":false`, `\"isPaused\":true`, 1)
	}
	var entityPool entity.Pool
	_ = json.Unmarshal([]byte(extraStr),
		&entityPool)
	return lo.Must(NewPoolSimulator(entityPool))
}

func TestPoolSimulator_CalcAmountOut(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name      string
		indexIn   int
		indexOut  int
		amountIn  string
		amountOut string
		poolSim   *PoolSimulator
	}{
		{name: "USDC -> iUSD (mint) - 1 USDC to 1 iUSD",
			indexIn: 0, indexOut: 1, amountIn: "1000000", amountOut: "1000000000000000000"},
		{name: "iUSD -> USDC (redeem) - 1 iUSD to 1 USDC",
			indexIn: 1, indexOut: 0, amountIn: "1000000000000000000", amountOut: "1000000"},
		{name: "iUSD -> siUSD (stake) - 1 iUSD to 0.5 siUSD (2:1 rate)",
			indexIn: 1, indexOut: 2, amountIn: "1000000000000000000", amountOut: "500000000000000000"},
		{name: "siUSD -> iUSD (unstake) - 1 siUSD to 2 iUSD (2:1 rate)",
			indexIn: 2, indexOut: 1, amountIn: "1000000000000000000", amountOut: "2000000000000000000"},
		{name: "iUSD -> liUSD-1mo (lock) - 1 iUSD to 1 liUSD",
			indexIn: 1, indexOut: 3, amountIn: "1000000000000000000", amountOut: "1000000000000000000"},
		{name: "iUSD -> liUSD-2mo (lock) - 1 iUSD to 0.8 liUSD (0.8:1 rate)",
			indexIn: 1, indexOut: 4, amountIn: "1000000000000000000", amountOut: "800000000000000000"},
		{name: "Contract paused should fail",
			indexIn: 0, indexOut: 1, amountIn: "1000000", amountOut: "", poolSim: getPoolSim(true)},
		{name: "USDC -> siUSD (mintAndStake) - 1 USDC to 0.5 siUSD",
			indexIn: 0, indexOut: 2, amountIn: "1000000", amountOut: "500000000000000000"},
		{name: "USDC -> liUSD-1mo (mintAndLock) - 1 USDC to 1 liUSD",
			indexIn: 0, indexOut: 3, amountIn: "1000000", amountOut: "1000000000000000000"},
		{name: "USDC -> liUSD-2mo (mintAndLock) - 1 USDC to 0.8 liUSD",
			indexIn: 0, indexOut: 4, amountIn: "1000000", amountOut: "800000000000000000"},
		{name: "Unsupported swap (siUSD -> USDC) should fail",
			indexIn: 2, indexOut: 0, amountIn: "1000000000000000000", amountOut: ""},
	}
	poolSim := getPoolSim(false)
	for _, tc := range testCases {
		t.Run("", func(t *testing.T) {
			testutil.TestCalcAmountOut(t, lo.Ternary(tc.poolSim != nil, tc.poolSim, poolSim), map[int]map[int]map[string]string{
				tc.indexIn: {
					tc.indexOut: {
						tc.amountIn: tc.amountOut,
					},
				},
			})
		})
	}
}

func TestPoolSimulator_UpdateBalance(t *testing.T) {
	// Actual token addresses from ethereum.json
	usdcAddr := "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"
	iusdAddr := "0x48f9e38f3070ad8945dfae3fa70987722e3d89c"
	siusdAddr := "0xdbdc1ef57537e34680b898e1febd3d68c7389bcb"
	liusd1moAddr := "0x12b004719fb632f1e7c010c6f5d6009fb4258442"
	gatewayAddr := "0x3f04b65ddbd87f9ce0a2e7eb24d80e7fb87625b5"

	initialIUSD := mustParseBig("1000000000000000000000000")
	initialSIUSDAssets := mustParseBig("1000000000000000000000000")
	initialSIUSDSupply := mustParseBig("500000000000000000000000")
	initialLIUSD1moSupply := mustParseBig("1000000000000000000000000")
	initialLIUSD1moReceipts := mustParseBig("1000000000000000000000000")

	poolExtra := Extra{
		IsPaused:         false,
		IUSDSupply:       initialIUSD,
		SIUSDTotalAssets: initialSIUSDAssets,
		SIUSDSupply:      initialSIUSDSupply,
		LIUSDBuckets:     []bucket{{TotalSupply: initialLIUSD1moSupply, BucketData: bucketData{TotalReceiptTokens: initialLIUSD1moReceipts}}},
	}

	poolEntity := entity.Pool{
		Address:  gatewayAddr,
		Exchange: "infinifi",
		Type:     DexType,
		Tokens: []*entity.PoolToken{
			{Address: usdcAddr, Decimals: 6, Swappable: true},
			{Address: iusdAddr, Decimals: 18, Swappable: true},
			{Address: siusdAddr, Decimals: 18, Swappable: true},
			{Address: liusd1moAddr, Decimals: 18, Swappable: true},
		},
		Reserves: []string{
			poolExtra.SIUSDTotalAssets.String(),
			poolExtra.SIUSDSupply.String(),
			poolExtra.LIUSDBuckets[0].TotalSupply.String(),
		},
	}

	extraBytes, err := json.Marshal(poolExtra)
	require.NoError(t, err)
	poolEntity.Extra = string(extraBytes)

	// Test 1: USDC → iUSD (mint)
	t.Run("USDC -> iUSD (mint)", func(t *testing.T) {
		simulator, err := NewPoolSimulator(poolEntity)
		require.NoError(t, err)

		amountInUSDC := mustParseBig("1000000")              // 1 USDC
		amountOutIUSD := mustParseBig("1000000000000000000") // 1 iUSD

		simulator.UpdateBalance(pool.UpdateBalanceParams{
			TokenAmountIn:  pool.TokenAmount{Token: usdcAddr, Amount: amountInUSDC},
			TokenAmountOut: pool.TokenAmount{Token: iusdAddr, Amount: amountOutIUSD},
			Fee:            pool.TokenAmount{Token: usdcAddr, Amount: big.NewInt(0)},
		})

		expectedIUSD := new(big.Int).Add(initialIUSD, amountOutIUSD)
		assert.Equal(t, expectedIUSD.String(), simulator.IUSDSupply.String())
	})

	// Test 2: iUSD → USDC (redeem)
	t.Run("iUSD -> USDC (redeem)", func(t *testing.T) {
		simulator, err := NewPoolSimulator(poolEntity)
		require.NoError(t, err)

		amountInIUSD := mustParseBig("1000000000000000000") // 1 iUSD
		amountOutUSDC := mustParseBig("1000000")            // 1 USDC

		simulator.UpdateBalance(pool.UpdateBalanceParams{
			TokenAmountIn:  pool.TokenAmount{Token: iusdAddr, Amount: amountInIUSD},
			TokenAmountOut: pool.TokenAmount{Token: usdcAddr, Amount: amountOutUSDC},
			Fee:            pool.TokenAmount{Token: iusdAddr, Amount: big.NewInt(0)},
		})

		expectedIUSD := new(big.Int).Sub(initialIUSD, amountInIUSD)
		assert.Equal(t, expectedIUSD.String(), simulator.IUSDSupply.String())
	})

	// Test 3: iUSD → siUSD (stake)
	t.Run("iUSD -> siUSD (stake)", func(t *testing.T) {
		simulator, err := NewPoolSimulator(poolEntity)
		require.NoError(t, err)

		amountInIUSD := mustParseBig("1000000000000000000")  // 1 iUSD
		amountOutSIUSD := mustParseBig("500000000000000000") // 0.5 siUSD

		simulator.UpdateBalance(pool.UpdateBalanceParams{
			TokenAmountIn:  pool.TokenAmount{Token: iusdAddr, Amount: amountInIUSD},
			TokenAmountOut: pool.TokenAmount{Token: siusdAddr, Amount: amountOutSIUSD},
			Fee:            pool.TokenAmount{Token: iusdAddr, Amount: big.NewInt(0)},
		})

		expectedSIUSDAssets := new(big.Int).Add(initialSIUSDAssets, amountInIUSD)
		expectedSIUSDSupply := new(big.Int).Add(initialSIUSDSupply, amountOutSIUSD)

		assert.Equal(t, expectedSIUSDAssets.String(), simulator.SIUSDTotalAssets.String())
		assert.Equal(t, expectedSIUSDSupply.String(), simulator.SIUSDSupply.String())
	})

	// Test 4: siUSD → iUSD (unstake)
	t.Run("siUSD -> iUSD (unstake)", func(t *testing.T) {
		simulator, err := NewPoolSimulator(poolEntity)
		require.NoError(t, err)

		amountInSIUSD := mustParseBig("1000000000000000000") // 1 siUSD
		amountOutIUSD := mustParseBig("2000000000000000000") // 2 iUSD

		simulator.UpdateBalance(pool.UpdateBalanceParams{
			TokenAmountIn:  pool.TokenAmount{Token: siusdAddr, Amount: amountInSIUSD},
			TokenAmountOut: pool.TokenAmount{Token: iusdAddr, Amount: amountOutIUSD},
			Fee:            pool.TokenAmount{Token: siusdAddr, Amount: big.NewInt(0)},
		})

		expectedSIUSDAssets := new(big.Int).Sub(initialSIUSDAssets, amountOutIUSD)
		expectedSIUSDSupply := new(big.Int).Sub(initialSIUSDSupply, amountInSIUSD)

		assert.Equal(t, expectedSIUSDAssets.String(), simulator.SIUSDTotalAssets.String())
		assert.Equal(t, expectedSIUSDSupply.String(), simulator.SIUSDSupply.String())
	})

	// Test 5: iUSD → liUSD (lock)
	t.Run("iUSD -> liUSD (lock)", func(t *testing.T) {
		simulator, err := NewPoolSimulator(poolEntity)
		require.NoError(t, err)

		amountInIUSD := mustParseBig("2000000000000000000")   // 2 iUSD
		amountOutLIUSD := mustParseBig("2000000000000000000") // 2 liUSD

		simulator.UpdateBalance(pool.UpdateBalanceParams{
			TokenAmountIn:  pool.TokenAmount{Token: iusdAddr, Amount: amountInIUSD},
			TokenAmountOut: pool.TokenAmount{Token: liusd1moAddr, Amount: amountOutLIUSD},
			Fee:            pool.TokenAmount{Token: iusdAddr, Amount: big.NewInt(0)},
		})

		expectedLIUSD1moSupply := new(big.Int).Add(initialLIUSD1moSupply, amountOutLIUSD)
		expectedLIUSD1moReceipts := new(big.Int).Add(initialLIUSD1moReceipts, amountInIUSD)

		assert.Equal(t, expectedLIUSD1moSupply.String(), simulator.LIUSDBuckets[0].TotalSupply.String())
		assert.Equal(t, expectedLIUSD1moReceipts.String(), simulator.LIUSDBuckets[0].BucketData.TotalReceiptTokens.String())
	})

	// Test 6: USDC → siUSD (mintAndStake)
	t.Run("USDC -> siUSD (mintAndStake)", func(t *testing.T) {
		simulator, err := NewPoolSimulator(poolEntity)
		require.NoError(t, err)

		amountInUSDC := mustParseBig("1000000")              // 1 USDC
		amountOutSIUSD := mustParseBig("500000000000000000") // 0.5 siUSD (based on 2:1 rate)

		simulator.UpdateBalance(pool.UpdateBalanceParams{
			TokenAmountIn:  pool.TokenAmount{Token: usdcAddr, Amount: amountInUSDC},
			TokenAmountOut: pool.TokenAmount{Token: siusdAddr, Amount: amountOutSIUSD},
			Fee:            pool.TokenAmount{Token: usdcAddr, Amount: big.NewInt(0)},
		})

		// Should increase iUSD supply (mint step)
		intermediateIUSD := mustParseBig("1000000000000000000") // 1 iUSD from mint
		expectedIUSD := new(big.Int).Add(initialIUSD, intermediateIUSD)
		assert.Equal(t, expectedIUSD.String(), simulator.IUSDSupply.String())

		// Should increase siUSD vault state (stake step)
		expectedSIUSDAssets := new(big.Int).Add(initialSIUSDAssets, intermediateIUSD)
		expectedSIUSDSupply := new(big.Int).Add(initialSIUSDSupply, amountOutSIUSD)
		assert.Equal(t, expectedSIUSDAssets.String(), simulator.SIUSDTotalAssets.String())
		assert.Equal(t, expectedSIUSDSupply.String(), simulator.SIUSDSupply.String())
	})

	// Test 7: USDC → liUSD (mintAndLock)
	t.Run("USDC -> liUSD (mintAndLock)", func(t *testing.T) {
		simulator, err := NewPoolSimulator(poolEntity)
		require.NoError(t, err)

		amountInUSDC := mustParseBig("1000000")               // 1 USDC
		amountOutLIUSD := mustParseBig("1000000000000000000") // 1 liUSD (based on 1:1 rate)

		simulator.UpdateBalance(pool.UpdateBalanceParams{
			TokenAmountIn:  pool.TokenAmount{Token: usdcAddr, Amount: amountInUSDC},
			TokenAmountOut: pool.TokenAmount{Token: liusd1moAddr, Amount: amountOutLIUSD},
			Fee:            pool.TokenAmount{Token: usdcAddr, Amount: big.NewInt(0)},
		})

		// Should increase iUSD supply (mint step)
		intermediateIUSD := mustParseBig("1000000000000000000") // 1 iUSD from mint
		expectedIUSD := new(big.Int).Add(initialIUSD, intermediateIUSD)
		assert.Equal(t, expectedIUSD.String(), simulator.IUSDSupply.String())

		// Should increase liUSD bucket state (lock step)
		expectedLIUSD1moSupply := new(big.Int).Add(initialLIUSD1moSupply, amountOutLIUSD)
		expectedLIUSD1moReceipts := new(big.Int).Add(initialLIUSD1moReceipts, intermediateIUSD)
		assert.Equal(t, expectedLIUSD1moSupply.String(), simulator.LIUSDBuckets[0].TotalSupply.String())
		assert.Equal(t, expectedLIUSD1moReceipts.String(), simulator.LIUSDBuckets[0].BucketData.TotalReceiptTokens.String())
	})
}

func mustParseBig(s string) *big.Int {
	b, ok := new(big.Int).SetString(s, 10)
	if !ok {
		panic("failed to parse big.Int: " + s)
	}
	return b
}
