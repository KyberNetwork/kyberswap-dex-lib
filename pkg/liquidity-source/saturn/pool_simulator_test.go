package saturn

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

func TestPoolSimulator_CalcAmountOut(t *testing.T) {
	t.Parallel()

	newSimulator := func(extra PoolExtra) *PoolSimulator {
		extraBytes, err := json.Marshal(extra)
		require.NoError(t, err)
		staticExtraBytes, err := json.Marshal(StaticExtra{Gas: defaultGas})
		require.NoError(t, err)

		sim, err := NewPoolSimulator(entity.Pool{
			Address:     SwapFacilityAddress,
			Exchange:    DexType,
			Type:        DexType,
			Reserves:    []string{defaultReserves, defaultReserves},
			Extra:       string(extraBytes),
			StaticExtra: string(staticExtraBytes),
			BlockNumber: 100,
			Tokens: []*entity.PoolToken{
				{Address: USDCAddress, Decimals: 6, Swappable: true},
				{Address: USDatAddress, Decimals: 6, Swappable: true},
			},
		})
		require.NoError(t, err)
		return sim
	}

	t.Run("mint 1:1 for USDC", func(t *testing.T) {
		sim := newSimulator(PoolExtra{
			ExtensionApproved:   true,
			AssetAllowed:        true,
			AssetCap:            uint256.NewInt(1_000_000_000),
			AssetBalance:        uint256.NewInt(0),
			AssetDecimals:       6,
			WhitelistEnabled:    true,
			ExecutorWhitelisted: true,
		})

		result, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: USDCAddress, Amount: big.NewInt(12_345_678)},
			TokenOut:      USDatAddress,
		})
		require.NoError(t, err)
		require.Equal(t, "12345678", result.TokenAmountOut.Amount.String())
		require.Nil(t, result.RemainingTokenAmountIn)
	})

	t.Run("round down and return remaining input", func(t *testing.T) {
		sim := newSimulator(PoolExtra{
			ExtensionApproved:   true,
			AssetAllowed:        true,
			AssetCap:            uint256.NewInt(1_000_000_000_000),
			AssetBalance:        uint256.NewInt(0),
			AssetDecimals:       8,
			WhitelistEnabled:    false,
			ExecutorWhitelisted: false,
		})

		result, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: USDCAddress, Amount: big.NewInt(123_456_789)},
			TokenOut:      USDatAddress,
		})
		require.NoError(t, err)
		require.Equal(t, "1234567", result.TokenAmountOut.Amount.String())
		require.NotNil(t, result.RemainingTokenAmountIn)
		require.Equal(t, "89", result.RemainingTokenAmountIn.Amount.String())
	})

	t.Run("reject when cap reached", func(t *testing.T) {
		sim := newSimulator(PoolExtra{
			ExtensionApproved:   true,
			AssetAllowed:        true,
			AssetCap:            uint256.NewInt(1000),
			AssetBalance:        uint256.NewInt(900),
			AssetDecimals:       6,
			WhitelistEnabled:    false,
			ExecutorWhitelisted: false,
		})

		_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: USDCAddress, Amount: big.NewInt(101)},
			TokenOut:      USDatAddress,
		})
		require.ErrorIs(t, err, ErrAssetCapReached)
	})

	t.Run("reject when executor is not whitelisted", func(t *testing.T) {
		sim := newSimulator(PoolExtra{
			ExtensionApproved:   true,
			AssetAllowed:        true,
			AssetCap:            uint256.NewInt(1000),
			AssetBalance:        uint256.NewInt(0),
			AssetDecimals:       6,
			WhitelistEnabled:    true,
			ExecutorWhitelisted: false,
		})

		_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: USDCAddress, Amount: big.NewInt(100)},
			TokenOut:      USDatAddress,
		})
		require.ErrorIs(t, err, ErrExecutorNotWhitelisted)
	})

	t.Run("reject when paused", func(t *testing.T) {
		sim := newSimulator(PoolExtra{
			SwapFacilityPaused:  true,
			ExtensionApproved:   true,
			AssetAllowed:        true,
			AssetCap:            uint256.NewInt(1000),
			AssetBalance:        uint256.NewInt(0),
			AssetDecimals:       6,
			WhitelistEnabled:    false,
			ExecutorWhitelisted: false,
		})

		_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: USDCAddress, Amount: big.NewInt(100)},
			TokenOut:      USDatAddress,
		})
		require.ErrorIs(t, err, ErrSwapFacilityPaused)
	})
}

func TestPoolSimulator_UpdateBalance(t *testing.T) {
	t.Parallel()

	extraBytes, err := json.Marshal(PoolExtra{
		ExtensionApproved:   true,
		AssetAllowed:        true,
		AssetCap:            uint256.NewInt(1000),
		AssetBalance:        uint256.NewInt(100),
		AssetDecimals:       6,
		WhitelistEnabled:    false,
		ExecutorWhitelisted: false,
	})
	require.NoError(t, err)
	staticExtraBytes, err := json.Marshal(StaticExtra{Gas: defaultGas})
	require.NoError(t, err)

	sim, err := NewPoolSimulator(entity.Pool{
		Address:     SwapFacilityAddress,
		Exchange:    DexType,
		Type:        DexType,
		Reserves:    []string{defaultReserves, defaultReserves},
		Extra:       string(extraBytes),
		StaticExtra: string(staticExtraBytes),
		Tokens:      []*entity.PoolToken{{Address: USDCAddress}, {Address: USDatAddress}},
	})
	require.NoError(t, err)

	sim.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: SwapInfo{EffectiveAssetAmount: uint256.NewInt(20)}})
	require.Equal(t, "120", sim.assetBalance.String())
}
