package valueobject

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/pooltypes"
	kyberpmm "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/kyber-pmm"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/uniswap"
)

func GenerateRandomTokenByAddress(nTokens int) map[string]*entity.Token {
	var (
		tokens       = make(map[string]*entity.Token)
		tokenAddress string
		tokenDecimal uint8
	)
	for i := 0; i < nTokens; i++ {
		tokenAddress = "token" + strconv.Itoa(i)
		tokenDecimal = uint8(RandInt(6, 10))
		tokens[tokenAddress] = &entity.Token{
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

// GenerateUniv2PoolByTokenAddress generate a tokenAddressList[i]-tokenAddressList[i+1] pools.
// all the pool will have the same reserve 1_000_000 - 1_000_000
func GenerateUniv2PoolByTokenAddress(tokenAddressList []string) (map[string]poolpkg.IPoolSimulator, error) {
	if len(tokenAddressList) <= 1 {
		return nil, fmt.Errorf("tokenAddressList must has at least 2 tokens")
	}
	var (
		poolByAddress          = make(map[string]poolpkg.IPoolSimulator)
		swapAddress, nextToken string
		swapFee                = 0.0
		data                   entity.Pool
		swap                   poolpkg.IPoolSimulator
		err                    error
	)
	for i := 0; i < len(tokenAddressList); i++ {
		if i == len(tokenAddressList)-1 {
			//Gen a tokenN-token0 pool
			nextToken = tokenAddressList[0]
		} else {
			nextToken = tokenAddressList[i+1]
		}
		swapAddress = "pool_" + strconv.Itoa(i)
		data = entity.Pool{
			Address: swapAddress,
			SwapFee: swapFee,
			Tokens: entity.PoolTokens{
				&entity.PoolToken{Address: tokenAddressList[i]},
				&entity.PoolToken{Address: nextToken},
			},
			Reserves: entity.PoolReserves{
				strconv.Itoa(1_000_000),
				strconv.Itoa(1_000_000),
			},
			Type: "uniswap",
		}
		// using uni pool for simplicity
		if swap, err = uniswap.NewPoolSimulator(data); err != nil {
			return nil, err
		}
		poolByAddress[swapAddress] = swap
	}
	return poolByAddress, nil
}

func GenerateRandomPoolByAddress(nPools int, tokenAddressList []string, poolType string) (map[string]poolpkg.IPoolSimulator, error) {
	if nPools < len(tokenAddressList)-1 {
		return nil, fmt.Errorf("not enough poolByAddress to make a connected graph")
	}
	var (
		nTokens                                   = len(tokenAddressList)
		poolByAddress                             = make(map[string]poolpkg.IPoolSimulator)
		data                                      entity.Pool
		swap                                      poolpkg.IPoolSimulator
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
			Type: poolType,
		}
		// using uni pool for simplicity
		if swap, err = uniswap.NewPoolSimulator(data); err != nil {
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

func GenPMMPool(token1, token2 *entity.Token) (*kyberpmm.PoolSimulator, error) {
	var entityPool = entity.Pool{
		Type:      pooltypes.PoolTypes.KyberPMM,
		Exchange:  kyberpmm.DexTypeKyberPMM,
		Address:   strings.Join([]string{"kyber_pmm", token1.Address, token2.Address}, "_"),
		Timestamp: time.Now().Unix(),
		Tokens: []*entity.PoolToken{
			{
				Address:   token1.Address,
				Name:      token1.Name,
				Symbol:    token1.Symbol,
				Decimals:  token1.Decimals,
				Weight:    0,
				Swappable: true,
			},
			{
				Address:   token2.Address,
				Name:      token2.Name,
				Symbol:    token2.Symbol,
				Decimals:  token2.Decimals,
				Weight:    0,
				Swappable: true,
			}},
		StaticExtra: `{
		"pairID":            "ID",
		"baseTokenAddress":  "base",
		"quoteTokenAddress": "quote"}`,
		Reserves: []string{"123554545", "5555555"},
		Extra: `{
  "baseToQuotePriceLevels": [
    {
      "price": 100.5,
      "amount": 10.0
    },
    {
      "price": 101.2,
      "amount": 15.0
    }
  ],
  "quoteToBasePriceLevels": [
    {
      "price": 0.0098,
      "amount": 500.0
    },
    {
      "price": 0.0099,
      "amount": 700.0
    }
  ]
}`,
	}
	return kyberpmm.NewPoolSimulator(entityPool)
}
