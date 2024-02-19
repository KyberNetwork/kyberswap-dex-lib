package spfa

import (
	"context"
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/KyberNetwork/router-service/pkg/mempool"
)

func TestSpfaFinder(t *testing.T) {
	ctx := context.TODO()
	var (
		nTokens = 100
		nPools  = 2000
	)
	tokenByAddress := valueobject.GenerateRandomTokenByAddress(nTokens)
	var tokenAddressList []string
	for tokenAddress := range tokenByAddress {
		tokenAddressList = append(tokenAddressList, tokenAddress)
	}
	priceUSDByAddress := valueobject.GenerateRandomPriceUSDByAddress(tokenAddressList)
	poolByAddress, err := valueobject.GenerateRandomPoolByAddress(nPools, tokenAddressList)
	assert.Nil(t, err)
	tokenToPoolAddress := make(map[string]*types.AddressList)
	for poolAddress, pool := range poolByAddress {

		for _, tokenAddress := range pool.GetTokens() {
			if _, ok := tokenToPoolAddress[tokenAddress]; !ok {
				tokenToPoolAddress[tokenAddress] = mempool.AddressListPool.Get().(*types.AddressList)
			}
			tokenToPoolAddress[tokenAddress].AddAddress(ctx, poolAddress)
		}
	}
	var (
		tokenIn  = tokenAddressList[valueobject.RandInt(0, nTokens)]
		tokenOut = tokenAddressList[valueobject.RandInt(0, nTokens)]
	)

	input := findroute.Input{
		TokenInAddress:   tokenIn,
		TokenOutAddress:  tokenOut,
		AmountIn:         big.NewInt(1_000_000_000),
		GasPrice:         big.NewFloat(8654684620),
		GasTokenPriceUSD: 1500,
		GasInclude:       true,
	}
	data := findroute.NewFinderData(ctx, tokenByAddress, priceUSDByAddress, &types.FindRouteState{
		Pools:     poolByAddress,
		SwapLimit: nil,
	})

	finder := NewDefaultSPFAFinder()
	routes, err := finder.Find(context.TODO(), input, data)
	assert.Nil(t, err)
	assert.Len(t, routes, 1)
	route := routes[0]
	assert.NotNil(t, route)
	fmt.Println("amountInUSD", route.Input.AmountUsd)
	fmt.Println("amountOutUSD", route.Output.AmountUsd)
	fmt.Println("number of paths on best route:", len(route.Paths))
	for _, path := range route.Paths {
		fmt.Println("path length", len(path.PoolAddresses))
		fmt.Print("pool on path ")
		for _, poolAddress := range path.PoolAddresses {
			fmt.Print(poolAddress, " ")
		}
		fmt.Println()
	}
}
