package valueobject

import (
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/pooltypes"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	clone "github.com/huandu/go-clone/generic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPoolBucket_RollBackPools(t *testing.T) {
	type fields struct {
		PerRequestPoolsByAddress map[string]poolpkg.IPoolSimulator
	}
	type args struct {
		backUpPools []poolpkg.IPoolSimulator
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
	poolByAddresses, err := GenerateRandomPoolByAddress(nPools, tokenAddressList, pooltypes.PoolTypes.KyberPMM)
	require.NoError(t, err)
	var (
		backUpPoolAddress string
		backUpPools       []poolpkg.IPoolSimulator
	)

	//we just backup 1 pool here
	for address := range poolByAddresses {
		backUpPools = []poolpkg.IPoolSimulator{clone.Slowly(poolByAddresses[address])}
		backUpPoolAddress = address
		break
	}

	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{name: "it should be able to roll back pools",
			fields: fields{PerRequestPoolsByAddress: poolByAddresses},
			args:   args{backUpPools},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := NewPoolBucket(tt.fields.PerRequestPoolsByAddress)
			oldPool, avail := b.GetPool(backUpPoolAddress)
			assert.Equal(t, true, avail)

			oldPool.UpdateBalance(poolpkg.UpdateBalanceParams{
				TokenAmountIn: poolpkg.TokenAmount{
					Token:     oldPool.GetTokens()[0],
					Amount:    oldPool.GetReserves()[0],
					AmountUsd: 0,
				},
				TokenAmountOut: poolpkg.TokenAmount{
					Token:     oldPool.GetTokens()[1],
					Amount:    oldPool.GetReserves()[1],
					AmountUsd: 0,
				},
				Fee:       poolpkg.TokenAmount{},
				SwapInfo:  nil,
				SwapLimit: nil,
			})
			b.ClonePool(backUpPoolAddress)
			b.RollBackPools(tt.args.backUpPools)
			rolledBack, avail := b.GetPool(backUpPoolAddress)
			assert.Equal(t, true, avail)
			assert.NotEqual(t, rolledBack.GetReserves(), oldPool.GetReserves())
		})
	}
}
