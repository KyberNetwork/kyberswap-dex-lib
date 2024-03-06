package valueobject

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/pooltypes"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/stretchr/testify/assert"

	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

func TestNewPath(t *testing.T) {
	//Push some nil here
	PathsPool.Put(nil)
	PathsPool.Put(nil)
	const (
		nPool        = 1000
		nToken       = 50
		nPoolOnPaths = 2
	)
	type args struct {
		poolBucket       *PoolBucket
		poolAddresses    []string
		tokens           []*entity.Token
		tokenAmountIn    pool.TokenAmount
		tokenOut         string
		tokenOutPrice    float64
		tokenOutDecimals uint8
		gasOption        GasOption
		limits           map[string]poolpkg.SwapLimit
	}
	tokenByAddress := GenerateRandomTokenByAddress(nToken)
	var tokenAddressList []string
	for tokenAddress := range tokenByAddress {
		tokenAddressList = append(tokenAddressList, tokenAddress)
	}
	poolByAddress, err := GenerateRandomPoolByAddress(nPool, tokenAddressList, pooltypes.PoolTypes.UniswapV2)
	assert.NoError(t, err, "must be able to generate random pool")
	poolBucket := NewPoolBucket(poolByAddress)
	poolAddressOnPaths := make([]string, nPoolOnPaths)
	tokensOnPaths := make([]*entity.Token, nPoolOnPaths+1)
	index := 0

	for address, pool := range poolByAddress {
		if index > 0 && pool.GetTokens()[0] != tokensOnPaths[index].Address {
			continue
		}
		poolAddressOnPaths[index] = address
		tokens := pool.GetTokens()
		tokensOnPaths[index] = tokenByAddress[tokens[0]]
		tokensOnPaths[index+1] = tokenByAddress[tokens[1]]

		index++
		if index == nPoolOnPaths {
			break
		}
	}
	tokenAmounIn := pool.TokenAmount{
		Token:     tokensOnPaths[0].Address,
		Amount:    big.NewInt(1_000_000_000),
		AmountUsd: 0,
	}
	fmt.Println(tokensOnPaths)
	fmt.Println(poolAddressOnPaths)
	tests := []struct {
		name    string
		args    args
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "should newPath successfully even if there is nil in PathPool",
			args: args{
				poolBucket:       poolBucket,
				poolAddresses:    poolAddressOnPaths,
				tokens:           tokensOnPaths,
				tokenAmountIn:    tokenAmounIn,
				tokenOut:         tokensOnPaths[nPoolOnPaths].Address,
				tokenOutPrice:    0,
				tokenOutDecimals: tokensOnPaths[nPoolOnPaths].Decimals,
				gasOption: GasOption{
					GasFeeInclude: false,
					Price:         big.NewFloat(1000),
					TokenPrice:    0,
				},
				limits: map[string]poolpkg.SwapLimit{},
			},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewPath(tt.args.poolBucket, tt.args.poolAddresses, tt.args.tokens, tt.args.tokenAmountIn, tt.args.tokenOut, tt.args.tokenOutPrice, tt.args.tokenOutDecimals, tt.args.gasOption, tt.args.limits)
			if !tt.wantErr(t, err, fmt.Sprintf("NewPath(%v, %v, %v, %v, %v, %v, %v, %v, %v)", tt.args.poolBucket, tt.args.poolAddresses, tt.args.tokens, tt.args.tokenAmountIn, tt.args.tokenOut, tt.args.tokenOutPrice, tt.args.tokenOutDecimals, tt.args.gasOption, tt.args.limits)) {
				return
			}
		})
	}
}
