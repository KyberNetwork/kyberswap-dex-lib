package uniswap

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

func TestUniswapFinder(t *testing.T) {
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
			tokenToPoolAddress[tokenAddress].AddAddress(poolAddress)
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
	data := findroute.NewFinderData(tokenByAddress, priceUSDByAddress, &types.FindRouteState{
		Pools:     poolByAddress,
		SwapLimit: nil,
	})
	finder := NewDefaultUniswapFinder()
	routes, err := finder.Find(context.TODO(), input, data)
	assert.Nil(t, err)
	assert.GreaterOrEqual(t, len(routes), 1)
	for _, route := range routes {
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
		fmt.Println()
	}
}

func BenchmarkUniswapFinder(b *testing.B) {
	var tests = []struct {
		nPools, nTokens int
	}{
		{nPools: 500, nTokens: 100},
		{nPools: 2000, nTokens: 100},
		{nPools: 5000, nTokens: 200},
	}
	finder := NewDefaultUniswapFinder()

	for _, test := range tests {
		var (
			nPools  = test.nPools
			nTokens = test.nTokens
		)
		b.Run(fmt.Sprintf("nPools_%d_nTokens_%d", nPools, nTokens), func(b *testing.B) {
			tokenByAddress := valueobject.GenerateRandomTokenByAddress(nTokens)
			var tokenAddressList []string
			for tokenAddress := range tokenByAddress {
				tokenAddressList = append(tokenAddressList, tokenAddress)
			}
			priceUSDByAddress := valueobject.GenerateRandomPriceUSDByAddress(tokenAddressList)
			poolByAddress, err := valueobject.GenerateRandomPoolByAddress(nPools, tokenAddressList)
			assert.Nil(b, err)
			tokenToPoolAddress := make(map[string][]string)
			for poolAddress, pool := range poolByAddress {
				for _, tokenAddress := range pool.GetTokens() {
					tokenToPoolAddress[tokenAddress] = append(tokenToPoolAddress[tokenAddress], poolAddress)
				}
			}
			var (
				tokenIn  = tokenAddressList[valueobject.RandInt(0, nTokens)]
				tokenOut = tokenAddressList[valueobject.RandInt(0, nTokens)]
			)
			for tokenIn == tokenOut {
				tokenOut = tokenAddressList[valueobject.RandInt(0, nTokens)]
			}
			input := findroute.Input{
				TokenInAddress:   tokenIn,
				TokenOutAddress:  tokenOut,
				AmountIn:         big.NewInt(int64(valueobject.RandInt(100_000_000, 1_000_000_000))),
				GasPrice:         big.NewFloat(8654684620),
				GasTokenPriceUSD: 1500,
				GasInclude:       true,
			}
			data := findroute.FinderData{
				PoolBucket:        valueobject.NewPoolBucket(poolByAddress),
				TokenByAddress:    tokenByAddress,
				PriceUSDByAddress: priceUSDByAddress,
			}
			_, err = finder.Find(context.TODO(), input, data)
			assert.Nil(b, err)
		})
	}
}
