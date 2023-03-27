package common

import (
	"context"
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	poolPkg "github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/core/pool"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/usecase/findroute"
)

func TestGenKthBestPaths(t *testing.T) {
	t.Run("stress test GenKthBestPaths", func(t *testing.T) {
		var (
			nTokens                  = 100
			nPools                   = 2000
			maxHop            uint32 = 3
			maxPathToGenerate uint32 = 5
		)
		tokenByAddress := GenerateRandomTokenByAddress(nTokens)
		var tokenAddressList []string
		for tokenAddress := range tokenByAddress {
			tokenAddressList = append(tokenAddressList, tokenAddress)
		}
		priceUSDByAddress := GenerateRandomPriceUSDByAddress(tokenAddressList)
		poolByAddress, err := GenerateRandomPoolByAddress(nPools, tokenAddressList)
		assert.Nil(t, err)
		tokenToPoolAddress := make(map[string][]string)
		for poolAddress, pool := range poolByAddress {
			for _, tokenAddress := range pool.GetTokens() {
				tokenToPoolAddress[tokenAddress] = append(tokenToPoolAddress[tokenAddress], poolAddress)
			}
		}
		var (
			tokenIn  = tokenAddressList[RandInt(0, nTokens)]
			tokenOut = tokenAddressList[RandInt(0, nTokens)]
		)
		for tokenIn == tokenOut {
			tokenOut = tokenAddressList[RandInt(0, nTokens)]
		}
		minHopToTokenOut, err := MinHopsToTokenOut(poolByAddress, tokenByAddress, tokenToPoolAddress, tokenOut)
		assert.Nil(t, err)
		input := findroute.Input{
			TokenInAddress:   tokenIn,
			TokenOutAddress:  tokenOut,
			GasPrice:         big.NewFloat(8654684620),
			GasTokenPriceUSD: 1500,
			GasInclude:       true,
		}
		data := findroute.FinderData{
			PoolByAddress:     poolByAddress,
			TokenByAddress:    tokenByAddress,
			PriceUSDByAddress: priceUSDByAddress,
		}
		tokenAmountIn := poolPkg.TokenAmount{
			Token:  tokenIn,
			Amount: big.NewInt(1_000_000_000),
		}
		paths, err := GenKthBestPaths(
			context.TODO(),
			input, data, tokenAmountIn,
			tokenToPoolAddress, minHopToTokenOut,
			maxHop, maxPathToGenerate,
		)
		assert.Nil(t, err)
		fmt.Println("tokenIn", tokenIn)
		fmt.Println("tokenOut", tokenOut)
		fmt.Println("number of generated paths", len(paths))
		for _, path := range paths {
			fmt.Println("path length", len(path.Pools))
			fmt.Println("output:", path.Output.Amount, path.Output.AmountUsd)
			fmt.Println(path.Tokens)
		}
	})
}
