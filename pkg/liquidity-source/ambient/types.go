package ambient

import (
	"fmt"
	"math/big"
	"strings"
	"sync"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/ethereum/go-ethereum/common"
)

type SubgraphPoolsResponse struct {
	Pools []SubgraphPool `json:"pools"`
}

type SubgraphPool struct {
	ID          string `json:"id"`
	BlockCreate string `json:"blockCreate"`
	TimeCreate  uint64 `json:"timeCreate,string"`
	Base        string `json:"base"`
	Quote       string `json:"quote"`
	PoolIdx     string `json:"poolIdx"`
}

type PoolListUpdaterMetadata struct {
	LastCreateTime uint64 `json:"lastCreateTime"`
}

type TokenPair struct {
	Base  common.Address
	Quote common.Address
}

func (p TokenPair) String() string {
	return strings.ToLower(fmt.Sprintf("%s:%s", p.Base, p.Quote))
}

func (p TokenPair) MarshalText() ([]byte, error) {
	s := strings.ToLower(fmt.Sprintf("%s:%s", p.Base, p.Quote))
	return []byte(s), nil
}

func (p *TokenPair) UnmarshalText(text []byte) error {
	splited := strings.SplitN(string(text), ":", 2)
	if len(splited) != 2 {
		return fmt.Errorf("expect <base>:<quote>")
	}
	p.Base = common.HexToAddress(splited[0])
	p.Quote = common.HexToAddress(splited[1])
	return nil
}

type StaticExtra struct {
	// ERC20 native wrapper token
	NativeTokenAddress string `json:"nativeTokenAddress"`
}

type TokenPairInfo struct {
	SqrtPriceX64 string `json:"sqrtPriceX64"`
	Liquidity    string `json:"liquidity"`
	// we assume that there is 1 pool per token pair
	PoolIdx *big.Int `json:"poolIdx"`
}

type Extra struct {
	TokenPairs map[TokenPair]*TokenPairInfo `json:"tokenPairs"`
}

// NTokenPool is extended from pool.Pool with custom CanSwapTo()
type NTokenPool struct {
	pool.Pool

	pairs              []TokenPair
	nativeTokenAddress common.Address
	cache              sync.Map `msgpack:"-"` // map[common.Address][]string
}

func NewNTokenPool(pool pool.Pool, pairs []TokenPair, nativeTokenAddress common.Address) *NTokenPool {
	return &NTokenPool{
		Pool:               pool,
		pairs:              pairs,
		nativeTokenAddress: nativeTokenAddress,
	}
}

func (p *NTokenPool) CanSwapTo(address string) []string {
	addr := common.HexToAddress(address)

	if adjs, ok := p.cache.Load(addr); ok {
		return adjs.([]string)
	}

	var adjs []string
	for _, pair := range p.pairs {
		if pair.Base == addr || (pair.Base == NativeTokenPlaceholderAddress && addr == p.nativeTokenAddress) {
			adjs = append(adjs, strings.ToLower(pair.Quote.Hex()))
		}
		if pair.Quote == addr {
			if pair.Base == NativeTokenPlaceholderAddress {
				adjs = append(adjs, strings.ToLower(p.nativeTokenAddress.Hex()))
			} else {
				adjs = append(adjs, strings.ToLower(pair.Base.Hex()))
			}
		}
	}
	p.cache.Store(addr, adjs)

	return adjs
}
