package clear

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

func TestNewPoolSimulator(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		entityPool  entity.Pool
		expectError bool
	}{
		{
			name: "valid pool",
			entityPool: entity.Pool{
				Address:  "0xvault_0xtoken0_0xtoken1",
				Exchange: "clear",
				Type:     DexType,
				Reserves: entity.PoolReserves{"1000000000000000000000000", "1000000000000000000000000"},
				Tokens: []*entity.PoolToken{
					{Address: "0xtoken0", Decimals: 18, Swappable: true},
					{Address: "0xtoken1", Decimals: 18, Swappable: true},
				},
				StaticExtra: `{"vaultAddress":"0xvault","swapAddress":"0xswap","tokens":["0xtoken0","0xtoken1"]}`,
				Extra:       `{"reserves":{},"paused":false}`,
			},
			expectError: false,
		},
		{
			name: "empty static extra",
			entityPool: entity.Pool{
				Address:     "0xvault_0xtoken0_0xtoken1",
				Exchange:    "clear",
				Type:        DexType,
				StaticExtra: "",
			},
			expectError: true,
		},
		{
			name: "invalid static extra json",
			entityPool: entity.Pool{
				Address:     "0xvault_0xtoken0_0xtoken1",
				Exchange:    "clear",
				Type:        DexType,
				StaticExtra: "invalid json",
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sim, err := NewPoolSimulator(tc.entityPool)
			if tc.expectError {
				require.Error(t, err)
				require.Nil(t, sim)
			} else {
				require.NoError(t, err)
				require.NotNil(t, sim)
			}
		})
	}
}

func TestPoolSimulator_CalcAmountOut(t *testing.T) {
	t.Parallel()

	// Create a pool simulator with cached reserves for 1:1 ratio
	staticExtra := StaticExtra{
		VaultAddress: "0xvault",
		SwapAddress:  "0xswap",
		Tokens:       []string{"0xtoken0", "0xtoken1"},
	}
	staticExtraBytes, _ := json.Marshal(staticExtra)

	extra := Extra{
		Reserves: map[string]*uint256.Int{
			"0xtoken0": uint256.NewInt(1000000000000000000), // 1e18
			"0xtoken1": uint256.NewInt(1000000000000000000), // 1e18
		},
		Paused: false,
	}
	extraBytes, _ := json.Marshal(extra)

	entityPool := entity.Pool{
		Address:  "0xvault_0xtoken0_0xtoken1",
		Exchange: "clear",
		Type:     DexType,
		Reserves: entity.PoolReserves{"1000000000000000000000000", "1000000000000000000000000"},
		Tokens: []*entity.PoolToken{
			{Address: "0xtoken0", Decimals: 18, Swappable: true},
			{Address: "0xtoken1", Decimals: 18, Swappable: true},
		},
		StaticExtra: string(staticExtraBytes),
		Extra:       string(extraBytes),
	}

	sim, err := NewPoolSimulator(entityPool)
	require.NoError(t, err)

	tests := []struct {
		name          string
		tokenIn       string
		tokenOut      string
		amountIn      *big.Int
		expectedOut   *big.Int
		expectError   bool
		expectedError error
	}{
		{
			name:        "valid swap token0 to token1",
			tokenIn:     "0xtoken0",
			tokenOut:    "0xtoken1",
			amountIn:    big.NewInt(1000000000000000000), // 1e18
			expectedOut: big.NewInt(1000000000000000000), // 1:1 ratio
			expectError: false,
		},
		{
			name:        "valid swap token1 to token0",
			tokenIn:     "0xtoken1",
			tokenOut:    "0xtoken0",
			amountIn:    big.NewInt(500000000000000000), // 0.5e18
			expectedOut: big.NewInt(500000000000000000), // 1:1 ratio
			expectError: false,
		},
		{
			name:          "invalid token in",
			tokenIn:       "0xinvalid",
			tokenOut:      "0xtoken1",
			amountIn:      big.NewInt(1000000000000000000),
			expectError:   true,
			expectedError: ErrInvalidToken,
		},
		{
			name:          "invalid token out",
			tokenIn:       "0xtoken0",
			tokenOut:      "0xinvalid",
			amountIn:      big.NewInt(1000000000000000000),
			expectError:   true,
			expectedError: ErrInvalidToken,
		},
		{
			name:          "zero amount in",
			tokenIn:       "0xtoken0",
			tokenOut:      "0xtoken1",
			amountIn:      big.NewInt(0),
			expectError:   true,
			expectedError: ErrInvalidAmountIn,
		},
		{
			name:          "nil amount in",
			tokenIn:       "0xtoken0",
			tokenOut:      "0xtoken1",
			amountIn:      nil,
			expectError:   true,
			expectedError: ErrInvalidAmountIn,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{
					Token:  tc.tokenIn,
					Amount: tc.amountIn,
				},
				TokenOut: tc.tokenOut,
			})

			if tc.expectError {
				require.Error(t, err)
				if tc.expectedError != nil {
					assert.ErrorIs(t, err, tc.expectedError)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, tc.expectedOut.String(), result.TokenAmountOut.Amount.String())
			}
		})
	}
}

func TestPoolSimulator_CalcAmountOut_Paused(t *testing.T) {
	t.Parallel()

	staticExtra := StaticExtra{
		VaultAddress: "0xvault",
		SwapAddress:  "0xswap",
		Tokens:       []string{"0xtoken0", "0xtoken1"},
	}
	staticExtraBytes, _ := json.Marshal(staticExtra)

	extra := Extra{
		Reserves: map[string]*uint256.Int{},
		Paused:   true,
	}
	extraBytes, _ := json.Marshal(extra)

	entityPool := entity.Pool{
		Address:  "0xvault_0xtoken0_0xtoken1",
		Exchange: "clear",
		Type:     DexType,
		Reserves: entity.PoolReserves{"0", "0"},
		Tokens: []*entity.PoolToken{
			{Address: "0xtoken0", Decimals: 18, Swappable: true},
			{Address: "0xtoken1", Decimals: 18, Swappable: true},
		},
		StaticExtra: string(staticExtraBytes),
		Extra:       string(extraBytes),
	}

	sim, err := NewPoolSimulator(entityPool)
	require.NoError(t, err)

	result, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  "0xtoken0",
			Amount: big.NewInt(1000000000000000000),
		},
		TokenOut: "0xtoken1",
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInsufficientOutput)
	assert.Nil(t, result)
}

func TestPoolSimulator_CloneState(t *testing.T) {
	t.Parallel()

	staticExtra := StaticExtra{
		VaultAddress: "0xvault",
		SwapAddress:  "0xswap",
		Tokens:       []string{"0xtoken0", "0xtoken1"},
	}
	staticExtraBytes, _ := json.Marshal(staticExtra)

	extra := Extra{
		Reserves: map[string]*uint256.Int{
			"0xtoken0": uint256.NewInt(1000000000000000000),
			"0xtoken1": uint256.NewInt(1000000000000000000),
		},
		Paused: false,
	}
	extraBytes, _ := json.Marshal(extra)

	entityPool := entity.Pool{
		Address:  "0xvault_0xtoken0_0xtoken1",
		Exchange: "clear",
		Type:     DexType,
		Reserves: entity.PoolReserves{"1000000000000000000000000", "1000000000000000000000000"},
		Tokens: []*entity.PoolToken{
			{Address: "0xtoken0", Decimals: 18, Swappable: true},
			{Address: "0xtoken1", Decimals: 18, Swappable: true},
		},
		StaticExtra: string(staticExtraBytes),
		Extra:       string(extraBytes),
	}

	sim, err := NewPoolSimulator(entityPool)
	require.NoError(t, err)

	cloned := sim.CloneState()
	require.NotNil(t, cloned)

	clonedSim, ok := cloned.(*PoolSimulator)
	require.True(t, ok)

	// Verify it's a different instance
	assert.NotSame(t, sim, clonedSim)
	assert.NotSame(t, sim.RWMutex, clonedSim.RWMutex)

	// Verify reserves are cloned
	assert.Equal(t, len(sim.extra.Reserves), len(clonedSim.extra.Reserves))
}

func TestPoolSimulator_UpdateBalance(t *testing.T) {
	t.Parallel()

	staticExtra := StaticExtra{
		VaultAddress: "0xvault",
		SwapAddress:  "0xswap",
		Tokens:       []string{"0xtoken0", "0xtoken1"},
	}
	staticExtraBytes, _ := json.Marshal(staticExtra)

	entityPool := entity.Pool{
		Address:  "0xvault_0xtoken0_0xtoken1",
		Exchange: "clear",
		Type:     DexType,
		Reserves: entity.PoolReserves{"1000000000000000000", "1000000000000000000"},
		Tokens: []*entity.PoolToken{
			{Address: "0xtoken0", Decimals: 18, Swappable: true},
			{Address: "0xtoken1", Decimals: 18, Swappable: true},
		},
		StaticExtra: string(staticExtraBytes),
		Extra:       `{"reserves":{},"paused":false}`,
	}

	sim, err := NewPoolSimulator(entityPool)
	require.NoError(t, err)

	initialReserve0 := new(big.Int).Set(sim.Info.Reserves[0])
	initialReserve1 := new(big.Int).Set(sim.Info.Reserves[1])

	amountIn := big.NewInt(100000000000000000)  // 0.1e18
	amountOut := big.NewInt(100000000000000000) // 0.1e18

	sim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  "0xtoken0",
			Amount: amountIn,
		},
		TokenAmountOut: pool.TokenAmount{
			Token:  "0xtoken1",
			Amount: amountOut,
		},
	})

	// Reserve0 should increase by amountIn
	expectedReserve0 := new(big.Int).Add(initialReserve0, amountIn)
	assert.Equal(t, expectedReserve0.String(), sim.Info.Reserves[0].String())

	// Reserve1 should decrease by amountOut
	expectedReserve1 := new(big.Int).Sub(initialReserve1, amountOut)
	assert.Equal(t, expectedReserve1.String(), sim.Info.Reserves[1].String())
}

func TestPoolSimulator_GetMetaInfo(t *testing.T) {
	t.Parallel()

	staticExtra := StaticExtra{
		VaultAddress: "0xvault",
		SwapAddress:  "0xswap",
		Tokens:       []string{"0xtoken0", "0xtoken1"},
	}
	staticExtraBytes, _ := json.Marshal(staticExtra)

	entityPool := entity.Pool{
		Address:  "0xvault_0xtoken0_0xtoken1",
		Exchange: "clear",
		Type:     DexType,
		Reserves: entity.PoolReserves{"1000000000000000000", "1000000000000000000"},
		Tokens: []*entity.PoolToken{
			{Address: "0xtoken0", Decimals: 18, Swappable: true},
			{Address: "0xtoken1", Decimals: 18, Swappable: true},
		},
		StaticExtra: string(staticExtraBytes),
		Extra:       `{"reserves":{},"paused":false}`,
	}

	sim, err := NewPoolSimulator(entityPool)
	require.NoError(t, err)

	meta := sim.GetMetaInfo("0xtoken0", "0xtoken1")
	require.NotNil(t, meta)

	poolMeta, ok := meta.(PoolMeta)
	require.True(t, ok)
	assert.Equal(t, "0xvault", poolMeta.VaultAddress)
	assert.Equal(t, "0xswap", poolMeta.SwapAddress)
}
