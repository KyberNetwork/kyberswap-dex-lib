package integral

import (
	"math/big"
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
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

var (
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
	t.Parallel()
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
				liquidity:  mockLiquidity,
				ticks:      mockTicks,
				tickMin:    mockTickmin,
				tickMax:    mockTickmax,
				timepoints: mockTimepoints,
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
					Amount: big.NewInt(999561),
				},
				Fee: &pool.TokenAmount{
					Token:  "0x06efdbff2a14a7c8e15944d1f4a48f9f95f663a4",
					Amount: big.NewInt(15),
				},
				SwapInfo: StateUpdate{
					Liquidity: uint256.NewInt(98862330578),
					Price:     uint256.MustFromDecimal("79214336503197110663621051079"),
					Tick:      -4,
				},
				Gas: 281278,
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
				liquidity:  mockLiquidity,
				ticks:      mockTicks,
				tickMin:    mockTickmin,
				tickMax:    mockTickmax,
				timepoints: mockTimepoints,
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
					Amount: big.NewInt(997317),
				},
				Fee: &pool.TokenAmount{
					Token:  "0xf55bec9cafdbe8730f096aa55dad6d22d44099df",
					Amount: big.NewInt(450),
				},
				SwapInfo: StateUpdate{
					Liquidity: uint256.NewInt(98862330578),
					Price:     uint256.MustFromDecimal("79215936545101674541845231019"),
					Tick:      -4,
				},
				Gas: 281278,
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
				liquidity:  mockLiquidity,
				ticks:      mockTicks,
				tickMin:    mockTickmin,
				tickMax:    mockTickmax,
				timepoints: mockTimepoints,
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
					Amount: big.NewInt(1425476892),
				},
				Fee: &pool.TokenAmount{
					Token:  "0x06efdbff2a14a7c8e15944d1f4a48f9f95f663a4",
					Amount: big.NewInt(641565247949996),
				},
				SwapInfo: StateUpdate{
					Liquidity: uint256.NewInt(35733795),
					Price:     uint256.MustFromDecimal("1991751945353340918"),
					Tick:      -488157,
				},
				Gas: 344516,
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
				liquidity:  mockLiquidity,
				ticks:      mockTicks,
				tickMin:    mockTickmin,
				tickMax:    mockTickmax,
				timepoints: mockTimepoints,
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
					Amount: big.NewInt(776240305),
				},
				Fee: &pool.TokenAmount{
					Token:  "0xf55bec9cafdbe8730f096aa55dad6d22d44099df",
					Amount: big.NewInt(367831),
				},
				SwapInfo: StateUpdate{
					Liquidity: uint256.NewInt(3480992933),
					Price:     uint256.MustFromDecimal("86573656772143189240293883608"),
					Tick:      1773,
				},
				Gas: 317414,
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
	ps = lo.Must(NewPoolSimulator(p))
)

func TestCalcAmountOut_FromPool(t *testing.T) {
	t.Parallel()
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

var (
	thenaEp entity.Pool
	_       = lo.Must(0,
		json.Unmarshal([]byte(`{"address":"0x9ea0f51fd2133d995cf00229bc523737415ad318","exchange":"thena-fusion-v3","type":"algebra-integral","timestamp":1737562946,"reserves":["18414865277861570689","35620318087431674"],"tokens":[{"address":"0x55d398326f99059ff775485246999027b3197955","name":"Tether USD","symbol":"USDT","decimals":18,"weight":50,"swappable":true},{"address":"0xbb4cdb9cbd36b01bd1cbaebf2de08d9173bc095c","name":"Wrapped BNB","symbol":"WBNB","decimals":18,"weight":50,"swappable":true}],"extra":"{\"liq\":\"7454466039228971588\",\"gS\":{\"price\":\"3013079375406544485683250193\",\"tick\":-65391,\"lF\":840,\"pC\":195,\"cF\":900,\"un\":true},\"ticks\":[{\"Index\":-887220,\"LiquidityGross\":\"54665134789121271\",\"LiquidityNet\":\"54665134789121271\"},{\"Index\":-604800,\"LiquidityGross\":\"378334690498943168\",\"LiquidityNet\":\"378334690498943168\"},{\"Index\":-83460,\"LiquidityGross\":\"114198463289361161\",\"LiquidityNet\":\"114198463289361161\"},{\"Index\":-81360,\"LiquidityGross\":\"1028162166888171618\",\"LiquidityNet\":\"1028162166888171618\"},{\"Index\":-74460,\"LiquidityGross\":\"114198463289361161\",\"LiquidityNet\":\"-114198463289361161\"},{\"Index\":-72360,\"LiquidityGross\":\"1028162166888171618\",\"LiquidityNet\":\"-1028162166888171618\"},{\"Index\":-67440,\"LiquidityGross\":\"30061094172819371\",\"LiquidityNet\":\"30061094172819371\"},{\"Index\":-66900,\"LiquidityGross\":\"516014702169853064\",\"LiquidityNet\":\"516014702169853064\"},{\"Index\":-66240,\"LiquidityGross\":\"369944374456874395\",\"LiquidityNet\":\"369944374456874395\"},{\"Index\":-66060,\"LiquidityGross\":\"2161245792615067038\",\"LiquidityNet\":\"2161245792615067038\"},{\"Index\":-65940,\"LiquidityGross\":\"443254816312717934\",\"LiquidityNet\":\"443254816312717934\"},{\"Index\":-65820,\"LiquidityGross\":\"1262134329124297016\",\"LiquidityNet\":\"1262134329124297016\"},{\"Index\":-65640,\"LiquidityGross\":\"1408506298369874820\",\"LiquidityNet\":\"1408506298369874820\"},{\"Index\":-65460,\"LiquidityGross\":\"830304806719403511\",\"LiquidityNet\":\"830304806719403511\"},{\"Index\":-65340,\"LiquidityGross\":\"72467299639106158\",\"LiquidityNet\":\"64833397926204928\"},{\"Index\":-65280,\"LiquidityGross\":\"2161245792615067038\",\"LiquidityNet\":\"-2161245792615067038\"},{\"Index\":-65040,\"LiquidityGross\":\"1262134329124297016\",\"LiquidityNet\":\"-1262134329124297016\"},{\"Index\":-64920,\"LiquidityGross\":\"1477156647152530363\",\"LiquidityNet\":\"-1477156647152530363\"},{\"Index\":-64860,\"LiquidityGross\":\"516014702169853064\",\"LiquidityNet\":\"-516014702169853064\"},{\"Index\":-64740,\"LiquidityGross\":\"585016684454068041\",\"LiquidityNet\":\"-585016684454068041\"},{\"Index\":-64200,\"LiquidityGross\":\"369944374456874395\",\"LiquidityNet\":\"-369944374456874395\"},{\"Index\":-63360,\"LiquidityGross\":\"30061094172819371\",\"LiquidityNet\":\"-30061094172819371\"},{\"Index\":-63060,\"LiquidityGross\":\"688542938578053404\",\"LiquidityNet\":\"-688542938578053404\"},{\"Index\":-55260,\"LiquidityGross\":\"453104336449262994\",\"LiquidityNet\":\"453104336449262994\"},{\"Index\":-52260,\"LiquidityGross\":\"453104336449262994\",\"LiquidityNet\":\"-453104336449262994\"},{\"Index\":-50520,\"LiquidityGross\":\"2670414588260124137\",\"LiquidityNet\":\"2670414588260124137\"},{\"Index\":-50460,\"LiquidityGross\":\"2670414588260124137\",\"LiquidityNet\":\"-2670414588260124137\"},{\"Index\":604800,\"LiquidityGross\":\"378334690498943168\",\"LiquidityNet\":\"-378334690498943168\"},{\"Index\":887220,\"LiquidityGross\":\"50848183932670656\",\"LiquidityNet\":\"-50848183932670656\"}],\"tS\":60,\"tP\":{\"0\":{\"init\":true,\"ts\":1737324773,\"vo\":\"0\",\"tick\":-65495,\"avgT\":-65495},\"54\":{\"init\":true,\"ts\":1737458202,\"cum\":-8719633295,\"vo\":\"1348335241\",\"tick\":-65324,\"avgT\":-65323,\"wsI\":32},\"55\":{\"init\":true,\"ts\":1737477357,\"cum\":-9971048600,\"vo\":\"1356463794\",\"tick\":-65331,\"avgT\":-65300,\"wsI\":39},\"56\":{\"init\":true,\"ts\":1737477411,\"cum\":-9974578742,\"vo\":\"1356751560\",\"tick\":-65373,\"avgT\":-65300,\"wsI\":39},\"57\":{\"init\":true,\"ts\":1737505454,\"cum\":-11808983544,\"vo\":\"1649854548\",\"tick\":-65414,\"avgT\":-65324,\"wsI\":44},\"58\":{\"init\":true,\"ts\":1737505604,\"cum\":-11818784394,\"vo\":\"1649888298\",\"tick\":-65339,\"avgT\":-65324,\"wsI\":44},\"59\":{\"init\":true,\"ts\":1737505976,\"cum\":-11843086410,\"vo\":\"1649892882\",\"tick\":-65328,\"avgT\":-65325,\"wsI\":44},\"60\":{\"init\":true,\"ts\":1737506084,\"cum\":-11850139566,\"vo\":\"1649927874\",\"tick\":-65307,\"avgT\":-65325,\"wsI\":44},\"61\":{\"init\":true,\"ts\":1737506432,\"cum\":-11872863270,\"vo\":\"1650181566\",\"tick\":-65298,\"avgT\":-65325,\"wsI\":44},\"62\":{\"init\":true,\"ts\":1737506573,\"cum\":-11882072544,\"vo\":\"1650198627\",\"tick\":-65314,\"avgT\":-65325,\"wsI\":44},\"63\":{\"init\":true,\"ts\":1737513536,\"cum\":-12337390077,\"vo\":\"1676165365\",\"tick\":-65391,\"avgT\":-65335,\"wsI\":44},\"64\":{\"init\":true,\"ts\":1737528797,\"cum\":-13335352650,\"vo\":\"1710491407\",\"tick\":-65393,\"avgT\":-65357,\"wsI\":45},\"65\":{\"init\":true,\"ts\":1737528803,\"cum\":-13335744780,\"vo\":\"1710491431\",\"tick\":-65355,\"avgT\":-65357,\"wsI\":45},\"66\":{\"init\":true,\"ts\":1737533351,\"cum\":-13632979320,\"vo\":\"1710618805\",\"tick\":-65355,\"avgT\":-65363,\"wsI\":45},\"67\":{\"init\":true,\"ts\":1737537416,\"cum\":-13899005115,\"vo\":\"1734424264\",\"tick\":-65443,\"avgT\":-65370,\"wsI\":46},\"68\":{\"init\":true,\"ts\":1737543071,\"cum\":-14269266240,\"vo\":\"1787219588\",\"tick\":-65475,\"avgT\":-65387,\"wsI\":50},\"69\":{\"init\":true,\"ts\":1737546995,\"cum\":-14526158748,\"vo\":\"1809904932\",\"tick\":-65467,\"avgT\":-65395,\"wsI\":54},\"70\":{\"init\":true,\"ts\":1737548408,\"cum\":-14618737095,\"vo\":\"1831109455\",\"tick\":-65519,\"avgT\":-65398,\"wsI\":54},\"71\":{\"init\":true,\"ts\":1737553181,\"cum\":-14931402006,\"vo\":\"1882772958\",\"tick\":-65507,\"avgT\":-65408,\"wsI\":54},\"72\":{\"init\":true,\"ts\":1737553955,\"cum\":-14982078882,\"vo\":\"1886093610\",\"tick\":-65474,\"avgT\":-65409,\"wsI\":54},\"73\":{\"init\":true,\"ts\":1737557573,\"cum\":-15218815476,\"vo\":\"1887773460\",\"tick\":-65433,\"avgT\":-65414,\"wsI\":54},\"74\":{\"init\":true,\"ts\":1737562367,\"cum\":-15532443750,\"vo\":\"1887880503\",\"tick\":-65421,\"avgT\":-65419,\"wsI\":54},\"75\":{\"init\":true,\"ts\":1737562850,\"cum\":-15564033882,\"vo\":\"1887989178\",\"tick\":-65404,\"avgT\":-65419,\"wsI\":54},\"76\":{\"vo\":\"0\"},\"77\":{\"vo\":\"0\"}},\"vo\":{\"tpIdx\":75,\"lastTs\":1737562850,\"init\":true},\"dF\":{\"a1\":500,\"a2\":200,\"b1\":360,\"b2\":60000,\"g1\":59,\"g2\":8500,\"bF\":490},\"sF\":{\"0to1fF\":\"79228162514264337593543950336\",\"1to0fF\":\"79228162514264337593543950336\",\"pCF\":1000,\"bF\":3000,\"feeType\":false}}","staticExtra":"{\"pluginV2\":true}"}`),
			&thenaEp))
	thenaPS = lo.Must(NewPoolSimulator(thenaEp))
	_       = func() bool { blockTimestamp = func() uint32 { return 1737563754 }; return true }()
)

func TestCalcAmountOut_Ver_1_2(t *testing.T) {
	t.Parallel()
	res, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountOutResult, error) {
		return thenaPS.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{
				Token:  "0xbb4cdb9cbd36b01bd1cbaebf2de08d9173bc095c",
				Amount: big.NewInt(1e16),
			},
			TokenOut: "0x55d398326f99059ff775485246999027b3197955",
		})
	})

	require.NoError(t, err)
	assert.Equal(t, big.NewInt(6427705112340899769), res.TokenAmountOut.Amount)
}

// TrebleSwap (Base chain) - Algebra Integral v1.2.2
// Pool: WETH/USDC 0x256f399754f7ed5baa75b911ae6fd3c1a63b169c
var (
	trebleEp entity.Pool
	_        = lo.Must(0,
		json.Unmarshal([]byte(`{"address":"0x256f399754f7ed5baa75b911ae6fd3c1a63b169c","exchange":"trebleswap","type":"algebra-integral","timestamp":1771332717,"reserves":["6500000000000000000","13000000000"],"tokens":[{"address":"0x4200000000000000000000000000000000000006","name":"Wrapped Ether","symbol":"WETH","decimals":18,"weight":50,"swappable":true},{"address":"0x833589fcd6edb6e08f4c7c32d4f71b54bda02913","name":"USD Coin","symbol":"USDC","decimals":6,"weight":50,"swappable":true}],"extra":"{\"liq\":\"864523148948605\",\"gS\":{\"price\":\"3516580587790024869507051\",\"tick\":-200462,\"lF\":500,\"pC\":215,\"cF\":100,\"un\":true},\"ticks\":[{\"Index\":-887220,\"LiquidityGross\":\"13544724490664\",\"LiquidityNet\":\"13544724490664\"},{\"Index\":-209460,\"LiquidityGross\":\"850966927497294\",\"LiquidityNet\":\"850966927497294\"},{\"Index\":-200640,\"LiquidityGross\":\"11623664953\",\"LiquidityNet\":\"11496960647\"},{\"Index\":-200460,\"LiquidityGross\":\"12666966315\",\"LiquidityNet\":\"9423968815\"},{\"Index\":-199560,\"LiquidityGross\":\"11045467565\",\"LiquidityNet\":\"-11045467565\"},{\"Index\":-198240,\"LiquidityGross\":\"11560312800\",\"LiquidityNet\":\"-11560312800\"},{\"Index\":-193380,\"LiquidityGross\":\"850966927497294\",\"LiquidityNet\":\"-850966927497294\"},{\"Index\":887220,\"LiquidityGross\":\"13543039639761\",\"LiquidityNet\":\"-13543039639761\"}],\"tS\":60,\"tP\":{\"21197\":{\"init\":true,\"ts\":1771246279,\"cum\":-544971382236,\"vo\":\"210183180748\",\"tick\":-200299,\"avgT\":-200445,\"wsI\":15784},\"23879\":{\"init\":true,\"ts\":1771332617,\"cum\":-562274074386,\"vo\":\"210669818829\",\"tick\":-200455,\"avgT\":-200406,\"wsI\":21187},\"23880\":{\"init\":true,\"ts\":1771332619,\"cum\":-562274475306,\"vo\":\"210669824661\",\"tick\":-200460,\"avgT\":-200406,\"wsI\":21188},\"23881\":{\"init\":true,\"ts\":1771332717,\"cum\":-562294120582,\"vo\":\"210670131989\",\"tick\":-200462,\"avgT\":-200406,\"wsI\":21197}},\"vo\":{\"tpIdx\":23881,\"lastTs\":1771332717,\"init\":true},\"dF\":{\"a1\":100,\"a2\":200,\"b1\":360,\"b2\":60000,\"g1\":59,\"g2\":8500,\"bF\":200},\"sF\":{\"0to1fF\":\"79228162514264337593543950336\",\"1to0fF\":\"79228162514264337593543950336\",\"pCF\":0,\"bF\":0,\"feeType\":false}}","staticExtra":"{\"pluginV2\":true}"}`),
			&trebleEp))
	treblePS = lo.Must(NewPoolSimulator(trebleEp))
)

func TestCalcAmountOut_TrebleSwap(t *testing.T) {
	t.Parallel()
	// Swap 0.01 WETH -> USDC on TrebleSwap (Base)
	res, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountOutResult, error) {
		return treblePS.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{
				Token:  "0x4200000000000000000000000000000000000006", // WETH
				Amount: big.NewInt(1e16),                            // 0.01 WETH
			},
			TokenOut: "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913", // USDC
		})
	})

	require.NoError(t, err)
	assert.Equal(t, big.NewInt(19680765), res.TokenAmountOut.Amount)
}

// TrebleSwap Pool: USDC/TREB 0x6d354e51dd1e390851353ba5da4cb5737e62909e
var (
	trebleUsdcTrebEp entity.Pool
	_                = lo.Must(0,
		json.Unmarshal([]byte(`{"address":"0x6d354e51dd1e390851353ba5da4cb5737e62909e","exchange":"trebleswap","type":"algebra-integral","timestamp":1771326329,"reserves":["16700000000","25000000000000000000000"],"tokens":[{"address":"0x833589fcd6edb6e08f4c7c32d4f71b54bda02913","name":"USD Coin","symbol":"USDC","decimals":6,"weight":50,"swappable":true},{"address":"0xdd2fc771ddab2b787aedfd100a67d8a4754a380c","name":"Treble","symbol":"TREB","decimals":18,"weight":50,"swappable":true}],"extra":"{\"liq\":\"11985586389793724\",\"gS\":{\"price\":\"390096791648205279858193227186312903\",\"tick\":308206,\"lF\":500,\"pC\":215,\"cF\":100,\"un\":true},\"ticks\":[{\"Index\":-887220,\"LiquidityGross\":\"338820095940591621\",\"LiquidityNet\":\"338820095940591621\"},{\"Index\":292440,\"LiquidityGross\":\"331648476665730872\",\"LiquidityNet\":\"-331648476665730872\"},{\"Index\":304380,\"LiquidityGross\":\"102432555408153\",\"LiquidityNet\":\"102432555408153\"},{\"Index\":304980,\"LiquidityGross\":\"1732083318745\",\"LiquidityNet\":\"1296767196881\"},{\"Index\":306060,\"LiquidityGross\":\"495781564396439\",\"LiquidityNet\":\"495781564396439\"},{\"Index\":306540,\"LiquidityGross\":\"102432555408153\",\"LiquidityNet\":\"-102432555408153\"},{\"Index\":306780,\"LiquidityGross\":\"467528137008934\",\"LiquidityNet\":\"467528137008934\"},{\"Index\":307500,\"LiquidityGross\":\"495781564396439\",\"LiquidityNet\":\"-495781564396439\"},{\"Index\":307740,\"LiquidityGross\":\"4344756322246972\",\"LiquidityNet\":\"4344756322246972\"},{\"Index\":308160,\"LiquidityGross\":\"1787834197756\",\"LiquidityNet\":\"385888480188\"},{\"Index\":309180,\"LiquidityGross\":\"4344756322246972\",\"LiquidityNet\":\"-4344756322246972\"},{\"Index\":309840,\"LiquidityGross\":\"89181637287761001\",\"LiquidityNet\":\"89181637287761001\"},{\"Index\":309900,\"LiquidityGross\":\"89181637287761001\",\"LiquidityNet\":\"-89181637287761001\"},{\"Index\":310560,\"LiquidityGross\":\"467528137008934\",\"LiquidityNet\":\"-467528137008934\"},{\"Index\":311400,\"LiquidityGross\":\"94487594555140975\",\"LiquidityNet\":\"94487594555140975\"},{\"Index\":311460,\"LiquidityGross\":\"94487594555140975\",\"LiquidityNet\":\"-94487594555140975\"},{\"Index\":313380,\"LiquidityGross\":\"1086861338972\",\"LiquidityNet\":\"-1086861338972\"},{\"Index\":320100,\"LiquidityGross\":\"1514425257813\",\"LiquidityNet\":\"-1514425257813\"},{\"Index\":887220,\"LiquidityGross\":\"7170700643941033\",\"LiquidityNet\":\"-7170700643941033\"}],\"tS\":60,\"tP\":{\"1421\":{\"init\":true,\"ts\":1771326315,\"cum\":121683264476,\"vo\":\"672591899740\",\"tick\":308211,\"avgT\":307966,\"wsI\":1374},\"1422\":{\"init\":true,\"ts\":1771326329,\"cum\":121687583672,\"vo\":\"672596103996\",\"tick\":308514,\"avgT\":307966,\"wsI\":1374}},\"vo\":{\"tpIdx\":1422,\"lastTs\":1771326329,\"init\":true},\"dF\":{\"a1\":0,\"a2\":0,\"b1\":360,\"b2\":60000,\"g1\":59,\"g2\":8500,\"bF\":20000},\"sF\":{\"0to1fF\":\"79228162514264337593543950336\",\"1to0fF\":\"79228162514264337593543950336\",\"pCF\":0,\"bF\":0,\"feeType\":false}}","staticExtra":"{\"pluginV2\":true}"}`),
			&trebleUsdcTrebEp))
	trebleUsdcTrebPS = lo.Must(NewPoolSimulator(trebleUsdcTrebEp))
)

func TestCalcAmountOut_TrebleSwap_USDC_TREB(t *testing.T) {
	t.Parallel()
	// Swap 10 USDC -> TREB on TrebleSwap (Base)
	res, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountOutResult, error) {
		return trebleUsdcTrebPS.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{
				Token:  "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913", // USDC
				Amount: big.NewInt(10_000_000),                        // 10 USDC
			},
			TokenOut: "0xdd2fc771ddab2b787aedfd100a67d8a4754a380c", // TREB
		})
	})

	require.NoError(t, err)
	expected, _ := new(big.Int).SetString("236628303958585824752", 10)
	assert.Equal(t, expected, res.TokenAmountOut.Amount)
}

// TrebleSwap Pool: ETH/TREB 0x86622de331f75a2c0cd3d59dca862b82efd970d9
var (
	trebleEthTrebEp entity.Pool
	_               = lo.Must(0,
		json.Unmarshal([]byte(`{"address":"0x86622de331f75a2c0cd3d59dca862b82efd970d9","exchange":"trebleswap","type":"algebra-integral","timestamp":1771332537,"reserves":["6600000000000000000","20000000000000000000000"],"tokens":[{"address":"0x4200000000000000000000000000000000000006","name":"Wrapped Ether","symbol":"WETH","decimals":18,"weight":50,"swappable":true},{"address":"0xdd2fc771ddab2b787aedfd100a67d8a4754a380c","name":"Treble","symbol":"TREB","decimals":18,"weight":50,"swappable":true}],"extra":"{\"liq\":\"1367695178967723519455\",\"gS\":{\"price\":\"17368862755344960704543608531493\",\"tick\":107807,\"lF\":500,\"pC\":215,\"cF\":100,\"un\":true},\"ticks\":[{\"Index\":-887220,\"LiquidityGross\":\"14913288986969304337704\",\"LiquidityNet\":\"14913288986969304337704\"},{\"Index\":23040,\"LiquidityGross\":\"13559488140812048306952\",\"LiquidityNet\":\"-13559488140812048306952\"},{\"Index\":101880,\"LiquidityGross\":\"31774224174149071\",\"LiquidityNet\":\"31774224174149071\"},{\"Index\":105720,\"LiquidityGross\":\"13808063828146394592\",\"LiquidityNet\":\"13808063828146394592\"},{\"Index\":107400,\"LiquidityGross\":\"55878136434746916\",\"LiquidityNet\":\"54494758146945040\"},{\"Index\":108660,\"LiquidityGross\":\"32481436244952349\",\"LiquidityNet\":\"-31067012103345793\"},{\"Index\":109020,\"LiquidityGross\":\"13808063828146394592\",\"LiquidityNet\":\"-13808063828146394592\"},{\"Index\":111360,\"LiquidityGross\":\"55186447290845978\",\"LiquidityNet\":\"-55186447290845978\"},{\"Index\":887220,\"LiquidityGross\":\"1353800861680182933092\",\"LiquidityNet\":\"-1353800861680182933092\"}],\"tS\":60,\"tP\":{\"922\":{\"init\":true,\"ts\":1771327457,\"cum\":41954069246,\"vo\":\"533134328944\",\"tick\":107824,\"avgT\":107655,\"wsI\":893},\"923\":{\"init\":true,\"ts\":1771332537,\"cum\":42501810086,\"vo\":\"533209284397\",\"tick\":107823,\"avgT\":107755,\"wsI\":895}},\"vo\":{\"tpIdx\":923,\"lastTs\":1771332537,\"init\":true},\"dF\":{\"a1\":0,\"a2\":0,\"b1\":360,\"b2\":60000,\"g1\":59,\"g2\":8500,\"bF\":20000},\"sF\":{\"0to1fF\":\"79228162514264337593543950336\",\"1to0fF\":\"79228162514264337593543950336\",\"pCF\":0,\"bF\":0,\"feeType\":false}}","staticExtra":"{\"pluginV2\":true}"}`),
			&trebleEthTrebEp))
	trebleEthTrebPS = lo.Must(NewPoolSimulator(trebleEthTrebEp))
)

func TestCalcAmountOut_TrebleSwap_ETH_TREB(t *testing.T) {
	t.Parallel()
	// Swap 0.01 WETH -> TREB on TrebleSwap (Base)
	res, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountOutResult, error) {
		return trebleEthTrebPS.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{
				Token:  "0x4200000000000000000000000000000000000006", // WETH
				Amount: big.NewInt(1e16),                            // 0.01 WETH
			},
			TokenOut: "0xdd2fc771ddab2b787aedfd100a67d8a4754a380c", // TREB
		})
	})

	require.NoError(t, err)
	expected, _ := new(big.Int).SetString("470249138528089062086", 10)
	assert.Equal(t, expected, res.TokenAmountOut.Amount)
}

func TestPoolSimulator_CalcAmountIn(t *testing.T) {
	t.Parallel()
	testutil.TestCalcAmountIn(t, ps)
}
