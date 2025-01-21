package integral

import (
	"fmt"
	"math/big"
	"math/rand"
	"sync"
	"testing"

	"github.com/KyberNetwork/int256"
	v3Entities "github.com/KyberNetwork/uniswapv3-sdk-uint256/entities"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
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
		{Index: -887220, LiquidityGross: uint256.NewInt(35733795), LiquidityNet: int256.NewInt(35733795)},
		{Index: -4500, LiquidityGross: uint256.NewInt(1469002688), LiquidityNet: int256.NewInt(1469002688)},
		{Index: -1740, LiquidityGross: uint256.NewInt(815264000), LiquidityNet: int256.NewInt(815264000)},
		{Index: -1080, LiquidityGross: uint256.NewInt(4716862354), LiquidityNet: int256.NewInt(4716862354)},
		{Index: -960, LiquidityGross: uint256.NewInt(2130488), LiquidityNet: int256.NewInt(2130488)},
		{Index: -540, LiquidityGross: uint256.NewInt(59681565), LiquidityNet: int256.NewInt(59681565)},
		{Index: -120, LiquidityGross: uint256.NewInt(173321441467), LiquidityNet: int256.NewInt(173321441467)},
		{Index: -60, LiquidityGross: uint256.NewInt(265085097155), LiquidityNet: int256.NewInt(-81557785779)},
		{Index: 60, LiquidityGross: uint256.NewInt(91763655688), LiquidityNet: int256.NewInt(-91763655688)},
		{Index: 540, LiquidityGross: uint256.NewInt(2130488), LiquidityNet: int256.NewInt(-2130488)},
		{Index: 960, LiquidityGross: uint256.NewInt(59681565), LiquidityNet: int256.NewInt(-59681565)},
		{Index: 1080, LiquidityGross: uint256.NewInt(3555869904), LiquidityNet: int256.NewInt(-3555869904)},
		{Index: 1800, LiquidityGross: uint256.NewInt(1976256450), LiquidityNet: int256.NewInt(-1976256450)},
		{Index: 1860, LiquidityGross: uint256.NewInt(1469002688), LiquidityNet: int256.NewInt(-1469002688)},
		{Index: 887220, LiquidityGross: uint256.NewInt(35733795), LiquidityNet: int256.NewInt(-35733795)},
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

	mockAlpha1      uint16 = 2900
	mockAlpha2      uint16 = 12000
	mockBeta1       uint32 = 360
	mockBeta2       uint32 = 60000
	mockGamma1      uint16 = 59
	mockGamma2      uint16 = 8500
	mockVolumeBeta  uint32 = 0
	mockVolumeGamma uint16 = 0
	mockBaseFee     uint16 = 100
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
				volatilityOracle: &VolatilityOraclePlugin{
					TimepointIndex:         mockTimepointIndex,
					LastTimepointTimestamp: mockLastTimepointTimestamp,
					IsInitialized:          true,
				},
				dynamicFee: &DynamicFeeConfig{
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
				writeTimePointOnce: new(sync.Once),
				useBasePluginV2:    false,
				gas:                mockGas,
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
					Amount: big.NewInt(984666),
				},
				Fee: &pool.TokenAmount{
					Token:  "0x06efdbff2a14a7c8e15944d1f4a48f9f95f663a4",
					Amount: big.NewInt(2250),
				},
				SwapInfo: StateUpdate{
					Liquidity: uint256.NewInt(98862330578),
					Price:     uint256.MustFromDecimal("79214348439875248928556576945"),
					Tick:      -4,
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
				volatilityOracle: &VolatilityOraclePlugin{
					TimepointIndex:         mockTimepointIndex,
					LastTimepointTimestamp: mockLastTimepointTimestamp,
					IsInitialized:          true,
				},
				dynamicFee: &DynamicFeeConfig{
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
				writeTimePointOnce: new(sync.Once),
				useBasePluginV2:    false,
				gas:                mockGas,
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
					Amount: big.NewInt(985314),
				},
				Fee: &pool.TokenAmount{
					Token:  "0xf55bec9cafdbe8730f096aa55dad6d22d44099df",
					Amount: big.NewInt(2250),
				},
				SwapInfo: StateUpdate{
					Liquidity: uint256.NewInt(98862330578),
					Price:     uint256.MustFromDecimal("79215926928314930666100919082"),
					Tick:      -4,
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
				volatilityOracle: &VolatilityOraclePlugin{
					TimepointIndex:         mockTimepointIndex,
					LastTimepointTimestamp: mockLastTimepointTimestamp,
					IsInitialized:          true,
				},
				dynamicFee: &DynamicFeeConfig{
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
				writeTimePointOnce: new(sync.Once),
				useBasePluginV2:    false,
				gas:                mockGas,
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
					Amount: big.NewInt(279874986),
				},
				Fee: &pool.TokenAmount{
					Token:  "0x06efdbff2a14a7c8e15944d1f4a48f9f95f663a4",
					Amount: big.NewInt(641334),
				},
				SwapInfo: StateUpdate{
					Liquidity: uint256.NewInt(0),
					Price:     uint256.MustFromDecimal("4306310044"),
					Tick:      -887221,
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
				volatilityOracle: &VolatilityOraclePlugin{
					TimepointIndex:         mockTimepointIndex,
					LastTimepointTimestamp: mockLastTimepointTimestamp,
					IsInitialized:          true,
				},
				dynamicFee: &DynamicFeeConfig{
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
				writeTimePointOnce: new(sync.Once),
				useBasePluginV2:    false,
				gas:                mockGas,
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
					Amount: big.NewInt(312383229),
				},
				Fee: &pool.TokenAmount{
					Token:  "0xf55bec9cafdbe8730f096aa55dad6d22d44099df",
					Amount: big.NewInt(715591),
				},
				SwapInfo: StateUpdate{
					Liquidity: uint256.NewInt(0),
					Price:     uint256.MustFromDecimal("1457652066949847389969617340386294118487833376468"),
					Tick:      887220,
				},
				Gas: mockGas,
			},
			expectedErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountOutResult, error) {
				return tt.simulator.CalcAmountOut(tt.input)
			})
			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.ErrorIs(t, err, tt.expectedErr)
			} else {
				assert.NoError(t, err)
				require.NotEmpty(t, result.Fee)
				assert.Equal(t, tt.expectedResult.Fee, result.Fee)

				require.NotEmpty(t, result.Gas)
				assert.Equal(t, tt.expectedResult.Gas, result.Gas)

				require.NotEmpty(t, result.SwapInfo)

				expectedSwapInfo := tt.expectedResult.SwapInfo.(StateUpdate)
				actualSwapInfo := result.SwapInfo.(StateUpdate)

				assert.Equal(t, expectedSwapInfo, actualSwapInfo)

				require.NotEmpty(t, result.SwapInfo)
				assert.Equal(t, tt.expectedResult.TokenAmountOut, result.TokenAmountOut)
			}
		})
	}
}

var mockPool = []byte(`{"address":"0xbe9c1d237d002c8d9402f30c16ace1436d008f0c","exchange":"silverswap","type":"algebra-integral","timestamp":1733225338,"reserves":["9999999999999944","2620057588865"],"tokens":[{"address":"0x21be370d5312f44cb42ce377bc9b8a0cef1a4c83","name":"Wrapped Fantom","symbol":"WFTM","decimals":18,"weight":50,"swappable":true},{"address":"0xfe7eda5f2c56160d406869a8aa4b2f365d544c7b","name":"Axelar Wrapped ETH","symbol":"axlETH","decimals":18,"weight":50,"swappable":true}],"extra":"{\"liq\":161865919478591,\"gS\":{\"price\":\"1282433937397070526017841373\",\"tick\":82476,\"lF\":100,\"pC\":193,\"cF\":100,\"un\":true},\"ticks\":[{\"Index\":-887220,\"LiquidityGross\":161865919478591,\"LiquidityNet\":161865919478591},{\"Index\":887220,\"LiquidityGross\":161865919478591,\"LiquidityNet\":-161865919478591}],\"tS\":60,\"tP\":{\"0\":{\"init\":true,\"ts\":1712116096,\"cum\":0,\"vo\":\"0\",\"tick\":-82476,\"avgT\":-82476,\"wsI\":0},\"1\":{\"init\":false,\"ts\":0,\"cum\":0,\"vo\":\"0\",\"tick\":0,\"avgT\":0,\"wsI\":0},\"2\":{\"init\":false,\"ts\":0,\"cum\":0,\"vo\":\"0\",\"tick\":0,\"avgT\":0,\"wsI\":0},\"65535\":{\"init\":false,\"ts\":0,\"cum\":0,\"vo\":\"0\",\"tick\":0,\"avgT\":0,\"wsI\":0}},\"vo\":{\"tpIdx\":0,\"lastTs\":1712116096,\"init\":true},\"sF\":{\"0to1fF\":null,\"1to0fF\":null},\"dF\":{\"a1\":2900,\"a2\":12000,\"b1\":360,\"b2\":60000,\"g1\":59,\"g2\":8500,\"vB\":0,\"vG\":0,\"bF\":100}}","staticExtra":"{\"pluginV2\":false}","blockNumber":99019509}`)

var (
	p  entity.Pool
	_  = json.Unmarshal(mockPool, &p)
	ps = lo.Must(NewPoolSimulator(p, 280000))
)

func TestCalcAmountOut_FromPool(t *testing.T) {
	res, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountOutResult, error) {
		return ps.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{
				Token:  "0x21be370d5312f44cb42ce377bc9b8a0cef1a4c83",
				Amount: big.NewInt(100000000000000),
			},
			TokenOut: "0xfe7eda5f2c56160d406869a8aa4b2f365d544c7b",
		})
	})

	require.NoError(t, err)
	assert.Equal(t, big.NewInt(25555842204), res.TokenAmountOut.Amount)
}

func TestPoolSimulator_CalcAmountIn(t *testing.T) {
	for i := 0; i < 64; i++ {
		tokenIn := p.Tokens[i%2].Address
		tokenOut := p.Tokens[(i+1)%2].Address
		amountOut := big.NewInt(int64(rand.Uint32()))
		t.Run(fmt.Sprintf("token%d -> %s token%d", i%2, amountOut, (i+1)%2), func(t *testing.T) {
			resIn, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountInResult, error) {
				return ps.CalcAmountIn(pool.CalcAmountInParams{
					TokenAmountOut: pool.TokenAmount{
						Token:  tokenOut,
						Amount: amountOut,
					},
					TokenIn: tokenIn,
				})
			})
			require.NoError(t, err)

			if resIn.RemainingTokenAmountOut.Amount.Sign() > 0 {
				resIn, err = testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountInResult, error) {
					return ps.CalcAmountIn(pool.CalcAmountInParams{
						TokenAmountOut: pool.TokenAmount{
							Token: tokenOut,
							Amount: amountOut.Sub(amountOut, resIn.RemainingTokenAmountOut.Amount).Div(amountOut,
								bignumber.Two),
						},
						TokenIn: tokenIn,
					})
				})
				require.NoError(t, err)
			}

			resOut, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountOutResult, error) {
				return ps.CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: pool.TokenAmount{
						Token:  tokenIn,
						Amount: resIn.TokenAmountIn.Amount,
					},
					TokenOut: tokenOut,
				})
			})
			require.NoError(t, err)

			finalAmtOut := resOut.TokenAmountOut.Amount
			finalAmtOut.Sub(finalAmtOut, resIn.RemainingTokenAmountOut.Amount)
			origAmountOutF, _ := amountOut.Float64()
			finalAmountOutF, _ := finalAmtOut.Float64()
			assert.InEpsilonf(t, origAmountOutF, finalAmountOutF, 1e-4,
				"expected ~%s, got %s", amountOut, finalAmtOut)
		})
	}
}
