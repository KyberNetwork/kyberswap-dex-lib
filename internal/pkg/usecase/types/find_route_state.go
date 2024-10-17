package types

import (
	"context"
	"math"
	"sync"

	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"

	"github.com/KyberNetwork/router-service/pkg/logger"
)

// FindRouteState enclose the data we need for a findRoute rquest
type FindRouteState struct {
	// map PoolAddress - IPoolSimulator implementation
	Pools map[string]poolpkg.IPoolSimulator
	// map LimitType-SwapLimit
	SwapLimit map[string]poolpkg.SwapLimit
	// PoolsStorageID represents the last published pools
	PublishedPoolsStorageID string
}

type pooledSlice[T any] struct {
	data []T
}

func (s *pooledSlice[T]) reset() {
	// truncate slice and reuse the underlying buffer https://go.dev/play/p/AwRq45JcRQZ
	s.data = s.data[:0]
}

type (
	indexedAddressList = pooledSlice[uint16]
	addressList        = pooledSlice[string]
)

var (
	indexedAddressListPool = sync.Pool{
		New: func() interface{} {
			return &indexedAddressList{
				data: make([]uint16, 100),
			}
		},
	}
	poolAddrsPool = sync.Pool{
		New: func() interface{} {
			return &addressList{
				data: []string{},
			}
		},
	}
)

func indexedAddressListPoolGet() *indexedAddressList {
	l := indexedAddressListPool.Get().(*indexedAddressList)
	l.reset()
	return l
}

func poolAddrsPoolGet() *addressList {
	l := poolAddrsPool.Get().(*addressList)
	l.reset()
	return l
}

func indexedAddressListAppendUint16(l *indexedAddressList, v uint16)      { l.data = append(l.data, v) }
func indexedAddressListGetUint16(l *indexedAddressList, index int) uint16 { return l.data[index] }

func indexedAddressListAppendUint32(l *indexedAddressList, v uint32) {
	l.data = append(l.data, uint16(v>>16), uint16(v))
}

func indexedAddressListGetUint32(l *indexedAddressList, index int) uint32 {
	return (uint32(l.data[2*index]) << 16) | uint32(l.data[2*index+1])
}

// TokenToPoolAddressMap is a memory-optimized structure that holds tokens and theirs corresponding pool addresses.
// Since a pool address belongs to many tokens, this struct stores pool addresses by their 2-byte or 4-byte index (in a static list) instead of their full string (66-byte).
type TokenToPoolAddressMap struct {
	poolAddressList *addressList
	// 0 < addressIndexLists[token][i] < len(poolAddrs)
	addressIndexLists map[string]*indexedAddressList
	// if true then 0 < (poolAddrLists[token][2*i] << 16 | poolAddrLists[token][2*i+1]) < len(poolAddrs)
	use32BitIndex bool
}

// MakeTokenToPoolAddressMapFromPools for a list of pool:[tokenAddress] as
//
//	{
//	 "pool0":["token0","token1"],
//	 "pool1":["token1","token2"],
//	 "pool2":["token2","token1"],
//	}
//
// make a TokenToPoolAddress map whose poolAddressList shall be ["pool0","pool1","pool2"]
// and addressIndexLists shall be
//
//	map{
//	  "token0": [0]
//	  "token1": [0,1,2]
//	  "token2": [1,2]
//	}
//
// where values are pool indexes in poolAddressList.
//
// If the max pool index is larger than uint16, then we use 2 elements (element 2i and 2i + 1) in the uint16 slice to store indexes.
func MakeTokenToPoolAddressMapFromPools(pools map[string]poolpkg.IPoolSimulator) *TokenToPoolAddressMap {
	use32BitIndex := false
	if len(pools) >= math.MaxUint16 {
		if len(pools) >= math.MaxUint32 {
			panic("MakeTokenToPoolAddressMapFromPools: len(pools) exceeded math.MaxUint32")
		}
		use32BitIndex = true
	}

	addressIndexLists := make(map[string]*indexedAddressList)
	poolAddrs := poolAddrsPoolGet()
	for _, pool := range pools {
		poolAddrIndex := uint32(len(poolAddrs.data))
		poolAddrs.data = append(poolAddrs.data, pool.GetAddress())
		for _, tokenAddress := range pool.GetTokens() {
			if _, ok := addressIndexLists[tokenAddress]; !ok {
				addressIndexLists[tokenAddress] = indexedAddressListPoolGet()
			}
			if use32BitIndex {
				indexedAddressListAppendUint32(addressIndexLists[tokenAddress], poolAddrIndex)
			} else {
				indexedAddressListAppendUint16(addressIndexLists[tokenAddress], uint16(poolAddrIndex))
			}
		}
	}

	return &TokenToPoolAddressMap{
		poolAddressList:   poolAddrs,
		addressIndexLists: addressIndexLists,
		use32BitIndex:     use32BitIndex,
	}
}

func (m *TokenToPoolAddressMap) NumPools(token string) int {
	if m == nil {
		return 0
	}
	if _, ok := m.addressIndexLists[token]; !ok {
		return 0
	}
	if m.use32BitIndex {
		return len(m.addressIndexLists[token].data) / 2
	}
	return len(m.addressIndexLists[token].data)
}

// GetPoolAddressAt gets `i`-th pool address of token `token`.
// The index `i` is stored efficiently using an uint16 slice.
// for indexes pool < 16, we store the index as normal in addressIndexLists (as previous example MakeTokenToPoolAddressMapFromPools)
// map{
//
//	  "token0": [0]
//	  "token1": [0,1,2]
//	  "token2": [1,2]
//	  ...
//	}
//
// if the indexes of pools is > MaxUint16, the addressIndexLists will store 2 number: the first 16 bit and the remainder.
// for example: a total of 1_200_000 pools, the addressIndexLists map look like: {"token0" : [1114111],....}.
// the poolIndex 1114111>maxUint16, hence addressIndexLists look like:
// addressIndexLists = map { "token0": [16, 65535] }
// to look for the first pool of token0, reverse the process.
func (m *TokenToPoolAddressMap) GetPoolAddressAt(token string, i int) string {
	if i >= m.NumPools(token) {
		return ""
	}
	var index uint32
	if m.use32BitIndex {
		index = indexedAddressListGetUint32(m.addressIndexLists[token], i)
	} else {
		index = uint32(indexedAddressListGetUint16(m.addressIndexLists[token], i))
	}
	return m.poolAddressList.data[index]
}

// ReleaseResources returns address lists from the pool.
func (m *TokenToPoolAddressMap) ReleaseResources() {
	poolAddrsPool.Put(m.poolAddressList)
	for token, al := range m.addressIndexLists {
		indexedAddressListPool.Put(al)
		m.addressIndexLists[token] = nil
	}
}

// AddressList will store the list of Pool to use for Adjacent map.
// It is managed by mempool and once it's out of scope, call memPool.ReturnAddressList() to return it.
// Note that the Arr underneath will expand to max N pool with N equal to number of best pools selected for each request.
// TrueLen denotes the real len of the AddressList, it is set to 0 when mempool need to zero-out the memory before re-using.
type AddressList struct {
	Arr     []string
	TrueLen int
}

// AddAddress is a bit hacky, we use the underlying arr to store the address
func (a *AddressList) AddAddress(ctx context.Context, address string) {
	if a.TrueLen == len(a.Arr) {
		// have to do append here
		a.Arr = append(a.Arr, address)
	} else if a.TrueLen < len(a.Arr) {
		a.Arr[a.TrueLen] = address
	} else {
		logger.Errorf(ctx, "AddressList TrueLen %d is greater than underlying a.Arr's len %d", a.TrueLen, len(a.Arr))
	}

	a.TrueLen++

}

type StateAfterSwap struct {
	UpdatedBalancePools map[string]poolpkg.IPoolSimulator
	UpdatedSwapLimits   map[string]poolpkg.SwapLimit
}
