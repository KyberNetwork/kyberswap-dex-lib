package lo1inch

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/swaplimit"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

func TestPoolSimulator_CalcAmountOut_RealPool(t *testing.T) {
	// prepare data for test case 2
	takingAmount := new(big.Int).SetInt64(100000000 - 101)
	orderMakingAmount, _ := new(big.Int).SetString("100000000000", 10)
	orderTakingAmount, _ := new(big.Int).SetString("100247731166", 10)

	makingAmount := new(big.Int).Div(new(big.Int).Mul(takingAmount, orderMakingAmount), orderTakingAmount)
	makerBalance, _ := new(big.Int).SetString("722627607117", 10)
	makerAllowance, _ := new(big.Int).SetString("115792089237316195423570985008687907853269984665640564039457584007913129639935", 10)

	remainingMakerAmount := new(big.Int).Sub(orderMakingAmount, makingAmount)
	remainingMakerBalance := new(big.Int).Sub(makerBalance, makingAmount)
	remainingMakerAllowance := new(big.Int).Sub(makerAllowance, makingAmount)

	amountOut := new(big.Int).Add(big.NewInt(10000), makingAmount)
	// end of prepare data for test case 2

	type fields struct {
		poolEntity entity.Pool
	}
	type args struct {
		param pool.CalcAmountOutParams
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *pool.CalcAmountOutResult
		wantErr error
	}{
		{
			name: "test case 1: it should return correct result when the amountIn can be filled by 1 order",
			fields: fields{
				poolEntity: entity.Pool{
					Address:   "lo1inch_0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48_0xdac17f958d2ee523a2206206994597c13d831ec7",
					Exchange:  "lo1inch",
					Type:      "lo1inch",
					Timestamp: 1732175620,
					Reserves:  []string{"10000000000000000000", "10000000000000000000"},
					Tokens: []*entity.PoolToken{
						{
							Address:   "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
							Swappable: true,
						},
						{
							Address:   "0xdac17f958d2ee523a2206206994597c13d831ec7",
							Swappable: true,
						},
					},
					Extra: marshalPoolExtra(&Extra{
						TakeToken0Orders: []*Order{
							{
								Signature:            "0x3f31467bce6bb134944a8c3c57a8c2786ffadf31a7c39cb22a9c51cceb7e3c0f7ed91bba74a8227aae8933fa72cc8c6e3796bd4c4e734fcbe22bf5061ef9e8971c",
								OrderHash:            "0x177af74e4d3880743ac6603323a9a50f6999968e499f44966dd00d642e933285",
								CreateDateTime:       "2024-07-15T13:41:09.572Z",
								RemainingMakerAmount: big.NewInt(10000),
								MakerBalance:         big.NewInt(10437135),
								MakerAllowance:       big.NewInt(900000),
								MakerAsset:           "0xdac17f958d2ee523a2206206994597c13d831ec7",
								TakerAsset:           "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
								Salt:                 "54304030",
								Receiver:             "0x0000000000000000000000000000000000000000",
								MakingAmount:         big.NewInt(10000),
								TakingAmount:         big.NewInt(101),
								Maker:                "0xdf4039a454d58868dfd43f076ee46c92a35fdfd9",
								MakerRate:            "0.010100000000000000",
								TakerRate:            "99.009900990099009901",
							},
						},
					}),
					StaticExtra: `{"token0":"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48","token1":"0xdac17f958d2ee523a2206206994597c13d831ec7"}`,
				},
			},
			args: args{
				param: pool.CalcAmountOutParams{
					TokenAmountIn: pool.TokenAmount{
						Token:  "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
						Amount: big.NewInt(101),
					},
					TokenOut: "0xdac17f958d2ee523a2206206994597c13d831ec7",
					Limit:    nil,
				},
			},
			want: &pool.CalcAmountOutResult{
				TokenAmountOut: &pool.TokenAmount{
					Token:     "0xdac17f958d2ee523a2206206994597c13d831ec7",
					Amount:    big.NewInt(10000),
					AmountUsd: 0,
				},
				Fee: &pool.TokenAmount{
					Token:     "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
					Amount:    big.NewInt(0),
					AmountUsd: 0,
				},
				Gas: 113308,
				SwapInfo: SwapInfo{
					AmountIn: "101",
					SwapSide: SwapSideTakeToken0,
					FilledOrders: []*FilledOrderInfo{
						{
							Signature:            "0x3f31467bce6bb134944a8c3c57a8c2786ffadf31a7c39cb22a9c51cceb7e3c0f7ed91bba74a8227aae8933fa72cc8c6e3796bd4c4e734fcbe22bf5061ef9e8971c",
							OrderHash:            "0x177af74e4d3880743ac6603323a9a50f6999968e499f44966dd00d642e933285",
							CreateDateTime:       "2024-07-15T13:41:09.572Z",
							RemainingMakerAmount: new(big.Int).Sub(big.NewInt(10000), big.NewInt(10000)),
							MakerBalance:         big.NewInt(10437135 - 10000),
							MakerAllowance:       big.NewInt(900000 - 10000),
							MakerAsset:           "0xdac17f958d2ee523a2206206994597c13d831ec7",
							TakerAsset:           "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
							Salt:                 "54304030",
							Receiver:             "0x0000000000000000000000000000000000000000",
							MakingAmount:         big.NewInt(10000),
							TakingAmount:         big.NewInt(101),
							Maker:                "0xdf4039a454d58868dfd43f076ee46c92a35fdfd9",
							MakerRate:            "0.010100000000000000",
							TakerRate:            "99.009900990099009901",

							FilledMakingAmount: big.NewInt(10000),
							FilledTakingAmount: big.NewInt(101),
						},
					},
				},
			},
			wantErr: nil,
		},
		{
			name: "test case 2: it should return correct result when the amountIn can be filled by 2 orders",
			fields: fields{
				poolEntity: entity.Pool{
					Address:   "lo1inch_0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48_0xdac17f958d2ee523a2206206994597c13d831ec7",
					Exchange:  "lo1inch",
					Type:      "lo1inch",
					Timestamp: 1732175620,
					Reserves:  []string{"10000000000000000000", "10000000000000000000"},
					Tokens: []*entity.PoolToken{
						{
							Address:   "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
							Swappable: true,
						},
						{
							Address:   "0xdac17f958d2ee523a2206206994597c13d831ec7",
							Swappable: true,
						},
					},
					Extra:       `{"takeToken0Orders":[{"signature":"0x3f31467bce6bb134944a8c3c57a8c2786ffadf31a7c39cb22a9c51cceb7e3c0f7ed91bba74a8227aae8933fa72cc8c6e3796bd4c4e734fcbe22bf5061ef9e8971c","orderHash":"0x177af74e4d3880743ac6603323a9a50f6999968e499f44966dd00d642e933285","createDateTime":"2024-07-15T13:41:09.572Z","remainingMakerAmount":10000,"makerBalance":10437135,"makerAllowance":900000,"makerAsset":"0xdac17f958d2ee523a2206206994597c13d831ec7","takerAsset":"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48","salt":"54304030","receiver":"0x0000000000000000000000000000000000000000","makingAmount":10000,"takingAmount":101,"maker":"0xdf4039a454d58868dfd43f076ee46c92a35fdfd9","makerRate":"0.010100000000000000","takerRate":"99.009900990099009901"},{"signature":"0xd6a593b5bcdbe12600f09c421a769fdc2c5dd10399e71a873b7fbad1cb764b0b484452ab925d3420a9a58188b38a8b44eb91892c938ffe622c7cb6b4dd2634511b","orderHash":"0x8066b141446ead126c0aa0aeb2a1c268632be0e8d9d3ce0eace15671e06624eb","createDateTime":"2024-11-15T18:21:56.905Z","remainingMakerAmount":100000000000,"makerBalance":722627607117,"makerAllowance":115792089237316195423570985008687907853269984665640564039457584007913129639935,"makerAsset":"0xdac17f958d2ee523a2206206994597c13d831ec7","takerAsset":"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48","salt":"67123001626078665156821660044248014421358635920891485277615109939076199471724","receiver":"0x0000000000000000000000000000000000000000","makingAmount":100000000000,"takingAmount":100247731166,"maker":"0x29eba388141f070e6824dd7628f11cb946bc548b","makerRate":"1.002477311660000000","takerRate":"0.997528810247188712"}]}`,
					StaticExtra: `{"token0":"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48","token1":"0xdac17f958d2ee523a2206206994597c13d831ec7"}`,
				},
			},
			args: args{
				param: pool.CalcAmountOutParams{
					TokenAmountIn: pool.TokenAmount{
						Token:  "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
						Amount: big.NewInt(100000000),
					},
					TokenOut: "0xdac17f958d2ee523a2206206994597c13d831ec7",
					Limit:    nil,
				},
			},
			want: &pool.CalcAmountOutResult{
				TokenAmountOut: &pool.TokenAmount{
					Token:     "0xdac17f958d2ee523a2206206994597c13d831ec7",
					Amount:    amountOut,
					AmountUsd: 0,
				},
				Fee: &pool.TokenAmount{
					Token:     "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
					Amount:    big.NewInt(0),
					AmountUsd: 0,
				},
				Gas: 136616,
				SwapInfo: SwapInfo{
					AmountIn: "100000000",
					SwapSide: SwapSideTakeToken0,
					FilledOrders: []*FilledOrderInfo{
						{
							Signature:            "0x3f31467bce6bb134944a8c3c57a8c2786ffadf31a7c39cb22a9c51cceb7e3c0f7ed91bba74a8227aae8933fa72cc8c6e3796bd4c4e734fcbe22bf5061ef9e8971c",
							OrderHash:            "0x177af74e4d3880743ac6603323a9a50f6999968e499f44966dd00d642e933285",
							CreateDateTime:       "2024-07-15T13:41:09.572Z",
							RemainingMakerAmount: new(big.Int).Sub(big.NewInt(10000), big.NewInt(10000)),
							MakerBalance:         big.NewInt(10437135 - 10000),
							MakerAllowance:       big.NewInt(900000 - 10000),
							MakerAsset:           "0xdac17f958d2ee523a2206206994597c13d831ec7",
							TakerAsset:           "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
							Salt:                 "54304030",
							Receiver:             "0x0000000000000000000000000000000000000000",
							MakingAmount:         big.NewInt(10000),
							TakingAmount:         big.NewInt(101),
							Maker:                "0xdf4039a454d58868dfd43f076ee46c92a35fdfd9",
							MakerRate:            "0.010100000000000000",
							TakerRate:            "99.009900990099009901",

							FilledMakingAmount: big.NewInt(10000),
							FilledTakingAmount: big.NewInt(101),
						},
						{
							Signature:            "0xd6a593b5bcdbe12600f09c421a769fdc2c5dd10399e71a873b7fbad1cb764b0b484452ab925d3420a9a58188b38a8b44eb91892c938ffe622c7cb6b4dd2634511b",
							OrderHash:            "0x8066b141446ead126c0aa0aeb2a1c268632be0e8d9d3ce0eace15671e06624eb",
							CreateDateTime:       "2024-11-15T18:21:56.905Z",
							RemainingMakerAmount: remainingMakerAmount,
							MakerBalance:         remainingMakerBalance,
							MakerAllowance:       remainingMakerAllowance,
							MakerAsset:           "0xdac17f958d2ee523a2206206994597c13d831ec7",
							TakerAsset:           "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
							Salt:                 "67123001626078665156821660044248014421358635920891485277615109939076199471724",
							Receiver:             "0x0000000000000000000000000000000000000000",
							MakingAmount:         big.NewInt(100000000000),
							TakingAmount:         big.NewInt(100247731166),
							Maker:                "0x29eba388141f070e6824dd7628f11cb946bc548b",
							MakerRate:            "1.002477311660000000",
							TakerRate:            "0.997528810247188712",

							FilledMakingAmount: makingAmount,
							FilledTakingAmount: takingAmount,
						},
					},
				},
			},
			wantErr: nil,
		},
		{
			name: "test case 3: it should return error ErrTokenInNotSupported when tokenIn is not token0 or token1",
			fields: fields{
				poolEntity: entity.Pool{
					Address:   "lo1inch_0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48_0xdac17f958d2ee523a2206206994597c13d831ec7",
					Exchange:  "lo1inch",
					Type:      "lo1inch",
					Timestamp: 1732175620,
					Reserves:  []string{"10000000000000000000", "10000000000000000000"},
					Tokens: []*entity.PoolToken{
						{
							Address:   "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
							Swappable: true,
						},
						{
							Address:   "0xdac17f958d2ee523a2206206994597c13d831ec7",
							Swappable: true,
						},
					},
					Extra:       "{}",
					StaticExtra: `{"token0":"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48","token1":"0xdac17f958d2ee523a2206206994597c13d831ec7"}`,
				},
			},
			args: args{
				param: pool.CalcAmountOutParams{
					TokenAmountIn: pool.TokenAmount{
						Token:  "0x6b175474e89094c44da98b954eedeac495271d0f", // this token is not in pool
						Amount: big.NewInt(1000000000000000000),
					},
					TokenOut: "0xdac17f958d2ee523a2206206994597c13d831ec7",
					Limit:    nil,
				},
			},
			want:    nil,
			wantErr: ErrTokenInNotSupported,
		},
		{
			name: "test case 4: it should return error ErrCannotFulfillAmountIn when this pool can not fulfill amountIn",
			fields: fields{
				poolEntity: entity.Pool{
					Address:   "lo1inch_0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48_0xdac17f958d2ee523a2206206994597c13d831ec7",
					Exchange:  "lo1inch",
					Type:      "lo1inch",
					Timestamp: 1732175620,
					Reserves:  []string{"10000000000000000000", "10000000000000000000"},
					Tokens: []*entity.PoolToken{
						{
							Address:   "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
							Swappable: true,
						},
						{
							Address:   "0xdac17f958d2ee523a2206206994597c13d831ec7",
							Swappable: true,
						},
					},
					Extra: marshalPoolExtra(&Extra{
						TakeToken0Orders: []*Order{
							{
								Signature:            "0x3f31467bce6bb134944a8c3c57a8c2786ffadf31a7c39cb22a9c51cceb7e3c0f7ed91bba74a8227aae8933fa72cc8c6e3796bd4c4e734fcbe22bf5061ef9e8971c",
								OrderHash:            "0x177af74e4d3880743ac6603323a9a50f6999968e499f44966dd00d642e933285",
								CreateDateTime:       "2024-07-15T13:41:09.572Z",
								RemainingMakerAmount: big.NewInt(10000),
								MakerBalance:         big.NewInt(10437135),
								MakerAllowance:       big.NewInt(900000),
								MakerAsset:           "0xdac17f958d2ee523a2206206994597c13d831ec7",
								TakerAsset:           "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
								Salt:                 "54304030",
								Receiver:             "0x0000000000000000000000000000000000000000",
								MakingAmount:         big.NewInt(10000),
								TakingAmount:         big.NewInt(101),
								Maker:                "0xdf4039a454d58868dfd43f076ee46c92a35fdfd9",
								MakerRate:            "0.010100000000000000",
								TakerRate:            "99.009900990099009901",
							},
						},
					}),
					StaticExtra: `{"token0":"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48","token1":"0xdac17f958d2ee523a2206206994597c13d831ec7"}`,
				},
			},
			args: args{
				param: pool.CalcAmountOutParams{
					TokenAmountIn: pool.TokenAmount{
						Token:  "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
						Amount: big.NewInt(102),
					},
					TokenOut: "0xdac17f958d2ee523a2206206994597c13d831ec7",
					Limit:    nil,
				},
			},
			want:    nil,
			wantErr: ErrCannotFulfillAmountIn,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := NewPoolSimulator(tt.fields.poolEntity)
			assert.NoError(t, err)

			got, err := p.CalcAmountOut(tt.args.param)
			if err != nil {
				assert.Equalf(t, tt.wantErr, err, "PoolSimulator.CalcAmountOut() error = %+v, wantErr %+v", err, tt.wantErr)
				return
			}

			assert.Equalf(t, tt.want, got, "PoolSimulator.CalcAmountOut() = %+v, want %+v", got, tt.want)
		})
	}
}

func marshalPoolExtra(extra *Extra) string {
	bytesData, _ := json.Marshal(extra)
	return string(bytesData)
}

func TestPoolSimulator_CalcAmountOut(t *testing.T) {
	type testorder struct {
		hash                 string
		makingAmount         string
		takingAmount         string
		remainingMakerAmount string
	}

	pools := map[string][]testorder{
		"pool1": {
			{"1001", "100", "1000", "100"},
			{"1002", "100", "2000", "50"},
		},
		"pool2": {
			{"1001", "100", "1000", "100"},
			{"1002", "100", "2000", "50"},
			{"1003", "100", "2000", "100"},
		},
		"pool3": {
			{"1001", "39", "777000000000000000000", "39"},
		},
		"pool4": {
			{"1001", "39", "777000000000000000000", "39"},
			{"1002", "390000", "777000000000000000", "390000"},
		},
	}

	testcases := []struct {
		name           string
		pool           string
		amountIn       string
		expAmountOut   string
		expOrderHashes []string
	}{
		{"fully fill 1st order", "pool1", "100", "10", []string{"1001"}},
		{"f-fill 1st order, p-fill 2nd one", "pool1", "1100", "105", []string{"1001", "1002"}},
		{"f-fill both orders", "pool1", "2000", "150", []string{"1001", "1002"}}, // reach 1002's avaiFillAmount (50)
		{"cannot be filled", "pool1", "2001", "", nil},                           // cannot exceed 1002's avaiFillAmount (50)

		{"f-fill 1st order, p-fill 2nd one", "pool2", "1100", "105", []string{"1001", "1002"}},
		{"f-fill 1st/2nd order", "pool2", "2000", "150", []string{"1001", "1002", "1003"}}, // include 1003 with fill=0 as fallback
		{"f-fill 1st/2nd order, p-fill 3rd one", "pool2", "2100", "155", []string{"1001", "1002", "1003"}},
		{"f-fill all order", "pool2", "4000", "250", []string{"1001", "1002", "1003"}},
		{"cannot be filled", "pool1", "4001", "", nil},

		{"cannot be filled (too small, round to 0)", "pool3", "5874584652643", "", nil},

		{"skip 1st order (too small, round to 0) and use 2nd one", "pool4", "5874584652643", "2", []string{"1002"}},
	}

	sims := lo.MapValues(pools, func(orders []testorder, _ string) *PoolSimulator {
		extra := Extra{
			TakeToken0Orders: lo.Map(orders, func(o testorder, _ int) *Order {
				return &Order{
					OrderHash:            o.hash,
					MakingAmount:         bignumber.NewBig10(o.makingAmount),
					TakingAmount:         bignumber.NewBig10(o.takingAmount),
					RemainingMakerAmount: bignumber.NewBig10(o.remainingMakerAmount),
					MakerBalance:         bignumber.NewBig10("100000000000000000000"),
					MakerAllowance:       bignumber.NewBig10("100000000000000000000"),
				}
			}),
		}
		sExtra, _ := json.Marshal(extra)
		poolEnt := entity.Pool{
			Tokens:      []*entity.PoolToken{{Address: "A"}, {Address: "B"}},
			Reserves:    entity.PoolReserves{"0", "0"},
			StaticExtra: `{"token0":"A","token1":"B"}`,
			Extra:       string(sExtra),
		}
		p, err := NewPoolSimulator(poolEnt)
		require.Nil(t, err)
		return p
	})

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			limit := swaplimit.NewInventory("", sims[tc.pool].CalculateLimit())
			res, err := sims[tc.pool].CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{
					Token:  "A",
					Amount: bignumber.NewBig10(tc.amountIn),
				},
				TokenOut: "B",
				Limit:    limit,
			})

			if tc.expOrderHashes == nil {
				require.NotNil(t, err)
				return
			}

			require.Nil(t, err)

			assert.Equal(t, tc.expAmountOut, res.TokenAmountOut.Amount.String())

			si := res.SwapInfo.(SwapInfo)
			orderHashes := make([]string, 0, len(si.FilledOrders))
			orderInfo := ""
			for _, o := range si.FilledOrders {
				orderHashes = append(orderHashes, o.OrderHash)
				orderInfo += fmt.Sprintf("order %v %v\n", o.OrderHash, o.RemainingMakerAmount)
			}
			assert.Equal(t, tc.expOrderHashes, orderHashes, orderInfo)
			fmt.Println(orderInfo)
		})
	}
}

func TestPoolSimulator_UpdateBalance(t *testing.T) {
	type testorder struct {
		hash                 string
		makingAmount         string
		takingAmount         string
		remainingMakerAmount string
	}

	type testswap struct {
		amountIn       string
		expAmountOut   string
		expOrderHashes []string
	}

	pools := map[string][]testorder{
		"pool1": {
			{"1001", "100", "1000", "100"},
			{"1002", "100", "2000", "50"},
		},
	}

	testcases := []struct {
		name string
		pool string

		swaps []testswap
	}{
		{"case 1", "pool1", []testswap{
			{"1000", "100", []string{"1001", "1002"}}, // 1st swap full fill 1001 (1002 included as backup)
			{"500", "25", []string{"1002"}},           // after update balance, 1001 has been fully used , can only use 1002
			{"600", "", nil},                          // only 25 makingAmount left (500 takingAmount), so this swap will fail
		}},
		{"case 2", "pool1", []testswap{
			{"1000", "100", []string{"1001", "1002"}}, // 1st swap full fill 1001 (1002 included as backup)
			{"500", "25", []string{"1002"}},           // after update balance, 1001 has been fully used , can only use 1002
			{"300", "15", []string{"1002"}},           // still use 1002
			{"200", "10", []string{"1002"}},           // still use 1002
		}},
	}

	var sims map[string]*PoolSimulator
	resetSim := func() {
		sims = lo.MapValues(pools, func(orders []testorder, _ string) *PoolSimulator {
			extra := Extra{
				TakeToken0Orders: lo.Map(orders, func(o testorder, _ int) *Order {
					return &Order{
						OrderHash:            o.hash,
						MakingAmount:         bignumber.NewBig10(o.makingAmount),
						TakingAmount:         bignumber.NewBig10(o.takingAmount),
						RemainingMakerAmount: bignumber.NewBig10(o.remainingMakerAmount),
						MakerBalance:         bignumber.NewBig10("100000000000000000000"),
						MakerAllowance:       bignumber.NewBig10("100000000000000000000"),
					}
				}),
			}
			sExtra, _ := json.Marshal(extra)
			poolEnt := entity.Pool{
				Tokens:      []*entity.PoolToken{{Address: "A"}, {Address: "B"}},
				Reserves:    entity.PoolReserves{"0", "0"},
				StaticExtra: `{"token0":"A","token1":"B"}`,
				Extra:       string(sExtra),
			}
			p, err := NewPoolSimulator(poolEnt)
			require.Nil(t, err)
			return p
		})
	}

	for _, tc := range testcases {
		resetSim()
		limit := swaplimit.NewInventory("", sims[tc.pool].CalculateLimit())
		for i, swap := range tc.swaps {
			t.Run(fmt.Sprintf("%v swap %d", tc.name, i), func(t *testing.T) {
				res, err := sims[tc.pool].CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: pool.TokenAmount{
						Token:  "A",
						Amount: bignumber.NewBig10(swap.amountIn),
					},
					TokenOut: "B",
					Limit:    limit,
				})

				if swap.expOrderHashes == nil {
					require.NotNil(t, err)
					return
				}

				require.Nil(t, err)

				assert.Equal(t, swap.expAmountOut, res.TokenAmountOut.Amount.String())

				si := res.SwapInfo.(SwapInfo)
				oid := make([]string, 0, len(si.FilledOrders))
				oinfo := ""
				for _, o := range si.FilledOrders {
					oid = append(oid, o.OrderHash)
					oinfo += fmt.Sprintf("order %v %v\n", o.OrderHash, o.RemainingMakerAmount)
				}
				assert.Equal(t, swap.expOrderHashes, oid, oinfo)
				fmt.Println(oinfo)

				sims[tc.pool].UpdateBalance(pool.UpdateBalanceParams{
					TokenAmountIn: pool.TokenAmount{
						Token:  "A",
						Amount: bignumber.NewBig10(swap.amountIn),
					},
					TokenAmountOut: *res.TokenAmountOut,
					Fee:            *res.Fee,
					SwapInfo:       res.SwapInfo,
					SwapLimit:      limit,
				})
			})
		}
	}
}

func TestPoolSimulator_Inventory(t *testing.T) {
	type testorder struct {
		hash                 string
		maker                string
		makingAmount         string
		takingAmount         string
		remainingMakerAmount string
	}

	type testswap struct {
		amountIn       string
		expAmountOut   string
		expOrderHashes []string
	}

	pools := map[string][]testorder{
		"pool1": {
			{"1001", "maker1", "100", "1000", "100"},
			{"1002", "maker1", "100", "2000", "100"},
			{"1003", "maker1", "100", "4000", "50"},
			{"1004", "maker2", "100", "5000", "100"},
		},
	}

	makerBalances := map[string]*big.Int{
		"maker1": bignumber.NewBig10("250"),
		"maker2": bignumber.NewBig10("100"),
	}
	minBalanceAllowanceByMakerAndAsset := map[makerAndAsset]*big.Int{
		newMakerAndAsset("maker1", "B"): bignumber.NewBig10("150"), // maker1 original balance is 250, but 100 has been spent in previous paths (order with same makerAsset but different takerAsset)
		newMakerAndAsset("maker2", "B"): bignumber.NewBig10("50"),  // same, maker2 has spent 50 already
	}

	testcases := []struct {
		name string
		pool string

		swaps []testswap
	}{
		{"case 1", "pool1", []testswap{
			{"1000", "100", []string{"1001", "1002"}}, // 1st swap full fill 1001 (1002 included as backup)
			{"500", "25", []string{"1002"}},           // after update balance, 1001 has been fully used , can only use 1002
			{"500", "25", []string{"1002", "1004"}},   // maker1 has 25 makingAmount left, so will fully filled the rest of 1002 (1004 included as backup instead of 1003 because 1003 is from maker1 who has no balance left)
			{"500", "10", []string{"1004"}},           // only 1004 is available now
		}},

		{"case 2", "pool1", []testswap{
			{"1000", "100", []string{"1001", "1002"}}, // 1st swap full fill 1001 (1002 included as backup)
			{"500", "25", []string{"1002"}},           // after update balance, 1001 has been fully used , can only use 1002
			{"600", "27", []string{"1002", "1004"}},   // 500-25 from 1002 and 100-2 from 1004
		}},

		{"case 3", "pool1", []testswap{
			{"1000", "100", []string{"1001", "1002"}}, // 1st swap full fill 1001 (1002 included as backup)
			{"500", "25", []string{"1002"}},           // after update balance, 1001 has been fully used , can only use 1002
			{"3000", "75", []string{"1002", "1004"}},  // 500-25 from 1002 and 2500-50 from 1004 (fully filled)
		}},

		{"case 4", "pool1", []testswap{
			{"1000", "100", []string{"1001", "1002"}}, // 1st swap full fill 1001 (1002 included as backup)
			{"500", "25", []string{"1002"}},           // after update balance, 1001 has been fully used , can only use 1002
			{"3001", "", nil},                         // a bit larger than case 3 above, so no balance left and swap fail
		}},
	}

	var sims map[string]*PoolSimulator
	resetSim := func() {
		sims = lo.MapValues(pools, func(orders []testorder, _ string) *PoolSimulator {
			extra := Extra{
				TakeToken0Orders: lo.Map(orders, func(o testorder, _ int) *Order {
					return &Order{
						OrderHash:            o.hash,
						Maker:                o.maker,
						MakerAsset:           "B",
						TakerAsset:           "A",
						MakingAmount:         bignumber.NewBig10(o.makingAmount),
						TakingAmount:         bignumber.NewBig10(o.takingAmount),
						RemainingMakerAmount: bignumber.NewBig10(o.remainingMakerAmount),
						MakerBalance:         makerBalances[o.maker],
						MakerAllowance:       makerBalances[o.maker],
					}
				}),
			}
			sExtra, _ := json.Marshal(extra)
			poolEnt := entity.Pool{
				Tokens:      []*entity.PoolToken{{Address: "A"}, {Address: "B"}},
				Reserves:    entity.PoolReserves{"0", "0"},
				StaticExtra: `{"token0":"A","token1":"B"}`,
				Extra:       string(sExtra),
			}
			p, err := NewPoolSimulator(poolEnt)
			require.Nil(t, err)
			// fake spent balance
			p.minBalanceAllowanceByMakerAndAsset = minBalanceAllowanceByMakerAndAsset
			return p
		})
	}

	for _, tc := range testcases {
		resetSim()
		limit := swaplimit.NewInventory("", sims[tc.pool].CalculateLimit())
		for i, swap := range tc.swaps {
			t.Run(fmt.Sprintf("%v swap %d", tc.name, i), func(t *testing.T) {
				res, err := sims[tc.pool].CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: pool.TokenAmount{
						Token:  "A",
						Amount: bignumber.NewBig10(swap.amountIn),
					},
					TokenOut: "B",
					Limit:    limit,
				})

				if len(swap.expOrderHashes) == 0 {
					require.NotNil(t, err)
					return
				}

				require.Nil(t, err)

				assert.Equal(t, swap.expAmountOut, res.TokenAmountOut.Amount.String())

				si := res.SwapInfo.(SwapInfo)
				oid := make([]string, 0, len(si.FilledOrders))
				oinfo := ""
				for _, o := range si.FilledOrders {
					oid = append(oid, o.OrderHash)
					oinfo += fmt.Sprintf("order %v %v\n", o.OrderHash, o.RemainingMakerAmount)
				}
				assert.Equal(t, swap.expOrderHashes, oid, oinfo)
				fmt.Println(oinfo)

				sims[tc.pool].UpdateBalance(pool.UpdateBalanceParams{
					TokenAmountIn: pool.TokenAmount{
						Token:  "A",
						Amount: bignumber.NewBig10(swap.amountIn),
					},
					TokenAmountOut: *res.TokenAmountOut,
					Fee:            *res.Fee,
					SwapInfo:       res.SwapInfo,
					SwapLimit:      limit,
				})
			})
		}
	}
}
