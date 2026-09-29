// Package fables integrates Fables dynamic-fee hooks (Uniswap v4) on Robinhood Chain.
//
// # Discovery and auto-tracking
//
// Fables deploys a new immutable hook per pool, so a static address list would go stale as the
// protocol grows. The on-chain source of truth is the FablesPoolRegistry
// (0x159A113E012593D9B3cC63ad45E30F0467e13Ef3), an append-only registry that records every
// pool's full v4 PoolKey — hook address included — and exposes:
//
//	function allPools()    view returns (PoolInfo[])  // every pool ever registered
//	function activePools() view returns (PoolInfo[])  // currently active pools only
//	function poolCount()   view returns (uint256)
//	function poolAt(uint256) view returns (PoolInfo)
//	event    PoolRegistered(PoolId indexed id, PoolKey key, address indexed hook, uint256 index)
//
// where PoolInfo is {PoolKey key; PoolId id; bool active}. Because hook recognition in this
// module is an exact-address lookup populated at init (see HookAddresses), the registry is used
// to regenerate that list rather than to key off it at runtime: a maintenance step reads
// activePools() and rewrites constant.go. This keeps the hot swap/pricing path allocation-free
// while making "track all present and future Fables pools" a mechanical, reviewable diff.
//
// Regenerate HookAddresses (dedup, registration order preserved):
//
//	RPC=https://rpc.mainnet.chain.robinhood.com
//	REG=0x159A113E012593D9B3cC63ad45E30F0467e13Ef3
//	N=$(cast call $REG "poolCount()(uint256)" --rpc-url $RPC)
//	for i in $(seq 0 $((N-1))); do
//	  cast call $REG "poolAt(uint256)((address,address,uint24,int24,address),bytes32,bool)" \
//	    "$i" --rpc-url $RPC   # 5th key field is the hook address
//	done
//
// # Pool onboarding vs hook recognition
//
// These are two independent concerns in the uniswap-v4 module:
//
//   - Pool onboarding (finding pools to add) is a platform concern handled by the parent module
//     via either a v4 subgraph (PoolsListUpdater) or on-chain PoolManager `Initialize` events
//     (PoolFactory) + StateView. Robinhood Chain has no hosted v4 subgraph, so onboarding runs
//     off the on-chain event/StateView path; the FablesPoolRegistry can also seed the initial
//     pool set directly.
//   - Hook recognition (this package) maps a pool's hook address to the Fables handler so its
//     fee is read from currentFee() instead of a stale slot0 lpFee.
//
// # Fee model
//
// Every Fables pool uses the dynamic-fee flag and resolves its LP fee inside beforeSwap via the
// fee-override return only — no delta flags, no hook fee, no custom calldata. currentFee(poolId,
// zeroForOne) returns the fully-resolved per-direction fee (autonomous curve + optional
// directional premium + TTL-bounded operator poke, clamped to the pool cap). Track snapshots
// both directions; BeforeSwap replays the matching side.
package fables
