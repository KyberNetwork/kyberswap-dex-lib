package flywheelfun

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

const (
	DexType       = "flywheel-fun"
	Factory       = "0xe7743b4039dbcd05c5242939aa8db274c65fcbfa"
	Settlement    = "0xad06b86264411e0278dbcebce556c913b44da004"
	MarketAdapter = "0x734052ebae66b883da9cb386313b03db82599c2a"
	WETH          = "0x0bd7d308f8e1639fab988df18a8011f41eacad73"
	Manager       = "0x8366a39cc670b4001a1121b8f6a443a643e40951"
	NativeHook    = "0x306e2a57d667558863c761fda22dc30d08b83fc0"
)

var ErrUnsupported = errors.New("flywheel: unsupported pairing or base pool")
var ErrState = errors.New("flywheel: invalid or unsupported market snapshot")
var ErrBasePool = errors.New("flywheel: route base pool not loaded")

type Config struct {
	DexID        string `json:"dexID"`
	NewPoolLimit int    `json:"newPoolLimit"`
	// Pairing token -> the indexed uniswapv3 pool address or uniswap-v4 pool id the settlement
	// swaps ETH through. WETH-paired launches need none.
	QuoteBasePools map[string]string `json:"quoteBasePools"`
}
type StaticExtra struct {
	Factory    string `json:"factory"`
	Settlement string `json:"settlement"`
	Quote      string `json:"quote"`
	// The route's base pool; a list because router-service reads "basePools" of meta pools.
	BasePools []string `json:"basePools,omitempty"`
}
type NativeParent struct {
	Token    string      `json:"token"`
	Quote    string      `json:"quote"`
	Pool     entity.Pool `json:"pool"`
	Protocol [2]uint32   `json:"protocol"`
}

type Extra struct {
	Parents            []NativeParent `json:"parents,omitempty"`
	Curve              CurveState     `json:"curve"`
	Valid              bool           `json:"valid"`
	MarketPool         *entity.Pool   `json:"marketPool,omitempty"`
	Protocol           [2]uint32      `json:"protocol"`
	Dependencies       []string       `json:"dependencies,omitempty"`
	DependenciesStored bool           `json:"dependenciesStored,omitempty"`
}
type SwapInfo struct {
	Token string `json:"token"`
	Buy   bool   `json:"buy"`
	// Settlement route bytes, used for the swap and a buy's graduation refund alike.
	Route hexutil.Bytes `json:"route,omitempty"`

	MinQuote, AmountOut, Refund, RefundRouteOutput uint256.Int `json:"-"`
	Next, Previous                                 CurveState  `json:"-"`
	Revision                                       uint64      `json:"-"`
	Steps                                          []SwapStep  `json:"-"`
}

type poolKey struct {
	Currency0   common.Address
	Currency1   common.Address
	Fee         *big.Int
	TickSpacing *big.Int
	Hooks       common.Address
}
type routeHop struct {
	Kind uint8
	Pool common.Address
	Key  poolKey
}
type SwapStep struct {
	Before       string
	NativeFeeETH uint256.Int
	Index        int // -1 = canonical market, 0 = external base, 1..2 = native parents
	Params       pool.UpdateBalanceParams
}
type Meta struct {
	Settlement  string `json:"settlement"`
	Token       string `json:"token"`
	BlockNumber uint64 `json:"blockNumber"`
}
