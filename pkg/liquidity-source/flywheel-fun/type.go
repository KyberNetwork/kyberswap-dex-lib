package flywheelfun

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

const (
	DexType       = "flywheel-fun"
	Factory       = "0xee54da52128dd851c71b1c58d371966231b66c40"
	Settlement    = "0x04111c295399582b2b702ad5de8d11be2b50dd5d"
	MarketAdapter = "0x20495fd86c527a6e0ddc90bbbf8f0f6857595865"
	WETH          = "0x0bd7d308f8e1639fab988df18a8011f41eacad73"
	Manager       = "0x8366a39cc670b4001a1121b8f6a443a643e40951"
	Market        = "0x19548d7029ee272eb03576f33f7c84a3eebfe10b"
	NativeHook    = "0x3c5af67244db99fc4ad9baa6a316f35467e4ffc0"
	V3Factory     = "0x1f7d7550b1b028f7571e69a784071f0205fd2efa"
)

var ErrUnsupported = errors.New("flywheel: unsupported route, pool or tick coverage")
var ErrState = errors.New("flywheel: invalid or unsupported market snapshot")

type Config struct {
	DexID        string `json:"dexID"`
	NewPoolLimit int    `json:"newPoolLimit"`
	// Ordered WETH -> quote routes. Pool identities are authenticated on chain.
	QuoteRoutes map[string][]RouteHop `json:"quoteRoutes"`
}
type StaticExtra struct {
	Factory    string `json:"factory"`
	Settlement string `json:"settlement"`
	Quote      string `json:"quote"`
}
type Extra struct {
	Curve              CurveState    `json:"curve"`
	Valid              bool          `json:"valid"`
	MarketPool         *entity.Pool  `json:"marketPool,omitempty"`
	Protocol           [2]uint32     `json:"protocol"`
	Route              []RouteHop    `json:"route,omitempty"`
	RoutePools         []entity.Pool `json:"routePools,omitempty"`
	Dependencies       []string      `json:"dependencies,omitempty"`
	DependenciesStored bool          `json:"dependenciesStored,omitempty"`
}
type SwapInfo struct {
	Next              CurveState   `json:"-"`
	Previous          CurveState   `json:"-"`
	Token             string       `json:"token"`
	MinQuote          *uint256.Int `json:"quotedQuoteAmount"`
	AmountOut         *uint256.Int `json:"amountOut"`
	Refund            *uint256.Int `json:"refund"`
	RefundRouteOutput *uint256.Int `json:"refundRouteOutput"`
	Route             []RouteHop   `json:"route,omitempty"`
	Buy               bool         `json:"buy"`
	Revision          uint64       `json:"-"`
	Steps             []SwapStep   `json:"-"`
}

type PoolKey struct {
	Currency0   common.Address `json:"currency0"`
	Currency1   common.Address `json:"currency1"`
	Fee         *big.Int       `json:"fee"`
	TickSpacing *big.Int       `json:"tickSpacing"`
	Hooks       common.Address `json:"hooks"`
}
type RouteHop struct {
	Kind uint8          `json:"kind"`
	Pool common.Address `json:"pool"`
	Key  PoolKey        `json:"key"`
	// V4 protocol and LP fees read at the same block as all liquidity.
	Protocol [2]uint32 `json:"protocol"`
	LPFee    uint32    `json:"lpFee"`
}
type SwapStep struct {
	Index  int // -1 = canonical market; otherwise the route's base-pool index
	Before string
	Params pool.UpdateBalanceParams
}
type Meta struct {
	Settlement  string `json:"settlement"`
	Token       string `json:"token"`
	BlockNumber uint64 `json:"blockNumber"`
}
