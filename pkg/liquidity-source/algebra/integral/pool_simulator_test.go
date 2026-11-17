package integral

import (
	"bytes"
	altjson "encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"

	"github.com/KyberNetwork/int256"
	"github.com/KyberNetwork/msgpack/v5"
	v3Entities "github.com/KyberNetwork/uniswapv3-sdk-uint256/entities"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/algebra/integral/nami"
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

	// fresh storage per simulator: the first quote writes a timepoint into it
	newMockTimepoints = func() *TimepointStorage {
		return NewTimepointStorage(map[uint16]Timepoint{
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
	}

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
				timepoints: newMockTimepoints(),
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
				useBasePluginV2: false,
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
				timepoints: newMockTimepoints(),
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
				useBasePluginV2: false,
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
					Amount: big.NewInt(1000218),
				},
				Fee: &pool.TokenAmount{
					Token:  "0xf55bec9cafdbe8730f096aa55dad6d22d44099df",
					Amount: big.NewInt(15),
				},
				SwapInfo: StateUpdate{
					Liquidity: uint256.NewInt(98862330578),
					Price:     uint256.MustFromDecimal("79215938869158470978483439737"),
					Tick:      -4,
				},
				Gas: 281278,
			},
			expectedErr: nil,
		},
		{
			name: "swap token 0 to token 1 with large amount in",
			simulator: &PoolSimulator{
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
				timepoints: newMockTimepoints(),
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
				useBasePluginV2: false,
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
					Amount: big.NewInt(21385508264997),
				},
				SwapInfo: StateUpdate{
					Liquidity: uint256.NewInt(35733795),
					Price:     uint256.MustFromDecimal("1985975287040092911"),
					Tick:      -488215,
				},
				Gas: 344516,
			},
			expectedErr: nil,
		},
		{
			name: "swap token 1 to token 0 with large amount in",
			simulator: &PoolSimulator{
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
				timepoints: newMockTimepoints(),
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
				useBasePluginV2: false,
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
					Amount: big.NewInt(778224361),
				},
				Fee: &pool.TokenAmount{
					Token:  "0xf55bec9cafdbe8730f096aa55dad6d22d44099df",
					Amount: big.NewInt(12259),
				},
				SwapInfo: StateUpdate{
					Liquidity: uint256.NewInt(3480992933),
					Price:     uint256.MustFromDecimal("86627609485054744257740325762"),
					Tick:      1785,
				},
				Gas: 317414,
			},
			expectedErr: nil,
		},
	}

	for _, tt := range tests {
		tt.simulator.pending = newPendingWrite(tt.simulator, blockTimestamp()) // as NewPoolSimulator does
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
				actualSwapInfo.write, actualSwapInfo.writeFee = nil, 0 // the pending write, for UpdateBalance

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
	// The clock is set before thenaPS is built: a simulator's first swap takes the time it was built.
	_       = func() bool { blockTimestamp = func() uint32 { return 1737563754 }; return true }()
	thenaPS = lo.Must(NewPoolSimulator(thenaEp))
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

func TestPoolSimulator_CalcAmountIn(t *testing.T) {
	t.Parallel()
	testutil.TestCalcAmountIn(t, ps)
}

// cloneTestPool is mockPool (a dynamic fee pool with a volatility oracle) with the clock at now.
func cloneTestPool(t *testing.T, now uint32) entity.Pool {
	old := blockTimestamp
	t.Cleanup(func() { blockTimestamp = old })
	setClock(now)
	var ep entity.Pool
	require.NoError(t, json.Unmarshal(mockPool, &ep))
	return ep
}

func setClock(ts uint32) { blockTimestamp = func() uint32 { return ts } }

func quoteOn(t *testing.T, sim pool.IPoolSimulator, ep entity.Pool, zeroForOne bool, amount *big.Int) *pool.CalcAmountOutResult {
	in, out := ep.Tokens[0].Address, ep.Tokens[1].Address
	if !zeroForOne {
		in, out = out, in
	}
	res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: in, Amount: amount}, TokenOut: out})
	require.NoError(t, err)
	return res
}

// A clone's quote writes the plugin's timepoint, as the swap would on-chain. It must write it into
// state of its own: the simulator it was cloned from, and every other clone of it, are other
// requests' pools, and a timepoint at a tick only one route reached would change their fees.
func TestCloneQuotesLeaveTheOriginalsTimepoints(t *testing.T) {
	const now = uint32(1733225400)
	ep := cloneTestPool(t, now)
	sim, control := lo.Must(NewPoolSimulator(ep)), lo.Must(NewPoolSimulator(ep))
	small, large := big.NewInt(1e12), big.NewInt(5e15)

	quoteOn(t, sim, ep, true, small)
	quoteOn(t, control, ep, true, small)
	oracle, timepoints := *sim.volatilityOracle, len(sim.timepoints.data)

	// another request: a clone takes a large swap, and a second later a clone of that clone is quoted
	c := sim.CloneState()
	res := quoteOn(t, c, ep, true, large)
	c.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: pool.TokenAmount{Token: ep.Tokens[0].Address,
		Amount: large}, TokenAmountOut: *res.TokenAmountOut, Fee: *res.Fee, SwapInfo: res.SwapInfo})
	setClock(now + 1)
	quoteOn(t, c.CloneState(), ep, false, small)

	require.Equal(t, oracle, *sim.volatilityOracle, "the original's oracle")
	require.Len(t, sim.timepoints.data, timepoints, "the original's timepoints")

	// a later request's clone quotes as one of a pool no other request touched
	setClock(now + 2)
	require.Equal(t, quoteOn(t, control.CloneState(), ep, false, small).TokenAmountOut.Amount.String(),
		quoteOn(t, sim.CloneState(), ep, false, small).TokenAmountOut.Amount.String())
}

// Clones quoted on concurrent requests share nothing they write (run with -race).
func TestConcurrentCloneQuotes(t *testing.T) {
	const now = uint32(1733225400)
	ep := cloneTestPool(t, now)
	sim := lo.Must(NewPoolSimulator(ep))
	quoteOn(t, sim, ep, true, big.NewInt(1e12))
	setClock(now + 1)
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			quoteOn(t, sim.CloneState(), ep, false, big.NewInt(1e12))
		}()
	}
	wg.Wait()
}

// A pool's first quote writes its timepoint, while a request may clone the same pool and quote the
// clone, which copies the pool's oracle and timepoints before its own write. Neither may race with the
// other (run with -race).
func TestFirstQuoteWhileACloneQuotes(t *testing.T) {
	const now = uint32(1733225400)
	ep := cloneTestPool(t, now)
	for range 50 {
		sim := lo.Must(NewPoolSimulator(ep))
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			quoteOn(t, sim, ep, true, big.NewInt(1e12))
		}()
		go func() {
			defer wg.Done()
			quoteOn(t, sim.CloneState(), ep, false, big.NewInt(1e12))
		}()
		wg.Wait()
	}
}

// TestSwapsInARowWhileTheClockTicks replays the swaps in a row of testdata/swaps, as
// TestSwapsInARowMatchChain does, with the wall clock one second on after each swap's UpdateBalance.
// On-chain the swaps of one transaction share one block timestamp, so a route's later swaps through
// the pool must take the fee factors of the timepoint its first swap wrote, whatever the clock does.
func TestSwapsInARowWhileTheClockTicks(t *testing.T) {
	fixtures := loadOnchainFeeFixtures(t)
	files, err := filepath.Glob("testdata/swaps/*.json")
	require.NoError(t, err)
	require.NotEmpty(t, files)
	for _, file := range files {
		name := filepath.Base(file)
		t.Run(name, func(t *testing.T) {
			fixture, ok := fixtures[name]
			require.True(t, ok, "no on-chain fixture for %s", name)
			b, err := os.ReadFile(file)
			require.NoError(t, err)
			var recorded struct {
				Blocks []struct {
					Block          uint64           `json:"block"`
					BlockTimestamp uint32           `json:"blockTimestamp"`
					Swaps          [][]recordedSwap `json:"swaps"`
				} `json:"blocks"`
			}
			require.NoError(t, altjson.Unmarshal(b, &recorded))
			token0, token1 := fixture.Pool.Tokens[0].Address, fixture.Pool.Tokens[1].Address

			for _, blk := range recorded.Blocks {
				setBlockTimestamp(t, blk.BlockTimestamp)
				sim, err := NewPoolSimulator(fixture.Pool)
				require.NoError(t, err)
				for _, swaps := range blk.Swaps {
					if len(swaps) < 2 {
						continue
					}
					setBlockTimestamp(t, blk.BlockTimestamp)
					s := sim.CloneState().(*PoolSimulator)
					for i, sw := range swaps {
						in, out := token0, token1
						if !sw.ZeroToOne {
							in, out = token1, token0
						}
						amount, _ := new(big.Int).SetString(sw.Amount, 10)
						var gotIn, gotOut *big.Int
						var info any
						if sw.ExactIn {
							res, err := s.CalcAmountOut(pool.CalcAmountOutParams{
								TokenAmountIn: pool.TokenAmount{Token: in, Amount: amount}, TokenOut: out})
							require.NoError(t, err, "block %d swap %d of %+v", blk.Block, i, swaps)
							gotIn, gotOut, info = amount, res.TokenAmountOut.Amount, res.SwapInfo
						} else {
							res, err := s.CalcAmountIn(pool.CalcAmountInParams{
								TokenAmountOut: pool.TokenAmount{Token: out, Amount: amount}, TokenIn: in})
							require.NoError(t, err, "block %d swap %d of %+v", blk.Block, i, swaps)
							gotIn, gotOut, info = res.TokenAmountIn.Amount, amount, res.SwapInfo
						}
						require.Equal(t, sw.In, gotIn.String(), "block %d swap %d of %+v: in", blk.Block, i, swaps)
						require.Equal(t, sw.Out, gotOut.String(), "block %d swap %d of %+v: out", blk.Block, i, swaps)
						s.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: info})
						setBlockTimestamp(t, blk.BlockTimestamp+uint32(i)+1)
					}
				}
			}
		})
	}
}

// A quote is read-only: a simulator quoted, either way and either side, is equal to one built from the
// same pool and never quoted. A host quotes its shared base simulator on a pool's first touch and
// clones it for other requests, so a quote that wrote a timepoint or a fee into it would change every
// later clone's quotes.
func TestQuoteLeavesTheReceiverUnchanged(t *testing.T) {
	pools := map[string]entity.Pool{"mockPool": cloneTestPool(t, 1733225400)}
	for name, f := range loadOnchainFeeFixtures(t) {
		pools[name] = f.Pool
	}
	for name, ep := range pools {
		t.Run(name, func(t *testing.T) {
			sim, fresh := lo.Must(NewPoolSimulator(ep)), lo.Must(NewPoolSimulator(ep))
			t0, t1 := ep.Tokens[0].Address, ep.Tokens[1].Address
			for _, dir := range [][2]string{{t0, t1}, {t1, t0}} {
				for _, amount := range []*big.Int{big.NewInt(1e6), big.NewInt(1e12), big.NewInt(1e15)} {
					_, _ = sim.CalcAmountOut(pool.CalcAmountOutParams{
						TokenAmountIn: pool.TokenAmount{Token: dir[0], Amount: amount}, TokenOut: dir[1]})
					_, _ = sim.CalcAmountIn(pool.CalcAmountInParams{
						TokenAmountOut: pool.TokenAmount{Token: dir[1], Amount: amount}, TokenIn: dir[0]})
				}
			}
			require.True(t, reflect.DeepEqual(sim, fresh), "the quoted simulator differs from a fresh one:\n"+
				"oracle %+v vs %+v\nlast fee %d vs %d\ntimepoints %d vs %d", sim.volatilityOracle,
				fresh.volatilityOracle, sim.globalState.LastFee, fresh.globalState.LastFee,
				len(sim.timepoints.data), len(fresh.timepoints.data))
		})
	}
}

func TestNamiQuoteTimepointPurity(t *testing.T) {
	oldTime := blockTimestamp
	blockTimestamp = func() uint32 { return 101 }
	t.Cleanup(func() { blockTimestamp = oldTime })
	sim := ps.CloneState().(*PoolSimulator)
	sim.globalState.PluginConfig |= BEFORE_SWAP_FLAG
	sim.volatilityOracle = &VolatilityOraclePlugin{IsInitialized: true, LastTimepointTimestamp: 100}
	sim.timepoints = NewTimepointStorage(map[uint16]Timepoint{0: {Initialized: true, BlockTimestamp: 100, Tick: sim.globalState.Tick, AverageTick: sim.globalState.Tick, VolatilityCumulative: uint256.NewInt(0)}})
	sim.dynamicFee = &DynamicFeeConfig{BaseFee: 300}
	sim.namiFee = &nami.Config{Floor: 100, Cap: 1000, Premium0For1: 100, Premium1For0: 200}
	tokens := sim.GetTokens()
	params := pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: tokens[0], Amount: big.NewInt(100000000000000)}, TokenOut: tokens[1]}
	first, err := sim.CalcAmountOut(params)
	require.NoError(t, err)
	second, err := sim.CalcAmountOut(params)
	require.NoError(t, err)
	require.Equal(t, first.TokenAmountOut.Amount, second.TokenAmountOut.Amount)
	require.Equal(t, uint16(0), sim.volatilityOracle.TimepointIndex)
	require.False(t, sim.timepoints.Get(1).Initialized)
	sim.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: first.SwapInfo})
	require.Equal(t, uint16(1), sim.volatilityOracle.TimepointIndex)
	require.True(t, sim.timepoints.Get(1).Initialized)
	third, err := sim.CalcAmountOut(params)
	require.NoError(t, err)
	require.NotEqual(t, first.TokenAmountOut.Amount, third.TokenAmountOut.Amount)
}

// Historical Base swaps of the METAc/USDC pool, using state from the preceding block.
// No initialized tick lies between the starting and ending ticks for either swap.
func TestNamiBaseSwapReceipts(t *testing.T) {
	oldTime := blockTimestamp
	t.Cleanup(func() { blockTimestamp = oldTime })
	for _, tc := range []struct {
		tx            string
		timestamp     uint32
		price         string
		tick          int32
		direction     int
		input, output int64
	}{
		{"0x4207ca5dbb2fc85e6eff3b41ed2eb3eb7727a503a4213712b729de4e974c7bee", 1793598911, "31900346563188641650020041102", -18196, 0, 4505817, 728168},
		{"0xffcb45b01a288e51ac6a1ca9ddf654b66770867fb733e41f28090db1f31217ae", 1793602535, "31831480227066460847131138237", -18239, 1, 763165, 4712445},
	} {
		t.Run(tc.tx, func(t *testing.T) {
			blockTimestamp = func() uint32 { return tc.timestamp }
			sim := ps.CloneState().(*PoolSimulator)
			sim.globalState = GlobalState{Price: uint256.MustFromDecimal(tc.price), Tick: tc.tick, LastFee: 500, PluginConfig: 195, CommunityFee: 1000, Unlocked: true}
			sim.liquidity = uint256.NewInt(837731207)
			ticks, err := v3Entities.NewTickListDataProvider([]v3Entities.Tick{
				{Index: -18540, LiquidityGross: uint256.NewInt(1), LiquidityNet: int256.NewInt(1)},
				{Index: -17800, LiquidityGross: uint256.NewInt(1), LiquidityNet: int256.NewInt(-1)},
			}, 1)
			require.NoError(t, err)
			sim.ticks = ticks
			sim.tickMin = -18540
			sim.tickMax = -17800
			sim.volatilityOracle = &VolatilityOraclePlugin{IsInitialized: true, LastTimepointTimestamp: tc.timestamp}
			sim.timepoints = NewTimepointStorage(map[uint16]Timepoint{0: {Initialized: true, BlockTimestamp: tc.timestamp, Tick: tc.tick, AverageTick: tc.tick, VolatilityCumulative: uint256.NewInt(0)}})
			sim.dynamicFee = &DynamicFeeConfig{BaseFee: 1000}
			sim.namiFee = &nami.Config{}
			tokens := sim.GetTokens()
			result, err := sim.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: tokens[tc.direction], Amount: big.NewInt(tc.input)}, TokenOut: tokens[1-tc.direction]})
			require.NoError(t, err)
			require.Equal(t, big.NewInt(tc.output), result.TokenAmountOut.Amount)
		})
	}
}

func TestNamiStateMsgpackRoundTrip(t *testing.T) {
	oldTime := blockTimestamp
	blockTimestamp = func() uint32 { return 101 }
	t.Cleanup(func() { blockTimestamp = oldTime })
	sim := ps.CloneState().(*PoolSimulator)
	sim.globalState.PluginConfig |= BEFORE_SWAP_FLAG
	sim.volatilityOracle = &VolatilityOraclePlugin{IsInitialized: true, LastTimepointTimestamp: 100}
	sim.timepoints = NewTimepointStorage(map[uint16]Timepoint{0: {Initialized: true, BlockTimestamp: 100, Tick: sim.globalState.Tick, AverageTick: sim.globalState.Tick, VolatilityCumulative: uint256.NewInt(0)}})
	sim.dynamicFee = &DynamicFeeConfig{BaseFee: 300}
	sim.namiFee = &nami.Config{Floor: 100, Cap: 1000, Premium0For1: 100, Premium1For0: 200}
	tokens := sim.GetTokens()
	params := pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: tokens[0], Amount: big.NewInt(100000000000000)}, TokenOut: tokens[1]}
	first, err := sim.CalcAmountOut(params)
	require.NoError(t, err)
	sim.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: first.SwapInfo})
	var buf bytes.Buffer
	enc := msgpack.NewEncoder(&buf)
	enc.IncludeUnexported(true)
	enc.SetForceAsArray(true)
	require.NoError(t, enc.Encode(sim))
	dec := msgpack.NewDecoder(&buf)
	dec.IncludeUnexported(true)
	dec.SetForceAsArray(true)
	var restored PoolSimulator
	require.NoError(t, dec.Decode(&restored))
	want, err := sim.CalcAmountOut(params)
	require.NoError(t, err)
	got, err := restored.CalcAmountOut(params)
	require.NoError(t, err)
	require.Equal(t, want.TokenAmountOut.Amount, got.TokenAmountOut.Amount)
}

func TestNamiUnsupportedStateCannotQuote(t *testing.T) {
	sim := ps.CloneState().(*PoolSimulator)
	sim.namiFee = &nami.Config{Unsupported: "Nami external fee source is not supported"}
	tokens := sim.GetTokens()
	_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: tokens[0], Amount: big.NewInt(1000000)}, TokenOut: tokens[1]})
	require.ErrorContains(t, err, "Nami external fee source")
}

func TestNamiSwapInfoDoesNotAliasUpdatedState(t *testing.T) {
	oldTime := blockTimestamp
	blockTimestamp = func() uint32 { return 101 }
	t.Cleanup(func() { blockTimestamp = oldTime })
	sim := ps.CloneState().(*PoolSimulator)
	sim.globalState.PluginConfig |= BEFORE_SWAP_FLAG
	sim.volatilityOracle = &VolatilityOraclePlugin{IsInitialized: true, LastTimepointTimestamp: 100}
	sim.timepoints = NewTimepointStorage(map[uint16]Timepoint{0: {Initialized: true, BlockTimestamp: 100, Tick: sim.globalState.Tick, AverageTick: sim.globalState.Tick, VolatilityCumulative: uint256.NewInt(5)}})
	sim.dynamicFee = &DynamicFeeConfig{BaseFee: 300}
	sim.namiFee = &nami.Config{}
	tokens := sim.GetTokens()
	result, err := sim.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: tokens[0], Amount: big.NewInt(100000000000000)}, TokenOut: tokens[1]})
	require.NoError(t, err)
	update := result.SwapInfo.(StateUpdate)
	sim.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: update})
	price, liquidity, volatility := sim.globalState.Price.Clone(), sim.liquidity.Clone(), sim.timepoints.Get(update.NamiTimepoint.Index).VolatilityCumulative.Clone()
	update.Price.Clear()
	update.Liquidity.Clear()
	update.NamiTimepoint.Timepoint.VolatilityCumulative.Clear()
	require.Equal(t, price, sim.globalState.Price)
	require.Equal(t, liquidity, sim.liquidity)
	require.Equal(t, volatility, sim.timepoints.Get(update.NamiTimepoint.Index).VolatilityCumulative)
}

// A Nami pool's swap writes its own timepoint. One that still carries the pending write of the
// pool it was cloned from must not also store that write, which would move its oracle twice.
func TestNamiSwapSkipsThePendingWrite(t *testing.T) {
	now := uint32(1700000000)
	p, err := NewPoolSimulator(cloneTestPool(t, now))
	require.NoError(t, err)
	require.NotNil(t, p.pending)
	sim := p.CloneState().(*PoolSimulator)
	sim.namiFee = &nami.Config{Floor: 100, Cap: 1000, Premium0For1: 100, Premium1For0: 200}
	tokens := sim.GetTokens()
	res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokens[0], Amount: big.NewInt(1e14)}, TokenOut: tokens[1]})
	require.NoError(t, err)
	si := res.SwapInfo.(StateUpdate)
	require.Nil(t, si.write)
	require.NotNil(t, si.NamiTimepoint)
	sim.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: si})
	require.Equal(t, si.NamiTimepoint.Index, sim.volatilityOracle.TimepointIndex)
	require.NotNil(t, sim.pending, "the pending write is left unused, not stored")
}
