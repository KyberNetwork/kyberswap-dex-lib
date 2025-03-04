package types

import (
	"fmt"
	"math/big"
	"os"
	"sync"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/pooltypes"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

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

func TestTokenToPoolAddressMap(t *testing.T) {
	t.Run("pool or token not in map", func(t *testing.T) {
		pools := map[string]poolpkg.IPoolSimulator{
			"p1": &testPoolSimulator{"p1", []string{"t1", "t2"}},
			"p2": &testPoolSimulator{"p2", []string{"t2", "t3"}},
			"p3": &testPoolSimulator{"p3", []string{"t3", "t1"}},
			"p4": &testPoolSimulator{"p4", []string{"t1", "t2", "t3"}},
		}
		m := MakeTokenToPoolAddressMapFromPools(pools)
		defer m.ReleaseResources()
		require.Empty(t, m.NumPools("t99"))
		require.Empty(t, m.GetPoolAddressAt("t99", 0))
		require.Empty(t, m.GetPoolAddressAt("t1", 3))
	})
	t.Run("len(pools) < MaxUint16", func(t *testing.T) {
		for i := 0; i < 10; i++ {
			pools := map[string]poolpkg.IPoolSimulator{
				"p1": &testPoolSimulator{"p1", []string{"t1", "t2"}},
				"p2": &testPoolSimulator{"p2", []string{"t2", "t3"}},
				"p3": &testPoolSimulator{"p3", []string{"t3", "t1"}},
				"p4": &testPoolSimulator{"p4", []string{"t1", "t2", "t3"}},
			}
			expected := map[string]sets.Set[string]{
				"t1": sets.New("p1", "p3", "p4"),
				"t2": sets.New("p1", "p2", "p4"),
				"t3": sets.New("p2", "p3", "p4"),
			}
			m := MakeTokenToPoolAddressMapFromPools(pools)
			defer m.ReleaseResources()
			for token, poolAddrSet := range expected {
				actualNumPools := m.NumPools(token)
				require.Equal(t, poolAddrSet.Len(), actualNumPools)
				for i := 0; i < actualNumPools; i++ {
					require.True(t, poolAddrSet.Has(m.GetPoolAddressAt(token, i)))
				}
			}
		}
	})
	t.Run("len(pools) >= MaxUint16", func(t *testing.T) {
		nTokens := 100_000
		nPools := 200_000
		tokenByAddress := valueobject.GenerateRandomTokenByAddress(nTokens)
		var tokenAddressList []string
		for tokenAddress := range tokenByAddress {
			tokenAddressList = append(tokenAddressList, tokenAddress)
		}
		poolByAddress, err := valueobject.GenerateRandomPoolByAddress(nPools, tokenAddressList, pooltypes.PoolTypes.UniswapV2)
		require.NoError(t, err)
		tokenToPoolAddress := MakeTokenToPoolAddressMapFromPools(poolByAddress)
		defer tokenToPoolAddress.ReleaseResources()

		expected := make(map[string]sets.Set[string])
		for _, pool := range poolByAddress {
			for _, token := range pool.GetTokens() {
				if _, ok := expected[token]; !ok {
					expected[token] = sets.Set[string]{}
				}
				expected[token].Insert(pool.GetAddress())
			}
		}

		for token, poolAddrSet := range expected {
			actualNumPools := tokenToPoolAddress.NumPools(token)
			require.Equal(t, poolAddrSet.Len(), actualNumPools)
			for i := 0; i < actualNumPools; i++ {
				require.True(t, poolAddrSet.Has(tokenToPoolAddress.GetPoolAddressAt(token, i)))
			}
		}
	})
}

func TestTokenToPoolAddressMapConcurrent(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skipf("compute intensive test, should only run on local machine")
	}
	var (
		wg sync.WaitGroup
		ch = make(chan struct{}, 100)
	)
	for k := 0; k < 10_000; k++ {
		wg.Add(1)
		k := k
		go func() {
			ch <- struct{}{}
			defer func() {
				<-ch
				fmt.Printf("k=%d\n", k)
				wg.Done()
			}()
			nTokens := 100
			nPools := 2_000
			tokenByAddress := valueobject.GenerateRandomTokenByAddress(nTokens)
			var tokenAddressList []string
			for tokenAddress := range tokenByAddress {
				tokenAddressList = append(tokenAddressList, tokenAddress)
			}
			poolByAddress, err := valueobject.GenerateRandomPoolByAddress(nPools, tokenAddressList, pooltypes.PoolTypes.UniswapV2)
			require.NoError(t, err)
			tokenToPoolAddress := MakeTokenToPoolAddressMapFromPools(poolByAddress)
			defer tokenToPoolAddress.ReleaseResources()

			expected := make(map[string]sets.Set[string])
			for _, pool := range poolByAddress {
				for _, token := range pool.GetTokens() {
					if _, ok := expected[token]; !ok {
						expected[token] = sets.Set[string]{}
					}
					expected[token].Insert(pool.GetAddress())
				}
			}

			for token, poolAddrSet := range expected {
				actualNumPools := tokenToPoolAddress.NumPools(token)
				require.Equal(t, poolAddrSet.Len(), actualNumPools)
				for i := 0; i < actualNumPools; i++ {
					require.True(t, poolAddrSet.Has(tokenToPoolAddress.GetPoolAddressAt(token, i)))
				}
			}
		}()
	}
	wg.Wait()
}

func TestIndexedAddressListExample(t *testing.T) {
	al := &indexedAddressList{}
	indexedAddressListAppendUint32(al, 1114111)

	poolAddrs := &addressList{data: make([]string, 1_200_000)}
	poolAddrs.data[1114111] = "pool_1114111"

	m := &TokenToPoolAddressMap{
		poolAddressList: poolAddrs,
		addressIndexLists: map[string]*indexedAddressList{
			"token0": al,
		},
		use32BitIndex: true,
	}

	require.Equal(t, []uint16{16, 65535}, al.data)
	require.Equal(t, uint32(1114111), indexedAddressListGetUint32(al, 0))
	require.Equal(t, "pool_1114111", m.GetPoolAddressAt("token0", 0))
}
