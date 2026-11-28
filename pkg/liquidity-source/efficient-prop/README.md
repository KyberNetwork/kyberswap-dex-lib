# efficient-prop

Efficient PropAMM listings. A PropAMM front holds any number of listings. Each listing is one pair on one pool: its
maker posts quotes and depth on chain, and the front fills a taker from them out of the pool's custody. Every listing
is reached through its own listing door, an immutable `IPropAMM` contract with one pair, created by the front's door
factory at the CREATE2 address of the listing id (`doorOf(listing)`):

| Entry | What it does |
|---|---|
| `quote(tokenIn, tokenOut, amountIn)` | View: what a fill of `amountIn` pays now; reverts where the listing does not fill it |
| `swapFrom(tokenIn, tokenOut, amountIn, minOut, recipient, deadline)` | Pull entry: takes `amountIn` from the caller by allowance and fills it |
| `FRONT`, `LISTING`, `TOKEN0`, `TOKEN1` | Immutables: the front, the listing id and the pair |

A fill is all or nothing. Nothing routes until the dex is enabled in config with an approved desk and a listing of
that desk is live on the front.

## Chains

The front (`0x0000000000403aeB66691063CC2fD42E71d6AcD6`, implementation
`0x48C6efab497EeC8501483Ceb23B6623f626d8371`) and its door factory (`0x39eD9993449C96503f6a629cCD2BB5F132D7cc93`)
have one address on every chain, each chain's own code behind it:

| Chain | The front's build | Read |
|---|---|---|
| Ethereum (1) | builder build: a maker's update lands only in a block a builder builds with it | `/state` (Ethereum) |
| Base (8453) | storage build with the packed desk: a maker's word stands in storage | the head (Base) |
| BNB Chain (56) | builder build, Ethereum's desk code | `/state` (BNB Chain) |

Everything below holds on every chain unless a chain's section says otherwise.

## What is listed and quoted

One pool per door: the pool address is the door, the tokens are `TOKEN0` and `TOKEN1`, and `StaticExtra` holds the
front, the listing id, the listing's pool (`custody`) and the fill mode. Listing ids run from 1 in order, so the
lister reads `front.doorView(id)` and `factory.doorOf(id)` in batches of 64 pinned to one block, at most `maxListings`
(4,096) a poll. Only the front's state says a listing is live, so the whole list is read on every poll.

Anyone may open a desk, a listing and a price source on the front, so nothing is quoted for being there. A listing is
listed and quoted only where:

- **its maker is approved**: its desk's contract is one `approvedDesks` names (Config). The set is empty by default;
- its door is genuine: the CREATE2 child of the configured factory under the factory's own `DOOR_CODEHASH()`, with
  immutables naming the configured front, the listing and its pair;
- every price source it reads is in `approvedSources`, read from the front's storage at each refresh, so a listing
  rebound to another source is dropped at the next refresh;
- the front's implementation (its proxy's ERC-1967 slot, read at every refresh) is in `implementations`: the source
  reads the front's storage by its layout, which an upgrade may change;
- its tip cap is at least `minTipCapWei`, or 0, which on a builder build (Ethereum, BNB Chain) means no cap and on
  Base's storage build fills nothing, so such a listing is idle there. A fill sent at a higher priority fee than its
  listing's cap reverts (`TipCapped`), and nothing in a route says what the taker will tip, so the cap is also in the
  pool meta (`tipCapWei`);
- its gas class has a measured figure in the front's fill mode (Execution and gas), or `listingGas` names it. On BNB
  Chain no class was measured: only a listing `listingGas` or `creditListings` names is quoted;
- on Ethereum and BNB Chain, its maker's update for the next block is in a `/state` answer (Ethereum, BNB Chain).

Not quoted at all: a SESSION listing in a closed phase, a tied listing, a listing with a credit mode that
`creditListings` does not name, and an asset-module pair (a tokenized stock's) that `listingGas` does not name.

**How to approve a source.** A source's code runs inside the quote and again inside the fill, and it can tell the two
apart, so it can fill a real transaction for less than the `eth_call` quoted, down to the route's minimum output.
Approve a source only where its code cannot change and its answer comes from its own stored prices and the clock
alone. Approval is by address. The same holds for a listing's two tokens, which this source does not screen: route
through a listing only where its tokens' code cannot change and moves exactly the amounts asked.

## Pricing

The source runs none of the front's math. Each refresh reads the listing, its pool's balances and the front's clock at
one block, then asks the door's own `quote` at a ladder of sizes in both directions in one multicall pinned to that
block:

- Sizes come from the shared `ladder` grid, dense at the last ladder's knee. A size the door refuses is no point. A
  direction's ladder is the run of passing sizes up to the first refusal or falling output, at most 64 points. On
  Ethereum and BNB Chain a state no answer covered holds no ladder, so the tracker keeps the listing's last ladder and
  draws the next quoted refresh's grid from it (sizes only, never a price).
- The grid also holds every rung boundary. A side fills from at most ten rungs, each at one price, and the output
  bends only where the fill moves to the next rung. The tracker reads the rung sizes from `doorView` and the side's
  fill memory from the front's storage (`extSloads`, one more call in the head read's multicall, no request of its
  own). The grid holds the middle of each rung too. The boundaries are sizes to ask the door at. They are never
  prices. They are read only while the front runs an implementation the config lists. Under any other the grid is
  the shared one alone.
- The grid holds a size at the listing's minimum fill, so a ladder starts where the listing does. The minimum is in
  the listing's settings word, read in that same storage call (a listing with a credit mode is not given it). With
  the quote token in it is an input, and the size is the minimum itself. With the base token in it is an output: the
  sizes are the two inputs that bracket it on the last ladder's first point, so the ladder reaches the minimum from the
  second refresh.
- The ladder holds for its whole window. A listing's levels may widen with its quote's age, so every size is sampled
  again at the window's last block (and around a premium's end, and at each block of a guarded source's hold) by an
  `eth_call` with a block override, and the least output of each size is kept. Each sample reads its block's time and
  number beside its quotes, so a node that ignores the override is caught.
- The simulator never pays more than the door at the refresh block. The front's output is a floored concave function
  of the input: at a sampled size the simulator pays the door's answer, between two sizes the floored chord, and under
  the smallest or past the largest nothing. With a point on every rung boundary the chord is the door's answer or one
  base unit of the output token less, on a ladder one block sampled. A listing with a credit mode keeps the shared grid
  alone, with no boundary, middle or minimum-fill size: it takes one fill a route whatever the state says, so they
  would buy it nothing, and each of its sizes costs a plan beside its quote in the one pinned call.
  The pinned call of a listing without a credit mode asks about 44 sizes a side at most, where the shared grid asked
  up to 22.
- It pays no more than the listing's pool holds of the output token. Listings of one pool share a swap limit keyed by
  pool and token.
- A direction takes more than one fill a route. A fill moves the side's fill memory by its input, rounded up to the
  side's board unit. The next fill is priced from the ladder shifted by that much: the ladder's answer at the end of
  the new fill, less the most the rungs before its start can have paid. It is never above what the front pays the
  fills in sequence, and each fill must still reach the ladder's first point. The state says per direction whether
  the ladder proves this (`repeat`). Where it does not, the direction takes one fill a route as before: a ladder whose
  sampled blocks disagree unevenly, a front whose implementation is not listed. A listing with a credit mode always
  takes one fill a route in all, since a fill moves the custody and the debt its route reads.
  Two more rules keep a later fill to what the chain pays. A side whose standing word may refill it at a later fill
  (the controls let a word refill, the word asks it of the side, and it acknowledges a sequence 1 to 1,023 ahead of the
  board's) takes one fill a route: a refill sends the next fill back to the first rung. And a later fill is quoted only
  where the ladder's answer at its end is at least n bucket units and 2 under the ladder's last output, n being the
  fills the direction took before it and the unit 10^max(0, decimals - 6) of the token paid out: the pool books each
  fill's payout apart, rounded up to that unit, so fills that end at the ladder's top could pass its bucket. A pool that
  takes a fee in the token it pays out (a spread rate, a notional rate or an operator's share) leaves the next fill of
  the quote's side less room than the samples show, and this source does not model that. It reads the fee instead: the
  lister reads the pool's `factory()` once, where the listing is scanned, and carries it in the static extra; each head
  read asks the Factory's `spreadFeeRatesOf(pool)` and the pool's `operatorShare()`, two more inner calls of the head
  multicall and no request of their own. A pool whose rates word or share is nonzero, or whose read fails, answers
  short or has no Factory read, is priced as one that takes a fee (the read fails closed): its base-in direction takes
  one fill a route (`ErrSecondFill`), and its quote-in direction keeps its later fills, since the fee is owed in the
  quote and comes off the room of the fill that pays the quote. A pool listed before its Factory was read has none in
  its static extra, so it is priced the same way until it is listed again. `oneFillListings` stays as an override that
  forces one fill a route in both directions.
- `StaleCheck`: route finding refuses a ladder older than 10 s and stops `inclusionAllowance` seconds before the
  ladder's `validUntil`. With `StaleCheck` off, no age gate applies.

## Refresh

- **Schedule.** The tracker implements `NextUpdateAt`: one second after the state's block time; on Ethereum a second
  from now, or at `validUntil` where that is sooner; on BNB Chain, whose second is two blocks and more, at once,
  including a state still waiting for its block's update. What that asks of the urls and the node is bounded by the
  router's scheduler tick and by the front's shared `/state` fetch and head probe (100 ms on BNB Chain). A listing not
  quoted for a standing reason (its desk not approved, its fill mode, its tip cap, no gas figure, not live, a BNB
  Chain front without `bundleOnly`) is read every 5 s.
- **Logs.** The event parser (a `poolfactory` decoder) maps each log that moves a listing to its door: the front's
  logs naming a listing, a desk or a pair; a desk contract's `DeskPush` and `LaneFill`; an approved price lane's
  `Posted`; and a fill's `Swapped`, which on Ethereum and BNB Chain the listing's quote contract logs. A `Swapped` from
  an address the parser does not know maps to nothing. Base's packed desk logs `DeskPush` only where its record asks
  and no `LaneFill`, so a post it does not log is quoted at the listing's next scheduled refresh.
- **Node load.** The tracker keeps, a listing, what the chain's head fixes, under the head's hash (never its number, so
  a head replaced at the same height reads again), and the result priced under the bytes of the listing's own override.
  The head's hash is asked of the node once a front per 250 ms, 100 ms on BNB Chain (`eth_getBlockByNumber`), whatever
  the number of its listings. The head read under an override asks, in its own multicall, the door's quotes at the sizes
  the listing's last quoted refresh drew for the next, and beside it, in the same batch, the window's samples at them; a
  refresh that needs a size not asked there reads the pinned read after it. Every gate is decided on the head read as
  before, and a listing a gate refused under its answer asks nothing ahead until it is quoted again. At a new head the
  head read is asked beside the head's facts, on the binding and slots of the head before, and is asked again where the
  facts name others. A refresh of a quoted listing that reads a price source (a primary and a backup), on Ethereum, then
  makes at a new head 2 `eth_call` and 3 `eth_getStorageAt` (on a tracker's first refresh, which reads the front's code
  hashes and holds no sizes, 5 and 7; 6 and 7 where the listing and its word carry into the next block); at a new head
  no answer covers yet, 1 `eth_call` and 3 `eth_getStorageAt`; at the same head under a new override (the maker's
  update), 1 `eth_call`; at the same head under the same override, no node call (`TestBuilderRefreshNodeLoad`,
  `TestBuilderHeadCacheSavesTheNodeCalls`, `TestBuilderPrefetchedReadIsTheReadItSaves`,
  `TestBuilderNewHeadAsksTheHeadReadBesideTheFacts`, `TestBuilderListingNoAnswerCoversReadsTheFactsAlone`,
  `TestPrefetchedReadPassesNoGate`, `TestBuilderListingsRefreshedTogetherShareOneStateFetch`). The listings of a front
  refreshed within 250 ms (100 ms on BNB Chain) share one `/state` read of each url. On BNB Chain the window also
  samples the second after the target block's, and the head read asks the millisecond precompile, one more inner call of
  its multicall. At the head a refresh makes 3 to 16 `eth_call` by the listing's kind.

## Execution and gas

Each front's config names its fill mode, `front` or `door`, with no default. Each pool carries it (`StaticExtra` and
`PoolMeta`: `door`, `front`, `listing`, `mode`), and the consumer builds the adapter's data from them,
`abi.encode(door, front, listing, mode)`:

- Mode 1, `front`: the adapter reads `front.payPoolOf(listing)`. Where it names a pool, the adapter pays the input to
  that pool and calls the front's paid entry, `swapExactInPaid`; where it names none (a listing with a credit mode),
  it approves the front and calls `swapSender`.
- Mode 0, `door`: the adapter approves the door and calls `swapFrom`.

Quotes come from the door's `quote` in both modes and a fill pays the same in both: a door forwards to the front's own
entries. `PoolMeta.approvalAddress` is the one contract the adapter may approve: the door in the door's mode, the
front in the front's. The front's mode is the cheaper, by 9,885 to 13,031 gas a fill at the receipt. The door's mode
stays for an executor that will only approve a contract that cannot change.

Gas is one figure a mode, an input token and a class of listing: what the fill adds to its transaction's receipt, that
is the adapter call frame's gas less the refund the fill earns through Kyber's executor (Refunds), the least first fill
measured of its class. On Ethereum, for a listing that reads a price source:

| Class | Input | Door mode | Front mode |
|---|---|---|---|
| on the pool's hook pair | the base, the pool's asset | 134,808 | 123,100 |
| on the pool's hook pair | the quote | 133,439 | 123,661 |
| past the hook pair, the base the pool's asset | the asset | 161,844 | 150,136 |
| past the hook pair, the base the pool's asset | the quote | 153,433 | 142,407 |
| past the hook pair, neither token the pool's asset | the base | 167,896 | 158,011 |
| past the hook pair, neither token the pool's asset | the quote | 171,112 | 158,141 |

Each was measured on a fork of Ethereum holding the deployed build, over wstETH, WETH, USDC and USDT pairs; a class's
figure is the least of its listings, so a new listing of an approved desk in one of these classes is quoted with no
config change. Same-size first fills vary by up to about 500 gas from run to run, and a listing whose token transfers
cheaper than these may fill a little under its class figure. Two listings of one class differ by their tokens'
transfer costs, by up to 2,005 gas, and `listingGas` gives a listing its own figures. A class never measured on the
deployed build has no figure and is quoted only where `listingGas` names the listing with its own two figures: a
listing that reads no price source, a pair past the hook pair whose quote is the pool's asset, and an asset-module
pair. The figures of a front read at the head (Base) are in `constant.go`: they were measured on the front's earlier
storage build on Base, a retired deployment, and not yet on the deployed one, whose desk is new code (the packed desk).
Between the two builds the project's own gas pins of a storage-build fill (not through the adapter) rose 0.3 % to
2.8 %, so these figures likely under-charge a Base fill by 1 % to 3 %.
No class was measured on BNB Chain: a listing there is quoted only with figures of its own (`listingGas`,
`creditListings`).

- **A pool holding none of the input.** Such a fill writes the pool's balance from zero and costs 17,044 more. The
  state marks it (`noneHeld`), and the simulator charges it until a fill of the route, through any listing of the
  pool, pays the pool some.
- **Refunds.** Kyber's executor runs the adapter in its own frame and keeps one base unit of each token it has routed
  (read on Ethereum, Base and BNB Chain). A fill returns the executor's input balance to that unit, which earns 2,800,
  and in the door's mode clears the door's allowance, which earns 19,900. Every figure nets those. The fills were
  measured from a caller that held none of the input, whose cleared balance earns 19,900, so each figure is that
  receipt plus 17,100 (`dustRefundGap`). A token the executor has never routed earns the larger refund on its first
  route alone. A refund is paid up to one fifth of its transaction's gas, which held for every figure's refund behind
  a bare test hop.
- **Gas limits.** Every figure is what a fill adds to a receipt, never a limit: the least gas limit that let a fill
  through the adapter was 21% to 29% above its receipt's gas, since each call forwards at most 63/64 of the gas left.
  Set a route's limit at 1.3 times its expected gas or more.

## Ethereum

A maker's update on Ethereum (its quote contract's quote for one block and, where the listing reads one, its price
print) lands only in a block a builder built with it ahead of a fill, and stands for that block alone. At the head a
door refuses every size. So on chain 1 the source reads the state the next block will hold:

- **The read path.** Each front names a `titan` block: the urls answering `titan_getPammStateOverrides` (or
  `titan_getPammVenues`, set by the front's `stateMethod`; the old method stays the default, and either answer shape is
  read whichever method asked), dialled and awaited as Kyber's `titan` package does, except that the merge window
  closes at once when url 0 has answered with an account for the highest block seen, so a slow later url cannot hold
  the front's lock. An answer names its target block, its beacon slot and, per quote contract, the slots that block's
  update writes; a venues answer names them per venue, and its venues are grouped by block.
- **One answer per listing.** The answers are never merged: a listing is read from the first answer, in the configured
  url order, of those for the highest block, that holds the listing's own update whole. A slot two entries of one
  answer disagree on is left out.
- **The slot rule** (`slot_rule.go`). A maker has one desk contract, the CREATE2 child of the front under its
  `CELL_CODEHASH()`, which must be the `cellCodehash` the config names. An answer covers a listing only where it
  writes the listing's quote word with the target block's stamp, of the desk epoch the head holds, with the desk not
  stopped. An answer that writes any other kind of slot of the desk contract covers none of that desk's listings.
- **What is quoted.** A listing is quoted only where the answer's block is the head's next and its `slot` the head's
  slot plus one. Every sample runs pinned to the head with the target block's number and time as block overrides and,
  as state overrides, the listing's quote contract and bound price contracts alone.
- **Validity.** A state is good for its target block alone, and the simulator holds it to the millisecond: its end
  (`validUntilMs`) is that block's time and its allowance (`allowanceMs`) is `inclusionAllowance` (1 s by default here),
  so route finding quotes a fill only while now plus the allowance is not past the end. `validUntil` stays beside them
  in whole seconds, the end less the allowance, rounded down. Where both the listing and the quote carry, the next block
  is sampled too and the lower output of each size kept.
- **The `NoWord` exposure.** A route that lands in a block without the maker's update reverts at the front (`NoWord`),
  and the taker pays the reverted transaction's gas: about 83,500 to 87,100 in the adapter's frame through the door on
  the price-source listings, and about 32,000 less through the front. The route's minimum output does not prevent it;
  the allowance bounds how often it happens.

### Credit-mode listings

A listing with a credit mode fills inside its pool's custody where that fills it whole and, past it, through the
pool's credit route, which may borrow the token it pays out. The door's `quote` answers what either fill pays and the
adapter fills both; the gas differs several times over. So such a listing is quoted, on every chain, only where
`creditListings` names it with figures measured on its own pool:

- Beside each `quote`, the refresh asks the front's plan for the size (`quoteFrom` with `PLAN_ONLY`), which says
  whether the size is routed: in each block a state is sampled in on a builder chain, and at each moment the window
  samples on Base, whose front has the same credit route and entries. A side's ladder is its custody points, then its
  routed points, and the simulator quotes nothing strictly between the two (`ErrCreditStep`).
- A route borrows or repays, which moves what every listing of the pool fills: a route is quoted only on a pool no
  fill of the route has touched, and once taken no listing of that pool is quoted again in that route.
- The route reads state the sample cannot pin (the lending venue's rate, a feed update). A fill that meets a change
  reverts at the taker's gas, as a `NoWord` does.
- **Measured** on a fork of Ethereum holding the deployed build, on the wstETH/WETH listing of a pool whose WETH is
  borrowed at Morpho Blue, through the door and through the front, from a caller that held none of the input: a
  custody fill of wstETH in 135,485 and 118,241, of WETH in 128,913 and 117,064, and a route fill of wstETH in 841,583
  and 824,339 (982,875 and 945,731 before refunds). Through Kyber's executor each receipt here is 17,100 higher
  (Refunds), and the Config example adds it.
  A fill of WETH in just past the custody is a route too. With no loan open it filled from 1.008 to 1.028 times the
  custody (to 1.036 with the pool holding some WETH) and reverted `NotEnoughDepth()` at the next size tried; with the
  pool owing 0.37 WETH it filled to 1.5 times the custody and reverted at 3. It cost 948,309 and 936,460 with the pool
  holding some WETH (1,072,509 and 1,040,760 before refunds), 966,765 and 954,916 with it holding none, and 1,061,009
  and 1,049,160 where it repays the loan (1,190,009 and 1,158,260 before refunds). One route figure a side stands for
  every state and path the route may take: it is the least, so a dearer fill is under-charged (the repaying one by
  112,700), and `routeLimitBaseIn` and `routeLimitQuoteIn` state the gas limit the dearest needs. Measure the figures
  again on the live pool after any change of its settings.
- **Not measured** on Base or BNB Chain. A route's gas is its pool's (its lending venue, its tokens), so a credit
  listing there is named with figures measured on its own pool once it is live. On a BNB Chain fork of a rehearsal, a
  route fill of QQQB into a pool that borrows USD1 at Lista paid the simulator's output to the wei, in a transaction of
  1,227,913 gas sent by an account straight to the door (`TestForkBNBFillsTheSampledQuote`): not an adapter figure.
  On Base no credit fill has run end to end on the deployed build: on a fork of its rehearsal the desk had no word,
  and the door refused every size. Enabling a Base credit listing means measuring its figures on its pool, and that
  measurement is also the first proof that a Base route fills as quoted.

## BNB Chain

**A fill on BNB Chain lands only as a bundle sent to its builder (48 Club).** The builder places a desk's update
ahead of the bundles that touch the desk, and only bundles. A fill sent to the public mempool finds no update in its
block, unless a carried word stands, and reverts (`NoWord`) at the taker's gas. So a BNB Chain front is quoted only
where its config sets `bundleOnly`, stating that every route through it goes to that builder as a bundle. It is off
by default, and while it is off the front's listings are listed and idle.

BNB Chain's front is the builder build with Ethereum's desk code, so it is read as on Ethereum, through `/state`
(`layout`, `cellCodehash` and `titan` are required), on BNB Chain's own clock:

- **Blocks.** 450 ms apart (450.4 ms on average over the 30,000 blocks to 135,693,393). An answer's `slot` is its
  block's number, or absent, and gives no time. Its block must be the head's next. The target block is taken at the
  head's milliseconds (`milliTimestamp` in the node's block; the start of its second where the node gives none) plus
  450 ms. The window samples that block's second and the second after, where a block comes late, and where the listing
  and its word carry, the block after, 450 ms on; each size keeps its least output.
- **Validity.** `validUntilMs` is the target block's time in milliseconds, and the allowance (`inclusionAllowanceMs`)
  is 100 ms by default: route finding stops 100 ms before the block. An allowance of a block or more is refused, as is
  one under 50 ms. The 100 ms is not measured: how long before a block the builder must hold a bundle to place the
  desk's update ahead of it is unknown (the builder's stated 100 ms is its cut-off for a desk's update). Where the
  `/state` answer for a block comes later than the allowance before it, nothing is quoted from it, and the tracker
  logs that once a minute at most a front. `validUntil`, in whole seconds, is the last second wholly before the end
  less the allowance, so a reader that counts any moment of a second as in it never quotes past the end.
- **The clock.** The front's clock is mode 3: it reads BNB Chain's millisecond precompile (BEP-706, address `0x70`)
  and, while that answers nothing, the block's second. The precompile answers nothing today. Every head read asks it,
  and a front in mode 3 is quoted only while it answers nothing: once it answers, the front reads each block's own
  milliseconds, which no block override sets, and nothing is quoted until this source places them.
- **Gas.** No class figure was measured on BNB Chain's build. A listing is quoted only where `listingGas` or
  `creditListings` names it with figures measured on its own pool, and is idle otherwise.
- **Tip cap.** `minTipCapWei` defaults to 0.1 gwei. BNB Chain's base fee is 0; on 25 Nov 2026 a block's tips were 0.05
  gwei at the 10th percentile and 0.1 gwei at the 90th. A cap of 0 is no cap, as on Ethereum.
- **Timings.** One `/state` fetch and one head probe serve a front's listings for 100 ms; a fetch waits 100 ms for the
  other urls once one answer holds an account, and times out at 400 ms unless `titan.timeout` says otherwise, at most
  450 ms, one block. A state that is not idle is asked for again at once (Refresh).
- **Logs.** A fill's `Swapped` is the desk contract's in either mode, as on Ethereum. The builder's own door, which
  settles the bundles it places, logs none.
- **Urls.** Nothing on chain 56 is quoted until the front's `titan.urls` name a url that answers for chain 56.

## Base

Base's front is the storage build with the packed desk: a maker's word stands in the desk's storage until it ends, so
every listing is read at the head and its window sampled by block overrides (Pricing). `layout`, `cellCodehash` and
`titan` are refused.

- **The clock.** The front's clock is the block count (mode 2) from an anchor block, 200 ms a block. `baseTime` names
  the OP Stack predeploy `0x4200000000000000000000000000000000000030`, whose stored sub-second a sample sets by an
  override.
- **Refresh.** The packed desk logs `DeskPush` only where its record asks and never `LaneFill`, so a post it does not
  log is quoted at the listing's next scheduled refresh, a second after its state's block. A fill through a door logs
  the door's `Swapped`; one through the front's entries logs the front's `PoolFill` alone. Both map to the door.
- **Gas.** The class figures were measured on the front's earlier storage build on Base, a retired deployment
  (Execution and gas). Measure them again on the deployed build, or give a listing its own (`listingGas`).
- **Tip cap.** A listing whose tip cap is 0 fills nothing on this build, so with no ladder it is idle.
- **Credit listings** are quoted where `creditListings` names them (Credit-mode listings).

## Config

Every address is a config field with no default. A front on Ethereum:

```json
{
  "dexID": "efficient-prop",
  "chainID": 1,
  "inclusionAllowance": 1,
  "approvedDesks": ["0x4f8dd270acE370dE2f6Ac5EBf3230cBE099a0753"],
  "fronts": [
    {
      "address": "0x0000000000403aeB66691063CC2fD42E71d6AcD6",
      "doorFactory": "0x39eD9993449C96503f6a629cCD2BB5F132D7cc93",
      "layout": "builder",
      "cellCodehash": "0xb4f7512e65eee793ac5939d62b8383d8544eb1ca850c276fd706e97a0b7938de",
      "fillMode": "front",
      "implementations": ["0x48C6efab497EeC8501483Ceb23B6623f626d8371"],
      "approvedSources": ["0xF166e802fb1994929b522f1ca5410a5f5BD1A366", "0xCa143657998DbeFb3AC6bb78342B8Cf3Db04dee8", "0xb9e048a763b0A6e57c8dEA7F53559b39B6c58949", "0x8f8F0f5c14F11024DBa87D3737Dd1c403F7d8D9B"],
      "titan": {"urls": ["https://<api host>/v1/1/state"]},
      "creditListings": [
        {"listing": 1, "custodyBaseIn": 135341, "custodyQuoteIn": 134164,
         "routeBaseIn": 841439, "routeQuoteIn": 953560}
      ]
    }
  ]
}
```

The sample leaves `approvedDesksURL` out: it is added when the api serves the path, and until then `approvedDesks`
rules.

These are Ethereum's deployed front (proxy and implementation), its door factory, its `CELL_CODEHASH()`, its four
price contracts and its first maker's desk contract (desk 1). The `creditListings` entry is the fork measurement above
plus 17,100 (Refunds) for a wstETH/WETH listing with id 1; ids and figures are checked against the live pool before the config is used.

A front on Base, read at the head: its deployed front, door factory, its price lane for USDe/USDC and its first
maker's desk contract (desk 1, the CREATE2 child of the front under the packed desk's code hash):

```json
{
  "dexID": "efficient-prop",
  "chainID": 8453,
  "approvedDesks": ["0x2b347C8e89520aF26d30e6932f6273B699Ae746A"],
  "fronts": [
    {
      "address": "0x0000000000403aeB66691063CC2fD42E71d6AcD6",
      "doorFactory": "0x39eD9993449C96503f6a629cCD2BB5F132D7cc93",
      "fillMode": "front",
      "implementations": ["0x48C6efab497EeC8501483Ceb23B6623f626d8371"],
      "approvedSources": ["0xd42d07791f520d86ac827463e40D07101c5f2329"],
      "baseTime": "0x4200000000000000000000000000000000000030"
    }
  ]
}
```

A front on BNB Chain, read through `/state`: its deployed front and door factory at the same addresses, its desk code
hash and desk 1 (both Ethereum's: the same desk code under the same front), its first price contract (QQQB/USD1), and
`bundleOnly`, which only an integrator that sends every route on chain 56 to the builder as a bundle may set. The
first listing planned (QQQB/USD1) borrows on fill: it is quoted only once `creditListings` names it with figures
measured on its pool.

```json
{
  "dexID": "efficient-prop",
  "chainID": 56,
  "approvedDesks": ["0x4f8dd270acE370dE2f6Ac5EBf3230cBE099a0753"],
  "fronts": [
    {
      "address": "0x0000000000403aeB66691063CC2fD42E71d6AcD6",
      "doorFactory": "0x39eD9993449C96503f6a629cCD2BB5F132D7cc93",
      "layout": "builder",
      "cellCodehash": "0xb4f7512e65eee793ac5939d62b8383d8544eb1ca850c276fd706e97a0b7938de",
      "fillMode": "door",
      "implementations": ["0x48C6efab497EeC8501483Ceb23B6623f626d8371"],
      "approvedSources": ["0xF166e802fb1994929b522f1ca5410a5f5BD1A366"],
      "titan": {"urls": ["https://<api host>/v1/56/state"]},
      "bundleOnly": true
    }
  ]
}
```

**`approvedDesks` is empty by default, and empty means nothing is listed or quoted.** A listing's maker is its desk, and
a desk's contract is the front's `deskCellOf` of the listing's desk; a listing bound to no desk is never quoted.
`approvedDesksURL` (optional, https only) answers the chain's approved desks as
`{"chainId":1,"updatedAt":<unix s>,"version":<n>,"bundleSha256":"0x…","desks":["0x…"]}`: `chainId`, `updatedAt` and
`desks` are read, any other field is ignored, an address is taken in either case, and an empty `desks` is a list that
approves nobody. Any other status than 200 is a failed fetch. It is fetched in the background every 15 s at most (a
failure is logged once a minute at most, and no refresh waits for the fetch), and its list rules while its last fetch is
at most `approvedDesksMaxAge` old (seconds, 600 by default), so a desk it drops is quoted no more within one fetch. Once
the list is older (the url failing, or answering a list that is refused), `approvedDesks` rules, less the desks that
list had left out: a failing url never brings a desk in. A list is refused whole where it names another chain, is above
64 KiB or 1,024 desks, names a bad address or no `updatedAt`, or is older than the one held.

| Field | Meaning |
|---|---|
| `fronts[].address`, `doorFactory` | Required. No door code hash is configured: the source reads `DOOR_CODEHASH()` from the factory |
| `fillMode` | Required, `front` or `door`. A pool is quoted only under the mode it was listed with |
| `implementations` | Required. List a new one only after checking its storage layout, clock and pricing rules are the ones this source reads |
| `approvedSources` | The price sources a listing may read and be quoted; none by default |
| `layout`, `cellCodehash`, `titan` | Required on Ethereum and BNB Chain, refused elsewhere. `layout` is `builder`; `titan.timeout` is in nanoseconds (unset: 1 s on Ethereum, 400 ms on BNB Chain; at most 10 s on Ethereum, 450 ms on BNB Chain) |
| `bundleOnly` | BNB Chain alone, off by default: set, it states that every route through the front goes to the chain's builder as a bundle, and the front's listings are quoted; off, they are idle. Refused elsewhere |
| `creditListings` | Every chain: `listing`, `custodyBaseIn`, `custodyQuoteIn`, `routeBaseIn`, `routeQuoteIn`, optional `routeLimitBaseIn`, `routeLimitQuoteIn`. A route figure of 0 leaves that side's routed sizes unquoted |
| `oneFillListings` | Listings that take one fill a route in each direction whatever their pool's fee, as an override of the fee read. A listing id in 1 to 2^24 - 1, named once. None by default |
| `listingGas` | A listing with no credit mode and its own `baseIn` and `quoteIn` figures, in place of its class's |
| `minTipCapWei` | The least nonzero tip cap listed. Unset, 2 gwei on Ethereum, 0.1 gwei on BNB Chain and none elsewhere; `1` lists every listing |
| `inclusionAllowance` | Seconds a routed fill takes to land: 1 on Ethereum and on chains with sub-second blocks (Base), 3 elsewhere |
| `inclusionAllowanceMs` | Builder chains alone, in place of `inclusionAllowance`: the same in milliseconds, under one block and at least 1,000 on Ethereum, 50 on BNB Chain. Unset, 1,000 on Ethereum and 100 on BNB Chain |
| `baseTime` | Head chains with a block-count clock: the chain's millisecond-time predeploy |

The 2 gwei default: over the day to block 26,486,699 the 90th-percentile priority fee of a block was at most 2.0 gwei
in 90% of blocks. The 0.1 gwei default on BNB Chain: BNB Chain, Tip cap.

## Tests

```sh
go test ./pkg/liquidity-source/efficient-prop
```

The unit tests run offline: the lister and the tracker against a fake chain whose doors walk concave levels, the
simulator against the door at random sizes, and the Ethereum and BNB Chain paths against fake `/state` servers and a
fake node (`TestBNB*`: the head's milliseconds, the late and carried blocks, the precompile's mode, gas of a listing's
own and no class figure, the tip floor and a cap of 0, `bundleOnly`, the fetch's merge window and timeout, the
whole-second end, the refresh schedule and the log of an empty window). The fork and live tests (`TestFork*`,
`TestLive*`) are skipped unless their `EFFICIENT_PROP_*` variables are set.

`TestLiveFrontsOnBaseAndBNBChain` (`EFFICIENT_PROP_LIVE=1`) reads the deployed front on Base and on BNB Chain from a
public node and checks it against each chain's deployment record: its desk and door code hashes, its implementation
and that implementation's code, desk 1's contract, its clock mode, and on BNB Chain that the millisecond precompile
answers nothing. It then lists every listing of desk 1 from the chain and refreshes each. A chain where the front, its
implementation, its door factory or desk 1's contract holds no code yet (not deployed, or deployed in part) is
skipped with a note. `EFFICIENT_PROP_LIVE_RPC_<chain id>` points a chain at another node, such as a rehearsal's fork.

`TestForkBNBFillsTheSampledQuote` (`EFFICIENT_PROP_BNB_FORK`, `EFFICIENT_PROP_TAKER`) runs on a local fork of BNB
Chain holding a rehearsal's deployment of its build with a listing of desk 1. It lists the listing from the front,
serves the desk's update for the head's next block as a `/state` answer, refreshes, lands the update and fills the
ladder's first size through the door in that block, which pays the simulator's output to the wei. The fork's blocks
carry no `milliTimestamp`, so it runs the start-of-second path alone: the state ends at the head's second plus 450 ms,
and the fill lands in the second after, past that end, which only a test with `StaleCheck` off can do. The path from
the head's milliseconds is covered by the fake node's tests alone.

`TestForkBuilderFillsTheSampledQuote` runs on a local fork of Ethereum that a stepper drives a block at a time with a
`/state` server answering for its next block: for each listing and direction it refreshes, checks the simulator
against `door.quote` under the same overrides, lands the update and fills the ladder's first size, which pays the
state's output to the wei, a credit listing's first routed size included.

`TestForkSimulatorAgainstTheDoor` and the other head-read fork tests run on a fork of Base.
`EFFICIENT_PROP_CREDIT_LISTINGS` (the config's JSON list) gives them a front's credit listings.
