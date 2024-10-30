package deltaswapv1

import (
	"math/big"
	"reflect"
	"strings"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	utils "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
)

func Test_calcEMA(t *testing.T) {
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := calcSingleSideLiquidity(tt.args.amount, tt.args.reserve0, tt.args.reserve1); !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("calcSingleSideLiquidity() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestPoolSimulator_CalcAmountOut(t *testing.T) {
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
	tests := []struct {
		name        string
		fields      fields
		params      pool.CalcAmountOutParams
		expected    *poolpkg.CalcAmountOutResult
		expectedErr bool
	}{
		{
			name: "Test 1",
			fields: fields{
				Pool: pool.Pool{Info: pool.PoolInfo{
					Address:  strings.ToLower("0x7c1c3f25686e992fc9a1a2069f877189b0b28990"),
					Tokens:   []string{"0x249c48e22e95514ca975de31f473f30c2f3c0916", "0xaf88d065e77c8cc2239327c5edb3a432268e5831"},
					Reserves: []*big.Int{utils.NewBig10("100000000000000000000"), utils.NewBig10("100000000000000000000")},
				}},
				feePrecision:             Number_1000,
				dsFee:                    uint256.NewInt(3),
				dsFeeThreshold:           uint256.NewInt(0),
				tradeLiquidityEMA:        uint256.NewInt(722321141048),
				liquidityEMA:             uint256.NewInt(113426914987188),
				lastLiquidityBlockNumber: 21074172,
				lastTradeLiquiditySum:    uint256.NewInt(722321141048),
				lastTradeBlockNumber:     21074172,
			},
			params: pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{
					Token:  "0x249c48e22e95514ca975de31f473f30c2f3c0916",
					Amount: utils.NewBig10("1000000000000000000"),
				},
				TokenOut: "0xaf88d065e77c8cc2239327c5edb3a432268e5831",
			},
			expected: &poolpkg.CalcAmountOutResult{
				TokenAmountOut: &poolpkg.TokenAmount{
					Token:  "0xaf88d065e77c8cc2239327c5edb3a432268e5831",
					Amount: utils.NewBig10("987158034397061298"),
				},
			},
			expectedErr: false,
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
			actual, err := s.CalcAmountOut(tt.params)
			if (err != nil) != tt.expectedErr {
				t.Errorf("CalcAmountOut() error = %v, expectedErr %v", err, tt.expectedErr)
				return
			}

			assert.EqualValues(t, actual.TokenAmountOut.Amount.String(), tt.expected.TokenAmountOut.Amount.String())
		})
	}
}
