package findroute

import (
	"context"
	"fmt"
	"math/big"
	"testing"

	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"

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

	b.Run("benchmark", func(b *testing.B) {
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
	})
}

type testPoolSimulator struct {
	addr   string
	tokens []string
}

func (p *testPoolSimulator) GetTokens() []string { return p.tokens }
func (p *testPoolSimulator) GetAddress() string  { return p.addr }

func (*testPoolSimulator) CalcAmountOut(poolpkg.CalcAmountOutParams) (*poolpkg.CalcAmountOutResult, error) {
	panic("unimplemented")
}
func (*testPoolSimulator) CloneState() poolpkg.IPoolSimulator        { panic("unimplemented") }
func (*testPoolSimulator) UpdateBalance(poolpkg.UpdateBalanceParams) { panic("unimplemented") }
func (*testPoolSimulator) CanSwapTo(string) []string                 { panic("unimplemented") }
func (*testPoolSimulator) CanSwapFrom(string) []string               { panic("unimplemented") }
func (*testPoolSimulator) GetReserves() []*big.Int                   { panic("unimplemented") }
func (*testPoolSimulator) GetExchange() string                       { panic("unimplemented") }
func (*testPoolSimulator) GetType() string                           { panic("unimplemented") }
func (*testPoolSimulator) GetMetaInfo(string, string) interface{}    { panic("unimplemented") }
func (*testPoolSimulator) GetTokenIndex(string) int                  { panic("unimplemented") }
func (*testPoolSimulator) CalculateLimit() map[string]*big.Int       { panic("unimplemented") }
func (*testPoolSimulator) GetApprovalAddress(string, string) string  { panic("unimplemented") }

/*
$ go test -benchmem -run=^$ -bench "^(BenchmarkTokenToPoolAddressWithMemPool|BenchmarkMakeTokenToPoolAddressMapFromPools)$" github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute -race -v -count 1
goos: darwin
goarch: amd64
pkg: github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute
cpu: Intel(R) Core(TM) i7-9750H CPU @ 2.60GHz
BenchmarkTokenToPoolAddressWithMemPool
BenchmarkTokenToPoolAddressWithMemPool/benchmark
BenchmarkTokenToPoolAddressWithMemPool/benchmark-12                   13          82749966 ns/op         2797734 B/op        984 allocs/op
BenchmarkMakeTokenToPoolAddressMapFromPools
BenchmarkMakeTokenToPoolAddressMapFromPools/benchmark
BenchmarkMakeTokenToPoolAddressMapFromPools/benchmark-12              10         105188065 ns/op         5263936 B/op       1256 allocs/op
PASS
ok      github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute   6.123s
*/
func BenchmarkMakeTokenToPoolAddressMapFromPools(b *testing.B) {
	perRequestPoolsByAddress := make(map[string]poolpkg.IPoolSimulator)
	for i := 0; i < 100000; i++ {
		key := fmt.Sprintf("address%d", i)
		tokens := []string{fmt.Sprintf("token%d", i%990), fmt.Sprintf("token%d", i%560)}
		perRequestPoolsByAddress[key] = &testPoolSimulator{key, tokens}
	}

	b.Run("benchmark", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			tokenToPoolAddress := types.MakeTokenToPoolAddressMapFromPools(perRequestPoolsByAddress)
			tokenToPoolAddress.ReleaseResources()
		}
	})
}
