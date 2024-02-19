package findroute

import (
	"context"
	"fmt"
	"testing"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/pkg/mempool"
)

func BenchmarkTokenToPoolAddressWithoutMemPool(b *testing.B) {

	perRequestPoolsToTokens := make(map[string][]string)
	for i := 0; i < 100000; i++ {
		key := fmt.Sprintf("address%d", i)
		perRequestPoolsToTokens[key] = []string{fmt.Sprintf("token%d", i%990), fmt.Sprintf("token%d", i%560)}
	}

	for i := 0; i < b.N; i++ {
		tokenToPoolAddress := make(map[string][]string)
		for poolAddress, tokens := range perRequestPoolsToTokens {
			for _, fromToken := range tokens {
				tokenToPoolAddress[fromToken] = append(tokenToPoolAddress[fromToken], poolAddress)
			}
		}
	}
}

func BenchmarkTokenToPoolAddressWithMemPool(b *testing.B) {
	ctx := context.TODO()

	perRequestPoolsByAddress := make(map[string][]string)
	for i := 0; i < 100000; i++ {
		key := fmt.Sprintf("address%d", i)
		perRequestPoolsByAddress[key] = []string{fmt.Sprintf("token%d", i%990), fmt.Sprintf("token%d", i%560)}
	}

	for i := 0; i < b.N; i++ {
		tokenToPoolAddress := make(map[string]*types.AddressList)
		for key, tokens := range perRequestPoolsByAddress {
			for _, tokenAddress := range tokens {
				if _, ok := tokenToPoolAddress[tokenAddress]; !ok {
					tokenToPoolAddress[tokenAddress] = mempool.AddressListPool.Get().(*types.AddressList)
				}
				tokenToPoolAddress[tokenAddress].AddAddress(ctx, key)
			}
		}

		for key := range tokenToPoolAddress {
			mempool.ReturnAddressList(tokenToPoolAddress[key])
		}
	}
}
