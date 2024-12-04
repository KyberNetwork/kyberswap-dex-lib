package integral

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
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
		{
			name: "swap token 0 to token 1 with large amount in",
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
					Amount: big.NewInt(1425700551000000000),
				},
				TokenOut: "0xf55bec9cafdbe8730f096aa55dad6d22d44099df", // USDT
			},
			expectedResult: &pool.CalcAmountOutResult{
				TokenAmountOut: &pool.TokenAmount{
					Token:  "0xf55bec9cafdbe8730f096aa55dad6d22d44099df",
					Amount: big.NewInt(279874986), // Expected amount after swap
				},
				Fee: &pool.TokenAmount{
					Token:  "0x06efdbff2a14a7c8e15944d1f4a48f9f95f663a4",
					Amount: big.NewInt(641334), // Expected fees
				},
				SwapInfo: StateUpdate{
					GlobalState: GlobalState{
						Unlocked:     true,
						LastFee:      15000,
						Tick:         -61,
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
			expectedResult: &pool.CalcAmountOutResult{
				TokenAmountOut: &pool.TokenAmount{
					Token:  "0x06efdbff2a14a7c8e15944d1f4a48f9f95f663a4",
					Amount: big.NewInt(768004061), // Expected amount after swap
				},
				Fee: &pool.TokenAmount{
					Token:  "0xf55bec9cafdbe8730f096aa55dad6d22d44099df",
					Amount: big.NewInt(1839166), // Expected fees
				},
				SwapInfo: StateUpdate{
					GlobalState: GlobalState{
						Unlocked:     true,
						LastFee:      15000,
						Tick:         1721,
						PluginConfig: mockPluginConfig,
						CommunityFee: mockCommunityFee,
					},
					Liquidity: uint256.NewInt(98862330578), // Expected liquidity
				},
				Gas: mockGas,
			},
			expectedErr: nil,
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

var mockPool = []byte(`{"address":"0xbe9c1d237d002c8d9402f30c16ace1436d008f0c","exchange":"silverswap","type":"algebra-integral","timestamp":1733225338,"reserves":["9999999999999944","2620057588865"],"tokens":[{"address":"0x21be370d5312f44cb42ce377bc9b8a0cef1a4c83","name":"Wrapped Fantom","symbol":"WFTM","decimals":18,"weight":50,"swappable":true},{"address":"0xfe7eda5f2c56160d406869a8aa4b2f365d544c7b","name":"Axelar Wrapped ETH","symbol":"axlETH","decimals":18,"weight":50,"swappable":true}],"extra":"{\"liquidity\":161865919478591,\"globalState\":{\"price\":\"1282433937397070526017841373\",\"tick\":82476,\"lastFee\":100,\"pluginConfig\":193,\"communityFee\":100,\"unlocked\":true},\"ticks\":[{\"Index\":-887220,\"LiquidityGross\":161865919478591,\"LiquidityNet\":161865919478591},{\"Index\":887220,\"LiquidityGross\":161865919478591,\"LiquidityNet\":-161865919478591}],\"tickSpacing\":60,\"timepoints\":{\"0\":{\"Initialized\":true,\"BlockTimestamp\":1712116096,\"TickCumulative\":0,\"VolatilityCumulative\":\"0\",\"Tick\":-82476,\"AverageTick\":-82476,\"WindowStartIndex\":0},\"1\":{\"Initialized\":false,\"BlockTimestamp\":0,\"TickCumulative\":0,\"VolatilityCumulative\":\"0\",\"Tick\":0,\"AverageTick\":0,\"WindowStartIndex\":0},\"2\":{\"Initialized\":false,\"BlockTimestamp\":0,\"TickCumulative\":0,\"VolatilityCumulative\":\"0\",\"Tick\":0,\"AverageTick\":0,\"WindowStartIndex\":0},\"65535\":{\"Initialized\":false,\"BlockTimestamp\":0,\"TickCumulative\":0,\"VolatilityCumulative\":\"0\",\"Tick\":0,\"AverageTick\":0,\"WindowStartIndex\":0}},\"votalityOracle\":{\"TimepointIndex\":0,\"LastTimepointTimestamp\":1712116096,\"IsInitialized\":true},\"slidingFee\":{\"FeeFactors\":{\"ZeroToOneFeeFactor\":null,\"OneToZeroFeeFactor\":null}},\"dynamicFee\":{\"FeeConfig\":{\"alpha1\":2900,\"alpha2\":12000,\"beta1\":360,\"beta2\":60000,\"gamma1\":59,\"gamma2\":8500,\"volumeBeta\":0,\"volumeGamma\":0,\"baseFee\":100}}}","staticExtra":"{\"useBasePluginV2\":false}","blockNumber":99019509}`)

func TestCalcAmountOut_FromPool(t *testing.T) {
	var p entity.Pool
	err := json.Unmarshal(mockPool, &p)
	require.NoError(t, err)

	ps, err := NewPoolSimulator(p, 280000)
	require.NoError(t, err)

	res, err := ps.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  "0x21be370d5312f44cb42ce377bc9b8a0cef1a4c83",
			Amount: big.NewInt(100000000000000),
		},
		TokenOut: "0xfe7eda5f2c56160d406869a8aa4b2f365d544c7b",
	})

	require.NoError(t, err)
	assert.Equal(t, big.NewInt(25555842204), res.TokenAmountOut.Amount)
}
