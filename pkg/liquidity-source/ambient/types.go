package ambient

import (
	"bytes"
	"fmt"
	"math/big"
	"slices"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
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
	return []byte(p.String()), nil
}

func (p *TokenPair) UnmarshalText(text []byte) error {
	parts := strings.SplitN(string(text), ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("expect <base>:<quote>")
	}
	p.Base = common.HexToAddress(parts[0])
	p.Quote = common.HexToAddress(parts[1])
	return nil
}

type StaticExtra struct {
	NativeTokenAddress common.Address `json:"nativeTokenAddress"`
	PoolIdx            uint64         `json:"poolIdx"`
	SwapDex            common.Address `json:"swapDex"`
	TickRange          int32          `json:"tickRange,omitempty"`
}

type TokenPairInfo struct {
	PoolIdx *big.Int      `json:"poolIdx"`
	State   *TrackerExtra `json:"state,omitempty"`
}

type Extra struct {
	TokenPairs map[TokenPair]*TokenPairInfo `json:"tokenPairs"`
}

type Meta struct {
	SwapDex common.Address `json:"swapDex"`
	Base    common.Address `json:"base"`
	Quote   common.Address `json:"quote"`
	PoolIdx *big.Int       `json:"poolIdx"`
}

type Gas struct {
	BaseGas int64
}

var defaultGas = Gas{BaseGas: 250_000}

// NTokenPool extends pool.Pool with pair-aware adjacency logic for Ambient's
// singleton contract model.
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
			adjs = append(adjs, hexutil.Encode(pair.Quote[:]))
		}
		if pair.Quote == addr {
			if pair.Base == NativeTokenPlaceholderAddress {
				adjs = append(adjs, hexutil.Encode(p.nativeTokenAddress[:]))
			} else {
				adjs = append(adjs, hexutil.Encode(pair.Base[:]))
			}
		}
	}
	p.cache.Store(addr, adjs)

	return adjs
}

func (p *NTokenPool) CanSwapFrom(address string) []string { return p.CanSwapTo(address) }

func (p *NTokenPool) GetPair(tokenIn, tokenOut common.Address) (TokenPair, bool) {
	base, quote := tokenIn, tokenOut
	if base == p.nativeTokenAddress {
		base = NativeTokenPlaceholderAddress
	}
	if quote == p.nativeTokenAddress {
		quote = NativeTokenPlaceholderAddress
	}
	if bytes.Compare(base[:], quote[:]) > 0 {
		base, quote = quote, base
	}
	pair := TokenPair{Base: base, Quote: quote}
	if slices.Contains(p.pairs, pair) {
		return pair, true
	}
	return TokenPair{}, false
}
