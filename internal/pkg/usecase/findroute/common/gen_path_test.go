package common

import (
	"context"
	"fmt"
	"math/big"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/pooltypes"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/KyberNetwork/router-service/pkg/mempool"
)

func TestGenKthBestPaths(t *testing.T) {
	ctx := context.TODO()
	t.Run("stress test GenKthBestPaths", func(t *testing.T) {
		var (
			nTokens                  = 100
			nPools                   = 2000
			maxHop            uint32 = 3
			maxPathToGenerate uint32 = 5
			maxPathToReturn   uint32 = 5
		)
		tokenByAddress := valueobject.GenerateRandomTokenByAddress(nTokens)
		var tokenAddressList []string
		for tokenAddress := range tokenByAddress {
			tokenAddressList = append(tokenAddressList, tokenAddress)
		}
		priceUSDByAddress := valueobject.GenerateRandomPriceUSDByAddress(tokenAddressList)
		poolByAddress, err := valueobject.GenerateRandomPoolByAddress(nPools, tokenAddressList, pooltypes.PoolTypes.UniswapV2)
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
		for tokenIn == tokenOut {
			tokenOut = tokenAddressList[valueobject.RandInt(0, nTokens)]
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
			PoolBucket:        valueobject.NewPoolBucket(poolByAddress),
			TokenByAddress:    tokenByAddress,
			PriceUSDByAddress: priceUSDByAddress,
		}
		tokenAmountIn := poolpkg.TokenAmount{
			Token:  tokenIn,
			Amount: big.NewInt(1_000_000_000),
		}
		paths, err := GenKthBestPaths(
			context.TODO(),
			input, data, tokenAmountIn,
			minHopToTokenOut,
			maxHop, maxPathToGenerate, maxPathToReturn,
		)
		assert.Nil(t, err)
		fmt.Println("tokenIn", tokenIn)
		fmt.Println("tokenOut", tokenOut)
		fmt.Println("number of generated paths", len(paths))
		for _, path := range paths {
			fmt.Println("path length", len(path.PoolAddresses))
			fmt.Println("output:", path.Output.Amount, path.Output.AmountUsd)
			fmt.Println(path.Tokens)
		}
	})
}
