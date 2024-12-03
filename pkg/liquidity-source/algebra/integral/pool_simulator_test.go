package integral

import (
	"math/big"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	v3Entities "github.com/daoleno/uniswapv3-sdk/entities"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	mockGas int64 = 1000

	mockPrice, _            = new(big.Int).SetString("79215137550403037532088647580", 10)
	mockTick         int32  = 4
	mockLastFee      uint16 = 100
	mockPluginConfig uint8  = 193
	mockCommunityFee uint16 = 150

	mockLiquidity = uint256.NewInt(98862330578)

	mockTickSpacing       = 60
	mockTickmin     int32 = -887220
	mockTickmax     int32 = 887220

	mockTicks, _ = v3Entities.NewTickListDataProvider([]v3Entities.Tick{
		{Index: -887220, LiquidityGross: big.NewInt(35733795), LiquidityNet: big.NewInt(35733795)},
		{Index: -4500, LiquidityGross: big.NewInt(1469002688), LiquidityNet: big.NewInt(1469002688)},
		{Index: -1740, LiquidityGross: big.NewInt(815264000), LiquidityNet: big.NewInt(815264000)},
		{Index: -1080, LiquidityGross: big.NewInt(4716862354), LiquidityNet: big.NewInt(4716862354)},
		{Index: -960, LiquidityGross: big.NewInt(2130488), LiquidityNet: big.NewInt(2130488)},
		{Index: -540, LiquidityGross: big.NewInt(59681565), LiquidityNet: big.NewInt(59681565)},
		{Index: -120, LiquidityGross: big.NewInt(173321441467), LiquidityNet: big.NewInt(173321441467)},
		{Index: -60, LiquidityGross: big.NewInt(265085097155), LiquidityNet: big.NewInt(-81557785779)},
		{Index: 60, LiquidityGross: big.NewInt(91763655688), LiquidityNet: big.NewInt(-91763655688)},
		{Index: 540, LiquidityGross: big.NewInt(2130488), LiquidityNet: big.NewInt(-2130488)},
		{Index: 960, LiquidityGross: big.NewInt(59681565), LiquidityNet: big.NewInt(-59681565)},
		{Index: 1080, LiquidityGross: big.NewInt(3555869904), LiquidityNet: big.NewInt(-3555869904)},
		{Index: 1800, LiquidityGross: big.NewInt(1976256450), LiquidityNet: big.NewInt(-1976256450)},
		{Index: 1860, LiquidityGross: big.NewInt(1469002688), LiquidityNet: big.NewInt(-1469002688)},
		{Index: 887220, LiquidityGross: big.NewInt(35733795), LiquidityNet: big.NewInt(-35733795)},
	}, mockTickSpacing)

	mockTimepoints = NewTimepointStorage(map[uint16]Timepoint{
		0: {
			Initialized:          true,
			BlockTimestamp:       1722423991,
			TickCumulative:       0,
			VolatilityCumulative: uZERO,
			Tick:                 0,
			AverageTick:          0,
			WindowStartIndex:     0,
		},
		19872: {
			Initialized:          true,
			BlockTimestamp:       1732902075,
			TickCumulative:       -7029297,
			VolatilityCumulative: uint256.NewInt(2411048939),
			Tick:                 -6,
			AverageTick:          -5,
			WindowStartIndex:     19865,
		},
		19873: {
			Initialized:          true,
			BlockTimestamp:       1733084987,
			TickCumulative:       -8126769,
			VolatilityCumulative: uint256.NewInt(2411109909),
			Tick:                 -6,
			AverageTick:          -6,
			WindowStartIndex:     19872,
		},
		19874: {
			Initialized:          true,
			BlockTimestamp:       1733131721,
			TickCumulative:       -8407173,
			VolatilityCumulative: uint256.NewInt(2411109909),
			Tick:                 -6,
			AverageTick:          -6,
			WindowStartIndex:     19872,
		},
		19875: {
			Initialized:          false,
			BlockTimestamp:       0,
			TickCumulative:       0,
			VolatilityCumulative: uZERO,
			Tick:                 0,
			AverageTick:          0,
			WindowStartIndex:     0,
		},
		19876: {
			Initialized:          false,
			BlockTimestamp:       0,
			TickCumulative:       0,
			VolatilityCumulative: uZERO,
			Tick:                 0,
			AverageTick:          0,
			WindowStartIndex:     0,
		},
	})

	mockTimepointIndex         uint16 = 19874
	mockLastTimepointTimestamp uint32 = 1733131721
	mockIsInitialized                 = true

	mockAlpha1      uint16 = 2900
	mockAlpha2      uint16 = 12000
	mockBeta1       uint32 = 360
	mockBeta2       uint32 = 60000
	mockGamma1      uint16 = 59
	mockGamma2      uint16 = 8500
	mockVolumeBeta  uint32 = 0
	mockVolumeGamma uint16 = 0
	mockBaseFee     uint16 = 100

	// mockSlidingFeeZeroToOneFeeFactor *int
	// mockSlidingFeeOneToZeroFeeFactor *int
)

func TestCalcAmountOut(t *testing.T) {
	tests := []struct {
		name           string
		simulator      *PoolSimulator
		input          pool.CalcAmountOutParams
		expectedResult *pool.CalcAmountOutResult
		expectedErr    error
	}{
		{
			name: "valid swap - token 0 to token 1",
			simulator: &PoolSimulator{
				Pool: pool.Pool{
					Info: pool.PoolInfo{
						Address:  "0x3b4d8548aa8dccd0ae7643a84049687bf16d1851",
						Exchange: "scribe",
						Type:     DexType,
						Reserves: []*big.Int{
							big.NewInt(817408052),
							big.NewInt(1425700551),
						},
						Tokens: []string{
							"0x06efdbff2a14a7c8e15944d1f4a48f9f95f663a4", // USDC
							"0xf55bec9cafdbe8730f096aa55dad6d22d44099df", // USDT
						},
						BlockNumber: 11587102,
					},
				},
				globalState: GlobalState{
					Price:        uint256.MustFromBig(mockPrice),
					Tick:         mockTick,
					LastFee:      mockLastFee,
					PluginConfig: mockPluginConfig,
					CommunityFee: mockCommunityFee,
					Unlocked:     true,
				},
				liquidity:   mockLiquidity,
				ticks:       mockTicks,
				tickMin:     mockTickmin,
				tickMax:     mockTickmax,
				tickSpacing: mockTickSpacing,
				timepoints:  mockTimepoints,
				volatilityOracle: &VotatilityOraclePlugin{
					TimepointIndex:         mockTimepointIndex,
					LastTimepointTimestamp: mockLastTimepointTimestamp,
					IsInitialized:          mockIsInitialized,
				},
				dynamicFee: &DynamicFeePlugin{
					FeeConfig: FeeConfiguration{
						Alpha1:      mockAlpha1,
						Alpha2:      mockAlpha2,
						Beta1:       mockBeta1,
						Beta2:       mockBeta2,
						Gamma1:      mockGamma1,
						Gamma2:      mockGamma2,
						VolumeBeta:  mockVolumeBeta,
						VolumeGamma: mockVolumeGamma,
						BaseFee:     mockBaseFee,
					},
				},
				useBasePluginV2: false,
				gas:             mockGas,
			},
			input: pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{
					Token:  "0x06efdbff2a14a7c8e15944d1f4a48f9f95f663a4", // USDC
					Amount: big.NewInt(1000000),
				},
				TokenOut: "0xf55bec9cafdbe8730f096aa55dad6d22d44099df", // USDT
			},
			expectedResult: &pool.CalcAmountOutResult{
				TokenAmountOut: &pool.TokenAmount{
					Token:  "0xf55bec9cafdbe8730f096aa55dad6d22d44099df",
					Amount: big.NewInt(984666), // Expected amount after swap
				},
				Fee: &pool.TokenAmount{
					Token:  "0x06efdbff2a14a7c8e15944d1f4a48f9f95f663a4",
					Amount: big.NewInt(2250), // Expected fees
				},
				SwapInfo: StateUpdate{
					GlobalState: GlobalState{
						Unlocked:     true,
						LastFee:      15000,
						Tick:         -4,
						PluginConfig: mockPluginConfig,
						CommunityFee: mockCommunityFee,
					},
					Liquidity: uint256.NewInt(98862330578), // Expected liquidity
				},
				Gas: mockGas,
			},
			expectedErr: nil,
		},
		{
			name: "valid swap - token 1 to token 0",
			simulator: &PoolSimulator{
				Pool: pool.Pool{
					Info: pool.PoolInfo{
						Address:  "0x3b4d8548aa8dccd0ae7643a84049687bf16d1851",
						Exchange: "scribe",
						Type:     DexType,
						Reserves: []*big.Int{
							big.NewInt(817408052),
							big.NewInt(1425700551),
						},
						Tokens: []string{
							"0x06efdbff2a14a7c8e15944d1f4a48f9f95f663a4", // USDC
							"0xf55bec9cafdbe8730f096aa55dad6d22d44099df", // USDT
						},
						BlockNumber: 11587102,
					},
				},
				globalState: GlobalState{
					Price:        uint256.MustFromBig(mockPrice),
					Tick:         mockTick,
					LastFee:      mockLastFee,
					PluginConfig: mockPluginConfig,
					CommunityFee: mockCommunityFee,
					Unlocked:     true,
				},
				liquidity:   mockLiquidity,
				ticks:       mockTicks,
				tickMin:     mockTickmin,
				tickMax:     mockTickmax,
				tickSpacing: mockTickSpacing,
				timepoints:  mockTimepoints,
				volatilityOracle: &VotatilityOraclePlugin{
					TimepointIndex:         mockTimepointIndex,
					LastTimepointTimestamp: mockLastTimepointTimestamp,
					IsInitialized:          mockIsInitialized,
				},
				dynamicFee: &DynamicFeePlugin{
					FeeConfig: FeeConfiguration{
						Alpha1:      mockAlpha1,
						Alpha2:      mockAlpha2,
						Beta1:       mockBeta1,
						Beta2:       mockBeta2,
						Gamma1:      mockGamma1,
						Gamma2:      mockGamma2,
						VolumeBeta:  mockVolumeBeta,
						VolumeGamma: mockVolumeGamma,
						BaseFee:     mockBaseFee,
					},
				},
				useBasePluginV2: false,
				gas:             mockGas,
			},
			input: pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{
					Token:  "0xf55bec9cafdbe8730f096aa55dad6d22d44099df", // USDT
					Amount: big.NewInt(1000000),
				},
				TokenOut: "0x06efdbff2a14a7c8e15944d1f4a48f9f95f663a4", // USDC
			},
			expectedResult: &pool.CalcAmountOutResult{
				TokenAmountOut: &pool.TokenAmount{
					Token:  "0x06efdbff2a14a7c8e15944d1f4a48f9f95f663a4",
					Amount: big.NewInt(985314), // Expected amount after swap
				},
				Fee: &pool.TokenAmount{
					Token:  "0xf55bec9cafdbe8730f096aa55dad6d22d44099df",
					Amount: big.NewInt(2250), // Expected fees
				},
				SwapInfo: StateUpdate{
					GlobalState: GlobalState{
						Unlocked:     true,
						LastFee:      15000,
						Tick:         -4,
						PluginConfig: mockPluginConfig,
						CommunityFee: mockCommunityFee,
					},
					Liquidity: uint256.NewInt(98862330578), // Expected liquidity
				},
				Gas: mockGas,
			},
			expectedErr: nil,
		},
		// {
		// 	name: "swap token 0 to token 1 with large amount in",
		// 	simulator: &PoolSimulator{
		// 		Pool: pool.Pool{
		// 			Info: pool.PoolInfo{
		// 				Address:  "0x3b4d8548aa8dccd0ae7643a84049687bf16d1851",
		// 				Exchange: "scribe",
		// 				Type:     DexType,
		// 				Reserves: []*big.Int{
		// 					big.NewInt(817408052),
		// 					big.NewInt(1425700551),
		// 				},
		// 				Tokens: []string{
		// 					"0x06efdbff2a14a7c8e15944d1f4a48f9f95f663a4", // USDC
		// 					"0xf55bec9cafdbe8730f096aa55dad6d22d44099df", // USDT
		// 				},
		// 				BlockNumber: 11587102,
		// 			},
		// 		},
		// 		globalState: GlobalState{
		// 			Price:        uint256.MustFromBig(mockPrice),
		// 			Tick:         mockTick,
		// 			LastFee:      mockLastFee,
		// 			PluginConfig: mockPluginConfig,
		// 			CommunityFee: mockCommunityFee,
		// 			Unlocked:     true,
		// 		},
		// 		liquidity:   mockLiquidity,
		// 		ticks:       mockTicks,
		// 		tickMin:     mockTickmin,
		// 		tickMax:     mockTickmax,
		// 		tickSpacing: mockTickSpacing,
		// 		timepoints:  mockTimepoints,
		// 		volatilityOracle: &VotatilityOraclePlugin{
		// 			TimepointIndex:         mockTimepointIndex,
		// 			LastTimepointTimestamp: mockLastTimepointTimestamp,
		// 			IsInitialized:          mockIsInitialized,
		// 		},
		// 		dynamicFee: &DynamicFeePlugin{
		// 			FeeConfig: FeeConfiguration{
		// 				Alpha1:      mockAlpha1,
		// 				Alpha2:      mockAlpha2,
		// 				Beta1:       mockBeta1,
		// 				Beta2:       mockBeta2,
		// 				Gamma1:      mockGamma1,
		// 				Gamma2:      mockGamma2,
		// 				VolumeBeta:  mockVolumeBeta,
		// 				VolumeGamma: mockVolumeGamma,
		// 				BaseFee:     mockBaseFee,
		// 			},
		// 		},
		// 		useBasePluginV2: false,
		// 		gas:             mockGas,
		// 	},
		// 	input: pool.CalcAmountOutParams{
		// 		TokenAmountIn: pool.TokenAmount{
		// 			Token:  "0x06efdbff2a14a7c8e15944d1f4a48f9f95f663a4", // USDC
		// 			Amount: big.NewInt(1425700551000000000),
		// 		},
		// 		TokenOut: "0xf55bec9cafdbe8730f096aa55dad6d22d44099df", // USDT
		// 	},

		// 	expectedErr: ErrLiquiditySub,
		// },
		{
			name: "swap token 1 to token 0 with large amount in",
			simulator: &PoolSimulator{
				Pool: pool.Pool{
					Info: pool.PoolInfo{
						Address:  "0x3b4d8548aa8dccd0ae7643a84049687bf16d1851",
						Exchange: "scribe",
						Type:     DexType,
						Reserves: []*big.Int{
							big.NewInt(817408052),
							big.NewInt(1425700551),
						},
						Tokens: []string{
							"0x06efdbff2a14a7c8e15944d1f4a48f9f95f663a4", // USDC
							"0xf55bec9cafdbe8730f096aa55dad6d22d44099df", // USDT
						},
						BlockNumber: 11587102,
					},
				},
				globalState: GlobalState{
					Price:        uint256.MustFromBig(mockPrice),
					Tick:         mockTick,
					LastFee:      mockLastFee,
					PluginConfig: mockPluginConfig,
					CommunityFee: mockCommunityFee,
					Unlocked:     true,
				},
				liquidity:   mockLiquidity,
				ticks:       mockTicks,
				tickMin:     mockTickmin,
				tickMax:     mockTickmax,
				tickSpacing: mockTickSpacing,
				timepoints:  mockTimepoints,
				volatilityOracle: &VotatilityOraclePlugin{
					TimepointIndex:         mockTimepointIndex,
					LastTimepointTimestamp: mockLastTimepointTimestamp,
					IsInitialized:          mockIsInitialized,
				},
				dynamicFee: &DynamicFeePlugin{
					FeeConfig: FeeConfiguration{
						Alpha1:      mockAlpha1,
						Alpha2:      mockAlpha2,
						Beta1:       mockBeta1,
						Beta2:       mockBeta2,
						Gamma1:      mockGamma1,
						Gamma2:      mockGamma2,
						VolumeBeta:  mockVolumeBeta,
						VolumeGamma: mockVolumeGamma,
						BaseFee:     mockBaseFee,
					},
				},
				useBasePluginV2: false,
				gas:             mockGas,
			},
			input: pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{
					Token:  "0xf55bec9cafdbe8730f096aa55dad6d22d44099df", // USDT
					Amount: big.NewInt(817408052),
				},
				TokenOut: "0x06efdbff2a14a7c8e15944d1f4a48f9f95f663a4", // USDC
			},

			expectedErr: ErrLiquiditySub,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.simulator.CalcAmountOut(tt.input)

			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.Equal(t, err.Error(), tt.expectedErr.Error())
			} else {
				require.NotEmpty(t, result.Fee)
				assert.Equal(t, tt.expectedResult.Fee, result.Fee)

				require.NotEmpty(t, result.Gas)
				assert.Equal(t, tt.expectedResult.Gas, result.Gas)

				require.NotEmpty(t, result.SwapInfo)

				expectedSwapInfo := tt.expectedResult.SwapInfo.(StateUpdate)
				actualSwapInfo := result.SwapInfo.(StateUpdate)

				require.NotEmpty(t, actualSwapInfo.GlobalState)
				assert.Equal(t, expectedSwapInfo.GlobalState.CommunityFee, actualSwapInfo.GlobalState.CommunityFee)
				assert.Equal(t, expectedSwapInfo.GlobalState.PluginConfig, actualSwapInfo.GlobalState.PluginConfig)
				assert.Equal(t, expectedSwapInfo.GlobalState.Unlocked, actualSwapInfo.GlobalState.Unlocked)
				assert.Equal(t, expectedSwapInfo.GlobalState.LastFee, actualSwapInfo.GlobalState.LastFee)
				assert.Equal(t, expectedSwapInfo.GlobalState.Tick, actualSwapInfo.GlobalState.Tick)

				require.NotEmpty(t, result.SwapInfo)
				assert.Equal(t, tt.expectedResult.TokenAmountOut, result.TokenAmountOut)
			}
		})
	}
}
