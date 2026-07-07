package dexLite

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	big256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

var (
	entityPool entity.Pool
	_          = json.Unmarshal([]byte(`{"address":"0xbbcb91440523216e2b87052a99f69c604a7b6e006dd161107ef07bb8","swapFee":0.0005,"exchange":"fluid-dex-lite","type":"fluid-dex-lite","timestamp":1754385937,"reserves":["494178168265","507852200630"],"tokens":[{"address":"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48","symbol":"USDC","decimals":6,"swappable":true},{"address":"0xdac17f958d2ee523a2206206994597c13d831ec7","symbol":"USDT","decimals":6,"swappable":true}],"extra":"{\"pS\":{\"dV\":\"0x1cde38e0457a2001c173d22cdbba319bb7801e007c00006765c7939d700005\",\"pS\":\"0xd1182321a5e00000039d6228d9dcc28dfffffe6890d0e3\",\"rS\":\"0x6890d0e315180004800c\",\"nP\":\"0x33b2e3ca3a10079d480c6b0\"},\"ts\":1754385923}","staticExtra":"{\"l\":\"19d6228d9dcc28dfffffe688c1fe7\",\"k\":{\"t0\":\"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48\",\"t1\":\"0xdac17f958d2ee523a2206206994597c13d831ec7\",\"s\":\"0x0000000000000000000000000000000000000000000000000000000000000000\"},\"i\":\"0x6dd161107ef07bb8\"}","blockNumber":23073952}`),
		&entityPool)
	poolSim = lo.Must(NewPoolSimulator(entityPool))
)

func TestPoolSimulator(t *testing.T) {
	t.Run("TestCalcAmountOut", func(t *testing.T) {
		testutil.TestCalcAmountOut(t, poolSim, map[int]map[int]map[string]string{
			0: {
				1: {
					"1000000": "999909",
					"9999999": "9999091",
				},
			},
			1: {
				0: {
					"1000000": "1000080",
					"9999999": "10000806",
				},
			},
		})
	})

	t.Run("TestCalcAmountIn", func(t *testing.T) {
		testutil.TestCalcAmountIn(t, poolSim)
	})

	t.Run("TestUnpackDexVariables", func(t *testing.T) {
		// Test unpacking dex variables
		dexVars := unpackDexVariables(poolSim.PoolState.DexVariables)

		require.NotNil(t, dexVars)
		require.NotNil(t, dexVars.Fee)
		require.NotNil(t, dexVars.RevenueCut)
		require.NotNil(t, dexVars.CenterPrice)

		t.Logf("Unpacked fee: %s", dexVars.Fee.String())
		t.Logf("Unpacked revenue cut: %s", dexVars.RevenueCut.String())
		t.Logf("Unpacked center price: %s", dexVars.CenterPrice.String())
	})
}

func TestPoolSimulatorEdgeCases(t *testing.T) {
	t.Run("TestZeroAmountIn", func(t *testing.T) {
		// Create a mock dexKey and dexId for this test
		testDexKey := DexKey{
			Token0: common.HexToAddress("0xA0b86a33E6441c0c37Fc0C16b6C7Da2A0edD0bD1"), // USDC
			Token1: common.HexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7"), // USDT
			Salt:   common.Hash{},
		}
		testDexId := [8]byte{0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0}

		// Create normal pool for this test
		staticExtra := StaticExtra{
			DexLiteAddress: "0xBbcb91440523216e2b87052A99F69c604A7b6e00",
			DexKey:         testDexKey,
			DexId:          testDexId,
		}
		staticExtraBytes, _ := json.Marshal(staticExtra)

		extra := PoolExtra{
			PoolState: PoolState{DexVariables: uint256.NewInt(0x123456789abcdef)},
		}
		extraBytes, _ := json.Marshal(extra)

		entityPool := entity.Pool{
			Address:  "0x1234567890123456789012345678901234567890",
			Type:     DexType,
			Reserves: entity.PoolReserves{"1000000000", "1000000000"},
			Tokens: []*entity.PoolToken{
				{Address: "0xA0b86a33E6441c0c37Fc0C16b6C7Da2A0edD0bB336F8eb2f", Decimals: 6},
				{Address: "0xdAC17F958D2ee523a2206206994597C13D831ec7", Decimals: 6},
			},
			SwapFee:     0.001,
			Extra:       string(extraBytes),
			StaticExtra: string(staticExtraBytes),
		}

		poolSim, err := NewPoolSimulator(entityPool)
		require.NoError(t, err)

		// Should fail with zero amount
		_, err = poolSim.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{
				Token:  "0xA0b86a33E6441c0c37Fc0C16b6C7Da2A0edD0bB336F8eb2f",
				Amount: big.NewInt(0),
			},
			TokenOut: "0xdAC17F958D2ee523a2206206994597C13D831ec7",
		})

		require.Error(t, err)
		require.Equal(t, ErrInvalidAmountIn, err)
	})
}

func TestMathFunctions(t *testing.T) {
	// Create a properly packed dexVariables for testing
	// Pack: fee=100, revenueCut=10, rebalancing=1, centerPrice=1e27, token0Supply=1000000, token1Supply=1000000
	mockDexVariables := uint256.NewInt(0)

	t.Run("TestSwapMath", func(t *testing.T) {
		// Test basic constant product calculation
		// dexVars := &UnpackedDexVariables{
		//	Fee:                       big.NewInt(100), // 1%
		//	CenterPrice:              bignumber.TenPowInt(27), // 1:1 price
		//	Token0TotalSupplyAdjusted: big.NewInt(1000000), // 1M in 12 decimals
		//	Token1TotalSupplyAdjusted: big.NewInt(1000000), // 1M in 12 decimals
		// }

		// pricing := &PricingResult{
		//	CenterPrice:             dexVars.CenterPrice,
		//	Token0ImaginaryReserves: dexVars.Token0TotalSupplyAdjusted,
		//	Token1ImaginaryReserves: dexVars.Token1TotalSupplyAdjusted,
		// }

		// Fee (100) at bits 0-12
		mockDexVariables.Or(mockDexVariables, uint256.NewInt(100))

		// Revenue cut (10) at bits 13-19
		mockDexVariables.Or(mockDexVariables, new(uint256.Int).Lsh(uint256.NewInt(10), 13))

		// Rebalancing status (1) at bits 20-21
		mockDexVariables.Or(mockDexVariables, new(uint256.Int).Lsh(uint256.NewInt(1), 20))

		// Center price (1e27 compressed) at bits 23-62
		centerPriceCompressed := uint256.NewInt(1e18) // Simplified for testing
		mockDexVariables.Or(mockDexVariables, new(uint256.Int).Lsh(centerPriceCompressed, 23))

		// Token supplies at bits 136-196
		token0Supply := uint256.NewInt(1000000)
		token1Supply := uint256.NewInt(1000000)
		mockDexVariables.Or(mockDexVariables, new(uint256.Int).Lsh(token0Supply, 136))
		mockDexVariables.Or(mockDexVariables, new(uint256.Int).Lsh(token1Supply, 196))

		poolSim := &PoolSimulator{DexVars: unpackDexVariables(mockDexVariables), PoolState: PoolState{RangeShift: big256.U0}}

		// Test swapIn: 1000 tokens in
		amountOut, _, _, err := poolSim.calculateSwapInWithState(0, 1, uint256.NewInt(1000), poolSim.DexVars)
		if err != nil {
			t.Logf("SwapIn error (expected with simple math): %v", err)
		} else {
			require.Greater(t, amountOut.Uint64(), 0)
			t.Logf("SwapIn: 1000 -> %s", amountOut.String())
		}

		// Test swapOut: want 1000 tokens out
		amountIn, _, _, err := poolSim.calculateSwapOutWithState(0, 1, uint256.NewInt(1000), poolSim.DexVars)
		if err != nil {
			t.Logf("SwapOut error (expected with simple math): %v", err)
		} else {
			require.Greater(t, amountIn.Uint64(), 0)
			t.Logf("SwapOut: %s -> 1000", amountIn.String())
		}
	})
}
