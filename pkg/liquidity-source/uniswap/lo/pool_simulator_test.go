package uniswaplo

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/blockchain-toolkit/integer"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

func TestPoolSimulator_CalcAmountOut_RealPool(t *testing.T) {
	// Create test DutchOrder for test case 2
	makingAmount := uint256.MustFromDecimal("100000000000")
	takingAmount := uint256.MustFromDecimal("100247731166")

	// Create a sample DutchOrder
	// dutchOrder := &DutchOrder{
	// 	OrderHash:   "0x3ab75c59d2c3ae497ae22523440c0e172039d86a38a4cbf40fa8b3620b44e18c",
	// 	Type:        "Dutch",
	// 	OrderStatus: OpenOrderStatus,
	// 	Swapper:     common.HexToAddress("0x687ed137c2cb1fa58db8423797b505a9de037b1e"),
	// 	ChainID:     1,
	// 	Input: Input{
	// 		Token:       common.HexToAddress("0x6c3ea9036406852006290770BEdFcAbA0e23A0e8"),
	// 		StartAmount: uint256.MustFromDecimal("1000000"),
	// 		EndAmount:   uint256.MustFromDecimal("1000000"),
	// 	},
	// 	Outputs: []Output{
	// 		{
	// 			Token:       common.HexToAddress("0x6B175474E89094C44Da98b954EedeAC495271d0F"),
	// 			StartAmount: uint256.MustFromDecimal("998770000000000000"),
	// 			EndAmount:   uint256.MustFromDecimal("998770000000000000"),
	// 			Recipient:   common.HexToAddress("0x687ed137c2cb1fa58db8423797b505a9de037b1e"),
	// 		},
	// 	},
	// 	CreatedAt: 1708630693,
	// }

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
					Address:   "uniswap-lo_0xa0b=86991c6218b36c1d19d4a2e9eb0ce3606eb48_0xdac17f958d2ee523a2206206994597c13d831ec7",
					Exchange:  "uniswap-lo",
					Type:      "uniswap-lo",
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
						TakeToken0Orders: []*DutchOrder{
							{
								OrderHash:   "0x177af74e4d3880743ac6603323a9a50f6999968e499f44966dd00d642e933285",
								Swapper:     common.HexToAddress("0xdf4039a454d58868dfd43f076ee46c92a35fdfd9"),
								Type:        "Dutch",
								OrderStatus: OpenOrderStatus,
								Input: Input{
									Token:       common.HexToAddress("0xdac17f958d2ee523a2206206994597c13d831ec7"),
									StartAmount: uint256.NewInt(10000),
									EndAmount:   uint256.NewInt(10000),
								},
								Outputs: []Output{
									{
										Token:       common.HexToAddress("0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"),
										StartAmount: uint256.NewInt(101),
										EndAmount:   uint256.NewInt(101),
										Recipient:   common.HexToAddress("0x0000000000000000000000000000000000000000"),
									},
								},
							},
						},
					}),
					StaticExtra: `{"token0":"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48","token1":"0xdac17f958d2ee523a2206206994597c13d831ec7","reactorAddress":"0x1111111111111111111111111111111111111111"}`,
				},
			},
			args: args{
				param: pool.CalcAmountOutParams{
					TokenAmountIn: pool.TokenAmount{
						Token:  "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
						Amount: uint256.NewInt(101).ToBig(),
					},
					TokenOut: "0xdac17f958d2ee523a2206206994597c13d831ec7",
					Limit:    nil,
				},
			},
			want: &pool.CalcAmountOutResult{
				TokenAmountOut: &pool.TokenAmount{
					Token:     "0xdac17f958d2ee523a2206206994597c13d831ec7",
					Amount:    uint256.NewInt(10000).ToBig(),
					AmountUsd: 0,
				},
				Fee: &pool.TokenAmount{
					Token:     "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
					Amount:    integer.Zero(),
					AmountUsd: 0,
				},
				Gas: 300000,
				SwapInfo: SwapInfo{
					AmountIn: "101",
					SwapSide: SwapSideTakeToken0,
					FilledOrders: []*DutchOrder{
						{
							OrderHash:   "0x177af74e4d3880743ac6603323a9a50f6999968e499f44966dd00d642e933285",
							Swapper:     common.HexToAddress("0xdf4039a454d58868dfd43f076ee46c92a35fdfd9"),
							Type:        "Dutch",
							OrderStatus: OpenOrderStatus,
							Input: Input{
								Token:       common.HexToAddress("0xdac17f958d2ee523a2206206994597c13d831ec7"),
								StartAmount: uint256.NewInt(10000),
								EndAmount:   uint256.NewInt(10000),
							},
							Outputs: []Output{
								{
									Token:       common.HexToAddress("0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"),
									StartAmount: uint256.NewInt(101),
									EndAmount:   uint256.NewInt(101),
									Recipient:   common.HexToAddress("0x0000000000000000000000000000000000000000"),
								},
							},
						},
					},
					IsAmountInFulfilled: true,
				},
				RemainingTokenAmountIn: &pool.TokenAmount{
					Token:  "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
					Amount: big.NewInt(0),
				},
			},
			wantErr: nil,
		},
		{
			name: "test case 2: it should return correct result when the amountIn can be filled by 2 orders",
			fields: fields{
				poolEntity: entity.Pool{
					Address:   "uniswap-lo_0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48_0xdac17f958d2ee523a2206206994597c13d831ec7",
					Exchange:  "uniswap-lo",
					Type:      "uniswap-lo",
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
						TakeToken0Orders: []*DutchOrder{
							{
								OrderHash:   "0x177af74e4d3880743ac6603323a9a50f6999968e499f44966dd00d642e933285",
								Swapper:     common.HexToAddress("0xdf4039a454d58868dfd43f076ee46c92a35fdfd9"),
								Type:        "Dutch",
								OrderStatus: OpenOrderStatus,
								Input: Input{
									Token:       common.HexToAddress("0xdac17f958d2ee523a2206206994597c13d831ec7"),
									StartAmount: uint256.NewInt(10000),
									EndAmount:   uint256.NewInt(10000),
								},
								Outputs: []Output{
									{
										Token:       common.HexToAddress("0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"),
										StartAmount: uint256.NewInt(101),
										EndAmount:   uint256.NewInt(101),
										Recipient:   common.HexToAddress("0x0000000000000000000000000000000000000000"),
									},
								},
							},
							{
								OrderHash:   "0x8066b141446ead126c0aa0aeb2a1c268632be0e8d9d3ce0eace15671e06624eb",
								Swapper:     common.HexToAddress("0x29eba388141f070e6824dd7628f11cb946bc548b"),
								Type:        "Dutch",
								OrderStatus: OpenOrderStatus,
								Input: Input{
									Token:       common.HexToAddress("0xdac17f958d2ee523a2206206994597c13d831ec7"),
									StartAmount: makingAmount,
									EndAmount:   makingAmount,
								},
								Outputs: []Output{
									{
										Token:       common.HexToAddress("0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"),
										StartAmount: takingAmount,
										EndAmount:   takingAmount,
										Recipient:   common.HexToAddress("0x0000000000000000000000000000000000000000"),
									},
								},
							},
						},
					}),
					StaticExtra: `{"token0":"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48","token1":"0xdac17f958d2ee523a2206206994597c13d831ec7","reactorAddress":"0x1111111111111111111111111111111111111111"}`,
				},
			},
			args: args{
				param: pool.CalcAmountOutParams{
					TokenAmountIn: pool.TokenAmount{
						Token:  "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
						Amount: uint256.NewInt(101).ToBig(),
					},
					TokenOut: "0xdac17f958d2ee523a2206206994597c13d831ec7",
					Limit:    nil,
				},
			},
			want: &pool.CalcAmountOutResult{
				TokenAmountOut: &pool.TokenAmount{
					Token:     "0xdac17f958d2ee523a2206206994597c13d831ec7",
					Amount:    uint256.NewInt(10000).ToBig(),
					AmountUsd: 0,
				},
				Fee: &pool.TokenAmount{
					Token:     "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
					Amount:    integer.Zero(),
					AmountUsd: 0,
				},
				Gas: 300000,
				SwapInfo: SwapInfo{
					AmountIn: "101",
					SwapSide: SwapSideTakeToken0,
					FilledOrders: []*DutchOrder{
						{
							OrderHash:   "0x177af74e4d3880743ac6603323a9a50f6999968e499f44966dd00d642e933285",
							Swapper:     common.HexToAddress("0xdf4039a454d58868dfd43f076ee46c92a35fdfd9"),
							Type:        "Dutch",
							OrderStatus: OpenOrderStatus,
							Input: Input{
								Token:       common.HexToAddress("0xdac17f958d2ee523a2206206994597c13d831ec7"),
								StartAmount: uint256.NewInt(10000),
								EndAmount:   uint256.NewInt(10000),
							},
							Outputs: []Output{
								{
									Token:       common.HexToAddress("0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"),
									StartAmount: uint256.NewInt(101),
									EndAmount:   uint256.NewInt(101),
									Recipient:   common.HexToAddress("0x0000000000000000000000000000000000000000"),
								},
							},
						},
					},
					IsAmountInFulfilled: true,
				},
				RemainingTokenAmountIn: &pool.TokenAmount{
					Token:  "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
					Amount: big.NewInt(0),
				},
			},
			wantErr: nil,
		},
		{
			name: "test case 3: it should return error ErrTokenInNotSupported when tokenIn is not token0 or token1",
			fields: fields{
				poolEntity: entity.Pool{
					Address:   "uniswap-lo_0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48_0xdac17f958d2ee523a2206206994597c13d831ec7",
					Exchange:  "uniswap-lo",
					Type:      "uniswap-lo",
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
					StaticExtra: `{"token0":"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48","token1":"0xdac17f958d2ee523a2206206994597c13d831ec7","reactorAddress":"0x1111111111111111111111111111111111111111"}`,
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
			name: "test case 4: it should return error ErrNoOrderAvailable when no orders are available",
			fields: fields{
				poolEntity: entity.Pool{
					Address:   "uniswap-lo_0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48_0xdac17f958d2ee523a2206206994597c13d831ec7",
					Exchange:  "uniswap-lo",
					Type:      "uniswap-lo",
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
					StaticExtra: `{"token0":"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48","token1":"0xdac17f958d2ee523a2206206994597c13d831ec7","reactorAddress":"0x1111111111111111111111111111111111111111"}`,
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
			want:    nil,
			wantErr: ErrNoOrderAvailable,
		},
		{
			name: "test case 5: it should return error ErrCannotFulfillAmountIn when amount cannot be fulfilled",
			fields: fields{
				poolEntity: entity.Pool{
					Address:   "uniswap-lo_0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48_0xdac17f958d2ee523a2206206994597c13d831ec7",
					Exchange:  "uniswap-lo",
					Type:      "uniswap-lo",
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
						TakeToken0Orders: []*DutchOrder{
							{
								OrderHash:   "0x177af74e4d3880743ac6603323a9a50f6999968e499f44966dd00d642e933285",
								Swapper:     common.HexToAddress("0xdf4039a454d58868dfd43f076ee46c92a35fdfd9"),
								Type:        "Dutch",
								OrderStatus: OpenOrderStatus,
								Input: Input{
									Token:       common.HexToAddress("0xdac17f958d2ee523a2206206994597c13d831ec7"),
									StartAmount: uint256.NewInt(10000),
									EndAmount:   uint256.NewInt(10000),
								},
								Outputs: []Output{
									{
										Token:       common.HexToAddress("0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"),
										StartAmount: uint256.NewInt(101),
										EndAmount:   uint256.NewInt(101),
										Recipient:   common.HexToAddress("0x0000000000000000000000000000000000000000"),
									},
								},
							},
						},
					}),
					StaticExtra: `{"token0":"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48","token1":"0xdac17f958d2ee523a2206206994597c13d831ec7","reactorAddress":"0x1111111111111111111111111111111111111111"}`,
				},
			},
			args: args{
				param: pool.CalcAmountOutParams{
					TokenAmountIn: pool.TokenAmount{
						Token:  "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
						Amount: big.NewInt(100), // More than what the order can handle
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
			// t.Log(got.TokenAmountOut.Amount, got.SwapInfo)
			// t.Log(tt.want.TokenAmountOut.Amount, tt.want.SwapInfo)
			gotJSON, _ := json.Marshal(got)
			wantJSON, _ := json.Marshal(tt.want)
			assert.JSONEq(t, string(wantJSON), string(gotJSON), "PoolSimulator.CalcAmountOut() results don't match")
		})
	}
}

func marshalPoolExtra(extra *Extra) string {
	bytesData, _ := json.Marshal(extra)
	return string(bytesData)
}

func TestPoolSimulator_CalcAmountOut(t *testing.T) {
	type testorder struct {
		hash         string
		makingToken  string
		takingToken  string
		makingAmount string
		takingAmount string
	}

	pools := map[string][]testorder{
		"pool1": {
			{"1001", "0xdac17f958d2ee523a2206206994597c13d831ec7", "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "100", "1000"},
			{"1002", "0xdac17f958d2ee523a2206206994597c13d831ec7", "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "100", "2000"},
		},
		"pool2": {
			{"1001", "0xdac17f958d2ee523a2206206994597c13d831ec7", "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "100", "1000"},
			{"1002", "0xdac17f958d2ee523a2206206994597c13d831ec7", "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "100", "2000"},
			{"1003", "0xdac17f958d2ee523a2206206994597c13d831ec7", "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "100", "2000"},
		},
		// Pool for testing partial fulfillment with remaining amount
		"pool3": {
			{"3001", "0xdac17f958d2ee523a2206206994597c13d831ec7", "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "100", "1000"},
		},
	}

	testcases := []struct {
		name                    string
		pool                    string
		amountIn                string
		tokenIn                 string
		tokenOut                string
		expAmountOut            string
		expOrderHashes          []string
		expRemainingTokenAmount string // For testing remaining amount after partial fill
		expAmountInFulfilled    bool
	}{
		{"fill 1st order", "pool1", "1000", "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "0xdac17f958d2ee523a2206206994597c13d831ec7", "100", []string{"1001"}, "0", true},
		{"fill 1st order and skip 2nd (too big)", "pool1", "2000", "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "0xdac17f958d2ee523a2206206994597c13d831ec7", "100", []string{"1001"}, "1000", false},
		{"fill both orders", "pool1", "3000", "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "0xdac17f958d2ee523a2206206994597c13d831ec7", "200", []string{"1001", "1002"}, "0", true},
		{"fill all orders", "pool2", "5000", "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "0xdac17f958d2ee523a2206206994597c13d831ec7", "300", []string{"1001", "1002", "1003"}, "0", true},

		// Test case where amountIn > order output and we get some remaining amount
		{"partial fill with remaining", "pool3", "1500", "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "0xdac17f958d2ee523a2206206994597c13d831ec7", "100", []string{"3001"}, "500", false},

		// Test case where amount is exactly equal to the order limit
		{"exact fill amount", "pool3", "1000", "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "0xdac17f958d2ee523a2206206994597c13d831ec7", "100", []string{"3001"}, "0", true},
	}

	sims := lo.MapValues(pools, func(orders []testorder, _ string) *PoolSimulator {
		extra := Extra{
			TakeToken0Orders: lo.Map(orders, func(o testorder, i int) *DutchOrder {
				return &DutchOrder{
					OrderHash:   o.hash,
					Type:        "Dutch",
					OrderStatus: OpenOrderStatus,
					Swapper:     common.HexToAddress(fmt.Sprintf("0x%d", i)),
					Input: Input{
						Token:       common.HexToAddress(o.makingToken),
						StartAmount: uint256.MustFromDecimal(o.makingAmount),
						EndAmount:   uint256.MustFromDecimal(o.makingAmount),
					},
					Outputs: []Output{
						{
							Token:       common.HexToAddress(o.takingToken),
							StartAmount: uint256.MustFromDecimal(o.takingAmount),
							EndAmount:   uint256.MustFromDecimal(o.takingAmount),
							Recipient:   common.HexToAddress("0x0000000000000000000000000000000000000000"),
						},
					},
				}
			}),
		}
		sExtra, _ := json.Marshal(extra)
		poolEnt := entity.Pool{
			Tokens:      []*entity.PoolToken{{Address: "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"}, {Address: "0xdac17f958d2ee523a2206206994597c13d831ec7"}},
			Reserves:    entity.PoolReserves{"0", "0"},
			StaticExtra: `{"token0":"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48","token1":"0xdac17f958d2ee523a2206206994597c13d831ec7","reactorAddress":"0x1111111111111111111111111111111111111111"}`,
			Extra:       string(sExtra),
		}
		p, err := NewPoolSimulator(poolEnt)
		require.Nil(t, err)
		return p
	})

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := sims[tc.pool].CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{
					Token:  tc.tokenIn,
					Amount: uint256.MustFromDecimal(tc.amountIn).ToBig(),
				},
				TokenOut: tc.tokenOut,
				Limit:    nil,
			})

			if tc.expOrderHashes == nil {
				require.NotNil(t, err)
				return
			}

			require.Nil(t, err)

			// Check the amountOut separately
			assert.Equal(t, tc.expAmountOut, res.TokenAmountOut.Amount.String())

			// Check the remaining amount
			assert.Equal(t, tc.expRemainingTokenAmount, res.RemainingTokenAmountIn.Amount.String(), "Remaining token amount doesn't match")

			// Check swap info
			si := res.SwapInfo.(SwapInfo)
			assert.Equal(t, tc.expAmountInFulfilled, si.IsAmountInFulfilled, "IsAmountInFulfilled doesn't match")

			// Check order hashes
			orderHashes := make([]string, 0, len(si.FilledOrders))
			orderInfo := ""
			for _, o := range si.FilledOrders {
				orderHashes = append(orderHashes, o.OrderHash)
				orderInfo += fmt.Sprintf("order %v\n", o.OrderHash)
			}
			orderHashesJSON, _ := json.Marshal(orderHashes)
			expectedOrderHashesJSON, _ := json.Marshal(tc.expOrderHashes)
			assert.JSONEq(t, string(expectedOrderHashesJSON), string(orderHashesJSON), fmt.Sprintf("Order hashes don't match.\nGot: %s\nExpected: %s\n%s", orderHashesJSON, expectedOrderHashesJSON, orderInfo))
			fmt.Println(orderInfo)
		})
	}
}

func TestPoolSimulator_UpdateBalance(t *testing.T) {
	type testorder struct {
		hash         string
		makingToken  string
		takingToken  string
		makingAmount string
		takingAmount string
	}

	type testswap struct {
		amountIn       string
		tokenIn        string
		tokenOut       string
		expAmountOut   string
		expOrderHashes []string
	}

	pools := map[string][]testorder{
		"pool1": {
			{"1001", "0xdac17f958d2ee523a2206206994597c13d831ec7", "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "100", "1000"},
			{"1002", "0xdac17f958d2ee523a2206206994597c13d831ec7", "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "100", "2000"},
		},
	}

	testcases := []struct {
		name  string
		pool  string
		swaps []testswap
	}{
		{"case 1", "pool1", []testswap{
			{"1000", "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "0xdac17f958d2ee523a2206206994597c13d831ec7", "100", []string{"1001"}},
			{"2000", "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "0xdac17f958d2ee523a2206206994597c13d831ec7", "100", []string{"1002"}},
			{"1000", "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "0xdac17f958d2ee523a2206206994597c13d831ec7", "", nil}, // No orders left
		}},
	}

	var sims map[string]*PoolSimulator
	resetSim := func() {
		sims = lo.MapValues(pools, func(orders []testorder, _ string) *PoolSimulator {
			extra := Extra{
				TakeToken0Orders: lo.Map(orders, func(o testorder, _ int) *DutchOrder {
					return &DutchOrder{
						OrderHash:   o.hash,
						Type:        "Dutch",
						OrderStatus: OpenOrderStatus,
						Swapper:     common.HexToAddress("0x0000000000000000000000000000000000000123"),
						Input: Input{
							Token:       common.HexToAddress(o.makingToken),
							StartAmount: uint256.MustFromDecimal(o.makingAmount),
							EndAmount:   uint256.MustFromDecimal(o.makingAmount),
						},
						Outputs: []Output{
							{
								Token:       common.HexToAddress(o.takingToken),
								StartAmount: uint256.MustFromDecimal(o.takingAmount),
								EndAmount:   uint256.MustFromDecimal(o.takingAmount),
								Recipient:   common.HexToAddress("0x0000000000000000000000000000000000000000"),
							},
						},
					}
				}),
			}
			sExtra, _ := json.Marshal(extra)
			poolEnt := entity.Pool{
				Tokens:      []*entity.PoolToken{{Address: "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"}, {Address: "0xdac17f958d2ee523a2206206994597c13d831ec7"}},
				Reserves:    entity.PoolReserves{"0", "0"},
				StaticExtra: `{"token0":"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48","token1":"0xdac17f958d2ee523a2206206994597c13d831ec7","reactorAddress":"0x1111111111111111111111111111111111111111"}`,
				Extra:       string(sExtra),
			}
			p, err := NewPoolSimulator(poolEnt)
			require.Nil(t, err)
			return p
		})
	}

	for _, tc := range testcases {
		resetSim()
		for i, swap := range tc.swaps {
			t.Run(fmt.Sprintf("%v swap %d", tc.name, i), func(t *testing.T) {
				res, err := sims[tc.pool].CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: pool.TokenAmount{
						Token:  swap.tokenIn,
						Amount: uint256.MustFromDecimal(swap.amountIn).ToBig(),
					},
					TokenOut: swap.tokenOut,
					Limit:    nil,
				})

				if swap.expOrderHashes == nil {
					require.NotNil(t, err)
					return
				}

				require.Nil(t, err)

				// Check the amountOut separately
				assert.Equal(t, swap.expAmountOut, res.TokenAmountOut.Amount.String())

				si := res.SwapInfo.(SwapInfo)
				oid := make([]string, 0, len(si.FilledOrders))
				oinfo := ""
				for _, o := range si.FilledOrders {
					oid = append(oid, o.OrderHash)
					oinfo += fmt.Sprintf("order %v\n", o.OrderHash)
				}
				oidJSON, _ := json.Marshal(oid)
				expectedOidJSON, _ := json.Marshal(swap.expOrderHashes)
				assert.JSONEq(t, string(expectedOidJSON), string(oidJSON), fmt.Sprintf("Order hashes don't match.\nGot: %s\nExpected: %s\n%s", oidJSON, expectedOidJSON, oinfo))
				fmt.Println(oinfo)

				sims[tc.pool].UpdateBalance(pool.UpdateBalanceParams{
					TokenAmountIn: pool.TokenAmount{
						Token:  swap.tokenIn,
						Amount: uint256.MustFromDecimal(swap.amountIn).ToBig(),
					},
					TokenAmountOut: *res.TokenAmountOut,
					Fee:            *res.Fee,
					SwapInfo:       res.SwapInfo,
				})
			})
		}
	}
}
