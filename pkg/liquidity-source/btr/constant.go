package btr

import "time"

// Monad mainnet records (BTR deployment records, verified 2026-10-08).
//
// The Router at RouterAddress holds no lane 0x400, so an aggregator must
// NOT route through Router.swap: the call reaches Pool.swap_qe with
// msg.sender = Router, which has no lane, and reverts NotAuthorized. The
// aggregator's own on-chain adapter must call Pool.swap_qe directly and hold
// lane 0x400, or SWAP_GATED must be cleared by BTR governance.
const (
	ChainIDMonad = 143

	// CreditPoolCore is BTR's single multi-asset AIMM book (tag `btr-core`).
	CreditPoolCore = "0xbbbbbbb04f5b762A4CdD1d89E341e2537e3267e4"
	// PoolFactory mints official pools; Router.swap checks isOfficialPool.
	PoolFactory = "0xbbbbbbbbbd5e955E20F323cA1F95f8191fD0E0BF"
	// RouterAddress is the atomic multi-hop Router. It holds no swap lane, so
	// an aggregator executor cannot use it while SWAP_GATED is set.
	RouterAddress = "0xbbbbbbbbb9687Ae7E49AE486e2e5C6408d2AA7fe"

	USDC  = "0x754704Bc059F8C67012fEd69BC8A327a5aafb603"
	WMON  = "0x3bd359C1119dA7Da1D913D1C4D2B7c461115433A"
	CBBTC = "0xd18B7EC58Cdf4876f6AFebd3Ed1730e4Ce10414b"
	WBTC  = "0x0555E30da8f98308EdB960aa94C0Db47230d2B9c"
	WETH  = "0xEE8c0E9f1BFFb4Eb878d8f15f368A02a35481242"
	XAUT0 = "0x01bFF41798a0BcF287b996046Ca68b395DbC1071"

	// SwapSelector is Pool.swap_qe, a suffix-mined dispatch (Pool.sol:270).
	SwapSelector = "0x01dc30b8"

	// DefaultGas is a conservative gas figure for one swap_qe at Monad's
	// cold-access pricing.
	DefaultGas = 350_000
	// MaxAge drops a stale sampled ladder rather than quoting marks the keeper
	// has since moved. Monad blockMs ~400ms; refresh every block.
	MaxAge = time.Minute
	// RPCTimeout bounds one tracker refresh.
	RPCTimeout = 15 * time.Second
)

// Risk-flag bits (BTR Solidity constants).
// Only the bits an open-path executor must gate on are listed.
const (
	SwapEnabledBit uint16 = 16   // SWAP_ENABLED_BIT
	SwapGatedBit   uint16 = 1024 // SWAP_GATED_BIT (lane-0x400 allowlist)
	HaltMask       uint16 = 5    // HALT_MASK (HALT_BIT | FEED_HALT_BIT)
	HaltBit        uint16 = 1    // HALT_BIT
)

// OpenPathMask is HALT_MASK | SWAP_ENABLED | SWAP_GATED. A normal (non-lane)
// executor may route a leg only when flags & OpenPathMask == SwapEnabledBit.
// The weaker `flags & 5 == 0 && flags & 16 != 0` is true on the live gated
// word and is NOT sufficient (ORACLE_MARKS.md §2 finding 0).
const OpenPathMask = HaltMask | SwapEnabledBit | SwapGatedBit
