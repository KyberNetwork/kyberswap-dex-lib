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
					Amount: big.NewInt(3207826239749998),
				},
				SwapInfo: StateUpdate{
					Liquidity: uint256.NewInt(35733795),
					Price:     uint256.MustFromDecimal("2016016943697492749"),
					Tick:      -487914,
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
					Amount: big.NewInt(768004061),
				},
				Fee: &pool.TokenAmount{
					Token:  "0xf55bec9cafdbe8730f096aa55dad6d22d44099df",
					Amount: big.NewInt(1839166),
				},
				SwapInfo: StateUpdate{
					Liquidity: uint256.NewInt(3480992933),
					Price:     uint256.MustFromDecimal("86350404125395664252363004498"),
					Tick:      1721,
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

var mockPool = []byte(`{"address":"0x5c332ec2387acd1e403682035c3b167d82725e89","exchange":"gliquid","type":"algebra-integral","timestamp":1754847461,"reserves":["79162517167825241112","11177859266908184431828"],"tokens":[{"address":"0x5555555555555555555555555555555555555555","symbol":"WHYPE","decimals":18,"swappable":true},{"address":"0xca79db4b49f608ef54a5cb813fbed3a6387bc645","symbol":"USDXL","decimals":18,"swappable":true}],"extra":"{\"liq\":\"18421115169652838402488\",\"gS\":{\"price\":\"537403680261167190078234329935\",\"tick\":38290,\"lF\":500,\"pC\":195,\"cF\":130,\"un\":true},\"ticks\":[{\"Index\":-887220,\"LiquidityGross\":\"93901449275686617\",\"LiquidityNet\":\"93901449275686617\"},{\"Index\":35820,\"LiquidityGross\":\"2183012775232623315\",\"LiquidityNet\":\"2183012775232623315\"},{\"Index\":36060,\"LiquidityGross\":\"722191179427725567388\",\"LiquidityNet\":\"722191179427725567388\"},{\"Index\":36180,\"LiquidityGross\":\"8646113766738642164434\",\"LiquidityNet\":\"8646113766738642164434\"},{\"Index\":36420,\"LiquidityGross\":\"2115388801331039529555\",\"LiquidityNet\":\"2115388801331039529555\"},{\"Index\":36540,\"LiquidityGross\":\"1321103055829383869631\",\"LiquidityNet\":\"1321103055829383869631\"},{\"Index\":36840,\"LiquidityGross\":\"39853856306612065893\",\"LiquidityNet\":\"39853856306612065893\"},{\"Index\":37080,\"LiquidityGross\":\"1549356644122390149105\",\"LiquidityNet\":\"1549356644122390149105\"},{\"Index\":37440,\"LiquidityGross\":\"1467073455938920359550\",\"LiquidityNet\":\"1467073455938920359550\"},{\"Index\":37500,\"LiquidityGross\":\"1576787145161724744077\",\"LiquidityNet\":\"1576787145161724744077\"},{\"Index\":37620,\"LiquidityGross\":\"528832787517289706174\",\"LiquidityNet\":\"528832787517289706174\"},{\"Index\":37680,\"LiquidityGross\":\"4396338968089263012815\",\"LiquidityNet\":\"4396338968089263012815\"},{\"Index\":37920,\"LiquidityGross\":\"2183012775232623315\",\"LiquidityNet\":\"-2183012775232623315\"},{\"Index\":38040,\"LiquidityGross\":\"8942418459770041919703\",\"LiquidityNet\":\"8942418459770041919703\"},{\"Index\":38100,\"LiquidityGross\":\"39853856306612065893\",\"LiquidityNet\":\"-39853856306612065893\"},{\"Index\":38160,\"LiquidityGross\":\"12844582995722858306561\",\"LiquidityNet\":\"-12844582995722858306561\"},{\"Index\":38400,\"LiquidityGross\":\"1324011543289034365283\",\"LiquidityNet\":\"-1260270606191416436343\"},{\"Index\":38460,\"LiquidityGross\":\"9755897317708830969141\",\"LiquidityNet\":\"-9755897317708830969141\"},{\"Index\":38700,\"LiquidityGross\":\"197869739105046870688\",\"LiquidityNet\":\"-197869739105046870688\"},{\"Index\":38880,\"LiquidityGross\":\"837311715344341609669\",\"LiquidityNet\":\"-837311715344341609669\"},{\"Index\":38940,\"LiquidityGross\":\"629761740594578749881\",\"LiquidityNet\":\"-629761740594578749881\"},{\"Index\":39360,\"LiquidityGross\":\"1352973524378192834101\",\"LiquidityNet\":\"-1352973524378192834101\"},{\"Index\":39540,\"LiquidityGross\":\"2115388801331039529555\",\"LiquidityNet\":\"-2115388801331039529555\"},{\"Index\":39960,\"LiquidityGross\":\"1549356644122390149105\",\"LiquidityNet\":\"-1549356644122390149105\"},{\"Index\":40140,\"LiquidityGross\":\"722191179427725567388\",\"LiquidityNet\":\"-722191179427725567388\"},{\"Index\":887220,\"LiquidityGross\":\"93901449275686617\",\"LiquidityNet\":\"-93901449275686617\"}],\"tS\":60,\"tP\":{\"0\":{\"init\":true,\"ts\":1752858180,\"vo\":\"0\",\"tick\":38068,\"avgT\":38068},\"13990\":{\"init\":true,\"ts\":1754759545,\"cum\":71449548487,\"vo\":\"93775609683\",\"tick\":38017,\"avgT\":37540,\"wsI\":13610},\"13991\":{\"init\":true,\"ts\":1754761182,\"cum\":71511790501,\"vo\":\"94144204218\",\"tick\":38022,\"avgT\":37555,\"wsI\":13617},\"13992\":{\"init\":true,\"ts\":1754761276,\"cum\":71515363911,\"vo\":\"94164050949\",\"tick\":38015,\"avgT\":37556,\"wsI\":13617},\"13993\":{\"init\":true,\"ts\":1754761361,\"cum\":71518594081,\"vo\":\"94180920481\",\"tick\":38002,\"avgT\":37557,\"wsI\":13617},\"13994\":{\"init\":true,\"ts\":1754761363,\"cum\":71518670159,\"vo\":\"94181385129\",\"tick\":38039,\"avgT\":37557,\"wsI\":13617},\"13995\":{\"init\":true,\"ts\":1754761364,\"cum\":71518708192,\"vo\":\"94181611705\",\"tick\":38033,\"avgT\":37557,\"wsI\":13617},\"13996\":{\"init\":true,\"ts\":1754761365,\"cum\":71518746209,\"vo\":\"94181823305\",\"tick\":38017,\"avgT\":37557,\"wsI\":13617},\"13997\":{\"init\":true,\"ts\":1754761378,\"cum\":71519240352,\"vo\":\"94184502813\",\"tick\":38011,\"avgT\":37557,\"wsI\":13617},\"13998\":{\"init\":true,\"ts\":1754761457,\"cum\":71522242984,\"vo\":\"94200535438\",\"tick\":38008,\"avgT\":37558,\"wsI\":13618},\"13999\":{\"init\":true,\"ts\":1754761518,\"cum\":71524560618,\"vo\":\"94212131294\",\"tick\":37994,\"avgT\":37558,\"wsI\":13618},\"14000\":{\"init\":true,\"ts\":1754761569,\"cum\":71526497955,\"vo\":\"94221495094\",\"tick\":37987,\"avgT\":37559,\"wsI\":13618},\"14001\":{\"init\":true,\"ts\":1754763168,\"cum\":71587218381,\"vo\":\"94488991693\",\"tick\":37974,\"avgT\":37571,\"wsI\":13626},\"14002\":{\"init\":true,\"ts\":1754763266,\"cum\":71590943557,\"vo\":\"94508050831\",\"tick\":38012,\"avgT\":37571,\"wsI\":13626},\"14003\":{\"init\":true,\"ts\":1754763273,\"cum\":71591209620,\"vo\":\"94509393739\",\"tick\":38009,\"avgT\":37571,\"wsI\":13626},\"14004\":{\"init\":true,\"ts\":1754763335,\"cum\":71593565868,\"vo\":\"94520990799\",\"tick\":38004,\"avgT\":37572,\"wsI\":13629},\"14005\":{\"init\":true,\"ts\":1754763337,\"cum\":71593641864,\"vo\":\"94521353751\",\"tick\":37998,\"avgT\":37572,\"wsI\":13629},\"14006\":{\"init\":true,\"ts\":1754763342,\"cum\":71593831819,\"vo\":\"94522231556\",\"tick\":37991,\"avgT\":37572,\"wsI\":13629},\"14007\":{\"init\":true,\"ts\":1754763390,\"cum\":71595655051,\"vo\":\"94530379268\",\"tick\":37984,\"avgT\":37572,\"wsI\":13629},\"14008\":{\"init\":true,\"ts\":1754763434,\"cum\":71597326215,\"vo\":\"94537721242\",\"tick\":37981,\"avgT\":37573,\"wsI\":13629},\"14009\":{\"init\":true,\"ts\":1754763582,\"cum\":71602946367,\"vo\":\"94561460090\",\"tick\":37974,\"avgT\":37574,\"wsI\":13630},\"14010\":{\"init\":true,\"ts\":1754763750,\"cum\":71609325999,\"vo\":\"94588272546\",\"tick\":37974,\"avgT\":37575,\"wsI\":13632},\"14011\":{\"init\":true,\"ts\":1754765186,\"cum\":71663875331,\"vo\":\"94825570425\",\"tick\":37987,\"avgT\":37586,\"wsI\":13646},\"14012\":{\"init\":true,\"ts\":1754765294,\"cum\":71667977603,\"vo\":\"94842634711\",\"tick\":37984,\"avgT\":37587,\"wsI\":13646},\"14013\":{\"init\":true,\"ts\":1754765440,\"cum\":71673523267,\"vo\":\"94865587315\",\"tick\":37984,\"avgT\":37588,\"wsI\":13646},\"14014\":{\"init\":true,\"ts\":1754765562,\"cum\":71678154997,\"vo\":\"94882880723\",\"tick\":37965,\"avgT\":37589,\"wsI\":13646},\"14015\":{\"init\":true,\"ts\":1754766533,\"cum\":71715023867,\"vo\":\"95020890144\",\"tick\":37970,\"avgT\":37597,\"wsI\":13649},\"14016\":{\"init\":true,\"ts\":1754766580,\"cum\":71716808692,\"vo\":\"95027605692\",\"tick\":37975,\"avgT\":37597,\"wsI\":13649},\"14017\":{\"init\":true,\"ts\":1754766622,\"cum\":71718403810,\"vo\":\"95033734500\",\"tick\":37979,\"avgT\":37597,\"wsI\":13649},\"14018\":{\"init\":true,\"ts\":1754766764,\"cum\":71723797396,\"vo\":\"95054836781\",\"tick\":37983,\"avgT\":37598,\"wsI\":13649},\"14019\":{\"init\":true,\"ts\":1754766780,\"cum\":71724405204,\"vo\":\"95057263756\",\"tick\":37988,\"avgT\":37599,\"wsI\":13649},\"14020\":{\"init\":true,\"ts\":1754766832,\"cum\":71726380580,\"vo\":\"95065132448\",\"tick\":37988,\"avgT\":37599,\"wsI\":13649},\"14021\":{\"init\":true,\"ts\":1754766877,\"cum\":71728090220,\"vo\":\"95072082653\",\"tick\":37992,\"avgT\":37599,\"wsI\":13649},\"14022\":{\"init\":true,\"ts\":1754767279,\"cum\":71743365014,\"vo\":\"95135121637\",\"tick\":37997,\"avgT\":37603,\"wsI\":13649},\"14023\":{\"init\":true,\"ts\":1754767498,\"cum\":71751685262,\"vo\":\"95168175429\",\"tick\":37992,\"avgT\":37604,\"wsI\":13649},\"14024\":{\"init\":true,\"ts\":1754767517,\"cum\":71752407034,\"vo\":\"95170969419\",\"tick\":37988,\"avgT\":37605,\"wsI\":13649},\"14025\":{\"init\":true,\"ts\":1754767584,\"cum\":71754952096,\"vo\":\"95180695206\",\"tick\":37986,\"avgT\":37605,\"wsI\":13649},\"14026\":{\"init\":true,\"ts\":1754767943,\"cum\":71768587275,\"vo\":\"95231044191\",\"tick\":37981,\"avgT\":37608,\"wsI\":13649},\"14027\":{\"init\":true,\"ts\":1754767944,\"cum\":71768625265,\"vo\":\"95231190115\",\"tick\":37990,\"avgT\":37608,\"wsI\":13649},\"14028\":{\"init\":true,\"ts\":1754768548,\"cum\":71791573641,\"vo\":\"95320021094\",\"tick\":37994,\"avgT\":37613,\"wsI\":13651},\"14029\":{\"init\":true,\"ts\":1754768784,\"cum\":71800541169,\"vo\":\"95354820020\",\"tick\":37998,\"avgT\":37615,\"wsI\":13651},\"14030\":{\"init\":true,\"ts\":1754768817,\"cum\":71801795400,\"vo\":\"95359890932\",\"tick\":38007,\"avgT\":37615,\"wsI\":13651},\"14031\":{\"init\":true,\"ts\":1754768821,\"cum\":71801947340,\"vo\":\"95360438532\",\"tick\":37985,\"avgT\":37615,\"wsI\":13651},\"14032\":{\"init\":true,\"ts\":1754768825,\"cum\":71802099316,\"vo\":\"95361013096\",\"tick\":37994,\"avgT\":37615,\"wsI\":13651},\"14033\":{\"init\":true,\"ts\":1754768835,\"cum\":71802479286,\"vo\":\"95362468137\",\"tick\":37997,\"avgT\":37616,\"wsI\":13651},\"14034\":{\"init\":true,\"ts\":1754768884,\"cum\":71804341335,\"vo\":\"95369731162\",\"tick\":38001,\"avgT\":37616,\"wsI\":13651},\"14035\":{\"init\":true,\"ts\":1754768906,\"cum\":71805177555,\"vo\":\"95373146354\",\"tick\":38010,\"avgT\":37616,\"wsI\":13651},\"14036\":{\"init\":true,\"ts\":1754769131,\"cum\":71813731605,\"vo\":\"95409325852\",\"tick\":38018,\"avgT\":37618,\"wsI\":13652},\"14037\":{\"init\":true,\"ts\":1754769134,\"cum\":71813845671,\"vo\":\"95409815500\",\"tick\":38022,\"avgT\":37618,\"wsI\":13652},\"14038\":{\"init\":true,\"ts\":1754769946,\"cum\":71844721159,\"vo\":\"95541039062\",\"tick\":38024,\"avgT\":37626,\"wsI\":13654},\"14039\":{\"init\":true,\"ts\":1754769951,\"cum\":71844911294,\"vo\":\"95541843067\",\"tick\":38027,\"avgT\":37626,\"wsI\":13654},\"14040\":{\"init\":true,\"ts\":1754769988,\"cum\":71846318367,\"vo\":\"95547852200\",\"tick\":38029,\"avgT\":37626,\"wsI\":13654},\"14041\":{\"init\":true,\"ts\":1754770009,\"cum\":71847117060,\"vo\":\"95551330829\",\"tick\":38033,\"avgT\":37626,\"wsI\":13654},\"14042\":{\"init\":true,\"ts\":1754770041,\"cum\":71848334212,\"vo\":\"95556696510\",\"tick\":38036,\"avgT\":37627,\"wsI\":13654},\"14043\":{\"init\":true,\"ts\":1754770177,\"cum\":71853507380,\"vo\":\"95579613504\",\"tick\":38038,\"avgT\":37628,\"wsI\":13654},\"14044\":{\"init\":true,\"ts\":1754770266,\"cum\":71856892940,\"vo\":\"95594683670\",\"tick\":38040,\"avgT\":37629,\"wsI\":13654},\"14045\":{\"init\":true,\"ts\":1754770771,\"cum\":71876105665,\"vo\":\"95681237667\",\"tick\":38045,\"avgT\":37633,\"wsI\":13655},\"14046\":{\"init\":true,\"ts\":1754772390,\"cum\":71937711853,\"vo\":\"95955410763\",\"tick\":38052,\"avgT\":37648,\"wsI\":13665},\"14047\":{\"init\":true,\"ts\":1754772422,\"cum\":71938929517,\"vo\":\"95960620354\",\"tick\":38052,\"avgT\":37649,\"wsI\":13665},\"14048\":{\"init\":true,\"ts\":1754772434,\"cum\":71939386249,\"vo\":\"95962657282\",\"tick\":38061,\"avgT\":37649,\"wsI\":13665},\"14049\":{\"init\":true,\"ts\":1754772519,\"cum\":71942622199,\"vo\":\"95977722767\",\"tick\":38070,\"avgT\":37649,\"wsI\":13667},\"14050\":{\"init\":true,\"ts\":1754772582,\"cum\":71945020735,\"vo\":\"95988968243\",\"tick\":38072,\"avgT\":37650,\"wsI\":13667},\"14051\":{\"init\":true,\"ts\":1754772629,\"cum\":71946810542,\"vo\":\"95997678338\",\"tick\":38081,\"avgT\":37651,\"wsI\":13667},\"14052\":{\"init\":true,\"ts\":1754773047,\"cum\":71962735506,\"vo\":\"96080637879\",\"tick\":38098,\"avgT\":37654,\"wsI\":13667},\"14053\":{\"init\":true,\"ts\":1754773127,\"cum\":71965783666,\"vo\":\"96096657938\",\"tick\":38102,\"avgT\":37655,\"wsI\":13667},\"14054\":{\"init\":true,\"ts\":1754773235,\"cum\":71969900734,\"vo\":\"96120060028\",\"tick\":38121,\"avgT\":37656,\"wsI\":13668},\"14055\":{\"init\":true,\"ts\":1754773339,\"cum\":71973865734,\"vo\":\"96142886762\",\"tick\":38125,\"avgT\":37657,\"wsI\":13668},\"14056\":{\"init\":true,\"ts\":1754773370,\"cum\":71975047764,\"vo\":\"96149807235\",\"tick\":38130,\"avgT\":37658,\"wsI\":13668},\"14057\":{\"init\":true,\"ts\":1754773813,\"cum\":71991944227,\"vo\":\"96252298824\",\"tick\":38141,\"avgT\":37662,\"wsI\":13668},\"14058\":{\"init\":true,\"ts\":1754773894,\"cum\":71995034377,\"vo\":\"96271548499\",\"tick\":38150,\"avgT\":37663,\"wsI\":13668},\"14059\":{\"init\":true,\"ts\":1754773994,\"cum\":71998849777,\"vo\":\"96295607041\",\"tick\":38154,\"avgT\":37664,\"wsI\":13668},\"14060\":{\"init\":true,\"ts\":1754774030,\"cum\":72000223393,\"vo\":\"96304321345\",\"tick\":38156,\"avgT\":37664,\"wsI\":13668},\"14061\":{\"init\":true,\"ts\":1754774266,\"cum\":72009232457,\"vo\":\"96365343047\",\"tick\":38174,\"avgT\":37667,\"wsI\":13668},\"14062\":{\"init\":true,\"ts\":1754774450,\"cum\":72016255369,\"vo\":\"96411434607\",\"tick\":38168,\"avgT\":37668,\"wsI\":13668},\"14063\":{\"init\":true,\"ts\":1754774471,\"cum\":72017056729,\"vo\":\"96416507134\",\"tick\":38160,\"avgT\":37669,\"wsI\":13668},\"14064\":{\"init\":true,\"ts\":1754774476,\"cum\":72017247484,\"vo\":\"96417668754\",\"tick\":38151,\"avgT\":37669,\"wsI\":13668},\"14065\":{\"init\":true,\"ts\":1754774479,\"cum\":72017361928,\"vo\":\"96418357077\",\"tick\":38148,\"avgT\":37669,\"wsI\":13668},\"14066\":{\"init\":true,\"ts\":1754774541,\"cum\":72019726856,\"vo\":\"96432345827\",\"tick\":38144,\"avgT\":37669,\"wsI\":13668},\"14067\":{\"init\":true,\"ts\":1754774727,\"cum\":72026820710,\"vo\":\"96473257697\",\"tick\":38139,\"avgT\":37671,\"wsI\":13668},\"14068\":{\"init\":true,\"ts\":1754774962,\"cum\":72035781260,\"vo\":\"96522443469\",\"tick\":38130,\"avgT\":37674,\"wsI\":13668},\"14069\":{\"init\":true,\"ts\":1754775011,\"cum\":72037649189,\"vo\":\"96532234110\",\"tick\":38121,\"avgT\":37674,\"wsI\":13668},\"14070\":{\"init\":true,\"ts\":1754775080,\"cum\":72040278917,\"vo\":\"96545440709\",\"tick\":38112,\"avgT\":37675,\"wsI\":13668},\"14071\":{\"init\":true,\"ts\":1754775170,\"cum\":72043708277,\"vo\":\"96561965390\",\"tick\":38104,\"avgT\":37676,\"wsI\":13668},\"14072\":{\"init\":true,\"ts\":1754775171,\"cum\":72043746425,\"vo\":\"96562188174\",\"tick\":38148,\"avgT\":37676,\"wsI\":13668},\"14073\":{\"init\":true,\"ts\":1754775172,\"cum\":72043784570,\"vo\":\"96562408135\",\"tick\":38145,\"avgT\":37676,\"wsI\":13668},\"14074\":{\"init\":true,\"ts\":1754775191,\"cum\":72044509268,\"vo\":\"96566534099\",\"tick\":38142,\"avgT\":37676,\"wsI\":13668},\"14075\":{\"init\":true,\"ts\":1754775196,\"cum\":72044699933,\"vo\":\"96567578344\",\"tick\":38133,\"avgT\":37676,\"wsI\":13668},\"14076\":{\"init\":true,\"ts\":1754775203,\"cum\":72044966801,\"vo\":\"96568983272\",\"tick\":38124,\"avgT\":37676,\"wsI\":13668},\"14077\":{\"init\":true,\"ts\":1754775531,\"cum\":72057468193,\"vo\":\"96631476786\",\"tick\":38114,\"avgT\":37679,\"wsI\":13670},\"14078\":{\"init\":true,\"ts\":1754775574,\"cum\":72059106708,\"vo\":\"96639280254\",\"tick\":38105,\"avgT\":37679,\"wsI\":13670},\"14079\":{\"init\":true,\"ts\":1754775607,\"cum\":72060364008,\"vo\":\"96645114904\",\"tick\":38100,\"avgT\":37680,\"wsI\":13670},\"14080\":{\"init\":true,\"ts\":1754775669,\"cum\":72062725650,\"vo\":\"96655588006\",\"tick\":38091,\"avgT\":37680,\"wsI\":13670},\"14081\":{\"init\":true,\"ts\":1754776417,\"cum\":72091201262,\"vo\":\"96766748828\",\"tick\":38069,\"avgT\":37687,\"wsI\":13671},\"14082\":{\"init\":true,\"ts\":1754776426,\"cum\":72091543919,\"vo\":\"96768089792\",\"tick\":38073,\"avgT\":37687,\"wsI\":13671},\"14083\":{\"init\":true,\"ts\":1754776446,\"cum\":72092305439,\"vo\":\"96771116212\",\"tick\":38076,\"avgT\":37687,\"wsI\":13671},\"14084\":{\"init\":true,\"ts\":1754776447,\"cum\":72092343446,\"vo\":\"96771218612\",\"tick\":38007,\"avgT\":37687,\"wsI\":13671},\"14085\":{\"init\":true,\"ts\":1754776517,\"cum\":72095008206,\"vo\":\"96781352854\",\"tick\":38068,\"avgT\":37688,\"wsI\":13672},\"14086\":{\"init\":true,\"ts\":1754776519,\"cum\":72095084352,\"vo\":\"96781649304\",\"tick\":38073,\"avgT\":37688,\"wsI\":13672},\"14087\":{\"init\":true,\"ts\":1754776525,\"cum\":72095312892,\"vo\":\"96782618928\",\"tick\":38090,\"avgT\":37688,\"wsI\":13672},\"14088\":{\"init\":true,\"ts\":1754776608,\"cum\":72098475109,\"vo\":\"96796604875\",\"tick\":38099,\"avgT\":37689,\"wsI\":13672},\"14089\":{\"init\":true,\"ts\":1754776737,\"cum\":72103390654,\"vo\":\"96818875062\",\"tick\":38105,\"avgT\":37690,\"wsI\":13672},\"14090\":{\"init\":true,\"ts\":1754776738,\"cum\":72103428664,\"vo\":\"96818977462\",\"tick\":38010,\"avgT\":37690,\"wsI\":13672},\"14091\":{\"init\":true,\"ts\":1754776757,\"cum\":72104152317,\"vo\":\"96821972033\",\"tick\":38087,\"avgT\":37690,\"wsI\":13672},\"14092\":{\"init\":true,\"ts\":1754777066,\"cum\":72115921818,\"vo\":\"96870795003\",\"tick\":38089,\"avgT\":37693,\"wsI\":13672},\"14093\":{\"init\":true,\"ts\":1754777067,\"cum\":72115959791,\"vo\":\"96870873403\",\"tick\":37973,\"avgT\":37693,\"wsI\":13672},\"14094\":{\"init\":true,\"ts\":1754777084,\"cum\":72116606709,\"vo\":\"96873088860\",\"tick\":38054,\"avgT\":37693,\"wsI\":13672},\"14095\":{\"init\":true,\"ts\":1754777138,\"cum\":72118661895,\"vo\":\"96880302372\",\"tick\":38059,\"avgT\":37694,\"wsI\":13672},\"14096\":{\"init\":true,\"ts\":1754777147,\"cum\":72119004444,\"vo\":\"96881514573\",\"tick\":38061,\"avgT\":37694,\"wsI\":13672},\"14097\":{\"init\":true,\"ts\":1754778159,\"cum\":72157540392,\"vo\":\"97028035592\",\"tick\":38079,\"avgT\":37703,\"wsI\":13676},\"14098\":{\"init\":true,\"ts\":1754778261,\"cum\":72161424756,\"vo\":\"97042647971\",\"tick\":38082,\"avgT\":37704,\"wsI\":13676},\"14099\":{\"init\":true,\"ts\":1754778316,\"cum\":72163519376,\"vo\":\"97050589971\",\"tick\":38084,\"avgT\":37704,\"wsI\":13676},\"14100\":{\"init\":true,\"ts\":1754778397,\"cum\":72166604585,\"vo\":\"97062564653\",\"tick\":38089,\"avgT\":37705,\"wsI\":13676},\"14101\":{\"init\":true,\"ts\":1754778399,\"cum\":72166680769,\"vo\":\"97062864191\",\"tick\":38092,\"avgT\":37705,\"wsI\":13676},\"14102\":{\"init\":true,\"ts\":1754778403,\"cum\":72166833145,\"vo\":\"97063469475\",\"tick\":38094,\"avgT\":37705,\"wsI\":13676},\"14103\":{\"init\":true,\"ts\":1754778407,\"cum\":72166985529,\"vo\":\"97064080999\",\"tick\":38096,\"avgT\":37705,\"wsI\":13676},\"14104\":{\"init\":true,\"ts\":1754778414,\"cum\":72167252229,\"vo\":\"97065173174\",\"tick\":38100,\"avgT\":37705,\"wsI\":13676},\"14105\":{\"init\":true,\"ts\":1754778418,\"cum\":72167404645,\"vo\":\"97065809978\",\"tick\":38104,\"avgT\":37705,\"wsI\":13676},\"14106\":{\"init\":true,\"ts\":1754778796,\"cum\":72181816273,\"vo\":\"97132170864\",\"tick\":38126,\"avgT\":37709,\"wsI\":13676},\"14107\":{\"init\":true,\"ts\":1754778797,\"cum\":72181854398,\"vo\":\"97132343920\",\"tick\":38125,\"avgT\":37709,\"wsI\":13676},\"14108\":{\"init\":true,\"ts\":1754778805,\"cum\":72182159302,\"vo\":\"97133649648\",\"tick\":38113,\"avgT\":37709,\"wsI\":13676},\"14109\":{\"init\":true,\"ts\":1754778871,\"cum\":72184674166,\"vo\":\"97143947298\",\"tick\":38104,\"avgT\":37709,\"wsI\":13676},\"14110\":{\"init\":true,\"ts\":1754778878,\"cum\":72184940880,\"vo\":\"97145028441\",\"tick\":38102,\"avgT\":37709,\"wsI\":13676},\"14111\":{\"init\":true,\"ts\":1754778988,\"cum\":72189131880,\"vo\":\"97161801987\",\"tick\":38100,\"avgT\":37710,\"wsI\":13676},\"14112\":{\"init\":true,\"ts\":1754779045,\"cum\":72191303181,\"vo\":\"97170141065\",\"tick\":38093,\"avgT\":37711,\"wsI\":13676},\"14113\":{\"init\":true,\"ts\":1754779541,\"cum\":72210191853,\"vo\":\"97237676106\",\"tick\":38082,\"avgT\":37715,\"wsI\":13677},\"14114\":{\"init\":true,\"ts\":1754779562,\"cum\":72210991470,\"vo\":\"97240420073\",\"tick\":38077,\"avgT\":37716,\"wsI\":13677},\"14115\":{\"init\":true,\"ts\":1754779688,\"cum\":72215788542,\"vo\":\"97256343639\",\"tick\":38072,\"avgT\":37717,\"wsI\":13677},\"14116\":{\"init\":true,\"ts\":1754779710,\"cum\":72216625906,\"vo\":\"97258962189\",\"tick\":38062,\"avgT\":37717,\"wsI\":13677},\"14117\":{\"init\":true,\"ts\":1754780561,\"cum\":72249009009,\"vo\":\"97353046705\",\"tick\":38053,\"avgT\":37724,\"wsI\":13678},\"14118\":{\"init\":true,\"ts\":1754780660,\"cum\":72252775761,\"vo\":\"97363406962\",\"tick\":38048,\"avgT\":37725,\"wsI\":13678},\"14119\":{\"init\":true,\"ts\":1754780961,\"cum\":72264221587,\"vo\":\"97390496462\",\"tick\":38026,\"avgT\":37727,\"wsI\":13678},\"14120\":{\"init\":true,\"ts\":1754780967,\"cum\":72264449695,\"vo\":\"97391004548\",\"tick\":38018,\"avgT\":37727,\"wsI\":13678},\"14121\":{\"init\":true,\"ts\":1754781182,\"cum\":72272622705,\"vo\":\"97408590187\",\"tick\":38014,\"avgT\":37729,\"wsI\":13679},\"14122\":{\"init\":true,\"ts\":1754781191,\"cum\":72272964795,\"vo\":\"97409300836\",\"tick\":38010,\"avgT\":37729,\"wsI\":13679},\"14123\":{\"init\":true,\"ts\":1754781229,\"cum\":72274409023,\"vo\":\"97412216538\",\"tick\":38006,\"avgT\":37729,\"wsI\":13679},\"14124\":{\"init\":true,\"ts\":1754781244,\"cum\":72274979083,\"vo\":\"97413346518\",\"tick\":38004,\"avgT\":37730,\"wsI\":13679},\"14125\":{\"init\":true,\"ts\":1754781413,\"cum\":72281400914,\"vo\":\"97425529853\",\"tick\":37999,\"avgT\":37731,\"wsI\":13679},\"14126\":{\"init\":true,\"ts\":1754781627,\"cum\":72289531844,\"vo\":\"97440331564\",\"tick\":37995,\"avgT\":37733,\"wsI\":13679},\"14127\":{\"init\":true,\"ts\":1754781717,\"cum\":72292950584,\"vo\":\"97446092374\",\"tick\":37986,\"avgT\":37733,\"wsI\":13679},\"14128\":{\"init\":true,\"ts\":1754781797,\"cum\":72295989544,\"vo\":\"97451233107\",\"tick\":37987,\"avgT\":37734,\"wsI\":13679},\"14129\":{\"init\":true,\"ts\":1754781798,\"cum\":72296027749,\"vo\":\"97451454948\",\"tick\":38205,\"avgT\":37734,\"wsI\":13679},\"14130\":{\"init\":true,\"ts\":1754781799,\"cum\":72296065836,\"vo\":\"97451579557\",\"tick\":38087,\"avgT\":37734,\"wsI\":13679},\"14131\":{\"init\":true,\"ts\":1754781800,\"cum\":72296103874,\"vo\":\"97451671973\",\"tick\":38038,\"avgT\":37734,\"wsI\":13679},\"14132\":{\"init\":true,\"ts\":1754781802,\"cum\":72296179936,\"vo\":\"97451848391\",\"tick\":38031,\"avgT\":37734,\"wsI\":13679},\"14133\":{\"init\":true,\"ts\":1754781804,\"cum\":72296255972,\"vo\":\"97452009703\",\"tick\":38018,\"avgT\":37734,\"wsI\":13679},\"14134\":{\"init\":true,\"ts\":1754781892,\"cum\":72299599620,\"vo\":\"97458027086\",\"tick\":37996,\"avgT\":37735,\"wsI\":13679},\"14135\":{\"init\":true,\"ts\":1754781936,\"cum\":72301271224,\"vo\":\"97460910670\",\"tick\":37991,\"avgT\":37735,\"wsI\":13679},\"14136\":{\"init\":true,\"ts\":1754782266,\"cum\":72313806934,\"vo\":\"97481700608\",\"tick\":37987,\"avgT\":37737,\"wsI\":13679},\"14137\":{\"init\":true,\"ts\":1754782268,\"cum\":72313883094,\"vo\":\"97481935906\",\"tick\":38080,\"avgT\":37737,\"wsI\":13679},\"14138\":{\"init\":true,\"ts\":1754782273,\"cum\":72314073179,\"vo\":\"97482327906\",\"tick\":38017,\"avgT\":37737,\"wsI\":13679},\"14139\":{\"init\":true,\"ts\":1754782280,\"cum\":72314339270,\"vo\":\"97482858932\",\"tick\":38013,\"avgT\":37738,\"wsI\":13679},\"14140\":{\"init\":true,\"ts\":1754782284,\"cum\":72314491318,\"vo\":\"97483159236\",\"tick\":38012,\"avgT\":37738,\"wsI\":13679},\"14141\":{\"init\":true,\"ts\":1754782555,\"cum\":72324792028,\"vo\":\"97503061295\",\"tick\":38010,\"avgT\":37740,\"wsI\":13680},\"14142\":{\"init\":true,\"ts\":1754783407,\"cum\":72357175696,\"vo\":\"97563120608\",\"tick\":38009,\"avgT\":37747,\"wsI\":13684},\"14143\":{\"init\":true,\"ts\":1754783469,\"cum\":72359532130,\"vo\":\"97567311808\",\"tick\":38007,\"avgT\":37747,\"wsI\":13684},\"14144\":{\"init\":true,\"ts\":1754783742,\"cum\":72369907495,\"vo\":\"97585342762\",\"tick\":38005,\"avgT\":37749,\"wsI\":13686},\"14145\":{\"init\":true,\"ts\":1754784109,\"cum\":72383854596,\"vo\":\"97608740823\",\"tick\":38003,\"avgT\":37752,\"wsI\":13686},\"14146\":{\"init\":true,\"ts\":1754784535,\"cum\":72400042596,\"vo\":\"97634520223\",\"tick\":38000,\"avgT\":37756,\"wsI\":13687},\"14147\":{\"init\":true,\"ts\":1754785211,\"cum\":72425730596,\"vo\":\"97673946264\",\"tick\":38000,\"avgT\":37761,\"wsI\":13694},\"14148\":{\"init\":true,\"ts\":1754785792,\"cum\":72447810920,\"vo\":\"97707412167\",\"tick\":38004,\"avgT\":37767,\"wsI\":13697},\"14149\":{\"init\":true,\"ts\":1754785982,\"cum\":72455033200,\"vo\":\"97718770185\",\"tick\":38012,\"avgT\":37768,\"wsI\":13697},\"14150\":{\"init\":true,\"ts\":1754786605,\"cum\":72478715299,\"vo\":\"97755255974\",\"tick\":38013,\"avgT\":37774,\"wsI\":13697},\"14151\":{\"init\":true,\"ts\":1754786632,\"cum\":72479741434,\"vo\":\"97756696721\",\"tick\":38005,\"avgT\":37774,\"wsI\":13697},\"14152\":{\"init\":true,\"ts\":1754787451,\"cum\":72510874081,\"vo\":\"97801928484\",\"tick\":38013,\"avgT\":37782,\"wsI\":13698},\"14153\":{\"init\":true,\"ts\":1754787463,\"cum\":72511330297,\"vo\":\"97802596836\",\"tick\":38018,\"avgT\":37782,\"wsI\":13698},\"14154\":{\"init\":true,\"ts\":1754787498,\"cum\":72512661137,\"vo\":\"97804646576\",\"tick\":38024,\"avgT\":37782,\"wsI\":13698},\"14155\":{\"init\":true,\"ts\":1754788160,\"cum\":72537848251,\"vo\":\"97850089318\",\"tick\":38047,\"avgT\":37788,\"wsI\":13700},\"14156\":{\"init\":true,\"ts\":1754788645,\"cum\":72556294741,\"vo\":\"97878845852\",\"tick\":38034,\"avgT\":37793,\"wsI\":13700},\"14157\":{\"init\":true,\"ts\":1754789190,\"cum\":72577018911,\"vo\":\"97907801821\",\"tick\":38026,\"avgT\":37798,\"wsI\":13700},\"14158\":{\"init\":true,\"ts\":1754789299,\"cum\":72581163091,\"vo\":\"97913149393\",\"tick\":38020,\"avgT\":37799,\"wsI\":13701},\"14159\":{\"init\":true,\"ts\":1754789582,\"cum\":72591924166,\"vo\":\"97927412202\",\"tick\":38025,\"avgT\":37802,\"wsI\":13701},\"14160\":{\"init\":true,\"ts\":1754789720,\"cum\":72597172168,\"vo\":\"97934491697\",\"tick\":38029,\"avgT\":37803,\"wsI\":13701},\"14161\":{\"init\":true,\"ts\":1754789998,\"cum\":72607745342,\"vo\":\"97949006225\",\"tick\":38033,\"avgT\":37806,\"wsI\":13704},\"14162\":{\"init\":true,\"ts\":1754790084,\"cum\":72611016954,\"vo\":\"97953775578\",\"tick\":38042,\"avgT\":37807,\"wsI\":13706},\"14163\":{\"init\":true,\"ts\":1754791653,\"cum\":72670704852,\"vo\":\"98035360909\",\"tick\":38042,\"avgT\":37821,\"wsI\":13716},\"14164\":{\"init\":true,\"ts\":1754792172,\"cum\":72690451245,\"vo\":\"98061286090\",\"tick\":38047,\"avgT\":37826,\"wsI\":13717},\"14165\":{\"init\":true,\"ts\":1754792525,\"cum\":72703883601,\"vo\":\"98079076969\",\"tick\":38052,\"avgT\":37829,\"wsI\":13717},\"14166\":{\"init\":true,\"ts\":1754792569,\"cum\":72705558065,\"vo\":\"98081344245\",\"tick\":38056,\"avgT\":37829,\"wsI\":13717},\"14167\":{\"init\":true,\"ts\":1754795298,\"cum\":72809426534,\"vo\":\"98212964795\",\"tick\":38061,\"avgT\":37854,\"wsI\":13724},\"14168\":{\"init\":true,\"ts\":1754795311,\"cum\":72809921392,\"vo\":\"98213549067\",\"tick\":38066,\"avgT\":37854,\"wsI\":13724},\"14169\":{\"init\":true,\"ts\":1754795321,\"cum\":72810302142,\"vo\":\"98214037477\",\"tick\":38075,\"avgT\":37854,\"wsI\":13724},\"14170\":{\"init\":true,\"ts\":1754795655,\"cum\":72823019860,\"vo\":\"98230423854\",\"tick\":38077,\"avgT\":37857,\"wsI\":13724},\"14171\":{\"init\":true,\"ts\":1754795764,\"cum\":72827170798,\"vo\":\"98235917265\",\"tick\":38082,\"avgT\":37858,\"wsI\":13724},\"14172\":{\"init\":true,\"ts\":1754795802,\"cum\":72828617990,\"vo\":\"98237858153\",\"tick\":38084,\"avgT\":37858,\"wsI\":13724},\"14173\":{\"init\":true,\"ts\":1754795892,\"cum\":72832045640,\"vo\":\"98242475136\",\"tick\":38085,\"avgT\":37859,\"wsI\":13724},\"14174\":{\"init\":true,\"ts\":1754795944,\"cum\":72834026268,\"vo\":\"98245225936\",\"tick\":38089,\"avgT\":37859,\"wsI\":13724},\"14175\":{\"init\":true,\"ts\":1754795952,\"cum\":72834331052,\"vo\":\"98245680756\",\"tick\":38098,\"avgT\":37860,\"wsI\":13724},\"14176\":{\"init\":true,\"ts\":1754795980,\"cum\":72835397936,\"vo\":\"98247334128\",\"tick\":38103,\"avgT\":37860,\"wsI\":13724},\"14177\":{\"init\":true,\"ts\":1754796150,\"cum\":72841878336,\"vo\":\"98258781725\",\"tick\":38120,\"avgT\":37861,\"wsI\":13724},\"14178\":{\"init\":true,\"ts\":1754796184,\"cum\":72843174722,\"vo\":\"98261214372\",\"tick\":38129,\"avgT\":37862,\"wsI\":13724},\"14179\":{\"init\":true,\"ts\":1754796247,\"cum\":72845577416,\"vo\":\"98266013460\",\"tick\":38138,\"avgT\":37862,\"wsI\":13724},\"14180\":{\"init\":true,\"ts\":1754796319,\"cum\":72848324000,\"vo\":\"98271840879\",\"tick\":38147,\"avgT\":37863,\"wsI\":13724},\"14181\":{\"init\":true,\"ts\":1754796402,\"cum\":72851490699,\"vo\":\"98278796847\",\"tick\":38153,\"avgT\":37864,\"wsI\":13724},\"14182\":{\"init\":true,\"ts\":1754796434,\"cum\":72852712331,\"vo\":\"98281911855\",\"tick\":38176,\"avgT\":37864,\"wsI\":13724},\"14183\":{\"init\":true,\"ts\":1754796440,\"cum\":72852941297,\"vo\":\"98282441109\",\"tick\":38161,\"avgT\":37864,\"wsI\":13724},\"14184\":{\"init\":true,\"ts\":1754796447,\"cum\":72853208403,\"vo\":\"98283046161\",\"tick\":38158,\"avgT\":37864,\"wsI\":13724},\"14185\":{\"init\":true,\"ts\":1754796459,\"cum\":72853666275,\"vo\":\"98284069329\",\"tick\":38156,\"avgT\":37864,\"wsI\":13724},\"14186\":{\"init\":true,\"ts\":1754796514,\"cum\":72855764635,\"vo\":\"98288615139\",\"tick\":38152,\"avgT\":37865,\"wsI\":13724},\"14187\":{\"init\":true,\"ts\":1754796531,\"cum\":72856413134,\"vo\":\"98289967047\",\"tick\":38147,\"avgT\":37865,\"wsI\":13724},\"14188\":{\"init\":true,\"ts\":1754797766,\"cum\":72903524679,\"vo\":\"98384055915\",\"tick\":38147,\"avgT\":37877,\"wsI\":13725},\"14189\":{\"init\":true,\"ts\":1754798076,\"cum\":72915350869,\"vo\":\"98406738113\",\"tick\":38149,\"avgT\":37880,\"wsI\":13725},\"14190\":{\"init\":true,\"ts\":1754798100,\"cum\":72916266661,\"vo\":\"98408592929\",\"tick\":38158,\"avgT\":37880,\"wsI\":13725},\"14191\":{\"init\":true,\"ts\":1754798837,\"cum\":72944386159,\"vo\":\"98462321999\",\"tick\":38154,\"avgT\":37888,\"wsI\":13727},\"14192\":{\"init\":true,\"ts\":1754799771,\"cum\":72980000513,\"vo\":\"98515454178\",\"tick\":38131,\"avgT\":37897,\"wsI\":13728},\"14193\":{\"init\":true,\"ts\":1754799870,\"cum\":72983775779,\"vo\":\"98520991242\",\"tick\":38134,\"avgT\":37898,\"wsI\":13728},\"14194\":{\"init\":true,\"ts\":1754799924,\"cum\":72985835123,\"vo\":\"98524050018\",\"tick\":38136,\"avgT\":37898,\"wsI\":13728},\"14195\":{\"init\":true,\"ts\":1754799935,\"cum\":72986254663,\"vo\":\"98524694222\",\"tick\":38140,\"avgT\":37898,\"wsI\":13728},\"14196\":{\"init\":true,\"ts\":1754799978,\"cum\":72987894812,\"vo\":\"98527264531\",\"tick\":38143,\"avgT\":37899,\"wsI\":13728},\"14197\":{\"init\":true,\"ts\":1754800164,\"cum\":72994989782,\"vo\":\"98538428753\",\"tick\":38145,\"avgT\":37901,\"wsI\":13728},\"14198\":{\"init\":true,\"ts\":1754800206,\"cum\":72996591956,\"vo\":\"98540970425\",\"tick\":38147,\"avgT\":37901,\"wsI\":13728},\"14199\":{\"init\":true,\"ts\":1754800445,\"cum\":73005710284,\"vo\":\"98555847665\",\"tick\":38152,\"avgT\":37904,\"wsI\":13730},\"14200\":{\"init\":true,\"ts\":1754800488,\"cum\":73007350820,\"vo\":\"98558492337\",\"tick\":38152,\"avgT\":37904,\"wsI\":13730},\"14201\":{\"init\":true,\"ts\":1754801365,\"cum\":73040811878,\"vo\":\"98611353056\",\"tick\":38154,\"avgT\":37913,\"wsI\":13733},\"14202\":{\"init\":true,\"ts\":1754801937,\"cum\":73062635966,\"vo\":\"98643889702\",\"tick\":38154,\"avgT\":37918,\"wsI\":13733},\"14203\":{\"init\":true,\"ts\":1754802010,\"cum\":73065421354,\"vo\":\"98648007126\",\"tick\":38156,\"avgT\":37919,\"wsI\":13733},\"14204\":{\"init\":true,\"ts\":1754803069,\"cum\":73105828558,\"vo\":\"98705013247\",\"tick\":38156,\"avgT\":37929,\"wsI\":13748},\"14205\":{\"init\":true,\"ts\":1754803107,\"cum\":73107278296,\"vo\":\"98706886039\",\"tick\":38151,\"avgT\":37929,\"wsI\":13749},\"14206\":{\"init\":true,\"ts\":1754803134,\"cum\":73108308103,\"vo\":\"98708099527\",\"tick\":38141,\"avgT\":37929,\"wsI\":13749},\"14207\":{\"init\":true,\"ts\":1754803207,\"cum\":73111092104,\"vo\":\"98711242431\",\"tick\":38137,\"avgT\":37930,\"wsI\":13751},\"14208\":{\"init\":true,\"ts\":1754803225,\"cum\":73111778390,\"vo\":\"98711940993\",\"tick\":38127,\"avgT\":37930,\"wsI\":13751},\"14209\":{\"init\":true,\"ts\":1754803287,\"cum\":73114141954,\"vo\":\"98714214486\",\"tick\":38122,\"avgT\":37931,\"wsI\":13752},\"14210\":{\"init\":true,\"ts\":1754803311,\"cum\":73115056882,\"vo\":\"98715090030\",\"tick\":38122,\"avgT\":37931,\"wsI\":13752},\"14211\":{\"init\":true,\"ts\":1754803338,\"cum\":73116086041,\"vo\":\"98716024122\",\"tick\":38117,\"avgT\":37931,\"wsI\":13753},\"14212\":{\"init\":true,\"ts\":1754803390,\"cum\":73118067605,\"vo\":\"98717634874\",\"tick\":38107,\"avgT\":37931,\"wsI\":13754},\"14213\":{\"init\":true,\"ts\":1754803511,\"cum\":73122677463,\"vo\":\"98720989109\",\"tick\":38098,\"avgT\":37932,\"wsI\":13757},\"14214\":{\"init\":true,\"ts\":1754803523,\"cum\":73123134579,\"vo\":\"98721300161\",\"tick\":38093,\"avgT\":37932,\"wsI\":13757},\"14215\":{\"init\":true,\"ts\":1754803754,\"cum\":73131931983,\"vo\":\"98726566967\",\"tick\":38084,\"avgT\":37934,\"wsI\":13759},\"14216\":{\"init\":true,\"ts\":1754803880,\"cum\":73136729433,\"vo\":\"98729054108\",\"tick\":38075,\"avgT\":37935,\"wsI\":13761},\"14217\":{\"init\":true,\"ts\":1754803948,\"cum\":73139318261,\"vo\":\"98730311836\",\"tick\":38071,\"avgT\":37935,\"wsI\":13762},\"14218\":{\"init\":true,\"ts\":1754804295,\"cum\":73152514671,\"vo\":\"98733345376\",\"tick\":38030,\"avgT\":37938,\"wsI\":13766},\"14219\":{\"init\":true,\"ts\":1754804458,\"cum\":73158713887,\"vo\":\"98734770282\",\"tick\":38032,\"avgT\":37939,\"wsI\":13768},\"14220\":{\"init\":true,\"ts\":1754804666,\"cum\":73166625167,\"vo\":\"98736667215\",\"tick\":38035,\"avgT\":37940,\"wsI\":13768},\"14221\":{\"init\":true,\"ts\":1754804669,\"cum\":73166739284,\"vo\":\"98736696618\",\"tick\":38039,\"avgT\":37940,\"wsI\":13768},\"14222\":{\"init\":true,\"ts\":1754804688,\"cum\":73167462063,\"vo\":\"98736890437\",\"tick\":38041,\"avgT\":37940,\"wsI\":13769},\"14223\":{\"init\":true,\"ts\":1754804722,\"cum\":73168755593,\"vo\":\"98737265287\",\"tick\":38045,\"avgT\":37940,\"wsI\":13769},\"14224\":{\"init\":true,\"ts\":1754806396,\"cum\":73232449619,\"vo\":\"98755043605\",\"tick\":38049,\"avgT\":37952,\"wsI\":13774},\"14225\":{\"init\":true,\"ts\":1754806429,\"cum\":73233705170,\"vo\":\"98755341430\",\"tick\":38047,\"avgT\":37952,\"wsI\":13774},\"14226\":{\"init\":true,\"ts\":1754806582,\"cum\":73239523913,\"vo\":\"98756284188\",\"tick\":38031,\"avgT\":37953,\"wsI\":13775},\"14227\":{\"init\":true,\"ts\":1754806608,\"cum\":73240512511,\"vo\":\"98756411588\",\"tick\":38023,\"avgT\":37953,\"wsI\":13776},\"14228\":{\"init\":true,\"ts\":1754806635,\"cum\":73241539024,\"vo\":\"98756529200\",\"tick\":38019,\"avgT\":37953,\"wsI\":13776},\"14229\":{\"init\":true,\"ts\":1754806652,\"cum\":73242185279,\"vo\":\"98756594548\",\"tick\":38015,\"avgT\":37953,\"wsI\":13776},\"14230\":{\"init\":true,\"ts\":1754806758,\"cum\":73246212643,\"vo\":\"98756768382\",\"tick\":37994,\"avgT\":37954,\"wsI\":13776},\"14231\":{\"init\":true,\"ts\":1754806797,\"cum\":73247694253,\"vo\":\"98756818926\",\"tick\":37990,\"avgT\":37954,\"wsI\":13777},\"14232\":{\"init\":true,\"ts\":1754806957,\"cum\":73253772013,\"vo\":\"98756972677\",\"tick\":37986,\"avgT\":37956,\"wsI\":13777},\"14233\":{\"init\":true,\"ts\":1754807210,\"cum\":73263382471,\"vo\":\"98757192841\",\"tick\":37986,\"avgT\":37957,\"wsI\":13777},\"14234\":{\"init\":true,\"ts\":1754807482,\"cum\":73273713847,\"vo\":\"98757362881\",\"tick\":37983,\"avgT\":37959,\"wsI\":13777},\"14235\":{\"init\":true,\"ts\":1754807484,\"cum\":73273789679,\"vo\":\"98757366579\",\"tick\":37916,\"avgT\":37959,\"wsI\":13777},\"14236\":{\"init\":true,\"ts\":1754807485,\"cum\":73273827597,\"vo\":\"98757368260\",\"tick\":37918,\"avgT\":37959,\"wsI\":13777},\"14237\":{\"init\":true,\"ts\":1754807488,\"cum\":73273941372,\"vo\":\"98757371728\",\"tick\":37925,\"avgT\":37959,\"wsI\":13777},\"14238\":{\"init\":true,\"ts\":1754807489,\"cum\":73273979304,\"vo\":\"98757372457\",\"tick\":37932,\"avgT\":37959,\"wsI\":13777},\"14239\":{\"init\":true,\"ts\":1754807494,\"cum\":73274168969,\"vo\":\"98757375837\",\"tick\":37933,\"avgT\":37959,\"wsI\":13777},\"14240\":{\"init\":true,\"ts\":1754807510,\"cum\":73274776025,\"vo\":\"98757381021\",\"tick\":37941,\"avgT\":37959,\"wsI\":13777},\"14241\":{\"init\":true,\"ts\":1754807569,\"cum\":73277014544,\"vo\":\"98757401237\",\"tick\":37941,\"avgT\":37960,\"wsI\":13777},\"14242\":{\"init\":true,\"ts\":1754807599,\"cum\":73278152924,\"vo\":\"98757407117\",\"tick\":37946,\"avgT\":37960,\"wsI\":13777},\"14243\":{\"init\":true,\"ts\":1754807608,\"cum\":73278494528,\"vo\":\"98757407261\",\"tick\":37956,\"avgT\":37960,\"wsI\":13777},\"14244\":{\"init\":true,\"ts\":1754808039,\"cum\":73294854426,\"vo\":\"98757411289\",\"tick\":37958,\"avgT\":37962,\"wsI\":13777},\"14245\":{\"init\":true,\"ts\":1754808283,\"cum\":73304117886,\"vo\":\"98757412342\",\"tick\":37965,\"avgT\":37964,\"wsI\":13777},\"14246\":{\"init\":true,\"ts\":1754808290,\"cum\":73304383655,\"vo\":\"98757412405\",\"tick\":37967,\"avgT\":37964,\"wsI\":13777},\"14247\":{\"init\":true,\"ts\":1754808327,\"cum\":73305788508,\"vo\":\"98757413330\",\"tick\":37969,\"avgT\":37964,\"wsI\":13777},\"14248\":{\"init\":true,\"ts\":1754808342,\"cum\":73306358073,\"vo\":\"98757414065\",\"tick\":37971,\"avgT\":37964,\"wsI\":13777},\"14249\":{\"init\":true,\"ts\":1754808346,\"cum\":73306509965,\"vo\":\"98757414389\",\"tick\":37973,\"avgT\":37964,\"wsI\":13777},\"14250\":{\"init\":true,\"ts\":1754808369,\"cum\":73307383436,\"vo\":\"98757417972\",\"tick\":37977,\"avgT\":37965,\"wsI\":13777},\"14251\":{\"init\":true,\"ts\":1754808389,\"cum\":73308143016,\"vo\":\"98757421892\",\"tick\":37979,\"avgT\":37965,\"wsI\":13777},\"14252\":{\"init\":true,\"ts\":1754808504,\"cum\":73312510831,\"vo\":\"98757451332\",\"tick\":37981,\"avgT\":37965,\"wsI\":13777},\"14253\":{\"init\":true,\"ts\":1754808507,\"cum\":73312624801,\"vo\":\"98757453108\",\"tick\":37990,\"avgT\":37966,\"wsI\":13777},\"14254\":{\"init\":true,\"ts\":1754808514,\"cum\":73312890745,\"vo\":\"98757457840\",\"tick\":37992,\"avgT\":37966,\"wsI\":13777},\"14255\":{\"init\":true,\"ts\":1754808735,\"cum\":73321287861,\"vo\":\"98757650154\",\"tick\":37996,\"avgT\":37967,\"wsI\":13778},\"14256\":{\"init\":true,\"ts\":1754808798,\"cum\":73323681609,\"vo\":\"98757703137\",\"tick\":37996,\"avgT\":37967,\"wsI\":13778},\"14257\":{\"init\":true,\"ts\":1754809073,\"cum\":73334131609,\"vo\":\"98757984764\",\"tick\":38000,\"avgT\":37969,\"wsI\":13782},\"14258\":{\"init\":true,\"ts\":1754809305,\"cum\":73342948537,\"vo\":\"98758252965\",\"tick\":38004,\"avgT\":37971,\"wsI\":13787},\"14259\":{\"init\":true,\"ts\":1754809449,\"cum\":73348422121,\"vo\":\"98758477613\",\"tick\":38011,\"avgT\":37972,\"wsI\":13788},\"14260\":{\"init\":true,\"ts\":1754809518,\"cum\":73351045432,\"vo\":\"98758630034\",\"tick\":38019,\"avgT\":37972,\"wsI\":13788},\"14261\":{\"init\":true,\"ts\":1754809632,\"cum\":73355379598,\"vo\":\"98758876493\",\"tick\":38019,\"avgT\":37973,\"wsI\":13789},\"14262\":{\"init\":true,\"ts\":1754809846,\"cum\":73363517376,\"vo\":\"98759488978\",\"tick\":38027,\"avgT\":37974,\"wsI\":13790},\"14263\":{\"init\":true,\"ts\":1754809900,\"cum\":73365570942,\"vo\":\"98759652328\",\"tick\":38029,\"avgT\":37974,\"wsI\":13791},\"14264\":{\"init\":true,\"ts\":1754811453,\"cum\":73424636191,\"vo\":\"98764275118\",\"tick\":38033,\"avgT\":37983,\"wsI\":13808},\"14265\":{\"init\":true,\"ts\":1754811484,\"cum\":73425815090,\"vo\":\"98764340714\",\"tick\":38029,\"avgT\":37983,\"wsI\":13808},\"14266\":{\"init\":true,\"ts\":1754811523,\"cum\":73427297909,\"vo\":\"98764397030\",\"tick\":38021,\"avgT\":37983,\"wsI\":13808},\"14267\":{\"init\":true,\"ts\":1754811615,\"cum\":73430795013,\"vo\":\"98764474402\",\"tick\":38012,\"avgT\":37983,\"wsI\":13808},\"14268\":{\"init\":true,\"ts\":1754811625,\"cum\":73431175093,\"vo\":\"98764480652\",\"tick\":38008,\"avgT\":37983,\"wsI\":13808},\"14269\":{\"init\":true,\"ts\":1754811639,\"cum\":73431707177,\"vo\":\"98764488058\",\"tick\":38006,\"avgT\":37983,\"wsI\":13808},\"14270\":{\"init\":true,\"ts\":1754811669,\"cum\":73432847297,\"vo\":\"98764500647\",\"tick\":38004,\"avgT\":37984,\"wsI\":13808},\"14271\":{\"init\":true,\"ts\":1754811785,\"cum\":73437255297,\"vo\":\"98764530343\",\"tick\":38000,\"avgT\":37984,\"wsI\":13808},\"14272\":{\"init\":true,\"ts\":1754811803,\"cum\":73437939297,\"vo\":\"98764534951\",\"tick\":38000,\"avgT\":37984,\"wsI\":13809},\"14273\":{\"init\":true,\"ts\":1754811807,\"cum\":73438091281,\"vo\":\"98764535527\",\"tick\":37996,\"avgT\":37984,\"wsI\":13809},\"14274\":{\"init\":true,\"ts\":1754811873,\"cum\":73440598753,\"vo\":\"98764539751\",\"tick\":37992,\"avgT\":37984,\"wsI\":13809},\"14275\":{\"init\":true,\"ts\":1754811908,\"cum\":73441928298,\"vo\":\"98764539970\",\"tick\":37987,\"avgT\":37985,\"wsI\":13811},\"14276\":{\"init\":true,\"ts\":1754812050,\"cum\":73447321316,\"vo\":\"98764545082\",\"tick\":37979,\"avgT\":37985,\"wsI\":13811},\"14277\":{\"init\":true,\"ts\":1754812422,\"cum\":73461446900,\"vo\":\"98764618146\",\"tick\":37972,\"avgT\":37987,\"wsI\":13812},\"14278\":{\"init\":true,\"ts\":1754812619,\"cum\":73468925808,\"vo\":\"98764722359\",\"tick\":37964,\"avgT\":37987,\"wsI\":13813},\"14279\":{\"init\":true,\"ts\":1754812789,\"cum\":73475379008,\"vo\":\"98764850963\",\"tick\":37960,\"avgT\":37988,\"wsI\":13814},\"14280\":{\"init\":true,\"ts\":1754812853,\"cum\":73477808192,\"vo\":\"98764916499\",\"tick\":37956,\"avgT\":37988,\"wsI\":13816},\"14281\":{\"init\":true,\"ts\":1754812881,\"cum\":73478870848,\"vo\":\"98764952787\",\"tick\":37952,\"avgT\":37988,\"wsI\":13817},\"14282\":{\"init\":true,\"ts\":1754812909,\"cum\":73479933392,\"vo\":\"98764997587\",\"tick\":37948,\"avgT\":37988,\"wsI\":13817},\"14283\":{\"init\":true,\"ts\":1754812943,\"cum\":73481223488,\"vo\":\"98765064962\",\"tick\":37944,\"avgT\":37989,\"wsI\":13817},\"14284\":{\"init\":true,\"ts\":1754813003,\"cum\":73483499888,\"vo\":\"98765209022\",\"tick\":37940,\"avgT\":37989,\"wsI\":13819},\"14285\":{\"init\":true,\"ts\":1754813041,\"cum\":73484941304,\"vo\":\"98765332484\",\"tick\":37932,\"avgT\":37989,\"wsI\":13819},\"14286\":{\"init\":true,\"ts\":1754813107,\"cum\":73487444288,\"vo\":\"98765611334\",\"tick\":37924,\"avgT\":37989,\"wsI\":13821},\"14287\":{\"init\":true,\"ts\":1754813197,\"cum\":73490855558,\"vo\":\"98766276974\",\"tick\":37903,\"avgT\":37989,\"wsI\":13822},\"14288\":{\"init\":true,\"ts\":1754813283,\"cum\":73494114700,\"vo\":\"98767004878\",\"tick\":37897,\"avgT\":37989,\"wsI\":13823},\"14289\":{\"init\":true,\"ts\":1754813809,\"cum\":73514044314,\"vo\":\"98772317753\",\"tick\":37889,\"avgT\":37990,\"wsI\":13825},\"14290\":{\"init\":true,\"ts\":1754813820,\"cum\":73514461115,\"vo\":\"98772425564\",\"tick\":37891,\"avgT\":37990,\"wsI\":13825},\"14291\":{\"init\":true,\"ts\":1754813891,\"cum\":73517151518,\"vo\":\"98773093603\",\"tick\":37893,\"avgT\":37990,\"wsI\":13825},\"14292\":{\"init\":true,\"ts\":1754813922,\"cum\":73518326325,\"vo\":\"98773361722\",\"tick\":37897,\"avgT\":37990,\"wsI\":13826},\"14293\":{\"init\":true,\"ts\":1754814003,\"cum\":73521396306,\"vo\":\"98774010648\",\"tick\":37901,\"avgT\":37991,\"wsI\":13828},\"14294\":{\"init\":true,\"ts\":1754814008,\"cum\":73521585841,\"vo\":\"98774045928\",\"tick\":37907,\"avgT\":37991,\"wsI\":13828},\"14295\":{\"init\":true,\"ts\":1754814715,\"cum\":73548388918,\"vo\":\"98778627604\",\"tick\":37911,\"avgT\":37992,\"wsI\":13830},\"14296\":{\"init\":true,\"ts\":1754814837,\"cum\":73553014548,\"vo\":\"98779350942\",\"tick\":37915,\"avgT\":37992,\"wsI\":13830},\"14297\":{\"init\":true,\"ts\":1754814857,\"cum\":73553772928,\"vo\":\"98779457522\",\"tick\":37919,\"avgT\":37992,\"wsI\":13830},\"14298\":{\"init\":true,\"ts\":1754814894,\"cum\":73555176227,\"vo\":\"98779613847\",\"tick\":37927,\"avgT\":37992,\"wsI\":13831},\"14299\":{\"init\":true,\"ts\":1754815541,\"cum\":73579720172,\"vo\":\"98781790686\",\"tick\":37935,\"avgT\":37994,\"wsI\":13838},\"14300\":{\"init\":true,\"ts\":1754815615,\"cum\":73582527954,\"vo\":\"98781983160\",\"tick\":37943,\"avgT\":37994,\"wsI\":13840},\"14301\":{\"init\":true,\"ts\":1754815767,\"cum\":73588295746,\"vo\":\"98782333368\",\"tick\":37946,\"avgT\":37994,\"wsI\":13842},\"14302\":{\"init\":true,\"ts\":1754815793,\"cum\":73589282550,\"vo\":\"98782374968\",\"tick\":37954,\"avgT\":37994,\"wsI\":13842},\"14303\":{\"init\":true,\"ts\":1754816074,\"cum\":73599948748,\"vo\":\"98782739144\",\"tick\":37958,\"avgT\":37994,\"wsI\":13844},\"14304\":{\"init\":true,\"ts\":1754816112,\"cum\":73601391304,\"vo\":\"98782778056\",\"tick\":37962,\"avgT\":37994,\"wsI\":13844},\"14305\":{\"init\":true,\"ts\":1754816122,\"cum\":73601770974,\"vo\":\"98782785346\",\"tick\":37967,\"avgT\":37994,\"wsI\":13844},\"14306\":{\"init\":true,\"ts\":1754816208,\"cum\":73605036480,\"vo\":\"98782832870\",\"tick\":37971,\"avgT\":37995,\"wsI\":13845},\"14307\":{\"init\":true,\"ts\":1754816302,\"cum\":73608606130,\"vo\":\"98782870470\",\"tick\":37975,\"avgT\":37995,\"wsI\":13847},\"14308\":{\"init\":true,\"ts\":1754816303,\"cum\":73608644101,\"vo\":\"98782871046\",\"tick\":37971,\"avgT\":37995,\"wsI\":13847},\"14309\":{\"init\":true,\"ts\":1754817779,\"cum\":73664653873,\"vo\":\"98786488870\",\"tick\":37947,\"avgT\":37998,\"wsI\":13864},\"14310\":{\"init\":true,\"ts\":1754818132,\"cum\":73678050576,\"vo\":\"98787285403\",\"tick\":37951,\"avgT\":37999,\"wsI\":13868},\"14311\":{\"init\":true,\"ts\":1754818175,\"cum\":73679682469,\"vo\":\"98787384475\",\"tick\":37951,\"avgT\":37999,\"wsI\":13869},\"14312\":{\"init\":true,\"ts\":1754818222,\"cum\":73681466166,\"vo\":\"98787492763\",\"tick\":37951,\"avgT\":37999,\"wsI\":13869},\"14313\":{\"init\":true,\"ts\":1754818670,\"cum\":73698468662,\"vo\":\"98788503647\",\"tick\":37952,\"avgT\":38000,\"wsI\":13871},\"14314\":{\"init\":true,\"ts\":1754819068,\"cum\":73713573558,\"vo\":\"98789439924\",\"tick\":37952,\"avgT\":38001,\"wsI\":13872},\"14315\":{\"init\":true,\"ts\":1754819313,\"cum\":73722873758,\"vo\":\"98789851769\",\"tick\":37960,\"avgT\":38001,\"wsI\":13874},\"14316\":{\"init\":true,\"ts\":1754819326,\"cum\":73723367290,\"vo\":\"98789869566\",\"tick\":37964,\"avgT\":38001,\"wsI\":13874},\"14317\":{\"init\":true,\"ts\":1754819329,\"cum\":73723481155,\"vo\":\"98789875914\",\"tick\":37955,\"avgT\":38001,\"wsI\":13874},\"14318\":{\"init\":true,\"ts\":1754819361,\"cum\":73724695971,\"vo\":\"98789922122\",\"tick\":37963,\"avgT\":38001,\"wsI\":13874},\"14319\":{\"init\":true,\"ts\":1754819550,\"cum\":73731871923,\"vo\":\"98790134276\",\"tick\":37968,\"avgT\":38002,\"wsI\":13876},\"14320\":{\"init\":true,\"ts\":1754819717,\"cum\":73738212913,\"vo\":\"98790305284\",\"tick\":37970,\"avgT\":38002,\"wsI\":13877},\"14321\":{\"init\":true,\"ts\":1754821850,\"cum\":73819217854,\"vo\":\"98791804862\",\"tick\":37977,\"avgT\":38005,\"wsI\":13888},\"14322\":{\"init\":true,\"ts\":1754821870,\"cum\":73819977234,\"vo\":\"98791830782\",\"tick\":37969,\"avgT\":38005,\"wsI\":13888},\"14323\":{\"init\":true,\"ts\":1754822026,\"cum\":73825897278,\"vo\":\"98792319998\",\"tick\":37949,\"avgT\":38005,\"wsI\":13888},\"14324\":{\"init\":true,\"ts\":1754822077,\"cum\":73827832269,\"vo\":\"98792528894\",\"tick\":37941,\"avgT\":38005,\"wsI\":13889},\"14325\":{\"init\":true,\"ts\":1754822148,\"cum\":73830524589,\"vo\":\"98793041869\",\"tick\":37920,\"avgT\":38005,\"wsI\":13889},\"14326\":{\"init\":true,\"ts\":1754823089,\"cum\":73866203545,\"vo\":\"98800495530\",\"tick\":37916,\"avgT\":38005,\"wsI\":13891},\"14327\":{\"init\":true,\"ts\":1754824128,\"cum\":73905598269,\"vo\":\"98808725449\",\"tick\":37916,\"avgT\":38005,\"wsI\":13893},\"14328\":{\"init\":true,\"ts\":1754824427,\"cum\":73916933957,\"vo\":\"98811311500\",\"tick\":37912,\"avgT\":38005,\"wsI\":13894},\"14329\":{\"init\":true,\"ts\":1754824912,\"cum\":73935318367,\"vo\":\"98816064985\",\"tick\":37906,\"avgT\":38005,\"wsI\":13896},\"14330\":{\"init\":true,\"ts\":1754824985,\"cum\":73938085797,\"vo\":\"98816730864\",\"tick\":37910,\"avgT\":38006,\"wsI\":13896},\"14331\":{\"init\":true,\"ts\":1754824990,\"cum\":73938275362,\"vo\":\"98816774109\",\"tick\":37913,\"avgT\":38006,\"wsI\":13896},\"14332\":{\"init\":true,\"ts\":1754825009,\"cum\":73938995861,\"vo\":\"98816911384\",\"tick\":37921,\"avgT\":38006,\"wsI\":13896},\"14333\":{\"init\":true,\"ts\":1754825022,\"cum\":73939488873,\"vo\":\"98816998796\",\"tick\":37924,\"avgT\":38006,\"wsI\":13896},\"14334\":{\"init\":true,\"ts\":1754825265,\"cum\":73948705377,\"vo\":\"98818477208\",\"tick\":37928,\"avgT\":38006,\"wsI\":13896},\"14335\":{\"init\":true,\"ts\":1754825310,\"cum\":73950412227,\"vo\":\"98818737128\",\"tick\":37930,\"avgT\":38006,\"wsI\":13896},\"14336\":{\"init\":true,\"ts\":1754825375,\"cum\":73952878262,\"vo\":\"98819028913\",\"tick\":37939,\"avgT\":38006,\"wsI\":13896},\"14337\":{\"init\":true,\"ts\":1754825499,\"cum\":73957582698,\"vo\":\"98819585549\",\"tick\":37939,\"avgT\":38006,\"wsI\":13899},\"14338\":{\"init\":true,\"ts\":1754825574,\"cum\":73960428723,\"vo\":\"98819846624\",\"tick\":37947,\"avgT\":38006,\"wsI\":13899},\"14339\":{\"init\":true,\"ts\":1754825763,\"cum\":73967601840,\"vo\":\"98820377525\",\"tick\":37953,\"avgT\":38006,\"wsI\":13899},\"14340\":{\"init\":true,\"ts\":1754825779,\"cum\":73968209152,\"vo\":\"98820415941\",\"tick\":37957,\"avgT\":38006,\"wsI\":13899},\"14341\":{\"init\":true,\"ts\":1754825869,\"cum\":73971625372,\"vo\":\"98820623301\",\"tick\":37958,\"avgT\":38006,\"wsI\":13899},\"14342\":{\"init\":true,\"ts\":1754826111,\"cum\":73980812902,\"vo\":\"98821040147\",\"tick\":37965,\"avgT\":38007,\"wsI\":13899},\"14343\":{\"init\":true,\"ts\":1754826154,\"cum\":73982445483,\"vo\":\"98821108947\",\"tick\":37967,\"avgT\":38007,\"wsI\":13899},\"14344\":{\"init\":true,\"ts\":1754826206,\"cum\":73984420183,\"vo\":\"98821162195\",\"tick\":37975,\"avgT\":38007,\"wsI\":13899},\"14345\":{\"init\":true,\"ts\":1754826242,\"cum\":73985787571,\"vo\":\"98821182931\",\"tick\":37983,\"avgT\":38007,\"wsI\":13899},\"14346\":{\"init\":true,\"ts\":1754827563,\"cum\":74035976324,\"vo\":\"98821480626\",\"tick\":37993,\"avgT\":38009,\"wsI\":13903},\"14347\":{\"init\":true,\"ts\":1754827633,\"cum\":74038635974,\"vo\":\"98821494346\",\"tick\":37995,\"avgT\":38009,\"wsI\":13904},\"14348\":{\"init\":true,\"ts\":1754827642,\"cum\":74038977938,\"vo\":\"98821495867\",\"tick\":37996,\"avgT\":38009,\"wsI\":13904},\"14349\":{\"init\":true,\"ts\":1754828541,\"cum\":74073143534,\"vo\":\"98821528542\",\"tick\":38004,\"avgT\":38011,\"wsI\":13907},\"14350\":{\"init\":true,\"ts\":1754828829,\"cum\":74084087246,\"vo\":\"98821570014\",\"tick\":37999,\"avgT\":38011,\"wsI\":13910},\"14351\":{\"init\":true,\"ts\":1754828935,\"cum\":74088114716,\"vo\":\"98821597150\",\"tick\":37995,\"avgT\":38011,\"wsI\":13910},\"14352\":{\"init\":true,\"ts\":1754828983,\"cum\":74089938380,\"vo\":\"98821612702\",\"tick\":37993,\"avgT\":38011,\"wsI\":13911},\"14353\":{\"init\":true,\"ts\":1754829003,\"cum\":74090698200,\"vo\":\"98821620702\",\"tick\":37991,\"avgT\":38011,\"wsI\":13911},\"14354\":{\"init\":true,\"ts\":1754829006,\"cum\":74090812167,\"vo\":\"98821622154\",\"tick\":37989,\"avgT\":38011,\"wsI\":13911},\"14355\":{\"init\":true,\"ts\":1754829179,\"cum\":74097383572,\"vo\":\"98821743684\",\"tick\":37985,\"avgT\":38012,\"wsI\":13911},\"14356\":{\"init\":true,\"ts\":1754829818,\"cum\":74121655348,\"vo\":\"98822244660\",\"tick\":37984,\"avgT\":38012,\"wsI\":13913},\"14357\":{\"init\":true,\"ts\":1754830002,\"cum\":74128643852,\"vo\":\"98822427280\",\"tick\":37981,\"avgT\":38013,\"wsI\":13914},\"14358\":{\"init\":true,\"ts\":1754830008,\"cum\":74128871828,\"vo\":\"98822429014\",\"tick\":37996,\"avgT\":38013,\"wsI\":13914},\"14359\":{\"init\":true,\"ts\":1754830026,\"cum\":74129555792,\"vo\":\"98822433064\",\"tick\":37998,\"avgT\":38013,\"wsI\":13914},\"14360\":{\"init\":true,\"ts\":1754830056,\"cum\":74130695792,\"vo\":\"98822438134\",\"tick\":38000,\"avgT\":38013,\"wsI\":13915},\"14361\":{\"init\":true,\"ts\":1754830199,\"cum\":74136130364,\"vo\":\"98822449717\",\"tick\":38004,\"avgT\":38013,\"wsI\":13917},\"14362\":{\"init\":true,\"ts\":1754830466,\"cum\":74146278500,\"vo\":\"98822456392\",\"tick\":38008,\"avgT\":38013,\"wsI\":13920},\"14363\":{\"init\":true,\"ts\":1754830576,\"cum\":74150460150,\"vo\":\"98822456832\",\"tick\":38015,\"avgT\":38013,\"wsI\":13920},\"14364\":{\"init\":true,\"ts\":1754831198,\"cum\":74174107968,\"vo\":\"98822479224\",\"tick\":38019,\"avgT\":38013,\"wsI\":13922},\"14365\":{\"init\":true,\"ts\":1754831236,\"cum\":74175552690,\"vo\":\"98822480592\",\"tick\":38019,\"avgT\":38013,\"wsI\":13923},\"14366\":{\"init\":true,\"ts\":1754832457,\"cum\":74221973889,\"vo\":\"98822517623\",\"tick\":38019,\"avgT\":38014,\"wsI\":13929},\"14367\":{\"init\":true,\"ts\":1754832970,\"cum\":74241489435,\"vo\":\"98822919815\",\"tick\":38042,\"avgT\":38014,\"wsI\":13933},\"14368\":{\"init\":true,\"ts\":1754833027,\"cum\":74243657829,\"vo\":\"98822964503\",\"tick\":38042,\"avgT\":38014,\"wsI\":13934},\"14369\":{\"init\":true,\"ts\":1754833037,\"cum\":74244038209,\"vo\":\"98822970263\",\"tick\":38038,\"avgT\":38014,\"wsI\":13934},\"14370\":{\"init\":true,\"ts\":1754833083,\"cum\":74245787773,\"vo\":\"98822987738\",\"tick\":38034,\"avgT\":38015,\"wsI\":13934},\"14371\":{\"init\":true,\"ts\":1754833093,\"cum\":74246168023,\"vo\":\"98822988738\",\"tick\":38025,\"avgT\":38015,\"wsI\":13934},\"14372\":{\"init\":true,\"ts\":1754833119,\"cum\":74247156569,\"vo\":\"98822989674\",\"tick\":38021,\"avgT\":38015,\"wsI\":13934},\"14373\":{\"init\":true,\"ts\":1754833243,\"cum\":74251870677,\"vo\":\"98822990170\",\"tick\":38017,\"avgT\":38015,\"wsI\":13934},\"14374\":{\"init\":true,\"ts\":1754833416,\"cum\":74258446926,\"vo\":\"98822990862\",\"tick\":38013,\"avgT\":38015,\"wsI\":13934},\"14375\":{\"init\":true,\"ts\":1754833417,\"cum\":74258484883,\"vo\":\"98822994226\",\"tick\":37957,\"avgT\":38015,\"wsI\":13934},\"14376\":{\"init\":true,\"ts\":1754833419,\"cum\":74258560811,\"vo\":\"98822999428\",\"tick\":37964,\"avgT\":38015,\"wsI\":13934},\"14377\":{\"init\":true,\"ts\":1754833421,\"cum\":74258636747,\"vo\":\"98823003846\",\"tick\":37968,\"avgT\":38015,\"wsI\":13934},\"14378\":{\"init\":true,\"ts\":1754833428,\"cum\":74258902544,\"vo\":\"98823017398\",\"tick\":37971,\"avgT\":38015,\"wsI\":13934},\"14379\":{\"init\":true,\"ts\":1754833429,\"cum\":74258940518,\"vo\":\"98823019079\",\"tick\":37974,\"avgT\":38015,\"wsI\":13934},\"14380\":{\"init\":true,\"ts\":1754833497,\"cum\":74261522818,\"vo\":\"98823127879\",\"tick\":37975,\"avgT\":38015,\"wsI\":13934},\"14381\":{\"init\":true,\"ts\":1754833858,\"cum\":74275233237,\"vo\":\"98823595735\",\"tick\":37979,\"avgT\":38015,\"wsI\":13935},\"14382\":{\"init\":true,\"ts\":1754833869,\"cum\":74275651028,\"vo\":\"98823608451\",\"tick\":37981,\"avgT\":38015,\"wsI\":13935},\"14383\":{\"init\":true,\"ts\":1754834007,\"cum\":74280892544,\"vo\":\"98823758733\",\"tick\":37982,\"avgT\":38015,\"wsI\":13935},\"14384\":{\"init\":true,\"ts\":1754834045,\"cum\":74282335860,\"vo\":\"98823800115\",\"tick\":37982,\"avgT\":38015,\"wsI\":13935},\"14385\":{\"init\":true,\"ts\":1754834131,\"cum\":74285602312,\"vo\":\"98823893769\",\"tick\":37982,\"avgT\":38015,\"wsI\":13935},\"14386\":{\"init\":true,\"ts\":1754834451,\"cum\":74297757512,\"vo\":\"98824181769\",\"tick\":37985,\"avgT\":38015,\"wsI\":13936},\"14387\":{\"init\":true,\"ts\":1754834475,\"cum\":74298668936,\"vo\":\"98824218273\",\"tick\":37976,\"avgT\":38015,\"wsI\":13936},\"14388\":{\"init\":true,\"ts\":1754834546,\"cum\":74301364664,\"vo\":\"98824375112\",\"tick\":37968,\"avgT\":38015,\"wsI\":13936},\"14389\":{\"init\":true,\"ts\":1754835133,\"cum\":74323649532,\"vo\":\"98825932083\",\"tick\":37964,\"avgT\":38016,\"wsI\":13937},\"14390\":{\"init\":true,\"ts\":1754835259,\"cum\":74328433248,\"vo\":\"98826247083\",\"tick\":37966,\"avgT\":38016,\"wsI\":13940},\"14391\":{\"init\":true,\"ts\":1754835336,\"cum\":74331356784,\"vo\":\"98826424491\",\"tick\":37968,\"avgT\":38016,\"wsI\":13941},\"14392\":{\"init\":true,\"ts\":1754835666,\"cum\":74343892164,\"vo\":\"98826721491\",\"tick\":37986,\"avgT\":38016,\"wsI\":13943},\"14393\":{\"init\":true,\"ts\":1754835817,\"cum\":74349628654,\"vo\":\"98826823567\",\"tick\":37990,\"avgT\":38016,\"wsI\":13944},\"14394\":{\"init\":true,\"ts\":1754835873,\"cum\":74351756206,\"vo\":\"98826855823\",\"tick\":37992,\"avgT\":38016,\"wsI\":13944},\"14395\":{\"init\":true,\"ts\":1754835910,\"cum\":74353161984,\"vo\":\"98826873731\",\"tick\":37994,\"avgT\":38016,\"wsI\":13944},\"14396\":{\"init\":true,\"ts\":1754835937,\"cum\":74354187930,\"vo\":\"98826882479\",\"tick\":37998,\"avgT\":38016,\"wsI\":13944},\"14397\":{\"init\":true,\"ts\":1754835950,\"cum\":74354681917,\"vo\":\"98826886236\",\"tick\":37999,\"avgT\":38016,\"wsI\":13944},\"14398\":{\"init\":true,\"ts\":1754836031,\"cum\":74357759917,\"vo\":\"98826906972\",\"tick\":38000,\"avgT\":38016,\"wsI\":13945},\"14399\":{\"init\":true,\"ts\":1754836081,\"cum\":74359660117,\"vo\":\"98826914172\",\"tick\":38004,\"avgT\":38016,\"wsI\":13945},\"14400\":{\"init\":true,\"ts\":1754836122,\"cum\":74361218404,\"vo\":\"98826917493\",\"tick\":38007,\"avgT\":38016,\"wsI\":13945},\"14401\":{\"init\":true,\"ts\":1754836228,\"cum\":74365247570,\"vo\":\"98826920143\",\"tick\":38011,\"avgT\":38016,\"wsI\":13946},\"14402\":{\"init\":true,\"ts\":1754836271,\"cum\":74366882215,\"vo\":\"98826920186\",\"tick\":38015,\"avgT\":38016,\"wsI\":13946},\"14403\":{\"init\":true,\"ts\":1754836413,\"cum\":74372280913,\"vo\":\"98826921464\",\"tick\":38019,\"avgT\":38016,\"wsI\":13946},\"14404\":{\"init\":true,\"ts\":1754836654,\"cum\":74381443974,\"vo\":\"98826926359\",\"tick\":38021,\"avgT\":38017,\"wsI\":13947},\"14405\":{\"init\":true,\"ts\":1754836694,\"cum\":74382964974,\"vo\":\"98826928919\",\"tick\":38025,\"avgT\":38017,\"wsI\":13948},\"14406\":{\"init\":true,\"ts\":1754836729,\"cum\":74384295849,\"vo\":\"98826931159\",\"tick\":38025,\"avgT\":38017,\"wsI\":13948},\"14407\":{\"init\":true,\"ts\":1754836751,\"cum\":74385132487,\"vo\":\"98826934327\",\"tick\":38029,\"avgT\":38017,\"wsI\":13948},\"14408\":{\"init\":true,\"ts\":1754836788,\"cum\":74386539745,\"vo\":\"98826945020\",\"tick\":38034,\"avgT\":38017,\"wsI\":13948},\"14409\":{\"init\":true,\"ts\":1754836890,\"cum\":74390419621,\"vo\":\"98826990002\",\"tick\":38038,\"avgT\":38017,\"wsI\":13949},\"14410\":{\"init\":true,\"ts\":1754836925,\"cum\":74391751021,\"vo\":\"98827008517\",\"tick\":38040,\"avgT\":38017,\"wsI\":13949},\"14411\":{\"init\":true,\"ts\":1754836949,\"cum\":74392664077,\"vo\":\"98827026013\",\"tick\":38044,\"avgT\":38017,\"wsI\":13949},\"14412\":{\"init\":true,\"ts\":1754836955,\"cum\":74392892353,\"vo\":\"98827031059\",\"tick\":38046,\"avgT\":38017,\"wsI\":13949},\"14413\":{\"init\":true,\"ts\":1754836956,\"cum\":74392930403,\"vo\":\"98827032148\",\"tick\":38050,\"avgT\":38017,\"wsI\":13949},\"14414\":{\"init\":true,\"ts\":1754837058,\"cum\":74396812319,\"vo\":\"98827203610\",\"tick\":38058,\"avgT\":38017,\"wsI\":13950},\"14415\":{\"init\":true,\"ts\":1754837181,\"cum\":74401496036,\"vo\":\"98827676422\",\"tick\":38079,\"avgT\":38017,\"wsI\":13950},\"14416\":{\"init\":true,\"ts\":1754837439,\"cum\":74411324030,\"vo\":\"98829166630\",\"tick\":38093,\"avgT\":38017,\"wsI\":13950},\"14417\":{\"init\":true,\"ts\":1754837641,\"cum\":74419019018,\"vo\":\"98830348724\",\"tick\":38094,\"avgT\":38018,\"wsI\":13950},\"14418\":{\"init\":true,\"ts\":1754838468,\"cum\":74450531853,\"vo\":\"98836536527\",\"tick\":38105,\"avgT\":38019,\"wsI\":13956},\"14419\":{\"init\":true,\"ts\":1754838535,\"cum\":74453085156,\"vo\":\"98837079227\",\"tick\":38109,\"avgT\":38019,\"wsI\":13956},\"14420\":{\"init\":true,\"ts\":1754838654,\"cum\":74457622626,\"vo\":\"98838545426\",\"tick\":38130,\"avgT\":38019,\"wsI\":13956},\"14421\":{\"init\":true,\"ts\":1754838782,\"cum\":74462504546,\"vo\":\"98840419474\",\"tick\":38140,\"avgT\":38019,\"wsI\":13958},\"14422\":{\"init\":true,\"ts\":1754839020,\"cum\":74471587816,\"vo\":\"98845457867\",\"tick\":38165,\"avgT\":38020,\"wsI\":13958},\"14423\":{\"init\":true,\"ts\":1754839275,\"cum\":74481323206,\"vo\":\"98851823687\",\"tick\":38178,\"avgT\":38020,\"wsI\":13960},\"14424\":{\"init\":true,\"ts\":1754839434,\"cum\":74487394780,\"vo\":\"98856178584\",\"tick\":38186,\"avgT\":38021,\"wsI\":13962},\"14425\":{\"init\":true,\"ts\":1754839594,\"cum\":74493505500,\"vo\":\"98860857144\",\"tick\":38192,\"avgT\":38021,\"wsI\":13962},\"14426\":{\"init\":true,\"ts\":1754839764,\"cum\":74499999160,\"vo\":\"98866183074\",\"tick\":38198,\"avgT\":38021,\"wsI\":13962},\"14427\":{\"init\":true,\"ts\":1754839866,\"cum\":74503895254,\"vo\":\"98869324532\",\"tick\":38197,\"avgT\":38022,\"wsI\":13962},\"14428\":{\"init\":true,\"ts\":1754839997,\"cum\":74508899323,\"vo\":\"98873428631\",\"tick\":38199,\"avgT\":38022,\"wsI\":13962},\"14429\":{\"init\":true,\"ts\":1754840916,\"cum\":74543998690,\"vo\":\"98899987697\",\"tick\":38193,\"avgT\":38024,\"wsI\":13964},\"14430\":{\"init\":true,\"ts\":1754840954,\"cum\":74545450328,\"vo\":\"98901178199\",\"tick\":38201,\"avgT\":38024,\"wsI\":13964},\"14431\":{\"init\":true,\"ts\":1754840988,\"cum\":74546749264,\"vo\":\"98902279799\",\"tick\":38204,\"avgT\":38024,\"wsI\":13964},\"14432\":{\"init\":true,\"ts\":1754841480,\"cum\":74565549568,\"vo\":\"98919484337\",\"tick\":38212,\"avgT\":38026,\"wsI\":13967},\"14433\":{\"init\":true,\"ts\":1754841526,\"cum\":74567307964,\"vo\":\"98921324337\",\"tick\":38226,\"avgT\":38026,\"wsI\":13967},\"14434\":{\"init\":true,\"ts\":1754842523,\"cum\":74605433244,\"vo\":\"98966345228\",\"tick\":38240,\"avgT\":38029,\"wsI\":13971},\"14435\":{\"init\":true,\"ts\":1754842690,\"cum\":74611818155,\"vo\":\"98973260884\",\"tick\":38233,\"avgT\":38030,\"wsI\":13971},\"14436\":{\"init\":true,\"ts\":1754843923,\"cum\":74658950813,\"vo\":\"99019666940\",\"tick\":38226,\"avgT\":38034,\"wsI\":13978},\"14437\":{\"init\":true,\"ts\":1754843927,\"cum\":74659103717,\"vo\":\"99019814396\",\"tick\":38226,\"avgT\":38034,\"wsI\":13978},\"14438\":{\"init\":true,\"ts\":1754843931,\"cum\":74659256621,\"vo\":\"99019961852\",\"tick\":38226,\"avgT\":38034,\"wsI\":13978},\"14439\":{\"init\":true,\"ts\":1754843990,\"cum\":74661512014,\"vo\":\"99022159543\",\"tick\":38227,\"avgT\":38034,\"wsI\":13979},\"14440\":{\"init\":true,\"ts\":1754844181,\"cum\":74668813944,\"vo\":\"99029459431\",\"tick\":38230,\"avgT\":38035,\"wsI\":13980},\"14441\":{\"init\":true,\"ts\":1754844315,\"cum\":74673937300,\"vo\":\"99034765965\",\"tick\":38234,\"avgT\":38035,\"wsI\":13981},\"14442\":{\"init\":true,\"ts\":1754844325,\"cum\":74674319720,\"vo\":\"99035194455\",\"tick\":38242,\"avgT\":38035,\"wsI\":13981},\"14443\":{\"init\":true,\"ts\":1754844356,\"cum\":74675505811,\"vo\":\"99036777811\",\"tick\":38261,\"avgT\":38035,\"wsI\":13981},\"14444\":{\"init\":true,\"ts\":1754844866,\"cum\":74695021981,\"vo\":\"99063991629\",\"tick\":38267,\"avgT\":38037,\"wsI\":13981},\"14445\":{\"init\":true,\"ts\":1754844898,\"cum\":74696246525,\"vo\":\"99065684429\",\"tick\":38267,\"avgT\":38037,\"wsI\":13981},\"14446\":{\"init\":true,\"ts\":1754844962,\"cum\":74698695613,\"vo\":\"99069055100\",\"tick\":38267,\"avgT\":38038,\"wsI\":13981},\"14447\":{\"init\":true,\"ts\":1754844995,\"cum\":74699958259,\"vo\":\"99070710908\",\"tick\":38262,\"avgT\":38038,\"wsI\":13981},\"14448\":{\"init\":true,\"ts\":1754846964,\"cum\":74775272509,\"vo\":\"99157133864\",\"tick\":38250,\"avgT\":38043,\"wsI\":13990},\"14449\":{\"init\":true,\"ts\":1754847270,\"cum\":74786974867,\"vo\":\"99169312566\",\"tick\":38243,\"avgT\":38044,\"wsI\":13990},\"14450\":{\"init\":true,\"ts\":1754847345,\"cum\":74789844217,\"vo\":\"99172747266\",\"tick\":38258,\"avgT\":38044,\"wsI\":13990},\"14451\":{\"init\":true,\"ts\":1754847360,\"cum\":74790418312,\"vo\":\"99173533881\",\"tick\":38273,\"avgT\":38044,\"wsI\":13990},\"14452\":{\"init\":true,\"ts\":1754847456,\"cum\":74794094152,\"vo\":\"99179319587\",\"tick\":38290,\"avgT\":38045,\"wsI\":13990},\"14453\":{\"vo\":\"0\"},\"14454\":{\"vo\":\"0\"}},\"vo\":{\"init\":true,\"tpIdx\":14452,\"lastTs\":1754847456},\"dF\":{\"b1\":360,\"b2\":60000,\"g1\":59,\"g2\":8500,\"bF\":2400}}","staticExtra":"{\"poolId\":\"0x5c332ec2387acd1e403682035c3b167d82725e89\"}","blockNumber":10772444}`)

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
				Token:  "0xca79db4b49f608ef54a5cb813fbed3a6387bc645",
				Amount: big.NewInt(1211111111111111111),
			},
			TokenOut: "0x5555555555555555555555555555555555555555",
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

func TestPoolSimulator_CalcAmountIn(t *testing.T) {
	t.Parallel()
	testutil.TestCalcAmountIn(t, ps)
}
