package common

import (
	"fmt"
	"math/rand"
	"strconv"

	poolPkg "github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/core/pool"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/core/uni"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/entity"
)

func GenerateRandomTokenByAddress(nTokens int) map[string]entity.Token {
	var (
		tokens       = make(map[string]entity.Token)
		tokenAddress string
		tokenDecimal uint8
	)
	for i := 0; i < nTokens; i++ {
		tokenAddress = "token" + strconv.Itoa(i)
		tokenDecimal = uint8(RandInt(6, 10))
		tokens[tokenAddress] = entity.Token{
			Address:  tokenAddress,
			Decimals: tokenDecimal,
		}
	}
	return tokens
}

func GenerateRandomPriceUSDByAddress(tokenAddressList []string) map[string]float64 {
	var prices = make(map[string]float64)
	for _, tokenAddress := range tokenAddressList {
		prices[tokenAddress] = RandFloat(1, 100)
	}
	return prices
}

func GenerateRandomPoolByAddress(nPools int, tokenAddressList []string) (map[string]poolPkg.IPool, error) {
	if nPools < len(tokenAddressList)-1 {
		return nil, fmt.Errorf("not enough poolByAddress to make a connected graph")
	}
	var (
		nTokens                                   = len(tokenAddressList)
		poolByAddress                             = make(map[string]poolPkg.IPool)
		data                                      entity.Pool
		swap                                      poolPkg.IPool
		swapAddress, tokenAddress0, tokenAddress1 string
		swapFee                                   float64
		err                                       error
	)
	for i := 0; i < nPools; i++ {
		swapAddress = "pool " + strconv.Itoa(i)
		if i < nTokens-1 {
			// build tree
			tokenAddress0 = tokenAddressList[i]
			tokenAddress1 = tokenAddressList[RandInt(i+1, nTokens)]
		} else {
			for atLeastOnce := true; atLeastOnce; atLeastOnce = tokenAddress0 == tokenAddress1 {
				tokenAddress0 = tokenAddressList[RandInt(0, nTokens)]
				tokenAddress1 = tokenAddressList[RandInt(0, nTokens)]
			}
		}
		swapFee = RandFloat(0, 0.05)
		data = entity.Pool{
			Address: swapAddress,
			SwapFee: swapFee,
			Tokens: entity.PoolTokens{
				&entity.PoolToken{Address: tokenAddress0},
				&entity.PoolToken{Address: tokenAddress1},
			},
			Reserves: entity.PoolReserves{
				strconv.Itoa(RandInt(1_000_000, 1_000_000_000)),
				strconv.Itoa(RandInt(1_000_000, 1_000_000_000)),
			},
		}
		// using uni pool for simplicity
		if swap, err = uni.NewPool(data); err != nil {
			return nil, err
		}
		poolByAddress[swapAddress] = swap
	}
	return poolByAddress, nil
}

// RandInt return random integer within [min,max)
func RandInt(min int, max int) int {
	return min + rand.Intn(max-min)
}

// RandFloat return random float within [min,max)
func RandFloat(min float64, max float64) float64 {
	return min + rand.Float64()*(max-min)
}
