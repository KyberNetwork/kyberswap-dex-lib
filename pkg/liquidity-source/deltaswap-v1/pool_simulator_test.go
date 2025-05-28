package deltaswapv1

import (
	"math/big"
	"reflect"
	"strings"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	utils "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

func Test_calcEMA(t *testing.T) {
	t.Parallel()
	type args struct {
		last      *uint256.Int
		emaLast   *uint256.Int
		emaWeight *uint256.Int
	}
	tests := []struct {
		name     string
		args     args
		expected *uint256.Int
	}{
		{
			name: "Test 1",
			args: args{
				last:      uint256.NewInt(0),
				emaLast:   uint256.NewInt(0),
				emaWeight: uint256.NewInt(0),
			},
			expected: uint256.NewInt(0),
		},
		{
			name: "Test 2",
			args: args{
				last:      uint256.NewInt(100),
				emaLast:   uint256.NewInt(90),
				emaWeight: uint256.NewInt(20),
			},
			expected: uint256.NewInt(92),
		},
		{
			name: "Test 3",
			args: args{
				last:      uint256.NewInt(120),
				emaLast:   uint256.NewInt(100),
				emaWeight: uint256.NewInt(20),
			},
			expected: uint256.NewInt(104),
		},
		{
			name: "Test 4",
			args: args{
				last:      uint256.NewInt(0),
				emaLast:   uint256.NewInt(90),
				emaWeight: uint256.NewInt(20),
			},
			expected: uint256.NewInt(72),
		},
		{
			name: "Test 5",
			args: args{
				last:      uint256.NewInt(2),
				emaLast:   uint256.NewInt(0),
				emaWeight: uint256.NewInt(100),
			},
			expected: uint256.NewInt(2),
		},
		{
			name: "Test 6",
			args: args{
				last:      uint256.NewInt(100),
				emaLast:   uint256.NewInt(90),
				emaWeight: uint256.NewInt(100),
			},
			expected: uint256.NewInt(100),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := calcEMA(tt.args.last, tt.args.emaLast, tt.args.emaWeight); !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("calcEMA() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

func Test_calcSingleSideLiquidity(t *testing.T) {
	t.Parallel()
	type args struct {
		amount   *uint256.Int
		reserve0 *uint256.Int
		reserve1 *uint256.Int
	}
	tests := []struct {
		name     string
		args     args
		expected *uint256.Int
	}{
		{
			name: "Test 1",
			args: args{
				amount:   uint256.NewInt(0),
				reserve0: uint256.NewInt(0),
				reserve1: uint256.NewInt(0),
			},
			expected: uint256.NewInt(0),
		},
		{
			name: "Test 2",
			args: args{
				amount:   uint256.NewInt(100),
				reserve0: uint256.NewInt(50),
				reserve1: uint256.NewInt(50),
			},
			expected: uint256.NewInt(50),
		},
		{
			name: "Test 3",
			args: args{
				amount:   uint256.NewInt(473164),
				reserve0: uint256.NewInt(4234723486123),
				reserve1: uint256.NewInt(2132193),
			},
			expected: uint256.NewInt(0),
		},
		{
			name: "Test 4",
			args: args{
				amount:   uint256.NewInt(21321321334),
				reserve0: uint256.NewInt(32133213),
				reserve1: uint256.NewInt(532132130),
			},
			expected: uint256.NewInt(43382720670),
		},
		{
			name: "Test 5",
			args: args{
				amount:   uint256.NewInt(21321321334),
				reserve0: uint256.NewInt(32133213),
				reserve1: uint256.NewInt(532132130),
			},
			expected: uint256.NewInt(43382720670),
		},
		{
			name: "Test 6",
			args: args{
				amount:   uint256.NewInt(9219321993),
				reserve0: uint256.NewInt(996123),
				reserve1: uint256.NewInt(2132254245193),
			},
			expected: uint256.NewInt(6744224174616),
		},
		{
			name: "Test 7",
			args: args{
				amount:   uint256.NewInt(32),
				reserve0: uint256.MustFromDecimal("28801288458145955324"),
				reserve1: uint256.NewInt(1689894),
			},
			expected: uint256.NewInt(0),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := calcSingleSideLiquidity(tt.args.amount, tt.args.reserve0, tt.args.reserve1); !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("calcSingleSideLiquidity() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestPoolSimulator(t *testing.T) {
	t.Parallel()
	type fields struct {
		Pool                     pool.Pool
		feePrecision             *uint256.Int
		dsFee                    *uint256.Int
		dsFeeThreshold           *uint256.Int
		tradeLiquidityEMA        *uint256.Int
		liquidityEMA             *uint256.Int
		lastLiquidityBlockNumber uint64
		lastTradeLiquiditySum    *uint256.Int
		lastTradeBlockNumber     uint64
	}
	type expected struct {
		err            error
		amountOut      *uint256.Int
		tradeLiquidity *uint256.Int
		fee            *uint256.Int
		getAmountIn    *uint256.Int
	}
	tests := []struct {
		name                string
		fields              fields
		calcAmountOutParams pool.CalcAmountOutParams
		calcAmountInParams  pool.CalcAmountInParams
		expected            expected
	}{
		{
			name: "Test 1: mock tx with fee=0",
			fields: fields{
				Pool: pool.Pool{Info: pool.PoolInfo{
					Address:     strings.ToLower("0x755f72d7f22efaed6e00e589a8c7bd95a666fef0"),
					Tokens:      []string{"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8", "0x82af49447d8a07e3bd95bd0d56f35241523fbab1"},
					Reserves:    []*big.Int{utils.NewBig10("8436045165882568536"), utils.NewBig10("22690278552")},
					BlockNumber: 21074172,
				}},
				feePrecision:             Number_1000,
				dsFee:                    uint256.NewInt(0),
				dsFeeThreshold:           uint256.NewInt(200),
				tradeLiquidityEMA:        uint256.NewInt(19281887),
				liquidityEMA:             uint256.NewInt(1000),
				lastLiquidityBlockNumber: 21074172,
				lastTradeLiquiditySum:    uint256.NewInt(1),
				lastTradeBlockNumber:     21074172,
			},
			calcAmountOutParams: pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{
					Token:  "0x82af49447d8a07e3bd95bd0d56f35241523fbab1",
					Amount: utils.NewBig10("2000"),
				},
				TokenOut: "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8",
			},
			expected: expected{
				amountOut:      uint256.NewInt(743582270527),
				tradeLiquidity: uint256.NewInt(19281887),
				fee:            uint256.NewInt(0),
				getAmountIn:    uint256.NewInt(2000),
			},
		},
		{
			// https://dashboard.tenderly.co/tenderly_kyber/nhathm/fork/af8cc9f2-18ad-4ffd-8464-f27834a35af2/simulation/5fc3c7a6-4e9a-429a-817b-201d9d445da1
			name: "Test 2: ",
			fields: fields{
				Pool: pool.Pool{Info: pool.PoolInfo{
					Address:     strings.ToLower("0x755f72d7f22efaed6e00e589a8c7bd95a666fef0"),
					Tokens:      []string{"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8", "0x82af49447d8a07e3bd95bd0d56f35241523fbab1"},
					Reserves:    []*big.Int{utils.NewBig10("8472159523484284304"), utils.NewBig10("22594888023")},
					BlockNumber: 21074172,
				}},
				feePrecision:             Number_1000,
				dsFee:                    uint256.NewInt(3),
				dsFeeThreshold:           uint256.NewInt(0),
				tradeLiquidityEMA:        uint256.NewInt(19281887),
				liquidityEMA:             uint256.NewInt(1000),
				lastLiquidityBlockNumber: 21074172,
				lastTradeLiquiditySum:    uint256.NewInt(1),
				lastTradeBlockNumber:     21074172,
			},
			calcAmountOutParams: pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{
					Token:  "0x82af49447d8a07e3bd95bd0d56f35241523fbab1",
					Amount: utils.NewBig10("1000000"),
				},
				TokenOut: "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8",
			},
			expected: expected{
				amountOut:      uint256.NewInt(373817756477164),
				tradeLiquidity: uint256.NewInt(9681930703),
				fee:            uint256.NewInt(3),
				getAmountIn:    uint256.NewInt(1000000),
			},
		},
		{
			// https://dashboard.tenderly.co/tenderly_kyber/nhathm/fork/f640c723-ce75-417a-8c00-350362f27ab1/simulation/66b31778-4c9e-41d7-ac81-606f8430f447
			name: "Test 3: state override dsFee=0, dsFeeThreshold=0",
			fields: fields{
				Pool: pool.Pool{Info: pool.PoolInfo{
					Address:     strings.ToLower("0x755f72d7f22efaed6e00e589a8c7bd95a666fef0"),
					Tokens:      []string{"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8", "0x82af49447d8a07e3bd95bd0d56f35241523fbab1"},
					Reserves:    []*big.Int{utils.NewBig10("8436045165882568536"), utils.NewBig10("22690278552")},
					BlockNumber: 269352742,
				}},
				feePrecision:             Number_1000,
				dsFee:                    uint256.NewInt(0),
				dsFeeThreshold:           uint256.NewInt(0),
				tradeLiquidityEMA:        uint256.NewInt(186241029792),
				liquidityEMA:             uint256.NewInt(605595291285450),
				lastLiquidityBlockNumber: 269352742,
				lastTradeLiquiditySum:    uint256.NewInt(19281888),
				lastTradeBlockNumber:     269352742,
			},
			calcAmountOutParams: pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{
					Token:  "0x82af49447d8a07e3bd95bd0d56f35241523fbab1",
					Amount: utils.NewBig10("1000000"),
				},
				TokenOut: "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8",
			},
			expected: expected{
				amountOut:      uint256.NewInt(371774783274123),
				tradeLiquidity: uint256.NewInt(9640943522),
				fee:            uint256.NewInt(0),
				getAmountIn:    uint256.NewInt(1000000),
			},
		},
		{
			// https://dashboard.tenderly.co/tenderly_kyber/nhathm/fork/af8cc9f2-18ad-4ffd-8464-f27834a35af2/simulation/ec0ec5f6-804f-436b-bc4d-4bd125fe58e2
			name: "Test 4: state override dsFee=0",
			fields: fields{
				Pool: pool.Pool{Info: pool.PoolInfo{
					Address:     strings.ToLower("0x755f72d7f22efaed6e00e589a8c7bd95a666fef0"),
					Tokens:      []string{"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8", "0x82af49447d8a07e3bd95bd0d56f35241523fbab1"},
					Reserves:    []*big.Int{utils.NewBig10("8471785705727807140"), utils.NewBig10("22595888023")},
					BlockNumber: 51077999,
				}},
				feePrecision:             Number_1000,
				dsFee:                    uint256.NewInt(0),
				dsFeeThreshold:           uint256.NewInt(0),
				tradeLiquidityEMA:        uint256.NewInt(186241029792),
				liquidityEMA:             uint256.NewInt(605595291285450),
				lastLiquidityBlockNumber: 21077999,
				lastTradeLiquiditySum:    uint256.NewInt(19281888),
				lastTradeBlockNumber:     21077999,
			},
			calcAmountOutParams: pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{
					Token:  "0x82af49447d8a07e3bd95bd0d56f35241523fbab1",
					Amount: utils.NewBig10("10000000001"),
				},
				TokenOut: "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8",
			},
			expected: expected{
				amountOut:      uint256.NewInt(2599035099254636496),
				tradeLiquidity: uint256.NewInt(96815028641079),
				fee:            uint256.NewInt(0),
				getAmountIn:    uint256.NewInt(10000000001),
			},
		},
		{
			// https://dashboard.tenderly.co/tenderly_kyber/nhathm/fork/baa370fd-7a23-4c8e-98f0-46f94eefa1c6/simulation/c038cfd7-f09e-4bc7-bc81-14abd6c7381a
			name: "Test 5",
			fields: fields{
				Pool: pool.Pool{Info: pool.PoolInfo{
					Address:     strings.ToLower("0x18663c0dc489c349d5d201a60cb006e7e58c00e0"),
					Tokens:      []string{"0x82af49447d8a07e3bd95bd0d56f35241523fbab1", "0x912ce59144191c1204e64559fe8253a0e49e6548"},
					Reserves:    []*big.Int{utils.NewBig10("974183970825735312"), utils.NewBig10("4636847921500029773274")},
					BlockNumber: 51077999,
				}},
				feePrecision:             Number_1000,
				dsFee:                    uint256.NewInt(3),
				dsFeeThreshold:           uint256.NewInt(0),
				tradeLiquidityEMA:        uint256.NewInt(40142329862594041),
				liquidityEMA:             uint256.MustFromDecimal("67209693648178135590"),
				lastLiquidityBlockNumber: 21079985,
				lastTradeLiquiditySum:    uint256.NewInt(40142329862594041),
				lastTradeBlockNumber:     21079985,
			},
			calcAmountOutParams: pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{
					Token:  "0x912ce59144191c1204e64559fe8253a0e49e6548",
					Amount: big.NewInt(50000000),
				},
				TokenOut: "0x82af49447d8a07e3bd95bd0d56f35241523fbab1",
			},
			expected: expected{
				amountOut:      uint256.NewInt(10473),
				tradeLiquidity: uint256.NewInt(362353),
				fee:            uint256.NewInt(3),
				getAmountIn:    uint256.NewInt(49998598),
			},
		},
		{
			// https://dashboard.tenderly.co/tenderly_kyber/nhathm/fork/890cb929-a710-4a9c-9bf2-e5c31e063764/simulation/04a00a11-911e-4975-a5f2-02d0eae91f81
			name: "Test 6",
			fields: fields{
				Pool: pool.Pool{Info: pool.PoolInfo{
					Address:     strings.ToLower("0x18663c0dc489c349d5d201a60cb006e7e58c00e0"),
					Tokens:      []string{"0xfc5a1a6eb076a2c7ad06ed22c90d7e710e35ad0a", "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8"},
					Reserves:    []*big.Int{utils.NewBig10("5096889559760847010"), utils.NewBig10("124843209")},
					BlockNumber: 269399614,
				}},
				feePrecision:             Number_1000,
				dsFee:                    uint256.NewInt(3),
				dsFeeThreshold:           uint256.NewInt(0),
				tradeLiquidityEMA:        uint256.NewInt(82598440409),
				liquidityEMA:             uint256.MustFromDecimal("25225226432266"),
				lastLiquidityBlockNumber: 21079951,
				lastTradeLiquiditySum:    uint256.NewInt(40142329862594041),
				lastTradeBlockNumber:     21079951,
			},
			calcAmountOutParams: pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{
					Token:  "0xfc5a1a6eb076a2c7ad06ed22c90d7e710e35ad0a",
					Amount: big.NewInt(1000000000000000000),
				},
				TokenOut: "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8",
			},
			expected: expected{
				amountOut:      uint256.NewInt(20425161),
				tradeLiquidity: uint256.NewInt(2474570568805),
				fee:            uint256.NewInt(3),
				getAmountIn:    uint256.NewInt(999999961589995713),
			},
		},
		{
			// https://dashboard.tenderly.co/tenderly_kyber/nhathm/fork/afe748e3-7941-4b9b-bc07-f6195c2cc5bd/simulation/305aa27e-4dfc-4e39-89ab-8b0dd78cbfa1
			name: "Test 7",
			fields: fields{
				Pool: pool.Pool{Info: pool.PoolInfo{
					Address:     strings.ToLower("0xb737586e9ab03c2aa1e1a4f164dcec2fe1dfbeb7"),
					Tokens:      []string{"0x82af49447d8a07e3bd95bd0d56f35241523fbab1", "0xaf88d065e77c8cc2239327c5edb3a432268e5831"},
					Reserves:    []*big.Int{utils.NewBig10("133015199886255268118"), utils.NewBig10("354129255591")},
					BlockNumber: 269399614,
				}},
				feePrecision:             Number_1000,
				dsFee:                    uint256.NewInt(3),
				dsFeeThreshold:           uint256.NewInt(0),
				tradeLiquidityEMA:        uint256.NewInt(377591003459),
				liquidityEMA:             uint256.MustFromDecimal("6863273842930235"),
				lastLiquidityBlockNumber: 21081803,
				lastTradeLiquiditySum:    uint256.NewInt(484162813413),
				lastTradeBlockNumber:     21081803,
			},
			calcAmountOutParams: pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{
					Token:  "0xaf88d065e77c8cc2239327c5edb3a432268e5831",
					Amount: big.NewInt(200000),
				},
				TokenOut: "0x82af49447d8a07e3bd95bd0d56f35241523fbab1",
			},
			expected: expected{
				amountOut:      uint256.NewInt(74896991717318),
				tradeLiquidity: uint256.NewInt(1938071220),
				fee:            uint256.NewInt(3),
				getAmountIn:    uint256.NewInt(200000),
			},
		},
		{
			name: "Test 8",
			fields: fields{
				Pool: pool.Pool{Info: pool.PoolInfo{
					Address:     strings.ToLower("0x508d15186bc00d2a21c76f16c76a202169ecfff9"),
					Tokens:      []string{"0x95146881b86b3ee99e63705ec87afe29fcc044d9", "0xaf88d065e77c8cc2239327c5edb3a432268e5831"},
					Reserves:    []*big.Int{utils.NewBig10("28801288458145955324"), utils.NewBig10("1689894")},
					BlockNumber: 269399614,
				}},
				feePrecision:             Number_1000,
				dsFee:                    uint256.NewInt(3),
				dsFeeThreshold:           uint256.NewInt(0),
				tradeLiquidityEMA:        uint256.NewInt(6479739510165),
				liquidityEMA:             uint256.MustFromDecimal("1574913102224118"),
				lastLiquidityBlockNumber: 21049204,
				lastTradeLiquiditySum:    uint256.NewInt(6479739510165),
				lastTradeBlockNumber:     21049204,
			},
			calcAmountOutParams: pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{
					Token:  "0x95146881b86b3ee99e63705ec87afe29fcc044d9",
					Amount: big.NewInt(32),
				},
				TokenOut: "0xaf88d065e77c8cc2239327c5edb3a432268e5831",
			},
			expected: expected{
				err: ErrZeroTradeLiquidity,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &PoolSimulator{
				Pool:                     tt.fields.Pool,
				feePrecision:             tt.fields.feePrecision,
				dsFee:                    tt.fields.dsFee,
				dsFeeThreshold:           tt.fields.dsFeeThreshold,
				tradeLiquidityEMA:        tt.fields.tradeLiquidityEMA,
				liquidityEMA:             tt.fields.liquidityEMA,
				lastLiquidityBlockNumber: tt.fields.lastLiquidityBlockNumber,
				lastTradeLiquiditySum:    tt.fields.lastTradeLiquiditySum,
				lastTradeBlockNumber:     tt.fields.lastTradeBlockNumber,
			}

			indexIn, indexOut := s.GetTokenIndex(tt.calcAmountOutParams.TokenAmountIn.Token), s.GetTokenIndex(tt.calcAmountOutParams.TokenOut)
			assert.GreaterOrEqual(t, indexIn, 0)
			assert.GreaterOrEqual(t, indexOut, 0)

			actualCalcAmountOutResult, err := s.CalcAmountOut(tt.calcAmountOutParams)
			if tt.expected.err != nil {
				assert.NotNil(t, err)
				assert.Equal(t, tt.expected.err.Error(), err.Error())
				return
			} else {
				assert.Nil(t, err)
			}

			tradeLiquidity, actualFee, _ := s.calcPairTradingFee(
				uint256.MustFromBig(tt.calcAmountOutParams.TokenAmountIn.Amount),
				uint256.MustFromBig(s.Pool.Info.Reserves[indexIn]),
				uint256.MustFromBig(s.Pool.Info.Reserves[indexOut]))

			assert.EqualValues(t, tt.expected.tradeLiquidity.String(), tradeLiquidity.String())
			assert.EqualValues(t, tt.expected.amountOut.String(), actualCalcAmountOutResult.TokenAmountOut.Amount.String())
			assert.Zero(t, actualFee.Cmp(tt.expected.fee), "actual fee: %s, expected fee: %s", actualFee.String(), tt.expected.fee.String())

			actualCalcAmountInResult, _ := s.CalcAmountIn(pool.CalcAmountInParams{
				TokenAmountOut: *actualCalcAmountOutResult.TokenAmountOut,
				TokenIn:        tt.calcAmountOutParams.TokenAmountIn.Token,
			})
			assert.Nil(t, err)
			assert.EqualValues(t, tt.expected.getAmountIn.String(), actualCalcAmountInResult.TokenAmountIn.Amount.String())

			s.UpdateBalance(pool.UpdateBalanceParams{
				TokenAmountIn:  tt.calcAmountOutParams.TokenAmountIn,
				TokenAmountOut: *actualCalcAmountOutResult.TokenAmountOut,
				SwapInfo:       actualCalcAmountOutResult.SwapInfo,
			})
		})
	}
}

func TestPoolSimulator_UpdateBalance(t *testing.T) {
	t.Parallel()
	type fields struct {
		Pool                     pool.Pool
		feePrecision             *uint256.Int
		dsFee                    *uint256.Int
		dsFeeThreshold           *uint256.Int
		tradeLiquidityEMA        *uint256.Int
		liquidityEMA             *uint256.Int
		lastLiquidityBlockNumber uint64
		lastTradeLiquiditySum    *uint256.Int
		lastTradeBlockNumber     uint64
	}
	type expected struct {
		tradeLiquidity *uint256.Int
		fee            *uint256.Int
		liquidityEMA   *uint256.Int
	}
	tests := []struct {
		name                 string
		fields               fields
		calcAmountOutParams1 pool.CalcAmountOutParams
		calcAmountOutParams2 pool.CalcAmountOutParams
		expected             expected
	}{
		{
			// https://dashboard.tenderly.co/tenderly_kyber/nhathm/fork/6fe08508-524e-4530-9003-c410e00ed0a7/simulation/72dd4455-e499-49d6-8867-4e0f40bcb725
			name: "Test 1: direct pool",
			fields: fields{
				Pool: pool.Pool{Info: pool.PoolInfo{
					Address:     strings.ToLower("0x755f72d7f22efaed6e00e589a8c7bd95a666fef0"),
					Tokens:      []string{"0x82af49447d8a07e3bd95bd0d56f35241523fbab1", "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8"},
					Reserves:    []*big.Int{utils.NewBig10("8488356036231783622"), utils.NewBig10("22552079594")},
					BlockNumber: 269450712,
				}},
				feePrecision:             Number_1000,
				dsFee:                    uint256.NewInt(3),
				dsFeeThreshold:           uint256.NewInt(0),
				tradeLiquidityEMA:        uint256.NewInt(116404),
				liquidityEMA:             uint256.NewInt(437527234524896),
				lastLiquidityBlockNumber: 21082998,
				lastTradeLiquiditySum:    uint256.NewInt(206882633397),
				lastTradeBlockNumber:     21082998,
			},
			calcAmountOutParams1: pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{
					Token:  "0x82af49447d8a07e3bd95bd0d56f35241523fbab1",
					Amount: utils.NewBig10("900000000000000000000"),
				},
				TokenOut: "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8",
			},
			expected: expected{
				tradeLiquidity: uint256.NewInt(23194980829731245),
				fee:            uint256.NewInt(3),
				liquidityEMA:   uint256.NewInt(438178846339938),
			},
		},
		{
			// https://dashboard.tenderly.co/tenderly_kyber/nhathm/fork/6fe08508-524e-4530-9003-c410e00ed0a7/simulation/6751a5a4-70bd-417a-8659-938d5491b50e
			name: "Test 2: mock tx with multi-path route",
			fields: fields{
				Pool: pool.Pool{Info: pool.PoolInfo{
					Address:     strings.ToLower("0x755f72d7f22efaed6e00e589a8c7bd95a666fef0"),
					Tokens:      []string{"0x82af49447d8a07e3bd95bd0d56f35241523fbab1", "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8"},
					Reserves:    []*big.Int{utils.NewBig10("8488356036231783622"), utils.NewBig10("22552079594")},
					BlockNumber: 21083169,
				}},
				feePrecision:             Number_1000,
				dsFee:                    uint256.NewInt(3),
				dsFeeThreshold:           uint256.NewInt(0),
				tradeLiquidityEMA:        uint256.NewInt(10308880368769442),
				liquidityEMA:             uint256.NewInt(437527234524896),
				lastLiquidityBlockNumber: 21082998,
				lastTradeLiquiditySum:    uint256.NewInt(10308880368769442),
				lastTradeBlockNumber:     21082998,
			},
			calcAmountOutParams1: pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{
					Token:  "0x82af49447d8a07e3bd95bd0d56f35241523fbab1",
					Amount: utils.NewBig10("400000000000000000000"),
				},
				TokenOut: "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8",
			},
			calcAmountOutParams2: pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{
					Token:  "0x82af49447d8a07e3bd95bd0d56f35241523fbab1",
					Amount: utils.NewBig10("400000000000000000000"),
				},
				TokenOut: "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8",
			},
			expected: expected{
				fee:          uint256.NewInt(3),
				liquidityEMA: uint256.NewInt(438171307419070),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &PoolSimulator{
				Pool:                     tt.fields.Pool,
				feePrecision:             tt.fields.feePrecision,
				dsFee:                    tt.fields.dsFee,
				dsFeeThreshold:           tt.fields.dsFeeThreshold,
				tradeLiquidityEMA:        tt.fields.tradeLiquidityEMA,
				liquidityEMA:             tt.fields.liquidityEMA,
				lastLiquidityBlockNumber: tt.fields.lastLiquidityBlockNumber,
				lastTradeLiquiditySum:    tt.fields.lastTradeLiquiditySum,
				lastTradeBlockNumber:     tt.fields.lastTradeBlockNumber,
			}

			indexIn, indexOut := s.GetTokenIndex(tt.calcAmountOutParams1.TokenAmountIn.Token), s.GetTokenIndex(tt.calcAmountOutParams1.TokenOut)
			assert.GreaterOrEqual(t, indexIn, 0)
			assert.GreaterOrEqual(t, indexOut, 0)

			actualCalcAmountOutResult1, err := s.CalcAmountOut(tt.calcAmountOutParams1)
			assert.Nil(t, err)

			s.UpdateBalance(pool.UpdateBalanceParams{
				TokenAmountIn:  tt.calcAmountOutParams1.TokenAmountIn,
				TokenAmountOut: *actualCalcAmountOutResult1.TokenAmountOut,
				SwapInfo:       actualCalcAmountOutResult1.SwapInfo,
			})

			assert.Equal(t, tt.expected.liquidityEMA.String(), s.liquidityEMA.String())

			if tt.calcAmountOutParams2 == (pool.CalcAmountOutParams{}) {
				return
			}

			actualCalcAmountOutResult2, err := s.CalcAmountOut(tt.calcAmountOutParams2)
			assert.Nil(t, err)

			s.UpdateBalance(pool.UpdateBalanceParams{
				TokenAmountIn:  tt.calcAmountOutParams2.TokenAmountIn,
				TokenAmountOut: *actualCalcAmountOutResult2.TokenAmountOut,
				SwapInfo:       actualCalcAmountOutResult2.SwapInfo,
			})
			assert.Equal(t, tt.expected.liquidityEMA.String(), s.liquidityEMA.String()) // just update once
		})
	}
}
