package rangepool

import "errors"

const (
	// Factory getter for on-chain pool discovery (Range has no subgraph).
	factoryMethodGetPools = "getPools"

	// RangePool getters: one struct each carrying (almost) all pool state.
	poolMethodGetDynamicData   = "getRangePoolDynamicData"
	poolMethodGetImmutableData = "getRangePoolImmutableData"

	// Vault getter for the aggregate (protocol + creator) swap-fee share, which
	// is NOT part of getRangePoolDynamicData.
	vaultMethodGetPoolConfig = "getPoolConfig"

	// Vault getter for the minimum trade amount (scaled18). Range's custom Vault
	// deploys a NON-canonical value (1e12 on mainnet, vs Balancer's default 1e6),
	// enforced at both _ensureValidSwapAmount points of Vault._swap — see gotcha #13.
	vaultMethodGetMinimumTradeAmount = "getMinimumTradeAmount"

	// Standard Router query getters — the on-chain pricing oracle used by the
	// parity test (never called on the hot path).
	routerMethodQuerySwapExactIn  = "querySwapSingleTokenExactIn"
	routerMethodQuerySwapExactOut = "querySwapSingleTokenExactOut"

	// baseGas is the swap gas estimate. Range's swap path is the Balancer V3
	// weighted-product curve on virtual balances plus a fact-balance cap, so it
	// mirrors the weighted connector's estimate.
	baseGas = 213745
)

var (
	// ErrInvalidToken is returned when a token index is out of range for the pool.
	ErrInvalidToken = errors.New("invalid token")

	// ErrExactOutNotAllowed mirrors RangePool.onSwap's EXACT_OUT revert guards:
	// the requested output would drain the pool below ABSOLUTE_MIN_TOKEN_BALANCE
	// (AmountOutExceedsFactBalance) or is >= the virtual out balance
	// (VirtualBalanceOutTooLow). Either makes the on-chain swap revert, so we
	// return no price.
	ErrExactOutNotAllowed = errors.New("range-pool: exact-out amount not allowed (fact cap or virtual balance)")

	// ErrTradeAmountTooSmall mirrors the Vault's TradeAmountTooSmall() revert: a swap
	// leg (post-fee input or output) below the Vault's minimum trade amount reverts
	// on-chain (gotcha #13).
	ErrTradeAmountTooSmall = errors.New("range-pool: trade amount below vault minimum")

	// ErrIncompleteState is returned when the state multicall did not fully succeed.
	// TryBlockAndAggregate tolerates per-call reverts (its error stays nil), so the
	// tracker must reject a partial read rather than build a pool from zero/nil values
	// (which would panic in MustFromBig or yield a garbage quote).
	ErrIncompleteState = errors.New("range-pool: incomplete on-chain state read")
)
