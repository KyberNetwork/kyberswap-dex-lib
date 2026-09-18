package slyngfun

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

type Gas struct {
	BuyNative  int64
	SellNative int64
	BuyERC20   int64
	SellERC20  int64
	// Graduation is what the buy that fills the curve pays on top of a plain buy.
	Graduation int64
}

// StaticExtra is what never changes for a curve once TokenCreated has fired. Discovery is a pure
// log decode, so the launchpad's constants are filled in by the tracker on its first pass; the
// rest is known from the log itself.
type StaticExtra struct {
	// Launchpad is the contract every curve lives in. Pools are keyed by their token address, so
	// the executor needs this to know what to call.
	Launchpad string `json:"launchpad"`
	// IsNativeQuote is true when the curve is priced in ETH: buy() then takes msg.value and
	// sell() pays out native ETH. Everything else is a plain ERC-20 pulled with transferFrom.
	IsNativeQuote bool `json:"isNativeQuote"`
	// GraduationTarget is the quote reserve at which the curve closes and its liquidity moves to
	// a Uniswap v4 pool, snapshotted at creation.
	GraduationTarget *uint256.Int `json:"graduationTarget"`
	// VirtualQuote seeds the constant product and sets the opening price, snapshotted at creation.
	VirtualQuote *uint256.Int `json:"virtualQuote"`
	// CreatedAt is the curve's creation timestamp, which the opening surcharge is measured from.
	CreatedAt uint64 `json:"createdAt"`

	// The launchpad's trade constants. Compile-time constants on chain, read rather than assumed
	// so a launchpad deployed with different numbers prices correctly under the same code.
	TradeFeeBps        uint64 `json:"tradeFeeBps"`
	SnipeBps           uint64 `json:"snipeBps"`
	SnipeWindowSeconds uint64 `json:"snipeWindowSeconds"`
}

// Extra is the curve state that moves with every trade, refreshed on each tracker pass.
type Extra struct {
	// QuoteReserve is the real quote the curve holds, net of fees (Curve.quoteReserve).
	QuoteReserve *uint256.Int `json:"quoteReserve"`
	// TokenReserve is what the curve has left to sell (Curve.tokenReserve).
	TokenReserve *uint256.Int `json:"tokenReserve"`
	// Graduated is Curve.graduated. Once true, buy() and sell() revert for good.
	Graduated bool `json:"graduated"`
}

// SwapInfo is what CalcAmountOut hands UpdateBalance, and what aggregator-encoding reads to build
// the call. IsBuy, IsNativeQuote, Launchpad and Token carry real json tags on purpose:
// aggregator-encoding round-trips swap extra through json, and a "-" tag would decode to zero.
type SwapInfo struct {
	IsBuy         bool   `json:"isBuy"`
	IsNativeQuote bool   `json:"isNativeQuote"`
	Launchpad     string `json:"launchpad"`
	Token         string `json:"token"`

	NewQuoteReserve *uint256.Int `json:"-"`
	NewTokenReserve *uint256.Int `json:"-"`
	// NewGraduated is set when the buy lifts the reserve to the graduation target: the launchpad
	// graduates the curve inside that same buy, and nothing trades on it afterwards.
	NewGraduated bool `json:"-"`
}

// PoolMeta names the launchpad as the contract to approve for an ERC-20 leg, and which of its two
// calls the executor should build.
type PoolMeta struct {
	Launchpad       string `json:"launchpad"`
	Token           string `json:"token"`
	ApprovalAddress string `json:"approvalAddress"`
	IsBuy           bool   `json:"isBuy"`
	IsNativeQuote   bool   `json:"isNativeQuote"`
	BlockNumber     uint64 `json:"blockNumber"`
}

// curveResp mirrors the tuple Launchpad.curves(token) returns, in declaration order. Fields must
// stay *big.Int: go-ethereum's reflection-based unpacker only fills *big.Int for a uint256.
type curveResp struct {
	QuoteReserve     *big.Int
	TokenReserve     *big.Int
	GraduationTarget *big.Int
	VirtualQuote     *big.Int
	LpQuote          *big.Int
	Quote            common.Address
	Creator          common.Address
	CreatedAt        uint64
	Exists           bool
	Graduated        bool
}
