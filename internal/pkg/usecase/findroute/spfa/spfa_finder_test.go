package spfa

import (
	"context"
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute/common"
)

func TestSpfaFinder(t *testing.T) {
	var (
		nTokens = 100
		nPools  = 2000
	)
	tokenByAddress := common.GenerateRandomTokenByAddress(nTokens)
	var tokenAddressList []string
	for tokenAddress := range tokenByAddress {
		tokenAddressList = append(tokenAddressList, tokenAddress)
	}
	priceUSDByAddress := common.GenerateRandomPriceUSDByAddress(tokenAddressList)
	poolByAddress, err := common.GenerateRandomPoolByAddress(nPools, tokenAddressList)
	assert.Nil(t, err)
	tokenToPoolAddress := make(map[string][]string)
	for poolAddress, pool := range poolByAddress {
		for _, tokenAddress := range pool.GetTokens() {
			tokenToPoolAddress[tokenAddress] = append(tokenToPoolAddress[tokenAddress], poolAddress)
		}
	}
	var (
		tokenIn  = tokenAddressList[common.RandInt(0, nTokens)]
		tokenOut = tokenAddressList[common.RandInt(0, nTokens)]
	)

	input := findroute.Input{
		TokenInAddress:   tokenIn,
		TokenOutAddress:  tokenOut,
		AmountIn:         big.NewInt(1_000_000_000),
		GasPrice:         big.NewFloat(8654684620),
		GasTokenPriceUSD: 1500,
		GasInclude:       true,
	}
	data := findroute.FinderData{
		PoolByAddress:     poolByAddress,
		TokenByAddress:    tokenByAddress,
		PriceUSDByAddress: priceUSDByAddress,
	}
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
		fmt.Println("path length", len(path.Pools))
		fmt.Print("pool on path ")
		for _, pool := range path.Pools {
			fmt.Print(pool.GetAddress(), " ")
		}
		fmt.Println()
	}
}
