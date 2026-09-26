# Everlong FLAMM

`everlong-flamm` quotes Everlong FLAMM pools. Each is an exact-input AMM between one pool asset and one loan asset,
and its inventory is financed through a money-market router.

Which pools it quotes is set by a hand-maintained hook registry (`hook_registry.go`, see "Hook registry"). A pool is
listed and quoted only when every hook it lists is registered, is of the kind its slot requires, and is bound to that
pool. Each quote is then dispatched on those kinds.

The registry has one production pool today, `c104` on Base (chain 8453):

- pool `0xc0fdCB1799cCc2CEBaA1fe247157b0dF33D57572`;
- cbBTC (8 decimals) against USDC (6 decimals);
- financed on Morpho Blue through MMRouter and MorphoBlueAccount.

The port is a wei-exact transcription of the deployed Solidity (the c104 tree at `80abd43`), not an approximation: a
quote runs the pool's own settlement on a copy of the state.

## Venues

One pool exposes two venues on one shared book. The executor selects the venue through word 1 of the adapter's
`data` (`EverlongFlammAdapter.sol` `VENUE_SWAP` / `VENUE_LEVERAGE`; `VenueSwap` / `VenueLever` in `constant.go`).

| venue | pool entry | direction | fill |
| --- | --- | --- | --- |
| 0, swap | `swap` | both | EverlongHook fill on the AlmCurve book, fee from EverlongStrategy, Router legs through Morpho. A sell clipped by its ceiling (gate room, notional cap, funding) uses part of the input. |
| 1, leverage | `leverUp` / `leverDown` | pool asset in = lever-up, loan asset in = lever-down | EverlongLeverageHook frame on the frozen CollRebalancerMath curve, spread from LeverageSpreadHook. A lever-down uses only its net pay leg. |

`CalcAmountOut` quotes both venues (the leverage venue only with `LeverRouting`) and keeps the swap venue unless the
leverage venue pays more than `LeverMinEdgeBps` above it -- the edge its 2.3-2.6M extra gas has to earn, since
`CalcAmountOut` returns one result per pool and a router that is handed the leverage quote never sees the swap quote
it replaced. The venue is picked on the two settlements alone and only the picked one is then put through the
configured margins, so a margin refuses a quote instead of moving it to the other venue at a different price.

`RemainingTokenAmountIn` is the input the pool did not pull. It means two different things per venue, and the
difference matters to a router:

- **Swap venue.** Spare input. A sell clipped by its ceiling (gate room, notional cap, a debt cap, Morpho liquidity,
  the funding ceiling) uses part of the input, and re-quoting at the used amount returns the same output.
- **Leverage venue, a lever-down.** Required headroom, *not* spare input. `FLAMMLeverLib._planDown` sizes the hook
  fill on the whole `loanIn` (`FLAMMLeverLib.sol:124-141`) and then charges only
  `payNative = ceilDiv(amountInUsed - virtualLeg, scale)` (`:140`), so re-quoting at the used amount is a smaller
  trade: at the 51313000 snapshot under the `btc_up10` scenario, with leverage armed -- the chain's own state at
  that block is `LevPaused`, which refuses both directions -- a 125,814,386 uUSDC lever-down pays 77,857 sats and
  reports 52,237,196 unused, while the same quote at 73,577,190 pays 47,474 (-39%) and reports 31,852,243 unused of
  its own. A lever-down hop cannot be trimmed to what it used; its residual has to be re-routed or accepted
  (`TestSimulatorLeverDownInput`).

What router-service does with a non-zero `RemainingTokenAmountIn` is not visible from this repository, and it is the
one thing that decides whether the leverage venue's remainder is an accounting detail or a real cost (see "Open
questions").

Exact output is not supported: the simulator implements no `CalcAmountIn`, so the dex is not registered in
`pool.CanCalcAmountIn` and router-service's exact-out finder never picks it. The approval address is the pool, which
pulls the input with `transferFrom`.

### Enabling `LeverRouting`

It is off by default and needs three confirmations from Kyber, not one:

1. the executor encoder passes `SwapInfo.Venue` as the adapter's `data` word 1;
2. router-service can re-route or accept the residual of a lever-down hop, which cannot be trimmed (above);
3. venue selection is gas-aware on router-service's side. `LeverMinEdgeBps` only removes the thin-gain tail -- a bps
   edge cannot express a gas cost, which depends on the route's value.

## File map

Protocol port (Solidity paths are in the c104 tree unless noted):

| Go | Solidity counterpart |
| --- | --- |
| `almcurve.go` | `src/hooks/everlong/AlmCurve.sol` (swap and book half; recenter-only solvers are not ported) |
| `fee.go` | `src/hooks/everlong/EverlongStrategy.sol` fill-fee law (L132-198) |
| `hook.go` | `src/hooks/everlong/EverlongHook.sol` swap path: lazy book rescale, fee role, fill, commit. With `fee.go` and `almcurve.go` it is hook kind `everlong-swap-v1` |
| `levcurve.go` | `src/hooks/everlong/lev/CollRebalancerMath.sol`, `LevCurveTypes.sol` |
| `levhook.go` | `src/hooks/everlong/lev/EverlongLeverageHook.sol`. With `levcurve.go` it is hook kind `everlong-leverage-v1` |
| `swap.go` | `src/core/flamm/FLAMMSwapLib.sol` |
| `lever.go` | `src/core/flamm/FLAMMLeverLib.sol` |
| `gate.go` | `src/core/flamm/FLAMMGateLib.sol` |
| `state.go` | `src/core/flamm/FLAMMStore.sol` ledger, dials and switches; `IFLAMMHooks.sol` contexts; `spreadHookState`, the `src/hooks/everlong/lev/LeverageSpreadHook.sol` post, which is hook kind `everlong-spread-v1` |
| `pricefeed.go` | `src/core/PriceFeed.sol` (cross, peekCross, usd, peekUsd, pegOk) |
| `router.go` | `src/core/mm/MMRouter.sol`, `MMRouterLib.sol`; the FLAMMSwapLib settlement legs that drive them |
| `account.go` | `src/core/mm/MorphoBlueAccount.sol` |
| `morpho.go` | morpho-blue v1.0.0 `src/Morpho.sol`, `MathLib`, `SharesMathLib` |
| `irm.go` | morpho-blue-irm v1.0.0 `AdaptiveCurveIrm.sol`, `ExpLib.sol` |
| `math_common.go`, `errors.go` | OpenZeppelin `Math.mulDiv`; one sentinel per custom error of the deployed contracts (147: the 139 declared under `src/core`, `src/factory`, `src/hooks`, `src/interfaces` and `src/libraries` at `80abd43`, plus the eight OpenZeppelin ERC20 / Initializable errors; the c104 periphery is not on any quoted path and is left out) |

Kyber integration:

| Go | role |
| --- | --- |
| `hook_registry.go` | the registries: hooks by address (kind, runtime codehash, bound pool), financing accounts by address, and each factory's shared contracts. Also the slot rules (`resolveHooks`) and the identity checks every refresh and quote re-runs (`validStatic`, `validVenues`) |
| `hook_kinds.go` | one implementation per hook kind: its listing and refresh reads, binding check, state build and refresh-trigger flag (`hookKindSpec`), and the role ports the settlement calls. Also the per-pool hook state (`poolHooks`) |
| `pools_list_updater.go` | lists the pools the registered swap hooks are bound to, once the configured `FLAMMFactory` owns them (`isPool`) and their whole wiring is registered |
| `pool_tracker.go` | one refresh at one pinned block, and the dependency set it publishes |
| `tracker_reads.go`, `state_reads.go` | the view aggregate, the storage words and the state build |
| `attest.go` | wiring drift and the attestation probes |
| `multicall.go` | Multicall3 `tryBlockAndAggregate` with revert data, chunked JSON-RPC batches |
| `pool_simulator.go` | `CalcAmountOut`, `UpdateBalance`, `CloneState`, the envelope and the margins |
| `config.go`, `type.go`, `constant.go` | configuration, entity shapes, gas and margin defaults |
| `abi.go`, `embed.go`, `abis/` | minimal ABIs extracted from the c104 forge artifacts |

## Hook registry

FLAMM hooks are write-once and bound to one pool. Each hook carries its pool as an immutable (`EverlongHook.sol:93`,
`EverlongLeverageHook.sol:36`, `LeverageSpreadHook.sol:29`), and the pool lists its hooks in `hooks()`. The list is
`IFLAMM.HookSet` in field order: invariant, fee, recenter, controller, leverage, spread, loanSwap (`FLAMM.sol:222-225`).

So the port registers code by hook address and quotes each pool through the hooks the pool itself lists. There is no
per-pool configuration: a pool is known only through the hooks registered as bound to it.

### What is registered

`hook_registry.go` keeps three hand-maintained tables, each keyed by chain:

| table | keyed by | holds |
| --- | --- | --- |
| `hookRegistry` | hook address | the hook's kind, its runtime codehash (immutables included) and the pool it is bound to. Also the immutables of its kind that the lister and every refresh read back: the swap hook's pair, `LOAN_SCALE` and `genesisStrategyHash`, and the leverage hook's `LOAN_SCALE`. The swap hook's entry also names the pool's deployment: the factory that created it, which the pool binds at initialization (`FLAMMOpsLib.sol:124`) and reports as `factory()` |
| `financingAccounts` | account address | a financing account a venue may use (`MorphoBlueAccount`), with its runtime codehash and the pool it is bound to. The Router deploys one per pool, loan asset and venue kind (`MMRouter.sol:131-133`), and every account one deployer creates has that deployer's codehash (`MMRouterLib.sol:150`) |
| `flammDeployments` | factory address | what every pool of one `FLAMMFactory` shares, each by address and codehash: the factory, its Router (`FLAMMFactory.ROUTER`), each implementation its beacon may serve, and each PriceFeed. Also the Morpho singleton, and the pool proxy's codehash, which is the same for every pool because each pool is its own `FLAMMProxy` of the factory's beacon (`FLAMMDeployLib.sol:21-27`) |

The kinds are the three hook implementations the package ports. No kind fills the loanSwap slot.

| kind | Solidity | slot | Go | state in the pool state | refresh trigger |
| --- | --- | --- | --- | --- | --- |
| `everlong-swap-v1` | `EverlongHook.sol` | invariant, fee, recenter and controller (0-3), one hook | `hook.go`, `fee.go`, `almcurve.go` | the hook's storage (`hookState`) | yes: a keeper's retune emits only `TuningChanged` |
| `everlong-leverage-v1` | `EverlongLeverageHook.sol` | leverage (4) | `levhook.go`, `levcurve.go` | none: it quotes on the swap hook's book | no |
| `everlong-spread-v1` | `LeverageSpreadHook.sol` | spread (5) | `spreadHookState` (`state.go`) | the keeper's post: `spread`, `maxSpreadAge`, `lastSetTs` | yes: a keeper's post |

Each kind reads back what binds its hook and compares it with the hook's entry, at listing and on every refresh:

- a swap hook's `POOL()`, `LOAN_SCALE()` and `genesisStrategyHash()`;
- a leverage hook's `HOOK()`, which must be the pool's swap hook, plus its `POOL()` and `LOAN_SCALE()`;
- a spread hook's `POOL()`.

The genesis strategy is an immutable; the live `strategyHash` and `paramsHash` move with keeper tuning, so they are
not compared.

Registered today, all from the c104 deployment record (`c104.8453.json`, codehashes verified on chain) and all bound
to pool `0xc0fdCB1799cCc2CEBaA1fe247157b0dF33D57572`:

| address | registered as |
| --- | --- |
| `0x65CBD227cBC61248ae77a5fC813A29C54C092134` | hook, `everlong-swap-v1`, pair cbBTC/USDC, `LOAN_SCALE` 1e12, of the factory below |
| `0xE0A98d8e60035832B8BaD7f7af7B9B0b3A7308F3` | hook, `everlong-leverage-v1`, `LOAN_SCALE` 1e12 |
| `0x04988aF54ec88D2de77b191025EAef2fe488f93b` | hook, `everlong-spread-v1` |
| `0x6760E3b032eE2d670Cb684d9076b8f48cb066c48` | financing account |
| `0x1BfcE014774D0DD7e04bC595D46Fa09F7dCCF45f` | factory, with Router `0x19A9b39E6710AAD109C829294b0841F0851c6bB4`, implementation `0xaAD580BeAa2cbd8Ab5F3956a5c56EDa1D5ee7184`, PriceFeed `0xbED275459578C87a63F2f50A0b077C720e838816` and Morpho Blue `0xBBBBBbbBBb9cC5e90e3b3Af64bdAF62C37EEFFCb` |

### Slot rules

`resolveHooks` accepts a hook set only if:

- every non-zero hook is registered on this chain, is of a kind whose role matches its slot, and is bound to this pool.
  A kind with no port is refused.
- the swap hook's entry names the deployment the pool is being resolved under as the factory that created it. A
  pool is a candidate of that deployment's listing alone, and a listing that pins it to any other registered
  deployment of the chain is refused.
- there is a swap hook, and slots 0-3 all name it. The pool itself accepts a zero controller slot
  (`FLAMMOpsLib.sol:208`), but the port does not: the swap kind models the invariant and fee roles as one contract's
  storage.
- the loanSwap slot is empty, since no kind ports it. This is defence in depth over the one-loan envelope rather
  than the only thing standing between the port and a loan swap: with one loan asset `swapLoan` has no pair to swap
  and reverts `InvalidPair` before it reaches the hook (`FLAMMLoanSwapLib.sol:67`, `:127`), and the envelope is
  re-pinned on every quote. It is also slot 6's only check -- the role loop covers slots 0, 4 and 5 -- so admitting
  a hook there would admit the one hook address with no registration, no codehash pin and no `POOL()` binding.
- the leverage and spread hooks are both set or both empty, as the pool itself requires (`FLAMMOpsLib.sol:209`).
- the leverage kind can quote on the swap kind: `everlong-leverage-v1` reads `everlong-swap-v1`'s book.

A pool with no leverage or spread hook is supported. The listing, the refresh, the probes and the dependency set skip
those two roles, and its leverage venue refuses with the pool's own `LeverageDisabled` (`FLAMMLeverLib.sol:81`).

### Discovery

The lister never enumerates the factory. `createPool` is permissionless (`FLAMMFactory.sol:216-254`), so `pools()`
grows without bound and would eventually exceed a node's `eth_call` gas cap. Pools come from the registry instead:

1. **Candidates.** These are the distinct pools the chain's registered swap hooks are bound to and name the
   configured factory as the creator of, cut down to `Config.Pools` when that is set. A factory that is not a
   registered deployment has no candidates, and the run sends nothing.
2. **First aggregate.** One Multicall3 aggregate at `latest` fixes the listing block and reads, for every candidate,
   `FLAMMFactory.isPool(pool)`, `hooks()` and `priceFeed()`.
   - A pool the factory answers `isPool == false` for is dropped and leaves the cursor. A pool whose `isPool`,
     `hooks()` or `priceFeed()` read reverts or does not decode is skipped for the run and keeps its cursor entry:
     only an `isPool == false` answer forgets a pool.
   - The hook set is checked against the slot rules, and the PriceFeed must be registered for the factory. Both
     checks run before any read is addressed to a hook or a feed.
3. **View round, at that block.** It reads:
   - the pool's pair, loan count, Router, PriceFeed and factory;
   - its hook set, which must equal the first round's;
   - the factory's implementation, which must be registered;
   - each hook's bindings, by kind;
   - the Router's loan and venue counts;
   - the PriceFeed configuration for the pair named in the swap hook's entry.
4. **Digest.** A pool whose pinned identity still hashes to the cursor's digest stops here. A steady-state poll
   therefore costs the shared first aggregate plus one view aggregate per pool: two round trips for c104 alone.
5. **Venues and code.** The venue rounds read each venue's account, market params and oracle feeds. Then one
   `eth_getCode` batch checks each of these contracts against its registered codehash:
   - the pool proxy, the factory, the implementation and the Router;
   - each listed hook;
   - the PriceFeed;
   - each venue's account.

   That is nine contracts for c104, and seven for a pool with no leverage or spread hook.
6. **Identity.** `validStatic` and `validVenues` then run as they do on every quote.

A pool that fails any step is skipped with the warning `pool does not match the registry`, which names the reason.
Only a transport failure fails the run.

### Per-pool dispatch

- **Tracker.** For each listed hook, the refresh's view aggregate adds the reads its kind declares
  (`hookKindSpec.refreshReads`):
  - for a swap hook, its storage and its bindings;
  - for a leverage hook, its bindings;
  - for a spread hook, its post and its binding.

  The drift check compares each hook's bindings with its entry, by kind. The dependency set adds the hooks whose kind
  is a refresh trigger.
- **State.** `flammState.Hooks` (`poolHooks`) holds the listed addresses and, for each role, a kind tag plus that
  kind's concrete state: `Swap.EverlongSwap *hookState` and `Spread.EverlongSpread *spreadHookState`. The leverage
  kind has no state.
  - It is a tagged union rather than an interface because an interface-typed field survives Kyber's msgpack encoder
    only when its concrete type is registered with the encoder, by hand, in the shared
    `pkg/msgpack/register_types.go` (the generated `register_pool_types.gen.go` registers simulators alone).
    Unregistered, the encoder writes the value as a bare array with no type tag and the decoder panics on it
    (`reflect.Set: value of type []interface {} is not assignable to ...`), so every new kind would owe an entry in
    another package, whose absence fails at runtime and not at build time.
  - `CloneState` deep-copies both pointers.
  - In `Extra.Reads`, `hook` and `spread` appear only when the pool lists a hook of that kind.
- **Simulator.** Each settlement step resolves its role's tag to the kind's port:
  - the swap path calls `swapHookPort` (spot, fee, fill, commit);
  - the leverage path calls `leverageHookPort` (`previewLever`, over the swap port);
  - `FLAMMLeverLib._spread`, the spread margin and the deadline round call `spreadHookPort` (the answer, its
    liveness, its deadline).
- **Fail closed.** The hook set is resolved against the registry at listing, on every refresh, when a simulator is
  constructed, and on every quote. The per-quote check is needed because msgpack restores a simulator without
  `NewPoolSimulator`.
  - `usable()` re-runs `validStatic` and `validVenues`, and requires the state's hook addresses and kinds to equal the
    listing's, as resolved by the registry.
  - A hook removed from the registry, or re-registered with another kind, pool or factory, stops quoting at the
    next quote.
  - A hook set changed on chain is drift, and the pool is relisted only once the registry accepts the new set.

A wiring the registries do not accept is refused with `ErrInvalidProfile`; a refresh whose wiring moved publishes
`Extra.ProfileDrift`. `StaticExtra.ProfileVersion` versions the shape of the pinned identity.

### Adding a pool whose hooks are of a registered kind

1. **Check the hook set.** Read the pool's `hooks()`:
   - slots 0-3 must all be the same EverlongHook;
   - the loanSwap slot must be empty;
   - the leverage and spread hooks must both be set or both be empty.
2. **Register each hook.** For each one, take `keccak256(eth_getCode(hook))` and append a `hookRegistry` entry with
   `ChainID`, `Address`, `Kind`, `CodeHash` and `Pool` (the hook's `POOL()`). The codehash is the instance's own,
   even for a hook that reuses known code: its pool, loan scale and genesis hashes are immutables, which the compiler
   writes into the runtime code, so no two hooks share one. Then add the fields for its kind:
   - swap hook: `Factory` (the pool's `factory()`, which must be a registered deployment on the chain), `PoolAsset`
     and `LoanAsset` (the pool's `asset()` and `loanAsset()`), `LoanScale` (`LOAN_SCALE()`) and
     `GenesisStrategyHash` (`genesisStrategyHash()`);
   - leverage hook: `LoanScale`. Its `HOOK()` must be the pool's swap hook;
   - spread hook: nothing more.
3. **Register each financing account.** Append each venue's account (`MMRouter.venue(pool, i).account`) to
   `financingAccounts` with its codehash and pool. An account's bindings are storage, not immutables
   (`MorphoBlueAccount.sol:58-62`, `:91-100`), and the Router accepts only its deployer's runtime code
   (`MMRouterLib.sol:150`), so every account it deploys has the c104 account's codehash.
4. **Register any new shared contracts.** Extend `flammDeployments` for a PriceFeed, an implementation or a factory the
   registry does not know yet. Do this only after checking that the Go port matches that code. A new factory needs
   its Router, implementations, PriceFeeds, pool proxy codehash and Morpho.
5. **Nothing else changes.** The lister reaches the pool through its swap hook, the tracker builds the read plan from
   the kinds, and the simulator dispatches on them.
6. **The offline tests name their pool.** The recorded tapes answer for one pool's listing round, so the replays
   list through `tapeConfig` (`Config.Pools` set to that pool) rather than through the whole registry. A second
   registered pool of the same deployment therefore leaves them untouched; it needs its own recorded tape, and a
   test of its own if it is to be replayed. Nothing in the suite pins the number of registered pools.
7. **Verify.** `TestRegistryWellFormed` walks the tables and fails on an entry that is registered twice, has no
   codehash or misses a field of its kind, names an unregistered factory, or is bound to a pool no registered swap
   hook of the chain is. Add the entries to a `TestHookRegistry`-style test, then run a live listing and refresh
   against the pool as `TestLiveHead` does; the refresh must attest. `TestForkMultiPool` shows the whole procedure
   on a fork: it creates two pools through the live factory, registers their hooks and accounts, and lists,
   refreshes and quotes all three pools.

### Adding a hook kind

1. **Port the Solidity.** Port it wei-exactly into its own file, with a concrete state struct, fixtures and tests.
2. **Name the kind.** In `hook_registry.go`, add the kind before `hookKindCount` and its name to `hookKindNames`. Add
   any immutables the tracker must read back as `hookEntry` fields.
3. **Implement `hookKindSpec`.** In `hook_kinds.go`, write an empty struct that implements it and add it to
   `hookKindSpecs`. It supplies:
   - `role`;
   - `listingReads`;
   - `refreshReads`, whose state goes into a new pointer field on `flammReads` with its own JSON key and `omitempty`;
   - `bound`;
   - `build`, which sets the role's slot in the state;
   - `dependency`.
4. **Add the state to its role's slot.** Add the state as a new pointer field on the slot struct, resolve it in that
   slot's `port()`, and deep-copy it in `poolHooks.clone()`. Never add an interface-typed field.
5. **Implement the role's port.** That is `swapHookPort` (and `levBookSource` if a leverage kind is to read it),
   `leverageHookPort` (and `swapBoundKind.quotesOn`, which `resolveHooks` requires of a leverage kind: one whose
   spec does not implement it quotes on no swap kind and its hook sets are refused, `TestRegistryWellFormed`), or
   `spreadHookPort`.
6. **A slot no kind fills yet** (the loanSwap hook) also needs a new role, a relaxed slot rule, the settlement path
   that calls it, and its attestation probes.
7. **Register the new hooks** as above, and test:
   - the slot-rule refusals;
   - the state build;
   - a msgpack round trip;
   - a refresh that attests against the deployed hook.

## State sourcing and attestation

A pool is listed only when its whole wiring is registered:

- the runtime codehash of every contract on the swap path;
- the hook set, resolved by kind, and each hook's bindings (the swap hook's genesis strategy and loan scale among
  them);
- the pair its swap hook is registered for;
- the Router venues, each on a registered financing account of the pool, with their Morpho market params;
- the PriceFeed wiring.

Pools come from the registry, not from the factory's `pools()` array, and each one is confirmed with
`FLAMMFactory.isPool(pool)` (see "Discovery").

A listing pins only what identifies the pool (`StaticExtra`): the addresses, the hook set, the implementation and the
PriceFeed wiring.

What a curator can change on a live pool, the Router venue set, is published in `Extra` and re-read by every refresh.
That is because we do not assume pool-service replaces an existing pool's `StaticExtra`. Two Kyber-authored listers in
this repository, synthereum and curve/plain, say that pool-service filters out a pool it already has, and we could not
confirm the setting for this dex (see "Open questions").

- A pool that gains or retires a venue heals on its next refresh instead of waiting to be relisted.
- A venue that the registry does not accept is drift, and the pool stops quoting. That covers an account that is not a
  registered financing account of this pool, an IRM the port cannot price, and a market whose pair is not the pool's.
- The listing cursor keeps a digest of the pinned identity. Only a change to that identity leads to a relisting, and
  only once the registries accept the new identity.

A refresh at block B (`pool_tracker.go`):

1. one Multicall3 aggregate of every view the state is built from, at `latest`, which fixes B;
2. the storage words no view exposes (`FLAMMStore.lastLeverSpreadPpm`, the pending loan-asset ceremony, each
   venue's managed collateral and supply shares), with the FLAMMStore and MMRouter configuration words, the pending
   venue ceremony and the factory's beacon implementation and pending-upgrade pair checked against the views that
   report them (storage layout) -- which is what binds `Extra.ScheduledChangeAt`, whose source words no view
   attestation or probe otherwise reaches;
3. the drift check against the listing and the venue set (a set the round disagrees with is re-read from the Router
   at B and the round run again, so a curator's `addVenue` is adopted rather than reported), the state build, and
   the deployed views the state must reproduce
   (`peekCross`, `pegOk`, Router `positions`, `loanPosition`, `venuePosition`, the account's `tryPosition` and
   `borrowRateAfter`, `totalAssets`);
4. a dependency-resolve round and two forward rounds, all at B. The resolve round reads the Chainlink aggregator
   every price feed proxy currently resolves to (the dependency set below) at the block's own clock. The two
   forward rounds read at the same block with only `block.timestamp` overridden, which needs an `eth_call` that
   accepts `blockOverrides`: while `MaxSnapshotAgeSec` declares a window, what each venue's Morpho market oracle
   answers at the far end of it; and, at each clock the port believes a deadline word flips at inside that window,
   the deployed views that word decides -- the PriceFeed's `peekCross` and `peekUsd` of both tokens, the loan
   asset's `pegOk` and the spread hook's `spreadPpm` (the deadline round below). Each of those two reads the clock
   it ran at back from `Multicall3.getCurrentBlockTimestamp()` in the same aggregate and fails the refresh in
   transport unless it is the one it asked for, so a node that accepts the parameter and ignores it cannot leave
   the window guard silently disarmed;
5. one aggregate at B of the pool's own `previewSwap` / `previewLever` and the Router's `fundingCeiling` on grids and
   bisected band edges; the port must answer each exactly, reverts included. It names its own gas
   (`attest.go` `probeAggregateGas`, 30M), and a probe the chain answers with an *empty* revert is re-run on its own
   before it is read as agreement: Multicall3 reports a subcall that ran out of gas exactly as it reports the empty
   revert of OpenZeppelin's `Math.mulDiv`, while a call of its own comes back as a JSON-RPC revert or as the node's
   own failure.

Under the default policy that is five `eth_call`s and, at 33 storage words, four JSON-RPC batches of at most ten:
**nine** HTTP round trips, counted at the tape. A policy that declares no window (`maxSnapshotAgeSec: 0`) sends
neither the oracle-window round nor the deadline round and makes seven. The deadline round is one aggregate on the
deployed pool and grows only when a deadline actually falls inside the window (below).

A listing run that lists or relists a pool is six round trips: five `eth_call`s and one `eth_getCode` batch (nine
contracts for c104).

A poll that finds the pool already listed with the wiring the view round read stops there, at two round trips:
- the first aggregate (`isPool`, `hooks()`, `priceFeed()`);
- the view aggregate.

The venue rounds and the code batch are skipped because they verify only what an unchanged `StaticExtra` cannot have
changed.

### The deadline round

A word whose only effect is to fix a *future* second at which something changes moves no answer at the snapshot's
own clock, so neither the views nor the probes bind it: the spread hook's `lastSetTs` and `maxSpreadAge` (its post
answers through `lastSetTs + maxSpreadAge`, `LeverageSpreadHook.sol:75`), each aggregator round's `updatedAt` (its
quote is read through `updatedAt + heartbeat`, `PriceFeed.sol:171`) and the sequencer round's `startedAt` (the feed
answers nothing until `startedAt + SEQUENCER_GRACE`, `PriceFeed.sol:157-162` -- the mirror of a deadline). A value
on the permissive side is fail-open: the simulator keeps quoting fills the pool reverts, right up to the end of the
window the policy admits quoting in.

Every one of those predicates is monotone in the clock -- a live post and a fresh round only ever go dead, a
sequencer's grace only ever ends -- so, given that the chain and the port agree at the snapshot, agreeing at the
two seconds either side of the flip the port believes in pins it exactly. A predicate whose flip is outside the
window needs only the window's far end, which is why the deployed pool costs one extra round trip: its spread post
lapsed long ago, its sequencer came up 6.9M seconds before the snapshot, and both feed rounds outlive the window.
A policy that declares no window (`maxSnapshotAgeSec: 0`) has no end to bind against and sends no deadline round,
as it publishes no oracle-window answer either.

What the round reads is a function of the block's state and the clock alone: the pool's own two Chainlink feeds are
`AccessControlledOCR2Aggregator 1.0.0` (the aggregators behind `0x07DA0E54` and `0x7e860098`), not the SVR
DualAggregator the venue's market oracle prices from, so moving the clock forward reveals no round and changes no
answer but the deadlines themselves.

A refresh that drifts, fails a probe, or reads a required view that reverts or does not decode is published with
`Extra.Attested = false`, and the simulator refuses it. Only a transport failure returns an error. The refresh also
stamps `Extra.ScheduledChangeAt`, the earliest time any of the four delayed changes can execute: the factory's
pending implementation (which anyone may execute), the pool's pending hook set, a scheduled venue admission (a new
funding leg in every settlement) and a scheduled loan asset (which puts the pool outside the quoted envelope
altogether). Either word of each pair marks it pending, and the two of the pool's that a view reports -- the hook
set and the venue admission -- are cross-checked against their FLAMMStore slots, as is the factory's pair; the
loan-asset ceremony has no view at all and is read from storage only.

The four ceremony slots are the last four fields of the FLAMMStore struct, `+30`..`+33`, and which pair is which was
confirmed on a Base fork rather than inferred: the curator's scheduling call wrote `+32`/`+33` for `addVenue` and
`+30`/`+31` for `addLoanAsset`, and `+32`/`+33` equal what `pendingHookSet()` reports. `TestTrackerScheduledChange`
pins that equality, so a layout change under a future implementation fails the refresh closed instead of silently
dropping a scheduled change from `Extra.ScheduledChangeAt`.

The state holds no price or accrual evaluated at the snapshot -- feed staleness, the sequencer grace, Morpho and IRM
accrual and the spread's age are recomputed at the quote's timestamp -- with one exception: a venue's Morpho market
oracle answer. The BTC/USD feed behind the c104 market oracle is a Chainlink SVR DualAggregator, which withholds each
primary round for a fixed delay, so `price()` changes with the clock and no transaction. The forward round reads
that answer at the end of the window and the simulator re-runs every fill at it (`Extra.OracleAhead`).

## Refresh dependencies

The tracker implements `pool.IPoolTrackerWithDependencies`: besides the pool's own events, a refresh should be
triggered by logs from

| address | why |
| --- | --- |
| the aggregators behind the pool asset, loan asset and sequencer feed proxies | a new round re-prices every gate; at the 600 s cadence the snapshot is otherwise behind an unseen cbBTC round about 30% of the time |
| the aggregator behind each venue's market oracle feeds | the same, for `bandOk` and the venue's health |
| the swap hook (`Hooks[0]`, kind `everlong-swap-v1`) | a keeper's `setFeeRow` / `_setTuning` changes every fee and emits only the hook's `TuningChanged` |
| the spread hook (`Hooks[5]`, kind `everlong-spread-v1`), when the pool lists one | a keeper's spread post |
| the `MMRouter` | `setGlobalPaused` (`GlobalPauseSet`) stops all borrowing, and venue flags, caps and priorities are Router-only events. Every Router leg of another listed pool's fill emits one too (`Borrowed`, `Repaid`, `Supplied`, `CollateralPosted` and their withdrawals, `MMRouter.sol:206-255`, `MMRouterLib.sol:276-353`), which is what refreshes a pool whose venue market another pool of the factory also finances on |

A hook is in the set when its kind declares itself a refresh trigger (`hookKindSpec.dependency`). The leverage kind
does not: it has no storage and emits nothing that moves a quote.

None of those emits a pool event: over blocks 51324778-51367978 (24 h) the cbBTC feed had 90 rounds and **none**
shared a block with a pool event. The set is resolved, not configured: a Chainlink proxy emits nothing and its
`aggregator()` moves on a phase change with no log of its own, so every refresh resolves it again and
`Extra.DependenciesStored` is cleared when the resolved set changes. Morpho Blue and the AdaptiveCurveIrm are
excluded on purpose -- a trigger on them is a trigger on every Morpho borrow on Base, and the accrual they drive is
recomputed at the quote clock anyway.

## Exactness contract

- `CalcAmountOut` runs the chosen venue's full plan and settlement on a copy of the state at the quote clock: hook
  fill, fee floor and cap, Router funding passes and cascades through Morpho, gate re-assertions. Amounts equal the
  adapter's `amountOut` and `amountUnused` to the wei, and a refusal is the revert the fill would hit.
- `SwapInfo` carries that settlement's post-state. `UpdateBalance` adopts it verbatim, so several fills can be chained
  between refreshes, including fills at one timestamp. Both published reserves follow the adopted state.
- Every fill is also settled at the market oracle answer the refresh read at the end of the snapshot window, and
  refused (`ErrOracleDrift`) unless it settles identically. Only `MMRouterLib.bandOk` (an interval around the
  PriceFeed cross) and Morpho's `_isHealthy` (monotone) read that price, so agreeing at both answers means agreeing
  at every answer between them.
- A `SwapInfo` is accepted only by the state it was quoted on. That state is a *value*, not an instance: the
  lineage token is derived from the entity (its address and the whole published `Extra`) and folded with each
  adopted fill, so clones, a msgpack round trip and a second simulator built from the same entity all accept each
  other's `SwapInfo`, while an entity that differs in one byte, or a state that has adopted a fill since, does
  not. Any other `SwapInfo` leaves that simulator refusing until the next refresh.
- Margins never change an amount; they only refuse. That holds for the routed quote too, because the venue is picked
  before any margin runs.
- The feed does not price the swap curve, but it does price a **sell's size**: the sell ceiling is the gate room and
  the Router's funding at the cross (`FLAMMSwapLib.sol:164`, `FLAMMGateLib.sol:45-53`, `:248-252`), so a clipped
  sell's amounts follow an unseen round. Those quotes are refused rather than published
  (`PriceBandMarginBps`, below). A buy's ceiling is the gross pool asset and carries no price.
- `Reserves` are `[gross pool asset, sell capacity]`, and the second is a **lower bound**, not a capacity the pool
  guarantees: it is the ceiling of a sell that brings no collateral of its own, while the chain counts the incoming
  pool asset as collateral and re-plans on it, so an accepted sell is paid more (on the 51302915 snapshot the bound
  is 94,577,910 uUSDC and a 177,385-sat sell is paid 128,065,084). An entity no quote will be served from --
  drift, a state that does not build, a failed attestation -- publishes `["0", "0"]`, since pool-service ranks on
  reserves.
- The pool is refused outright (`ErrPoolRefused`) outside the quoted envelope: more than one loan asset, an unreadable
  IRM or oracle on any venue, or Morpho collateral or supply shares beyond what the Router manages (unless
  `QuoteDonatedVenues`); and over an `Extra.OracleAhead` that is neither empty nor one answer per venue, a shape no
  refresh writes, where quoting on would skip the end-of-window oracle guard instead of running it.

## Configuration

Keys are the JSON tags below (`config.go`). pool-service decodes them with `pool.PropertiesToStruct`, which matches
case-insensitively, so `chainID`, `chainId` and `ChainID` are the same key -- but only while `Config` stays at or
below sixteen JSON fields, goccy/go-json's limit for that (`allowOptimizeMaxFieldLen`). Past it only the exact tag
matches. `Config` has thirteen fields, the gas overrides nested in `gas` to keep it there, and
`TestConfigFieldCount` fails if one is added past the limit. `DexID` is tagged with the exact key the lister and
tracker factories inject, so the dex id lands either way.

| key | default | effect |
| --- | --- | --- |
| `DexID` | injected | the exchange id; set by pool-service's factory, not by the configuration |
| `chainID` | required | the chain whose registry entries are listed |
| `factory` | required | a registered `FLAMMFactory`; a pool is listed only while this factory owns it (`isPool`) |
| `pools` | all | optional allow-list, intersected with the pools the registered swap hooks are bound to |
| `leverRouting` | `false` | quote the leverage venue |
| `leverMinEdgeBps` | `10` | keep a routed quote on the swap venue unless the leverage venue pays more than this many bps above it |
| `priceBandMarginBps` | `50` | how far the pool asset's feed may move before a fill is refused (below) |
| `priceAgeMarginSec` | `60` | refuse once a feed round the fill reads expires this many seconds after the quote -- the pool's own `maxPriceAgeSec` and each aggregator's heartbeat, whichever binds (`PriceFeed.sol:171`) |
| `spreadAgeMarginSec` | `60` | refuse a leverage quote once the keeper's spread lapses this many seconds after the quote |
| `debtDriftSec` | `30` | re-run the fill this many seconds later with Morpho interest accrued; refuse unless it still settles (a swap must also pay the same amounts) |
| `maxSnapshotAgeSec` | `600` | refuse once the clock is this far past the refresh block's timestamp; also how far ahead the refresh reads each venue's market oracle (`0` turns both off) |
| `quoteDonatedVenues` | `false` | quote a pool whose venue account holds Morpho collateral or supply beyond what the Router manages |
| `gas` | `{}` | override gas defaults: `swapSell`, `swapSellCapEval`, `swapSellPass`, `swapBuy`, `leverUp`, `leverDown` (zero keeps the measured default) |

`priceBandMarginBps` is the one margin with two enforcement points, because the feed prices more than the band:

- every price band is checked against `band - margin`, on both venues and both directions;
- a fill whose *size* or whose other gates the feed prices is re-run with the pool asset's Chainlink answer moved by
  the margin in both directions and refused (`ErrFeedMoveMargin`) unless it settles to the identical amounts. That
  is a swap **sell**, whose ceiling is the gate room and the Router's funding at the cross, and **every leverage
  fill**, whose value leak (`FLAMMLeverLib.sol:108`), 1.82e18 CR floor (`:112`) and lever-down concession (`:138`)
  are valued at the feed rather than at the curve's reservation price. A swap buy needs neither: its ceiling is the
  gross pool asset and its amounts were unchanged by a feed move in every state measured.

The default is sized on the feed's own round sizes, since the quote has to survive one round the snapshot has not
seen. Over Base blocks 51066959-51369359 (167.6 h, 567 cbBTC/USD rounds on aggregator `0x51cE3091`) consecutive
rounds moved a median 9.7 bps, p75 28.8, p90 33.6, p95 37.8, p99 52.1 and max 72.5, with the mass at the feed's own
30 bps deviation threshold and 49.3% of rounds above the previous default of 10 bps. At 50 bps a round of +-10,
+-20, +-35 or +-50 bps reverts and re-prices no accepted swap quote on the committed grids at any drift of the
snapshot from the book (0, +200, +400 and +600 bps pinned), where at 10 bps a band-pinned state loses 57 of its 610
accepted quotes to a +50 bps round and 47 to a +35. It costs 23 of 1,463 accepted swap quotes and 23 of 374
leverage quotes on those grids, and 50 bps is a sixteenth of the deployed band
(`swapPriceBandWad` 8e16). Scaling the margin with the snapshot's age instead would be the alternative; a flat
default that covers the p99 round was chosen because the feed aggregators are also in the dependency set, so a
round triggers a refresh rather than only ageing the snapshot.

An unset margin takes its default and an explicit `0` turns it off. The simulator also stops quoting 1800 s
(`scheduledChangeLeadSec`) before `ScheduledChangeAt`, and refuses to be constructed at all on a snapshot already
past `maxSnapshotAgeSec` when the caller sets `pool.FactoryOpts.StaleCheck` (route finding does; indexing does not).

## Gas defaults

The largest receipt gas of `executeEverlongFlamm` measured through the adapter on anvil forks of Base, plus 25%
(`constant.go`):

| venue and direction | default |
| --- | --- |
| swap sell | 1,360,000, plus 10,300 per curve solve of the hook's cap bisection and 210,000 per funding pass after the first |
| swap buy | 1,510,000 |
| lever-up | 3,670,000 |
| lever-down | 4,150,000 |

A buy runs neither term. A sell the pool clips (notional cap, room, a debt cap, Morpho liquidity, the funding ceiling)
grows with its input; the simulator counts its solves and passes from its own settlement.

The measured maxima behind those numbers are swap sell 1,085,310, swap buy 1,202,312, lever-up 2,929,633 and
lever-down 3,318,066; the two clipped-sell terms are the steepest per-solve and per-pass slopes any scenario showed.
The 25% on top is a deliberate margin over a worst case, not an estimate of a typical fill: on the 645 mined clipped
sells the estimate came to 1.31x-1.56x the receipt across the two recorded fork runs, the highest 1.554x. `CalcAmountOutResult.Gas` is what router-service subtracts from
route value when it scores a route, so the margin costs this source a little ranking against one that quotes its
mean; sizing the defaults on an expected receipt instead would under-state the worst case. The trade is the
operator's: every default is overridable per venue and direction through `Config.gas`.

## Known limitations

- **One production pool.** The registries hold only the c104 pool on Base.
  - A pool is not listed if any of its hooks is unregistered, registered as another kind, or bound to another pool.
  - A listed pool whose wiring moves to something the registries do not hold is refused as drift. That covers a new
    hook set, implementation, PriceFeed or financing account.
  - A pool whose hooks are of a registered kind is added with registry lines and a release. A hook with new code
    needs a new kind (see "Hook registry").
- **Stricter than the pool.** Slots 0-3 must name one hook and the loanSwap slot must be empty. A pool with a zero
  controller slot or a loan-swap hook is therefore not quoted, although `FLAMMOpsLib.sol:204-215` accepts both.
- One loan asset per pool; exact-input only.
- The dependency set covers the feeds, the hooks whose kind is a refresh trigger and the Router (above). It does not
  cover third-party Morpho activity on the venue market (utilisation, liquidity) or the accrual up to the fill's
  block. Those stay bounded by `PriceAgeMarginSec`, `SpreadAgeMarginSec`, `DebtDriftSec` and `MaxSnapshotAgeSec`, and
  by pool-service's refresh cadence. A pool that has not been refreshed since it was listed has no dependency set yet.
  Another listed pool's fill on a market this pool also finances on is not third-party in that sense: it moves the
  market only through the Router, whose events are in the set, so it triggers a refresh. `TestForkMultiPool` puts
  two of its pools on one market and refreshes the second onto the state the first's fills left before running the
  second's own sequence.
- The oracle window reads one answer, at the end of the window. That is the complete candidate set while the market
  oracle's feed withholds at most one round at a time (its reveal delay is ~12 s against rounds every ~290 s), and
  the gates are an interval and a monotone bound in the price, so every answer between the two is covered. Two
  withheld rounds inside one window whose prices reverse would leave a candidate outside that interval.
- The probe grid binds one local edge per direction, not every one. The acceptance set is not an interval, and over
  every integer amount at 51470150 the deployed pool's buy direction holds 22 class transitions, of which the grid
  straddles one; the `FillInvalid` class below the grid's first buy rung of 1,000 (every buy up to 779) is not
  probed, and neither is either leverage direction's `NothingToFill`, each leverage grid carrying a single amount
  (on an armed pool at 51470153, lever-down reaches it at 1 and lever-up at 24,691,965, and lever-down holds 41
  transitions). A refresh therefore binds the priced interior and one edge per direction; the refusal classes
  themselves are bound against the chain by the core grids at three blocks (section 4 of `testdata/README.md`),
  which carry the chain's own answer from amount 0 upwards in all four directions. Extending the grid costs a
  re-recording of both tracker tapes and their digests, since the aggregate is one `eth_call` keyed by its calldata.
- The probe aggregate names 30M gas. Its cost is state-dependent -- 2.87M on the pool as deployed, 11.3M in a state
  an ordinary curator call reaches, 13.0M in one only a Router-internal call reaches -- so the node's own `eth_call`
  gas cap has to clear the largest of them, and its `eth_call` must accept *and apply* `blockOverrides` for the two
  forward rounds, which prove it (above). A node that does not apply `blockOverrides` fails the refresh in
  transport, leaving pool-service on the last snapshot until `MaxSnapshotAgeSec`. A gas cap below what the
  aggregate needs has two outcomes rather than one, and only the first is a transport death:
  - a cap below what the *aggregate's own frame* needs fails the `eth_call` outright: transport, last entity kept;
  - a cap between that and the aggregate's full cost lets the aggregate run and starves its tail subcalls, which
    Multicall3 reports as unsuccessful with empty returndata -- the shape of `Math.mulDiv`'s own revert. Each such
    probe is then re-called on its own with the whole 30M for that one call, which is strictly more than the
    subcall had, so the refresh publishes the pool as refusing (`attestFailure`, "empty revert unconfirmed") rather
    than attesting a fill the chain never refused. The one shape it cannot separate is a cap below what a *single*
    probe costs, where the confirmation is starved as well (`multicall.go` `confirmRevert`).
- **Venues added after listing.** Such a venue is admitted only on a registered financing account of the pool; every
  c104 venue shares one. The registry pins each account's codehash, and the listing checks it for every venue it
  reads.
- **Codehashes are checked only at listing.** No refresh re-checks one, and neither does a poll that finds the pool
  already listed with the wiring the view round read. This is sound: the financing accounts are registered by address
  and codehash, and every other address the listing hashes is fixed by the digest:
  - the pool, factory, Router, PriceFeed and hooks are part of the pinned identity;
  - the implementation is read back into that identity by every view round, so a beacon upgrade relists the pool;
  - EIP-6780 leaves a contract deployed before its own transaction undeletable, so runtime code at a fixed address
    cannot change while the digest stays the same.
- A one-wei donation of Morpho collateral or supply on behalf of a venue account refuses the pool by default until the
  curator withdraws it (`MorphoBlueAccount.sol:272` allows the collateral withdrawal only with no debt outstanding).
- Attestation binds every word that moves a probed answer, and the deadline round binds the words whose only effect
  is a future one (above). What is left is bound by the tracker's decoding, pinned field for field by
  `TestTrackerReplay`, and -- for the Router and FLAMMStore configuration and the factory's upgrade schedule -- by
  the layout check against the raw storage words. The deadline round covers the window the policy admits quoting
  in and nothing past it: a deadline outside that window moves no admitted quote, and `maxSnapshotAgeSec: 0`
  declares no window and sends no round.
- A code override on an address the pinned bytecode links is not reported as drift. The c104 contracts link
  ten library addresses (PUSH20 scan of the live runtime code, cross-checked against the deploy broadcast's
  `libraries` array); the port mirrors six of them:

  | library | address | bound through | mirrored by |
  | --- | --- | --- | --- |
  | `FLAMMSwapLib` | `0x89aA5f76765D16c460A5B4a7AC6a385e35d2D405` | implementation codehash | `swap.go` |
  | `FLAMMGateLib` | `0x50417cB978f856b3885AbfA97308e0377FC2844e` | implementation codehash | `gate.go` |
  | `FLAMMLeverLib` | `0x9d8D5BDD29A81dC3996c0FFD80a9Ddb472F8c39f` | implementation codehash | `lever.go` |
  | `FLAMMFlowLib` | `0xFE6fe06338bc103CB57Bd0c3cf2F5Cb8dc2D0E79` | implementation codehash | -- (LP flows) |
  | `FLAMMOpsLib` | `0x3beaDe9745Dd2aeec8Cbd773F3F919664Df71877` | implementation codehash | -- (governance) |
  | `FLAMMLoanSwapLib` | `0x473ff6E1798BB03C6C75519905bAbE72B414Bd0c` | implementation codehash | -- (loan-to-loan) |
  | `MMRouterLib` | `0x167740F8E6baB2e8080ac1332Fd66ad5B45D8591` | **Router** codehash | `router.go` |
  | `AlmCurve` | `0xf82DdF0A8a50bc2C3F163997766bA1839E527A17` | hook codehash | `almcurve.go` |
  | `CollRebalancerMath` | `0xc002d0731e6A2E6e80bE754779BCef6B01aFF0BB` | leverage hook codehash | `levcurve.go` |
  | `PoolDeployLib` | `0x4F26504C30999CdA1A16d0B09E8c85D4cCf27013` | factory codehash | -- (deployment) |

  The eleventh library of the deployment, `RouterDeployLib` `0xDe756432Cbe1d813E97e2B30148cba8dfe9Fd816`, is linked
  by none of the registered contracts.
- Gas defaults are measured maxima plus 25%, not proven bounds, and router-service scores routes on that number, so
  they are deliberately conservative rather than expected costs ("Gas defaults"). Recovery-branch lever-downs were
  not reached on a fork.
- Venue selection compares outputs, not receipts. `LeverMinEdgeBps` makes the leverage venue earn a fixed edge over
  the swap venue, which removes the tail where its extra gas (2.31M up, 2.64M down; about $0.033 and $0.038 at
  Base's observed 0.006 gwei and ETH at $2,407) costs more than the gain, but a bps edge is not a gas cost: it
  over-charges a large route and under-charges a tiny one. Gas-aware selection belongs in router-service.
- `Reserves[1]` is a lower bound on one sell's payout, not the sell capacity (above). It is recomputed from the
  adopted state by `UpdateBalance`, so a chain of fills keeps it consistent.
- `PriceBandMarginBps` covers a feed move of its own size, and nothing larger. A round above the margin can still
  revert or re-price a quote; the dependency set (above) is what bounds how long the snapshot can be behind one.
- `PriceAgeMarginSec` is inert on Base while the feeds behave. The pool asset's aggregator posts every ~1,064 s on
  average (567 rounds over the 167.6 h window measured for `PriceBandMarginBps`) against a 3,600 s heartbeat, and
  the loan asset's every ~86,418 s (the last eight rounds: 86,406-86,420 s) against a 90,000 s heartbeat, which
  leaves about 3,582 s of slack -- so no round expires within 60 s of a quote. The margin is the only guard against
  a feed that *stops*, and it is not the guard against a round the snapshot has not seen: that is
  `PriceBandMarginBps` and the refresh cadence.
- A refresh's rounds are pinned by block *number*. Multicall3's second output, the block hash, is
  `blockhash(block.number)`, which the EVM defines as zero for the block being executed, so it is 0x00..00 in every
  `eth_call` (checked on Base and on anvil) and pins nothing; a real pin would need EIP-1898 block parameters on
  every later `eth_call` and `eth_getStorageAt`, which not every node serves. A same-height reorg between two
  rounds of one refresh would therefore have to survive the attestation, which runs the deployed previews after the
  storage reads against a state built from the view round.
- **A quote is not free.** `BenchmarkCalcAmountOut` on an Apple M2 Ultra, under the shipped defaults, on a busy
  machine:
  - a swap sell takes 73 us and 1,864 allocations;
  - a swap buy takes 39 us and 691;
  - a routed sell takes 152-159 us and 5,118;
  - a routed buy takes 57 us and 1,072.

  With every margin off, the same four quotes take 18.5, 20, 100 and 37 us. The difference is the margins that re-run
  a settlement (`DebtDriftSec`, `PriceBandMarginBps` and the oracle window): each does the whole plan again, two or
  three times over.

  Most of the cost is the rate model. `irm.go` keeps AdaptiveCurveIrm's int256 arithmetic in `math/big` rather than
  `int256` as `AGENTS.md` prefers. In a default-policy swap sell, `mmIrmBorrowRate` is about half of the CPU time
  inside `CalcAmountOut`, and `math/big.nat.make` is 65% of its allocations.
- **Strict/non-strict boundaries are largely not pinned by fixtures.** Each of the 223 `uint256` comparisons in the
  twelve ported math modules was flipped from `>` to `>=` (or `<` to `<=`), and the offline suite run on each. That
  killed 100 mutants and left **123** green, most of them in `router.go`, `almcurve.go`, `levcurve.go`, `lever.go`
  and `swap.go`.
  - Each survivor is a comparison whose two sides differ only on an exact tie that no recorded state produces.
  - They are held by the line-by-line transcription against the Solidity and by the live and fork differentials,
    not by a fixture.
  - A fault in one would be a wrong *refusal* or acceptance on a tie, never a wrong amount.
- Not modelled: token custody (fee-on-transfer tokens, `FLAMMStore.reconcile`), reentrancy guards and access checks,
  and gas-capped oracle reads beyond the tracked `oraclePrice` flag.

## Open questions

Two things this package cannot answer from inside this repository. Both are asked in the PR description as well;
neither blocks the swap venue.

1. **How does router-service treat a non-zero `RemainingTokenAmountIn`?** Whether it re-routes the residual, drops
   the hop, or leaves the amount with the executor decides whether a clipped swap sell and (once `LeverRouting` is
   on) a lever-down's headroom are an accounting detail or a cost to the taker. The swap venue's remainder is
   ordinary spare input, so nothing here is unusual; the leverage venue's is not (above), which is why it is a
   condition on enabling `LeverRouting`.
2. **Does pool-service overwrite an existing pool's `StaticExtra` when the lister re-emits it?** This package is
   written not to need it: `StaticExtra` holds only immutable identity and every mutable piece of wiring lives in
   `Extra`, re-read by each refresh. The question is worth settling anyway, because it decides whether a future
   wiring change (a new implementation, a new hook set) can be rolled out by relisting or needs the pool deleted
   first.

## Tests

`testdata/README.md` documents every fixture, its generator and digest, the environment variables of the opt-in
network tests, and how to run each suite. The offline suite needs no network:

```sh
go test -race -count=1 ./pkg/liquidity-source/everlong/flamm/...
```

Dense fixtures are sampled under `-race` (`fixture_sample_test.go`), and the three core preview grids -- 82,545
`previewSwap` / `previewLever` rows over 3 blocks x 26 scenarios -- are replayed every third row by default,
because each row there is a full settlement rather than a preview. `EVERLONG_FLAMM_FIXTURES=full` replays every
row of every fixture: 20,015 fills identical to the pool's own previews and 60,532 reverts of the same class, the
remaining 1,998 rows being the two IRM-outage scenarios the envelope refuses as a whole.

Two things the network suites do that are worth knowing before reading their counts:

- The live comparison counts five outcomes apart, not two (`live_test.go` `liveTally`): a fill the pool's preview
  answers identically, a size the chain and the port both refuse with the same error, a size the chain's preview
  *accepts* while the port refuses with one of the pool's own settlement errors, and a size the port refuses by a
  declared policy of its own, counted by whether the chain reverts too or fills. The one declared policy is the
  lapsed spread post: the port quotes neither direction of the leverage venue on it, where the pool reverts a
  lever-up with `SpreadUnavailable` and plans a lever-down at the stored degrade value, which this port does not
  quote and the pool then fills or refuses on its own gates. A spread refusal while the chain's post answers at
  the block is a mismatch, whatever the chain answers. Each of the settlement-only refusals is confirmed by an
  `eth_call` of the pool's own `swap` / `leverUp` / `leverDown` at the same block, from an account the call's state
  overrides fund and approve: the pool has to refuse it too, or the test fails. A port that refused everything with
  a settlement error would otherwise be counted as agreeing.
- The fork harness freezes each venue's Morpho market oracle at the answer it gives at the fork block
  (`fork_harness_test.go` `pinOracles`), because its BTC/USD feed is a Chainlink SVR DualAggregator: a fork mines
  its fills at the timestamps the simulator quoted them at, so an upstream reveal can land inside a sequence and
  move `oraclePrice` with no transaction in between. The pinned answer is the oracle's own, so nothing else
  changes; the port's own handling of the reveal is `Extra.OracleAhead`, covered offline to the wei.
