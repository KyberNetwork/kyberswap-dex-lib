package retryfinder

import (
	"context"
	"math/big"
	"strconv"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/uniswap"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

//implement IpoolSimulator for testPool

func TestRetryFinder_retryDynamicPools(t *testing.T) {
	var (
		tokenIn = &entity.Token{
			Address:  "a",
			Symbol:   "a",
			Name:     "a",
			Decimals: 18,
		}
		midToken = &entity.Token{
			Address:  "b",
			Symbol:   "b",
			Name:     "b",
			Decimals: 18,
		}
		tokenOut = &entity.Token{
			Address:  "c",
			Symbol:   "c",
			Name:     "c",
			Decimals: 18,
		}

		tokenByAddress = map[string]*entity.Token{
			"a": tokenIn,
			"b": midToken,
			"c": tokenOut,
		}
		mapUSDprice = map[string]float64{
			"a": 100,
			"b": 100,
			"c": 100,
		}

		tokenList = []string{"a", "b", "c"}
		gasPrice  = big.NewFloat(10)
	)

	pools, err := valueobject.GenerateUniv2PoolByTokenAddress(tokenList)
	require.NoError(t, err)

	//this is a pool which is better than pool 0.
	betterPool := entity.Pool{
		Address: "pool_" + strconv.Itoa(len(pools)),
		SwapFee: 0.0,
		Tokens: entity.PoolTokens{
			&entity.PoolToken{Address: tokenList[0]},
			&entity.PoolToken{Address: tokenList[1]},
		},
		Reserves: entity.PoolReserves{
			strconv.Itoa(1_000_000),
			strconv.Itoa(1_000_000_0),
		},
		Type: "pmm",
	}
	// using uni pool for simplicity
	pmmPoolSim, pErr := uniswap.NewPoolSimulator(betterPool)

	require.NoError(t, pErr)

	var poolsWithPmm = make(map[string]poolpkg.IPoolSimulator)
	for _, pool := range pools {
		poolsWithPmm[pool.GetAddress()] = pool
	}
	poolsWithPmm[pmmPoolSim.GetAddress()] = pmmPoolSim

	nonModifiedRoute := &valueobject.Route{
		Input: valueobject.TokenAmount{
			Token:  tokenIn.Address,
			Amount: big.NewInt(100),
		},
		Output: valueobject.TokenAmount{
			Token:  tokenOut.Address,
			Amount: big.NewInt(98),
		},
		Paths: []*valueobject.Path{
			{
				Input: valueobject.TokenAmount{
					Token:  tokenIn.Address,
					Amount: big.NewInt(100),
				},
				Output: valueobject.TokenAmount{
					Token:  tokenOut.Address,
					Amount: big.NewInt(98),
				},
				TotalGas:      1000,
				PoolAddresses: []string{"pool_0", "pool_1"},
				Tokens:        []*entity.Token{tokenIn, midToken, tokenOut},
			},
		},
		TotalGas: 0,
		Extra:    valueobject.RouteExtraData{},
	}

	type args struct {
		ctx          context.Context
		input        findroute.Input
		route        *valueobject.Route
		dynamicTypes []string
		data         findroute.FinderData
		gasOption    valueobject.GasOption
	}
	var (
		tests = []struct {
			name string
			args args
			want *valueobject.Route
		}{
			{
				name: "bestRoute already, retry finder can't optimize",
				args: args{
					ctx: context.Background(),
					input: findroute.Input{
						TokenInAddress:         tokenIn.Address,
						TokenOutAddress:        tokenOut.Address,
						AmountIn:               big.NewInt(100),
						GasPrice:               gasPrice,
						GasTokenPriceUSD:       100,
						SaveGas:                false,
						GasInclude:             false,
						IsPathGeneratorEnabled: false,
						SourceHash:             0,
					},
					route:        nonModifiedRoute,
					dynamicTypes: []string{"pmm"},
					data: findroute.NewFinderData(context.Background(), tokenByAddress, mapUSDprice, nil, &types.FindRouteState{
						Pools:     pools,
						SwapLimit: nil,
					}),
					gasOption: valueobject.GasOption{
						GasFeeInclude: false,
						Price:         big.NewFloat(10),
						TokenPrice:    0,
					},
				},
				want: nil,
			}, {
				name: "there is a pool better than pool 0",
				args: args{
					ctx: context.Background(),
					input: findroute.Input{
						TokenInAddress:         tokenIn.Address,
						TokenOutAddress:        tokenOut.Address,
						AmountIn:               big.NewInt(100),
						GasPrice:               gasPrice,
						GasTokenPriceUSD:       100,
						SaveGas:                false,
						GasInclude:             false,
						IsPathGeneratorEnabled: false,
						SourceHash:             0,
					},
					route:        nonModifiedRoute,
					dynamicTypes: []string{"pmm"},
					data: findroute.NewFinderData(context.Background(), tokenByAddress, mapUSDprice, nil, &types.FindRouteState{
						Pools:     poolsWithPmm,
						SwapLimit: nil,
					}),
					gasOption: valueobject.GasOption{
						GasFeeInclude: false,
						Price:         big.NewFloat(10),
						TokenPrice:    0,
					},
				},
				want: &valueobject.Route{
					Input: valueobject.TokenAmount{
						Token:  tokenIn.Address,
						Amount: big.NewInt(100),
					},
					Output: valueobject.TokenAmount{
						Token:  tokenOut.Address,
						Amount: big.NewInt(998),
					}},
			},
		}
	)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RetryFinder{}
			got := r.retryDynamicPools(tt.args.ctx, tt.args.input, tt.args.route, tt.args.dynamicTypes, tt.args.data, tt.args.gasOption)
			if tt.want == nil {
				require.Nil(t, got)
			} else if got.CompareTo(tt.want, tt.args.input.GasInclude) != 0 {
				t.Errorf("retryDynamicPools() = %v, want %v", got, tt.want)
			}
		})
	}
}
