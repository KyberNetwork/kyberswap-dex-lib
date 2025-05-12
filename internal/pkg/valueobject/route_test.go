package valueobject

import (
	"context"
	"fmt"
	"math/big"
	"strconv"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/pooltypes"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/uniswap"
	clone "github.com/huandu/go-clone/generic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRoute_AddPathRollBack(t *testing.T) {
	type fields struct {
		Input    TokenAmount
		Output   TokenAmount
		Paths    []*Path
		TotalGas int64
	}

	type args struct {
		poolBucket *PoolBucket
		p          *Path
		swapLimits map[string]poolpkg.SwapLimit
	}
	var (
		nTokens = 10
		nPools  = 100
	)
	tokenByAddress := GenerateRandomTokenByAddress(nTokens)
	var (
		tokenAddressList = make([]string, len(tokenByAddress))
		i                = 0
	)

	for tokenAddress := range tokenByAddress {
		tokenAddressList[i] = tokenAddress
		i++
	}

	var (
		tokenIn     = tokenByAddress[tokenAddressList[0]]
		middleToken = tokenByAddress[tokenAddressList[1]]
		tokenOut    = tokenByAddress[tokenAddressList[2]]

		//tokensOnPaths = []*entity.SimplifiedToken{tokenIn, tokenOut}
		tokenAmountIn = TokenAmount{
			Token:     tokenIn.Address,
			Amount:    big.NewInt(100_000),
			AmountUsd: 0,
		}
		//gasOption = GasOption{
		//	GasFeeInclude: false,
		//	Price:         big.NewFloat(1000),
		//	TokenPrice:    0,
		//}
	)
	poolByAddresses, err := GenerateRandomPoolByAddress(nPools, tokenAddressList, pooltypes.PoolTypes.KyberPMM)
	require.NoError(t, err)

	entity1 := entity.Pool{
		Address: "thisPool",
		SwapFee: RandFloat(0, 0.05),
		Tokens: entity.PoolTokens{
			&entity.PoolToken{Address: tokenIn.Address},
			&entity.PoolToken{Address: middleToken.Address},
		},
		Reserves: entity.PoolReserves{
			strconv.Itoa(1_000_000),
			strconv.Itoa(1_000_000),
		},
	}
	poolToAdd, err := uniswap.NewPoolSimulator(entity1)
	poolByAddresses[poolToAdd.GetAddress()] = poolToAdd

	require.NoError(t, err)
	oldPool := clone.Slowly(poolToAdd)
	entity2 := entity.Pool{Address: "thatPool",
		SwapFee: RandFloat(0, 0.05),
		Tokens: entity.PoolTokens{
			&entity.PoolToken{Address: middleToken.Address},
			&entity.PoolToken{Address: tokenOut.Address},
		},
		Reserves: entity.PoolReserves{
			strconv.Itoa(0),
			strconv.Itoa(0),
		}}
	poolToFail, err := uniswap.NewPoolSimulator(entity2)
	require.NoError(t, err)

	poolByAddresses[poolToFail.GetAddress()] = poolToFail

	bucket := NewPoolBucket(poolByAddresses)
	//bucket.ClonePool(poolToAdd.GetAddress())
	path := Path{
		Input: tokenAmountIn,
		Output: TokenAmount{
			Token:     tokenOut.Address,
			Amount:    big.NewInt(1000),
			AmountUsd: 0,
		},
		TotalGas:      0,
		PoolAddresses: []string{poolToAdd.GetAddress(), poolToFail.GetAddress()},
		Tokens:        []*entity.SimplifiedToken{tokenIn, middleToken, tokenOut},
	}
	require.NoError(t, err)
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "Once path got error it will roll back successfully",
			fields: fields{
				Input: tokenAmountIn,
				Output: TokenAmount{
					Token:     tokenOut.Address,
					Amount:    big.NewInt(1000),
					AmountUsd: 0,
				},
				TotalGas: 0,
				Paths:    []*Path{&path},
			},
			args: args{
				poolBucket: bucket,
				p:          &path,
				swapLimits: nil,
			},
			wantErr: assert.Error,
		},
	}
	bucket.ClonePool(poolToAdd.GetAddress())
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Route{
				Input:    tt.fields.Input,
				Output:   tt.fields.Output,
				Paths:    tt.fields.Paths,
				TotalGas: tt.fields.TotalGas,
			}
			tt.wantErr(t, r.AddPath(context.TODO(), tt.args.poolBucket, tt.args.p, tt.args.swapLimits), fmt.Sprintf("AddPath(%v, %v, %v)", tt.args.poolBucket, tt.args.p, tt.args.swapLimits))
			rolledBackPool, avail := bucket.GetPool(oldPool.GetAddress())
			assert.Equal(t, true, avail)
			assert.Equal(t, oldPool.GetReserves(), rolledBackPool.GetReserves())

		})
	}
}
