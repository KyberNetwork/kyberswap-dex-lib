package limitorder

import (
	"math/big"
	"math/rand/v2"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/swaplimit"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

const (
	tokenUSDC = "0x2791bca1f2de4661ed88a30c99a7a9449aa84174"
	tokenUSDT = "0xc2132d05d31c914a87c6611c10748aeb04b58e8f"
)

func TestPool_CalcAmountIn(t *testing.T) {
	t.Parallel()
	type args struct {
		tokenIn        string
		tokenAmountOut pool.TokenAmount
	}
	tests := []struct {
		name       string
		poolEntity entity.Pool
		args       args
		want       *pool.CalcAmountInResult
		err        error
	}{
		{
			name: "Should return correct CalcAmountInResult when swapSide is BUY(strings.ToLower(tokeIn) <= strings.ToLower(TokenOut))",
			poolEntity: newExamplePool(t, nil, []*order{
				newExampleOrder(t, 1, tokenUSDT, big.NewInt(200), big.NewInt(0), tokenUSDC, big.NewInt(400), big.NewInt(0), 0, false),
				newExampleOrder(t, 2, tokenUSDT, big.NewInt(300), big.NewInt(0), tokenUSDC, big.NewInt(300), big.NewInt(0), 0, true),
			}),
			args: args{
				tokenAmountOut: pool.TokenAmount{
					Token:     tokenUSDT,
					Amount:    parseBigInt("500"),
					AmountUsd: 0,
				},
				tokenIn: tokenUSDC,
			},
			want: &pool.CalcAmountInResult{
				TokenAmountIn: &pool.TokenAmount{
					Token:     tokenUSDC,
					Amount:    parseBigInt("300"),
					AmountUsd: 0,
				},
				Fee: &pool.TokenAmount{
					Token:     tokenUSDT,
					Amount:    big.NewInt(0),
					AmountUsd: 0,
				},
				Gas: 314416,
				SwapInfo: SwapInfo{
					AmountIn: "300",
					SwapSide: Buy,
					FilledOrders: []*FilledOrderInfo{
						newExampleFilledOrderInfo(
							t,
							1,
							tokenUSDT, big.NewInt(200), big.NewInt(200),
							tokenUSDC, big.NewInt(400), big.NewInt(400),
							big.NewInt(0), 0,
						),
						newExampleFilledOrderInfo(
							t,
							2,
							tokenUSDT, big.NewInt(300), big.NewInt(100),
							tokenUSDC, big.NewInt(300), big.NewInt(100),
							big.NewInt(0), 0,
						),
					},
				},
			},
			err: nil,
		},
		{
			name: "Should return correct CalcAmountInResult when swapSide is SELL(strings.ToLower(tokeIn) > strings.ToLower(TokenOut))",
			poolEntity: newExamplePool(t, []*order{
				newExampleOrder(t, 1, tokenUSDC, big.NewInt(992_000), big.NewInt(0), tokenUSDT, big.NewInt(1_000_000), big.NewInt(0), 0, false),
				newExampleOrder(t, 2, tokenUSDC, big.NewInt(1_010_000), big.NewInt(0), tokenUSDT, big.NewInt(1_000_000), big.NewInt(0), 0, false),
			}, nil),
			args: args{
				tokenAmountOut: pool.TokenAmount{
					Token:     tokenUSDC,
					Amount:    parseBigInt("1215841"),
					AmountUsd: 0,
				},
				tokenIn: tokenUSDT,
			},
			want: &pool.CalcAmountInResult{
				TokenAmountIn: &pool.TokenAmount{
					Token:     tokenUSDT,
					Amount:    parseBigInt("1210000"),
					AmountUsd: 0,
				},
				Fee: &pool.TokenAmount{
					Token:     tokenUSDC,
					Amount:    big.NewInt(0),
					AmountUsd: 0,
				},
				Gas: 314416,
				SwapInfo: SwapInfo{
					AmountIn: "1210000",
					SwapSide: Sell,
					FilledOrders: []*FilledOrderInfo{
						newExampleFilledOrderInfo(
							t,
							1,
							tokenUSDC, big.NewInt(992_000), big.NewInt(992_000),
							tokenUSDT, big.NewInt(1_000_000), big.NewInt(1_000_000),
							big.NewInt(0), 0,
						),
						newExampleFilledOrderInfo(
							t,
							2,
							tokenUSDC, big.NewInt(1_010_000), big.NewInt(218_000),
							tokenUSDT, big.NewInt(1_000_000), big.NewInt(215_841),
							big.NewInt(0), 0,
						),
					},
				},
			},
			err: nil,
		},
		{
			name: "Should return correct CalcAmountInResult when swapSide is BUY(strings.ToLower(tokeIn) <= strings.ToLower(TokenOut)) and orders has MakerTokenFeePercent",
			poolEntity: newExamplePool(t, nil, []*order{
				newExampleOrder(t, 1383, tokenUSDT, big.NewInt(200), big.NewInt(0), tokenUSDC, big.NewInt(400), big.NewInt(0), 100, true),
				newExampleOrder(t, 1382, tokenUSDT, big.NewInt(300), big.NewInt(0), tokenUSDC, big.NewInt(300), big.NewInt(0), 0, true),
			}),
			args: args{
				tokenAmountOut: pool.TokenAmount{
					Token:     tokenUSDT,
					Amount:    parseBigInt("500"),
					AmountUsd: 0,
				},
				tokenIn: tokenUSDC,
			},
			want: &pool.CalcAmountInResult{
				TokenAmountIn: &pool.TokenAmount{
					Token:     tokenUSDC,
					Amount:    parseBigInt("302"),
					AmountUsd: 0,
				},
				Fee: &pool.TokenAmount{
					Token:     tokenUSDT,
					Amount:    big.NewInt(0),
					AmountUsd: 0,
				},
				Gas: 314416,
				SwapInfo: SwapInfo{
					AmountIn: "302",
					SwapSide: Buy,
					FilledOrders: []*FilledOrderInfo{
						newExampleFilledOrderInfo(t,
							1383,
							tokenUSDT, big.NewInt(200), big.NewInt(200),
							tokenUSDC, big.NewInt(400), big.NewInt(400),
							big.NewInt(0), 100,
						),
						newExampleFilledOrderInfo(t,
							1382,
							tokenUSDT, big.NewInt(300), big.NewInt(100),
							tokenUSDC, big.NewInt(300), big.NewInt(100),
							big.NewInt(0), 0,
						),
					},
				},
			},
			err: nil,
		},
		{
			name: "Should return correct CalcAmountInResult and list orders(include fallback orders) when swapSide is SELL(strings.ToLower(tokeIn) > strings.ToLower(TokenOut))",
			poolEntity: newExamplePool(t, []*order{
				newExampleOrder(t, 1383, tokenUSDC, big.NewInt(700), big.NewInt(0), tokenUSDT, big.NewInt(1400), big.NewInt(0), 0, false),
				newExampleOrder(t, 1382, tokenUSDC, big.NewInt(240), big.NewInt(0), tokenUSDT, big.NewInt(250), big.NewInt(0), 0, false),
				newExampleOrder(t, 1385, tokenUSDC, big.NewInt(200), big.NewInt(0), tokenUSDT, big.NewInt(300), big.NewInt(0), 0, false),
				newExampleOrder(t, 1389, tokenUSDC, big.NewInt(100), big.NewInt(0), tokenUSDT, big.NewInt(100), big.NewInt(0), 0, false),
			}, nil),
			args: args{
				tokenAmountOut: pool.TokenAmount{
					Token:     tokenUSDC,
					Amount:    parseBigInt("1400"),
					AmountUsd: 0,
				},
				tokenIn: tokenUSDT,
			},
			want: &pool.CalcAmountInResult{
				TokenAmountIn: &pool.TokenAmount{
					Token:     tokenUSDT,
					Amount:    parseBigInt("700"),
					AmountUsd: 0,
				},
				Fee: &pool.TokenAmount{
					Token:     tokenUSDC,
					Amount:    big.NewInt(0),
					AmountUsd: 0,
				},
				Gas: 426624,
				SwapInfo: SwapInfo{
					AmountIn: "700",
					SwapSide: Sell,
					FilledOrders: []*FilledOrderInfo{
						newExampleFilledOrderInfo(t,
							1383,
							tokenUSDC, big.NewInt(700), big.NewInt(700),
							tokenUSDT, big.NewInt(1400), big.NewInt(1400),
							big.NewInt(0), 0,
						),
						newExampleFallBackOrderInfo(t,
							1382,
							tokenUSDC, big.NewInt(240),
							tokenUSDT, big.NewInt(250),
							0,
						),
						newExampleFallBackOrderInfo(t,
							1385,
							tokenUSDC, big.NewInt(200),
							tokenUSDT, big.NewInt(300),
							0,
						),
					},
				},
			},
			err: nil,
		},
		{
			name: "Should return correct error(ErrCannotFulfillAmountIn) when cannot fulfill amountIn buy the orders",
			poolEntity: newExamplePool(t, []*order{
				newExampleOrder(t, 1383, tokenUSDC, big.NewInt(992_000), big.NewInt(0), tokenUSDT, big.NewInt(1_000_000), big.NewInt(0), 0, false),
				newExampleOrder(t, 1382, tokenUSDC, big.NewInt(1_010_000), big.NewInt(0), tokenUSDT, big.NewInt(1_000_000), big.NewInt(0), 0, false),
			}, nil),
			args: args{
				tokenAmountOut: pool.TokenAmount{
					Token:     tokenUSDC,
					Amount:    parseBigInt("121000000"),
					AmountUsd: 0,
				},
				tokenIn: tokenUSDT,
			},
			want: nil,
			err:  ErrCannotFulfillAmountOut,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := NewPoolSimulator(tt.poolEntity)
			assert.Equal(t, nil, err)
			got, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountInResult, error) {
				limit := swaplimit.NewInventory("", p.CalculateLimit())
				return p.CalcAmountIn(
					pool.CalcAmountInParams{
						TokenAmountOut: tt.args.tokenAmountOut,
						TokenIn:        tt.args.tokenIn,
						Limit:          limit,
					})
			})
			assert.Equal(t, tt.err, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPool_CalcAmountOut_CalcAmountIn(t *testing.T) {
	t.Parallel()
	type args struct {
		tokenIn   string
		amountIn  *big.Int
		tokenOut  string
		amountOut *big.Int
	}

	tests := []struct {
		name       string
		poolEntity entity.Pool
		args       args
		err        error
	}{
		{
			name: "Should return correct CalcAmountInResult and CalcAmountOutResult when swapSide is BUY(strings.ToLower(tokeIn) <= strings.ToLower(TokenOut))",
			poolEntity: newExamplePool(t, nil, []*order{
				newExampleOrder(t, 1, tokenUSDT, big.NewInt(200), big.NewInt(0), tokenUSDC, big.NewInt(400), big.NewInt(0), 0, false),
				newExampleOrder(t, 2, tokenUSDT, big.NewInt(300), big.NewInt(0), tokenUSDC, big.NewInt(300), big.NewInt(0), 0, true),
			}),
			args: args{
				tokenIn:   tokenUSDC,
				amountIn:  parseBigInt("300"),
				tokenOut:  tokenUSDT,
				amountOut: parseBigInt("500"),
			},
			err: nil,
		},
		{
			name: "Should return correct CalcAmountInResult and CalcAmountOutResult when swapSide is SELL(strings.ToLower(tokeIn) > strings.ToLower(TokenOut))",
			poolEntity: newExamplePool(t, []*order{
				newExampleOrder(t, 1, tokenUSDC, big.NewInt(992_000), big.NewInt(0), tokenUSDT, big.NewInt(1_000_000), big.NewInt(0), 0, false),
				newExampleOrder(t, 2, tokenUSDC, big.NewInt(1_010_000), big.NewInt(0), tokenUSDT, big.NewInt(1_000_000), big.NewInt(0), 0, false),
			}, nil),
			args: args{

				tokenIn:   tokenUSDT,
				amountIn:  parseBigInt("1210000"),
				tokenOut:  tokenUSDC,
				amountOut: parseBigInt("1215841"),
			},
			err: nil,
		},
		{
			name: "Should return correct CalcAmountInResult and CalcAmountOutResult when swapSide is BUY(strings.ToLower(tokeIn) <= strings.ToLower(TokenOut)) and orders has MakerTokenFeePercent",
			poolEntity: newExamplePool(t, nil, []*order{
				newExampleOrder(t, 1383, tokenUSDT, big.NewInt(200), big.NewInt(0), tokenUSDC, big.NewInt(400), big.NewInt(0), 100, true),
				newExampleOrder(t, 1382, tokenUSDT, big.NewInt(300), big.NewInt(0), tokenUSDC, big.NewInt(300), big.NewInt(0), 0, true),
			}),
			args: args{
				tokenIn:   tokenUSDC,
				amountIn:  parseBigInt("302"),
				tokenOut:  tokenUSDT,
				amountOut: parseBigInt("500"),
			},
			err: nil,
		},
		{
			name: "Should return correct CalcAmountInResult and CalcAmountOutResult and list orders(include fallback orders) when swapSide is SELL(strings.ToLower(tokeIn) > strings.ToLower(TokenOut))",
			poolEntity: newExamplePool(t, []*order{
				newExampleOrder(t, 1383, tokenUSDC, big.NewInt(700), big.NewInt(0), tokenUSDT, big.NewInt(1400), big.NewInt(0), 0, false),
				newExampleOrder(t, 1382, tokenUSDC, big.NewInt(240), big.NewInt(0), tokenUSDT, big.NewInt(250), big.NewInt(0), 0, false),
				newExampleOrder(t, 1385, tokenUSDC, big.NewInt(200), big.NewInt(0), tokenUSDT, big.NewInt(300), big.NewInt(0), 0, false),
				newExampleOrder(t, 1389, tokenUSDC, big.NewInt(100), big.NewInt(0), tokenUSDT, big.NewInt(100), big.NewInt(0), 0, false),
			}, nil),
			args: args{
				tokenIn:   tokenUSDT,
				amountIn:  parseBigInt("700"),
				tokenOut:  tokenUSDC,
				amountOut: parseBigInt("1400"),
			},
			err: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := NewPoolSimulator(tt.poolEntity)
			assert.Equal(t, nil, err)
			calcAmountOutResult, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountOutResult, error) {
				limit := swaplimit.NewInventory("", p.CalculateLimit())
				return p.CalcAmountOut(
					pool.CalcAmountOutParams{
						TokenAmountIn: pool.TokenAmount{
							Token:  tt.args.tokenIn,
							Amount: tt.args.amountIn,
						},
						TokenOut: tt.args.tokenOut,
						Limit:    limit,
					})
			})
			require.NoError(t, err)

			assert.Equal(t, tt.args.amountOut, calcAmountOutResult.TokenAmountOut.Amount)

			calcAmountInResult, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountInResult, error) {
				limit := swaplimit.NewInventory("", p.CalculateLimit())
				return p.CalcAmountIn(
					pool.CalcAmountInParams{
						TokenAmountOut: pool.TokenAmount{
							Token:  tt.args.tokenOut,
							Amount: tt.args.amountOut,
						},
						TokenIn: tt.args.tokenIn,
						Limit:   limit,
					})
			})
			require.NoError(t, err)

			assert.Equal(t, tt.args.amountIn, calcAmountInResult.TokenAmountIn.Amount)
		})
	}
}

func newExamplePool(t *testing.T, sellOrders, buyOrders []*order) entity.Pool {
	t.Helper()

	return entity.Pool{
		Address:      "pool_limit_order_",
		ReserveUsd:   1000000000,
		AmplifiedTvl: 0,
		SwapFee:      0,
		Exchange:     "kyberswap_limit-order",
		Type:         "limit-order",
		Timestamp:    0,
		Reserves:     []string{"10000000000000000000", "10000000000000000000"},
		Tokens: []*entity.PoolToken{
			{
				Address:   tokenUSDT,
				Symbol:    "USDT",
				Decimals:  6,
				Swappable: true,
			},
			{
				Address:   tokenUSDC,
				Symbol:    "USDC",
				Decimals:  6,
				Swappable: true,
			},
		},
		Extra: marshalPoolExtra(&Extra{
			BuyOrders:  buyOrders,
			SellOrders: sellOrders,
		}),
		TotalSupply: "",
	}
}

func newExampleOrder(
	t *testing.T,
	id int64,
	takerAsset string, takingAmount, filledTakingAmount *big.Int,
	makerAsset string, makingAmount, filledMakingAmount *big.Int,
	makerTokenFeePercent uint32, IsTakerAssetFee bool,
) *order {
	t.Helper()

	return &order{
		ID:                   id,
		ChainID:              "5",
		Salt:                 "185786982651412687203851465093295409688",
		Signature:            "signature" + strconv.Itoa(int(id)),
		TakerAsset:           takerAsset,
		MakerAsset:           makerAsset,
		Receiver:             "0xa246ec8bf7f2e54cc2f7bfdd869302ae4a08a590",
		Maker:                "0xa246ec8bf7f2e54cc2f7bfdd869302ae4a08a590",
		AllowedSenders:       "0x0000000000000000000000000000000000000000",
		TakingAmount:         takingAmount,
		MakingAmount:         makingAmount,
		FeeConfig:            parseBigInt("100"),
		FeeRecipient:         "0x0000000000000000000000000000000000000000",
		FilledMakingAmount:   filledMakingAmount,
		FilledTakingAmount:   filledTakingAmount,
		MakerTokenFeePercent: makerTokenFeePercent,
		MakerAssetData:       "",
		TakerAssetData:       "",
		GetMakerAmount:       "f4a215c3000000000000000000000000000000000000000000000001d7d843dc3b4800000000000000000000000000000000000000000000000000000de0b6b3a7640000",
		GetTakerAmount:       "296637bf000000000000000000000000000000000000000000000001d7d843dc3b4800000000000000000000000000000000000000000000000000000de0b6b3a7640000",
		Predicate:            "961d5b1e000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000000000020000000000000000000000002892e28b58ab329741f27fd1ea56dca0192a38840000000000000000000000002892e28b58ab329741f27fd1ea56dca0192a38840000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000c00000000000000000000000000000000000000000000000000000000000000044cf6fc6e3000000000000000000000000a246ec8bf7f2e54cc2f7bfdd869302ae4a08a590000000000000000000000000000000000000000000000000000000000000000600000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000002463592c2b0000000000000000000000000000000000000000000000000000000063c1169800000000000000000000000000000000000000000000000000000000",
		Permit:               "",
		Interaction:          "",
		ExpiredAt:            0,
		IsTakerAssetFee:      IsTakerAssetFee,
	}
}

func newExampleFallBackOrderInfo(
	t *testing.T,
	orderID int64,
	takerAsset string, takingAmount *big.Int,
	makerAsset string, makingAmount *big.Int,
	makerTokenFeePercent uint32,
) *FilledOrderInfo {
	t.Helper()
	o := newExampleFilledOrderInfo(
		t, orderID,
		takerAsset, takingAmount, big.NewInt(0),
		makerAsset, makingAmount, big.NewInt(0),
		big.NewInt(0), makerTokenFeePercent,
	)
	o.IsFallBack = true
	return o
}

func newExampleFilledOrderInfo(
	t *testing.T,
	orderID int64,
	takerAsset string, takingAmount, filledTakingAmount *big.Int,
	makerAsset string, makingAmount, filledMakingAmount *big.Int,
	feeAmount *big.Int, makerTokenFeePercent uint32,
) *FilledOrderInfo {
	t.Helper()

	return &FilledOrderInfo{
		OrderID:              orderID,
		FilledTakingAmount:   filledTakingAmount.String(),
		FilledMakingAmount:   filledMakingAmount.String(),
		TakingAmount:         takingAmount.String(),
		MakingAmount:         makingAmount.String(),
		Salt:                 "185786982651412687203851465093295409688",
		TakerAsset:           takerAsset,
		MakerAsset:           makerAsset,
		Maker:                "0xa246ec8bf7f2e54cc2f7bfdd869302ae4a08a590",
		Receiver:             "0xa246ec8bf7f2e54cc2f7bfdd869302ae4a08a590",
		AllowedSenders:       "0x0000000000000000000000000000000000000000",
		GetMakerAmount:       "f4a215c3000000000000000000000000000000000000000000000001d7d843dc3b4800000000000000000000000000000000000000000000000000000de0b6b3a7640000",
		GetTakerAmount:       "296637bf000000000000000000000000000000000000000000000001d7d843dc3b4800000000000000000000000000000000000000000000000000000de0b6b3a7640000",
		FeeConfig:            "100",
		FeeRecipient:         "0x0000000000000000000000000000000000000000",
		MakerTokenFeePercent: makerTokenFeePercent,
		MakerAssetData:       "",
		TakerAssetData:       "",
		Predicate:            "961d5b1e000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000000000020000000000000000000000002892e28b58ab329741f27fd1ea56dca0192a38840000000000000000000000002892e28b58ab329741f27fd1ea56dca0192a38840000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000c00000000000000000000000000000000000000000000000000000000000000044cf6fc6e3000000000000000000000000a246ec8bf7f2e54cc2f7bfdd869302ae4a08a590000000000000000000000000000000000000000000000000000000000000000600000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000002463592c2b0000000000000000000000000000000000000000000000000000000063c1169800000000000000000000000000000000000000000000000000000000",
		Permit:               "",
		Interaction:          "",
		Signature:            "signature" + strconv.Itoa(int(orderID)),
		FeeAmount:            feeAmount.String(),
	}
}

// Callers (e.g. onchain-price-service) check CalcAmountOut(CalcAmountIn(out)) against out, so amountIn must reach
// amountOut via exact-in, and 1 wei less must not along the same fill path. Exact-in skips orders that would fill
// 0 making wei, so a smaller amountIn can reach amountOut only by skipping to a later order.
func TestPool_CalcAmountIn_MinimalInverseOfCalcAmountOut(t *testing.T) {
	t.Parallel()
	makers := []string{"0x1111111111111111111111111111111111111111", "0x2222222222222222222222222222222222222222"}
	rng := rand.New(rand.NewPCG(1, 2))
	const maxAmount = 300 // small amounts make rounding edges frequent
	calcOut := func(p *PoolSimulator, amountIn *big.Int) (*big.Int, []int64, error) {
		res, err := p.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: tokenUSDC, Amount: amountIn},
			TokenOut:      tokenUSDT,
			Limit:         swaplimit.NewInventory("", p.CalculateLimit()),
		})
		if err != nil {
			return nil, nil, err
		}
		var filledIDs []int64
		for _, o := range res.SwapInfo.(SwapInfo).FilledOrders {
			if !o.IsFallBack {
				filledIDs = append(filledIDs, o.OrderID)
			}
		}
		return res.TokenAmountOut.Amount, filledIDs, nil
	}

	var succeeded, failed, skipped int
	for range 5000 {
		orders := make([]*order, 1+rng.IntN(5))
		balances := map[string]*big.Int{}
		totalMaking := int64(0)
		for i := range orders {
			taking, making := big.NewInt(1+rng.Int64N(maxAmount)), big.NewInt(1+rng.Int64N(maxAmount))
			orders[i] = newExampleOrder(t, int64(i+1), tokenUSDC, taking, big.NewInt(0), tokenUSDT, making,
				big.NewInt(0), uint32(rng.IntN(3)*50), rng.IntN(2) == 0)
			totalMaking += making.Int64()
			if rng.IntN(2) == 0 { // maker balance shared across orders caps remaining amounts
				o := orders[i]
				o.Maker = makers[rng.IntN(len(makers))]
				o.AvailableMakingAmount = new(big.Int).Set(making)
				if balances[o.Maker] == nil {
					balances[o.Maker] = big.NewInt(1 + rng.Int64N(maxAmount))
				}
				o.MakerBalanceAllowance = balances[o.Maker]
			}
		}
		p, err := NewPoolSimulator(newExamplePool(t, nil, orders))
		require.NoError(t, err)

		amountOut := big.NewInt(1 + rng.Int64N(totalMaking))
		res, err := p.CalcAmountIn(pool.CalcAmountInParams{
			TokenAmountOut: pool.TokenAmount{Token: tokenUSDT, Amount: amountOut},
			TokenIn:        tokenUSDC,
			Limit:          swaplimit.NewInventory("", p.CalculateLimit()),
		})
		if err != nil {
			require.ErrorIs(t, err, ErrCannotFulfillAmountOut)
			failed++
			continue
		}
		succeeded++
		amountIn := res.TokenAmountIn.Amount

		out, filledIDs, err := calcOut(p, amountIn)
		require.NoError(t, err)
		require.GreaterOrEqual(t, out.Cmp(amountOut), 0, "amountIn %v gives %v < %v", amountIn, out, amountOut)
		require.Equal(t, amountIn.String(), res.SwapInfo.(SwapInfo).AmountIn)

		outLess, filledIDsLess, err := calcOut(p, new(big.Int).Sub(amountIn, bignumber.One))
		if err != nil || outLess.Cmp(amountOut) < 0 {
			continue
		}
		require.NotEqual(t, filledIDs, filledIDsLess, "amountIn %v is not minimal", amountIn)
		require.Greater(t, filledIDsLess[len(filledIDsLess)-1], filledIDs[len(filledIDs)-1],
			"smaller amountIn must skip to a later order")
		skipped++
	}
	t.Logf("succeeded %d, failed %d, smaller amountIn via skipped order %d", succeeded, failed, skipped)
}

func TestPool_CalcAmountIn_FeesReachAmountOut(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		orders       []*order
		amountOut    int64
		wantAmountIn int64
	}{
		{
			// fully filled order 1 delivers 400 - 1% maker fee = 396, so order 2 must fill 104, not 100
			name: "maker asset fee is deducted from fully filled order",
			orders: []*order{
				newExampleOrder(t, 1, tokenUSDC, big.NewInt(400), big.NewInt(0), tokenUSDT, big.NewInt(400), big.NewInt(0), 100, false),
				newExampleOrder(t, 2, tokenUSDC, big.NewInt(300), big.NewInt(0), tokenUSDT, big.NewInt(300), big.NewInt(0), 0, false),
			},
			amountOut:    500,
			wantAmountIn: 504,
		},
		{
			// 10100 + 1 has takingAmountAfterFee = floor(10101 * 1e4 / 10100) = 10000, which exact-in fills
			// partially on order 1; order 1 is fully filled only from 10102
			name: "taker asset fee needs takingAmountAfterFee above remaining taking amount to fully fill",
			orders: []*order{
				newExampleOrder(t, 1, tokenUSDC, big.NewInt(10000), big.NewInt(0), tokenUSDT, big.NewInt(10000), big.NewInt(0), 100, true),
				newExampleOrder(t, 2, tokenUSDC, big.NewInt(1000), big.NewInt(0), tokenUSDT, big.NewInt(1000), big.NewInt(0), 0, false),
			},
			amountOut:    10001,
			wantAmountIn: 10102,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := NewPoolSimulator(newExamplePool(t, nil, tt.orders))
			require.NoError(t, err)
			res, err := p.CalcAmountIn(pool.CalcAmountInParams{
				TokenAmountOut: pool.TokenAmount{Token: tokenUSDT, Amount: big.NewInt(tt.amountOut)},
				TokenIn:        tokenUSDC,
			})
			require.NoError(t, err)
			assert.Equal(t, big.NewInt(tt.wantAmountIn), res.TokenAmountIn.Amount)

			for amountIn, reached := range map[int64]bool{tt.wantAmountIn: true, tt.wantAmountIn - 1: false} {
				out, err := p.CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: pool.TokenAmount{Token: tokenUSDC, Amount: big.NewInt(amountIn)},
					TokenOut:      tokenUSDT,
				})
				require.NoError(t, err)
				assert.Equal(t, reached, out.TokenAmountOut.Amount.Int64() >= tt.amountOut, "amountIn %d", amountIn)
			}
		})
	}
}

func TestPool_CalcAmountIn_Generic(t *testing.T) {
	t.Parallel()
	p, err := NewPoolSimulator(newExamplePool(t, []*order{
		newExampleOrder(t, 1, tokenUSDC, big.NewInt(992_000_000), big.NewInt(0), tokenUSDT, big.NewInt(1_000_000_000), big.NewInt(0), 0, false),
		newExampleOrder(t, 2, tokenUSDC, big.NewInt(1_010_000_000), big.NewInt(0), tokenUSDT, big.NewInt(1_000_000_000), big.NewInt(0), 30, true),
		newExampleOrder(t, 3, tokenUSDC, big.NewInt(5_050_000_000), big.NewInt(0), tokenUSDT, big.NewInt(5_000_000_000), big.NewInt(0), 50, false),
	}, []*order{
		newExampleOrder(t, 4, tokenUSDT, big.NewInt(2_000_000_000), big.NewInt(0), tokenUSDC, big.NewInt(1_990_000_000), big.NewInt(0), 100, false),
		newExampleOrder(t, 5, tokenUSDT, big.NewInt(3_000_000_000), big.NewInt(0), tokenUSDC, big.NewInt(2_950_000_000), big.NewInt(0), 20, true),
	}))
	require.NoError(t, err)
	testutil.TestCalcAmountIn(t, p)
}
