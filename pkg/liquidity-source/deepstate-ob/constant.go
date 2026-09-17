package deepstateob

import (
	"errors"

	orderbook "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/order-book"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// DexType covers DeepstateV1, a singleton multi-pool radix-tree order book.
// One router contract serves every pair; there is no per-pair pool contract
// and no factory/creation event, so pool discovery is static/config-seeded
// (see Config.Pools) rather than an on-chain scan.
const (
	DexType = valueobject.ExchangeDeepstateOb

	methodFeeConfig    = "feeConfig"
	methodActiveBookID = "activeBookId"
	methodPoolEpoch    = "poolEpoch"
	methodRoots        = "roots"
	methodTree         = "tree"
	methodTopOrder     = "topOrder"
	methodNextNonce    = "nextNonce"
	methodGetBook      = "getBook"

	// maxBFSNodes caps the total number of tree() reads performed while
	// walking one side of a book in the naive fallback path (no Lens
	// configured), guarding against an unexpectedly deep/wide tree consuming
	// unbounded RPC calls in a single poll.
	maxBFSNodes = 4096

	// lensMaxNodes is the maxNodes argument passed to a deployed
	// DeepstateBookLens.getBook(). Sized off a real measurement: the live
	// NVDA/USDG book (189 bid + 36 ask leaves) cost ~3.74M gas against this
	// cap with truncated=false and comfortable headroom under typical
	// eth_call gas limits (see ks-helper-sc's
	// test/lens/DeepstateBookLens.t.sol test_getBook_gasUsage). Revisit if
	// the book grows enough to approach it.
	lensMaxNodes = 4096

	// tickPriceExpNumerator/Denominator implement price(tick) = 2**(96*tick/2**31)
	// per DeepstateV1's NatSpec. float64 precision (not the contract's exact
	// Q128 TickMath32 table) is used here, matching this codebase's existing
	// order-book-family precedent (pkg/liquidity-source/kuru-ob) of pricing
	// resting levels in float64 for routing/estimation purposes.
	tickPriceExpNumerator   = 96
	tickPriceExpDenominator = 1 << 31
)

var defaultGas = orderbook.Gas{Base: 175625, Level: 17108}

var (
	ErrInvalidToken      = errors.New("invalid tokenIn/tokenOut for this pool")
	ErrInvalidPoolTokens = errors.New("pool must have exactly 2 tokens")
	ErrBookTooLarge      = errors.New("book tree exceeds maxBFSNodes during BFS walk; deploy the lens contract or raise the cap")
	ErrRPCCallReverted   = errors.New("deepstate-ob: required RPC call reverted")
)
