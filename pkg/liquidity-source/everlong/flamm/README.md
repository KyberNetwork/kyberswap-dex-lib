# everlong-flamm

Everlong's Efficient pools: one pool holds any number of tokens (token 0 the pool asset), each pair priced by a
hook, with lending venues (Morpho Blue, Aave V3) funding payouts past custody. Every pool sits behind its Factory's
one implementation (a beacon), so one generic source covers every pool, hook, price feed and venue.

## Pricing

The source runs none of the pool's math. Each refresh probes the pool's own quote,
`previewSwap(tokenIn, tokenOut, amountIn) -> (amountInUsed, amountOut, feeWad)`, which runs the swap's own plan,
hook, feed and venue checks, at a ladder of sizes per direction, all pinned to one block and each made at the
KyberSwap executor's address (a hook prices a swap for its caller). Settlement is the pool's `swap`, adapter kind 0.

- The probes run in one `eth_call` a pool: the sampler (`abi/FlammSampler.sol`, MIT), set at the executor's address by
  a state override and never deployed. It walks every direction the pool routes at once: the seeds below, round robin;
  where a probe carries a tier label (below), the pool's `pairRequest`, on a buy the hook's `levState`, and where the
  probes carry that one label alone, the curve's fee by the hook's `quotePair`; then sizes that split a change of
  route, sizes at the edge of the pool's ceiling and, while probes are left, sizes that split each span whose chord
  may lose more than 1 bp to the pool's curve (estimated from the slopes beside it); last, one size just above a
  tier-labelled top. Each call into the pool or the hook runs under its own gas cap, 5M at first and doubled, up to
  20M, for a pool whose probe ran out of it, and the walk stops before the call's own gas (45M, `sampleGas`) runs out,
  with the probes it has.
- Sizes follow the input token's decimals (no price is needed). A pool's first refresh seeds 32 sizes a direction,
  geometric from 10^(decimals-3) units to 10^(decimals+9) units or the token's notional cap. Each later refresh seeds
  one size an eighth of the last ladder's first point, so that a first point a feed move raised comes back down, then
  20 sizes (`rungs`) geometric from that first point to eight times the last top, the notional cap bounding both. A
  pool's directions share 64 probes, at most 32 each, the hook pair first, and past 16 directions the rest are not
  routed that refresh. A first refresh spends its whole budget on its seeds, so its ladders are coarser than the
  next refresh's (Fork check).
- A node that does not run the sampler (it refuses the state override, the method or the call's gas, or answers
  with something that is not the sampler's) gets the same reads as one `eth_call` each from the executor, in JSON-RPC
  batches of 50: 16 sizes a direction, then the `pairRequest` reads and the probes above a tier-labelled top, then
  `levState` on a buy, with no sizes that split a span, so its ladders are coarser. Eight such refusals in a row,
  from any pool, send every refresh down that path for 10 minutes, and `disableSampler` sends them always. A rate
  limit, a block the node lacks, the call's timeout (3 s) or any other failure of the node fails the refresh, and the
  last state stands until it ages out.
- The probes are settled into a ladder by the shape the hook's family states (`shape`). EverlongHook's, `everlong`,
  pays the better of two routes, its curve and its leverage row, and labels each fill by the fee rate it reports: the
  row's two spreads (the tiers) or the curve's. The ladder ends at the first refusal once a size has paid 10,000 units
  (under that the band test's rounding makes sizes alternate between a price and a refusal), at the first output that
  falls, but for a fall from a tier-labelled point to the curve, at a third label, and at a fill the pool cuts
  (`amountInUsed` below the size asked): a curve fill cut at the top caps the ladder, so a larger input fills at that
  point and returns the rest. A tier-labelled point stands only where the pool's room, read by `pairRequest` (and on a
  buy the hook's `levState`), shows the leverage route pays it, and one near the top only where the ladder itself
  prices a size at least 1/1024 above it, so a tier-labelled size the ladder would end at is dropped and the ladder
  runs past it; on a pool of more than two tokens, or a hook whose row fails its gate, the ladder is cut before its
  first tier label.
  Only an execution revert is a refusal. A probe that runs out of its gas, or whose answer cannot be a fill, ends its
  ladder there. Where a direction's first answer other than a refusal ran out of gas, the refresh fails, the last
  state stands until it ages out, and the sampler asks that pool's probes at twice the cap the next time.
- The simulator is exact at every point, but a tier-labelled one the settling lowered: where the leverage route gives
  way to the curve, the point under the change pays the curve's price. Between two points it takes the chord less a
  haircut (1 bp), never less than the point below, only where the points around the segment are consistent with an
  output concave in the input (up to the floors of the outputs), so the chord stays under the pool's curve. Elsewhere
  (a fixed cost, a size tier, a branch that pays more past a size) it quotes the point below, and so does a flat span,
  one across a change of label, where the probes cannot tell where the route changed. Under the first probe it quotes
  nothing, whatever the ladder's shape: that probe is the smallest size priced, and the pool may refuse any less, as
  the floor of a dust output falls past its price band. Past the top of a ladder the pool did not cut it quotes
  nothing: one that ended at a refusal, or at the input's own notional cap, where the pool fills the cap and returns
  the rest.
- Pairs of one pool share its custody and venue book. One entity per pool carries all its tokens, and the simulator
  keeps one payout room per token for every pair paying it; each direction takes one fill per route, since the chain
  charges a second fill its own fee on the book the first moved, which the ladder cannot say. A fill moves only
  its own direction and the paid token's room: sibling pairs keep the prices probed until the next refresh. No inventory spans pools: each pool has its own custody and venue accounts. The source
  registers a swap limit all the same, for the gas alone: it counts the route's fills on these pools (see Gas) and
  never refuses or resizes a fill.
- A route may cross one pool twice, but a direction takes one fill per route: a second fill in a direction that
  already has one is refused with no quote, because the chain charges it its own fee on the book the first fill moved,
  and the ladder walked at their sum could pass what the chain pays (on a Base fork by up to 1.5 bp on a calm pool and
  36.4 bp with the volatility term at its cap, for two fills of up to $50,000 each). A fill the other way is quoted:
  it only moves the pool in the second's favour.
- A probed payout can still revert on chain: the state may move after the refresh, and a lending venue may refuse a
  position under its minimum. The router's minimum output covers the first. For the second, the ladders of a pair
  that Lista Moolah lends keep out of the window its minimum opens, as the refresh read it (see Gas), with 1 % of
  margin for its oracle; a minimum that moves further before the fill shows as a failed route.

## Refresh

Refresh these pools every 10 s or less, event or none: set pool-service's interval for this source to that. A ladder is
one block's quote, and some of what moves it logs from elsewhere than the pool: a feed round, an operator's tuning on
the hook, the Router's venue state. The tracker asks for the same through `NextUpdateAt` (10 s after the last refresh),
and `MaxAge` (30 s) takes a state too old out of routing under `StaleCheck`. It publishes no dependency set: one Router
serves many pools, so its logs would refresh them all, and a feed round logs from the aggregators behind the feed.

A refresh costs five requests to the node, one of them the sampler's call, and four on a pool with no live venue; the
first to meet a hook costs two more, for its code and its `params()`, which are kept, and a pool whose output a Lista
Moolah venue lends one more where no walk read its request (see Gas). On the plain path the probes go as JSON-RPC
batches of 50 in the sampler call's place, with a batch more where a probe is tier-labelled and one more for a buy's
`levState`. `GetNewPoolStateWithOverrides` refreshes on the state its caller overrides: the overrides reach every read,
and the sampler's code at the executor's address takes the place of whatever the caller set there.

A refresh that routes anything stamps `lv` in the Extra. pool-service saves only a state that changed, and a quiet
pool's ladders can repeat to the byte (on a Base fork, a pool with no debt and both directions at their notional caps),
which would stop its `Timestamp` and age a live pool out of routing. A pool with nothing routed repeats, and costs no
save.

## Gas

A fill is charged per fill, never above what it spends inside the pool's `swap` call net of the refund it earns. Each
ladder carries its cells (`cs`): the paths a fill of its side may take, for its pricer's build, the chain and each
venue kind live on its pair, each path at its least over those kinds (the table below). The simulator charges a fill
the least cell its reads allow, at the payout counted from the direction's start; the reads are the pool's request
for the direction (`pairRequest`) and the Router's venue records, at the refresh block:

- past the output's custody (`ho`), a venue pays it; past what the pool frees before a draw (`fr`: its held, supplied
  and posted output, and 1/10,000 more for the supply interest until the fill), a borrow does;
- a borrow is a first one where the output owes nothing (`oo`), an owing one where it owes, and either where a second
  live venue lends it (`fw`). Where the output owes nothing and the input has collateral posted (`pi`), it is either
  too: the pool posts only what a borrow needs past what is posted, often nothing, unlike every first borrow
  measured, so such a borrow may cost less than those, and it costs no less than an owing one;
- no repay where the input owes nothing (`oi`), and a repay where it owes and every venue lending it is live and
  takes repays now (`venueActions` & `CAN_REPAY`: `rp`). Where one of them holds a minimum position (Lista Moolah),
  the repay is certain only where it is the only one and the debt is past its minimum by 1 % and 4 units, as it
  takes nothing of a partial repay that would leave less. A supply of the input is never assumed.

Where no cell passes, a fill is charged the ladder's least cell. Once the simulator has taken a fill on the pool, the
next is another fill of the same route, in the same transaction: the first moved the custody, supply and debts the
reads saw and warmed what the next touches, so it is charged the ladder's second fill (`sc`), the least second-fill
charge of the pair's live venue kinds (the table below).

A pool's first fill costs less where another pool of this source filled earlier in the same transaction: the two share
contracts and storage (the hook's code, the Router, the price feed and what it reads, the tokens), which that fill
left warm. A pool's simulator sees its own fills only, so the route's swap limit carries the news. The source
registers one (`RegisterUseSwapLimit`), every pool reports its one key, `filled`, at zero (`CalculateLimit`), and each
fill applied with the limit raises it by one (`UpdateBalance`). A fill quoted with a limit whose count is above zero,
on a pool that has not filled itself, is charged the ladder's cells after another pool (`ac`): the same paths,
narrowed by the same reads, each at the least gas measured of such fills and never above its cell (the table below).
The count is one for the whole source in a request: the saving was measured with the other pool on another venue kind
and does not turn on the measured pool's kind, so the kind of the pool that filled is not asked. The pool that filled
is charged its second fill as before, and so is a pool's second fill after another pool. With no limit, a limit no
fill was applied with, or one without the key, every charge is the cells': the quotes of a router that passes none do
not change. The limit holds no inventory: it never refuses or resizes a fill.

Each refresh reads the pool's venues from its Router (`venueCount`, `venue`, `venueActions`); retired and fenced ones
are left out. It then reads, in one multicall, each live venue account's `capabilities()` and `minimumDebt(id)`, which
only an account that holds a minimum position (`CAP_HELD_REPAY`) answers. A pair is routed only where every record and
every `capabilities()` read, some live venue holds or lends one of its tokens, and every such venue is of a kind its
chain routes with cells for the pricer's build and the side: Base and Ethereum route kinds 0 (Morpho Blue) and 1 (Aave
V3), BSC kinds 1 (Aave V3), 2 (Venus Core) and 3 (Lista Moolah). `venueKinds` narrows the kinds, never widens them. A
pair with a live venue of any other kind, or priced by a build the table does not measure, is not routed: no charge
stands in for a fill that was not measured.

A Lista Moolah venue holds a minimum position: the Router passes it over where a draw would leave its debt under
that minimum, or within 4 units over it while it owes, and the pool's quote then refuses. With F what the pool frees
of the output before a draw, a payout past the output's custody and under E = F + minimum - owed (plus 4 where it
owes) can refuse, while one within custody, or at or over E, passes. A chord or a flat span cannot tell where that
window falls between two probed sizes, so a direction whose output such a venue lends keeps one piece of its ladder:

- the sizes whose payout, with what the protocol's part and the operator's share may add (the larger of the label's
  fee, three tenths of the band and a thousandth of the payout, each rounded up, plus one), fits the custody;
- or the sizes from the first paying at least E, with the minimum taken 1 % over for its oracle, quoting nothing under
  it. This piece is kept wherever it has a point.

Where the minimum did not read, or a second live venue lends the output, only the first piece stands, as a draw split
between venues can leave the minimum's share under it; where E is not past F, the whole ladder stands. The guard
needs the pool's request for the pair (its legs and band). A walk reads it past a tier-labelled probe; where no walk of
the pair did, the refresh reads it for such a direction at its first priced probe, in one batch more, and where that
read is refused the direction publishes nothing. `TestHeldRepayGuardVectors` checks the guard on
`testdata/held_repay_guard_vectors.json`, the same bytes every client routing these pools checks it on.

<!-- Generated by the gas table generator with gas_table.go: regenerate, never edit. -->

Each cell's charge is the least gas measured of its fills on a local fork of its chain, less its chain's route saving,
then less 1 % and rounded down to 1,000: the gas inside the pool's `swap` call, the input pulled from its caller, net
of the refund the call earns, which the transaction gets back. In a route the call costs less than a fill measured on
its own where the executor received the input earlier in the same transaction, holding dust of it (the pull finds its
balance warm and already written, and returns it to its starting value, which earns a refund), and where an earlier
leg paid the recipient the same token. The route saving is the most a fill's net fell from the one form to the other,
on the same state and size: Base 25,400, Ethereum 25,400, BSC 14,900; for the EverlongIdleEntryHook build, measured on
its own fills, Base 27,992, BSC 14,900. The EverlongIdleEntryHook build's cells stand above EverlongHook's by the
leverage route's pure computation, and on an unlocked hook by its variance fold and its read of a posted row besides:
on like fills, the venue idle, locked 28,185 to 35,728, unlocked 28,149 to 35,590, unlocked with a keeper's row posted
30,823 to 38,264 gas a fill; each cell is the least measured over every setting. Fills measured in the route form
count as they are. A cell is the hook build (the Hook column), the chain, the venue kind, the side (a sale pays token
1 for token 0) and the fill's path: what pays the output, and whether the input repays a debt first. Every key has a
custody cell, measured on fills whose input stays in custody too, as a supply of the input is never assumed: a fill
that supplies its input is charged under its gas.

| Hook | Chain | Venue kind | Side | Output paid by | Input repays | Charge |
|---|---|---|---|---|---|---|
| EverlongHook | Base | 0, Morpho Blue | sale | custody | no | 204,000 |
| EverlongHook | Base | 0, Morpho Blue | sale | a venue withdrawal or release | no | 329,000 |
| EverlongHook | Base | 0, Morpho Blue | sale | a first borrow | no | 478,000 |
| EverlongHook | Base | 0, Morpho Blue | sale | an owing borrow | no | 405,000 |
| EverlongHook | Base | 0, Morpho Blue | buy | custody | no | 204,000 |
| EverlongHook | Base | 0, Morpho Blue | buy | custody | yes | 326,000 |
| EverlongHook | Base | 0, Morpho Blue | buy | a venue withdrawal or release | yes | 420,000 |
| EverlongHook | Base | 1, Aave V3 | sale | custody | no | 205,000 |
| EverlongHook | Base | 1, Aave V3 | sale | a venue withdrawal or release | no | 514,000 |
| EverlongHook | Base | 1, Aave V3 | sale | a first borrow | no | 977,000 |
| EverlongHook | Base | 1, Aave V3 | sale | an owing borrow | no | 781,000 |
| EverlongHook | Base | 1, Aave V3 | buy | custody | no | 195,000 |
| EverlongHook | Base | 1, Aave V3 | buy | custody | yes | 519,000 |
| EverlongHook | Base | 1, Aave V3 | buy | a venue withdrawal or release | yes | 811,000 |
| EverlongHook | Ethereum | 0, Morpho Blue | sale | custody | no | 188,000 |
| EverlongHook | Ethereum | 0, Morpho Blue | sale | a venue withdrawal or release | no | 313,000 |
| EverlongHook | Ethereum | 0, Morpho Blue | sale | a first borrow | no | 463,000 |
| EverlongHook | Ethereum | 0, Morpho Blue | sale | an owing borrow | no | 390,000 |
| EverlongHook | Ethereum | 0, Morpho Blue | buy | custody | no | 188,000 |
| EverlongHook | Ethereum | 0, Morpho Blue | buy | custody | yes | 311,000 |
| EverlongHook | Ethereum | 0, Morpho Blue | buy | a venue withdrawal or release | yes | 401,000 |
| EverlongHook | Ethereum | 1, Aave V3 | sale | custody | no | 188,000 |
| EverlongHook | Ethereum | 1, Aave V3 | sale | a venue withdrawal or release | no | 478,000 |
| EverlongHook | Ethereum | 1, Aave V3 | sale | a first borrow | no | 930,000 |
| EverlongHook | Ethereum | 1, Aave V3 | sale | an owing borrow | no | 729,000 |
| EverlongHook | Ethereum | 1, Aave V3 | buy | custody | no | 178,000 |
| EverlongHook | Ethereum | 1, Aave V3 | buy | custody | yes | 492,000 |
| EverlongHook | Ethereum | 1, Aave V3 | buy | a venue withdrawal or release | yes | 767,000 |
| EverlongHook | BSC | 1, Aave V3 | sale | custody | no | 186,000 |
| EverlongHook | BSC | 1, Aave V3 | sale | a venue withdrawal or release | no | 466,000 |
| EverlongHook | BSC | 1, Aave V3 | sale | a first borrow | no | 876,000 |
| EverlongHook | BSC | 1, Aave V3 | sale | an owing borrow | no | 676,000 |
| EverlongHook | BSC | 1, Aave V3 | buy | custody | no | 172,000 |
| EverlongHook | BSC | 1, Aave V3 | buy | custody | yes | 475,000 |
| EverlongHook | BSC | 1, Aave V3 | buy | a venue withdrawal or release | yes | 708,000 |
| EverlongHook | BSC | 2, Venus Core | sale | custody | no | 302,000 |
| EverlongHook | BSC | 2, Venus Core | sale | a venue withdrawal or release | no | 812,000 |
| EverlongHook | BSC | 2, Venus Core | sale | a first borrow | no | 1,890,000 |
| EverlongHook | BSC | 2, Venus Core | sale | an owing borrow | no | 1,625,000 |
| EverlongHook | BSC | 2, Venus Core | buy | custody | no | 288,000 |
| EverlongHook | BSC | 2, Venus Core | buy | custody | yes | 799,000 |
| EverlongHook | BSC | 2, Venus Core | buy | a venue withdrawal or release | yes | 1,602,000 |
| EverlongHook | BSC | 3, Lista Moolah | sale | custody | no | 200,000 |
| EverlongHook | BSC | 3, Lista Moolah | sale | a first borrow | no | 779,000 |
| EverlongHook | BSC | 3, Lista Moolah | sale | an owing borrow | no | 671,000 |
| EverlongHook | BSC | 3, Lista Moolah | buy | custody | no | 186,000 |
| EverlongHook | BSC | 3, Lista Moolah | buy | custody | yes | 438,000 |
| EverlongHook | BSC | 3, Lista Moolah | buy | a venue withdrawal or release | yes | 415,000 |
| EverlongIdleEntryHook | Base | 0, Morpho Blue | sale | custody | no | 237,000 |
| EverlongIdleEntryHook | Base | 0, Morpho Blue | sale | a venue withdrawal or release | no | 358,000 |
| EverlongIdleEntryHook | Base | 0, Morpho Blue | sale | a first borrow | no | 504,000 |
| EverlongIdleEntryHook | Base | 0, Morpho Blue | sale | an owing borrow | no | 480,000 |
| EverlongIdleEntryHook | Base | 0, Morpho Blue | buy | custody | no | 230,000 |
| EverlongIdleEntryHook | Base | 0, Morpho Blue | buy | custody | yes | 362,000 |
| EverlongIdleEntryHook | Base | 0, Morpho Blue | buy | a venue withdrawal or release | yes | 448,000 |
| EverlongIdleEntryHook | Base | 1, Aave V3 | sale | custody | no | 237,000 |
| EverlongIdleEntryHook | Base | 1, Aave V3 | sale | a venue withdrawal or release | no | 594,000 |
| EverlongIdleEntryHook | Base | 1, Aave V3 | sale | a first borrow | no | 1,047,000 |
| EverlongIdleEntryHook | Base | 1, Aave V3 | sale | an owing borrow | no | 962,000 |
| EverlongIdleEntryHook | Base | 1, Aave V3 | buy | custody | no | 221,000 |
| EverlongIdleEntryHook | Base | 1, Aave V3 | buy | custody | yes | 555,000 |
| EverlongIdleEntryHook | Base | 1, Aave V3 | buy | a venue withdrawal or release | yes | 843,000 |
| EverlongIdleEntryHook | BSC | 3, Lista Moolah | sale | custody | no | 235,000 |
| EverlongIdleEntryHook | BSC | 3, Lista Moolah | sale | a venue withdrawal or release | no | 379,000 |
| EverlongIdleEntryHook | BSC | 3, Lista Moolah | sale | a first borrow | no | 807,000 |
| EverlongIdleEntryHook | BSC | 3, Lista Moolah | sale | an owing borrow | no | 753,000 |
| EverlongIdleEntryHook | BSC | 3, Lista Moolah | buy | custody | no | 215,000 |
| EverlongIdleEntryHook | BSC | 3, Lista Moolah | buy | custody | yes | 476,000 |
| EverlongIdleEntryHook | BSC | 3, Lista Moolah | buy | a venue withdrawal or release | yes | 661,000 |

A fill on a pool that another fill of the same route moved runs in the same transaction, on what that fill warmed: it
is charged its key's second fill, the least net gas measured of the second of two swaps of one pool in one transaction
(the same direction or the other, in either form), less 1 % and rounded down to 1,000.

| Hook | Chain | Venue kind | Side | Charge |
|---|---|---|---|---|
| EverlongHook | Base | 0, Morpho Blue | sale | 80,000 |
| EverlongHook | Base | 0, Morpho Blue | buy | 76,000 |
| EverlongHook | Base | 1, Aave V3 | sale | 82,000 |
| EverlongHook | Base | 1, Aave V3 | buy | 73,000 |
| EverlongHook | Ethereum | 0, Morpho Blue | sale | 77,000 |
| EverlongHook | Ethereum | 0, Morpho Blue | buy | 72,000 |
| EverlongHook | Ethereum | 1, Aave V3 | sale | 79,000 |
| EverlongHook | Ethereum | 1, Aave V3 | buy | 69,000 |
| EverlongHook | BSC | 1, Aave V3 | sale | 74,000 |
| EverlongHook | BSC | 1, Aave V3 | buy | 65,000 |
| EverlongHook | BSC | 2, Venus Core | sale | 77,000 |
| EverlongHook | BSC | 2, Venus Core | buy | 68,000 |
| EverlongHook | BSC | 3, Lista Moolah | sale | 78,000 |
| EverlongHook | BSC | 3, Lista Moolah | buy | 68,000 |
| EverlongIdleEntryHook | Base | 0, Morpho Blue | sale | 116,000 |
| EverlongIdleEntryHook | Base | 0, Morpho Blue | buy | 114,000 |
| EverlongIdleEntryHook | Base | 1, Aave V3 | sale | 118,000 |
| EverlongIdleEntryHook | Base | 1, Aave V3 | buy | 111,000 |
| EverlongIdleEntryHook | BSC | 3, Lista Moolah | sale | 113,000 |
| EverlongIdleEntryHook | BSC | 3, Lista Moolah | buy | 106,000 |

A pool's first fill after another pool of this source filled earlier in the same transaction runs on what that fill
warmed, the contracts and storage the two share: it is charged its path's cell below where the route's swap limit says
so, the least net gas measured of such fills (the other pool on another venue kind, the same pair and side), less its
chain's route saving and 1 %, rounded down to 1,000; and never above the cell's own least fill less the least fall
measured on the path, from a fill alone to the same fill after another pool, as that least fill may be one not
measured again after another pool. A path with no such fill measured keeps its cell above.

| Hook | Chain | Venue kind | Side | Output paid by | Input repays | After another pool | Alone |
|---|---|---|---|---|---|---|---|
| EverlongHook | Base | 0, Morpho Blue | sale | custody | no | 123,000 | 204,000 |
| EverlongHook | Base | 0, Morpho Blue | sale | a venue withdrawal or release | no | 242,000 | 329,000 |
| EverlongHook | Base | 0, Morpho Blue | sale | a first borrow | no | 390,000 | 478,000 |
| EverlongHook | Base | 0, Morpho Blue | sale | an owing borrow | no | 318,000 | 405,000 |
| EverlongHook | Base | 0, Morpho Blue | buy | custody | no | 121,000 | 204,000 |
| EverlongHook | Base | 0, Morpho Blue | buy | custody | yes | 238,000 | 326,000 |
| EverlongHook | Base | 0, Morpho Blue | buy | a venue withdrawal or release | yes | 333,000 | 420,000 |
| EverlongHook | Base | 1, Aave V3 | sale | custody | no | 123,000 | 205,000 |
| EverlongHook | Base | 1, Aave V3 | sale | a venue withdrawal or release | no | 426,000 | 514,000 |
| EverlongHook | Base | 1, Aave V3 | sale | a first borrow | no | 889,000 | 977,000 |
| EverlongHook | Base | 1, Aave V3 | sale | an owing borrow | no | 693,000 | 781,000 |
| EverlongHook | Base | 1, Aave V3 | buy | custody | no | 111,000 | 195,000 |
| EverlongHook | Base | 1, Aave V3 | buy | custody | yes | 432,000 | 519,000 |
| EverlongHook | Base | 1, Aave V3 | buy | a venue withdrawal or release | yes | 724,000 | 811,000 |
| EverlongHook | Ethereum | 0, Morpho Blue | sale | custody | no | 120,000 | 188,000 |
| EverlongHook | Ethereum | 0, Morpho Blue | sale | a venue withdrawal or release | no | 239,000 | 313,000 |
| EverlongHook | Ethereum | 0, Morpho Blue | sale | a first borrow | no | 388,000 | 463,000 |
| EverlongHook | Ethereum | 0, Morpho Blue | sale | an owing borrow | no | 315,000 | 390,000 |
| EverlongHook | Ethereum | 0, Morpho Blue | buy | custody | no | 117,000 | 188,000 |
| EverlongHook | Ethereum | 0, Morpho Blue | buy | custody | yes | 236,000 | 311,000 |
| EverlongHook | Ethereum | 0, Morpho Blue | buy | a venue withdrawal or release | yes | 326,000 | 401,000 |
| EverlongHook | Ethereum | 1, Aave V3 | sale | custody | no | 120,000 | 188,000 |
| EverlongHook | Ethereum | 1, Aave V3 | sale | a venue withdrawal or release | no | 404,000 | 478,000 |
| EverlongHook | Ethereum | 1, Aave V3 | sale | a first borrow | no | 855,000 | 930,000 |
| EverlongHook | Ethereum | 1, Aave V3 | sale | an owing borrow | no | 654,000 | 729,000 |
| EverlongHook | Ethereum | 1, Aave V3 | buy | custody | no | 108,000 | 178,000 |
| EverlongHook | Ethereum | 1, Aave V3 | buy | custody | yes | 417,000 | 492,000 |
| EverlongHook | Ethereum | 1, Aave V3 | buy | a venue withdrawal or release | yes | 692,000 | 767,000 |
| EverlongHook | BSC | 1, Aave V3 | sale | custody | no | 123,000 | 186,000 |
| EverlongHook | BSC | 1, Aave V3 | sale | a venue withdrawal or release | no | 399,000 | 466,000 |
| EverlongHook | BSC | 1, Aave V3 | sale | a first borrow | no | 809,000 | 876,000 |
| EverlongHook | BSC | 1, Aave V3 | sale | an owing borrow | no | 610,000 | 676,000 |
| EverlongHook | BSC | 1, Aave V3 | buy | custody | no | 112,000 | 172,000 |
| EverlongHook | BSC | 1, Aave V3 | buy | custody | yes | 412,000 | 475,000 |
| EverlongHook | BSC | 1, Aave V3 | buy | a venue withdrawal or release | yes | 645,000 | 708,000 |
| EverlongHook | BSC | 2, Venus Core | sale | custody | no | 239,000 | 302,000 |
| EverlongHook | BSC | 2, Venus Core | sale | a venue withdrawal or release | no | 745,000 | 812,000 |
| EverlongHook | BSC | 2, Venus Core | sale | a first borrow | no | 1,823,000 | 1,890,000 |
| EverlongHook | BSC | 2, Venus Core | sale | an owing borrow | no | 1,558,000 | 1,625,000 |
| EverlongHook | BSC | 2, Venus Core | buy | custody | no | 228,000 | 288,000 |
| EverlongHook | BSC | 2, Venus Core | buy | custody | yes | 737,000 | 799,000 |
| EverlongHook | BSC | 2, Venus Core | buy | a venue withdrawal or release | yes | 1,539,000 | 1,602,000 |
| EverlongHook | BSC | 3, Lista Moolah | sale | custody | no | 137,000 | 200,000 |
| EverlongHook | BSC | 3, Lista Moolah | sale | a first borrow | no | 686,000 | 779,000 |
| EverlongHook | BSC | 3, Lista Moolah | sale | an owing borrow | no | 578,000 | 671,000 |
| EverlongHook | BSC | 3, Lista Moolah | buy | custody | no | 126,000 | 186,000 |
| EverlongHook | BSC | 3, Lista Moolah | buy | custody | yes | 376,000 | 438,000 |
| EverlongHook | BSC | 3, Lista Moolah | buy | a venue withdrawal or release | yes | 353,000 | 415,000 |
| EverlongIdleEntryHook | Base | 0, Morpho Blue | sale | custody | no | 154,000 | 237,000 |
| EverlongIdleEntryHook | Base | 0, Morpho Blue | sale | a venue withdrawal or release | no | 268,000 | 358,000 |
| EverlongIdleEntryHook | Base | 0, Morpho Blue | sale | a first borrow | no | 413,000 | 504,000 |
| EverlongIdleEntryHook | Base | 0, Morpho Blue | sale | an owing borrow | no | 390,000 | 480,000 |
| EverlongIdleEntryHook | Base | 0, Morpho Blue | buy | custody | no | 144,000 | 230,000 |
| EverlongIdleEntryHook | Base | 0, Morpho Blue | buy | custody | yes | 272,000 | 362,000 |
| EverlongIdleEntryHook | Base | 0, Morpho Blue | buy | a venue withdrawal or release | yes | 358,000 | 448,000 |
| EverlongIdleEntryHook | Base | 1, Aave V3 | sale | custody | no | 154,000 | 237,000 |
| EverlongIdleEntryHook | Base | 1, Aave V3 | sale | a venue withdrawal or release | no | 503,000 | 594,000 |
| EverlongIdleEntryHook | Base | 1, Aave V3 | sale | a first borrow | no | 956,000 | 1,047,000 |
| EverlongIdleEntryHook | Base | 1, Aave V3 | sale | an owing borrow | no | 872,000 | 962,000 |
| EverlongIdleEntryHook | Base | 1, Aave V3 | buy | custody | no | 135,000 | 221,000 |
| EverlongIdleEntryHook | Base | 1, Aave V3 | buy | custody | yes | 464,000 | 555,000 |
| EverlongIdleEntryHook | Base | 1, Aave V3 | buy | a venue withdrawal or release | yes | 752,000 | 843,000 |
| EverlongIdleEntryHook | BSC | 3, Lista Moolah | sale | custody | no | 170,000 | 235,000 |
| EverlongIdleEntryHook | BSC | 3, Lista Moolah | sale | a venue withdrawal or release | no | 310,000 | 379,000 |
| EverlongIdleEntryHook | BSC | 3, Lista Moolah | sale | a first borrow | no | 738,000 | 807,000 |
| EverlongIdleEntryHook | BSC | 3, Lista Moolah | sale | an owing borrow | no | 683,000 | 753,000 |
| EverlongIdleEntryHook | BSC | 3, Lista Moolah | buy | custody | no | 152,000 | 215,000 |
| EverlongIdleEntryHook | BSC | 3, Lista Moolah | buy | custody | yes | 411,000 | 476,000 |
| EverlongIdleEntryHook | BSC | 3, Lista Moolah | buy | a venue withdrawal or release | yes | 596,000 | 661,000 |

Gas limits, for the route's transaction: the dearest swap call measured of each key before refunds, over every path,
state, size and recipient, times 1.10 and rounded up to 5,000. A route result carries one gas figure and no limit, so
the package carries none of these.

| Hook | Chain | Venue kind | Side | Dearest | Limit |
|---|---|---|---|---|---|
| EverlongHook | Base | 0, Morpho Blue | sale | 698,551 | 770,000 |
| EverlongHook | Base | 0, Morpho Blue | buy | 785,914 | 865,000 |
| EverlongHook | Base | 1, Aave V3 | sale | 1,263,077 | 1,390,000 |
| EverlongHook | Base | 1, Aave V3 | buy | 1,260,197 | 1,390,000 |
| EverlongHook | Ethereum | 0, Morpho Blue | sale | 686,768 | 760,000 |
| EverlongHook | Ethereum | 0, Morpho Blue | buy | 774,153 | 855,000 |
| EverlongHook | Ethereum | 1, Aave V3 | sale | 1,201,472 | 1,325,000 |
| EverlongHook | Ethereum | 1, Aave V3 | buy | 1,187,211 | 1,310,000 |
| EverlongHook | BSC | 1, Aave V3 | sale | 1,289,060 | 1,420,000 |
| EverlongHook | BSC | 1, Aave V3 | buy | 1,151,654 | 1,270,000 |
| EverlongHook | BSC | 2, Venus Core | sale | 2,505,658 | 2,760,000 |
| EverlongHook | BSC | 2, Venus Core | buy | 2,618,365 | 2,885,000 |
| EverlongHook | BSC | 3, Lista Moolah | sale | 984,972 | 1,085,000 |
| EverlongHook | BSC | 3, Lista Moolah | buy | 1,049,739 | 1,155,000 |
| EverlongIdleEntryHook | Base | 0, Morpho Blue | sale | 739,141 | 815,000 |
| EverlongIdleEntryHook | Base | 0, Morpho Blue | buy | 795,521 | 880,000 |
| EverlongIdleEntryHook | Base | 1, Aave V3 | sale | 1,346,581 | 1,485,000 |
| EverlongIdleEntryHook | Base | 1, Aave V3 | buy | 1,292,704 | 1,425,000 |
| EverlongIdleEntryHook | BSC | 3, Lista Moolah | sale | 1,048,795 | 1,155,000 |
| EverlongIdleEntryHook | BSC | 3, Lista Moolah | buy | 1,056,125 | 1,165,000 |

<!-- End of the generated tables. -->

## Gates (all fail closed)

- Provenance: only pools of the configured `factories` (several per chain), `isPool` re-read on every refresh.
  List the Efficient pools' Factories only: the Factory of the cbBTC/USDC pool this source quoted before is not
  one, and that pool fails the reads.
- Implementation: `poolCodeVersion()` must be in `codeVersions` (default `[1]`); every upgrade raises it. The earlier
  pool also reports 1, so the Factory list, not the version, keeps it out.
- Price feed: the pool's `priceFeed()` must be in `priceFeeds`, by address; a pool behind any other is never probed.
  With none listed nothing is routed: the list has no default, a chain's protocol feeds are named beside its
  Factories. A pool's creator picks its feed at birth and the hook prices from it, so a feed that tells a probe from a
  swap makes an honest hook quote one price and fill another, by up to 2 % measured on a Base fork, and a code hash
  would not vet it, as a build of the protocol's feed takes any aggregators at construction. A pool with no market band
  is held to the list too: it reads no feed now, but its operator can attach a band at once.
- Hook code: the pricer of a pair (the pair hook where its mask names the pair, else the swap hook) must run code of a
  listed family: keccak256 of the runtime code with its immutable ranges zeroed. With no `hookFamilies` in the config,
  the families are the protocol-listed builds (the swap hook builds the gas table measures, `gas_table.go`, and each
  chain's PropAMM front, `families.go`); a `hookFamilies` list replaces them, so a new family is a config entry. The
  family must state the shape its output takes (`shape: everlong`, the only one): a family stating none, or another,
  is not routed. A family's hook is asked its `params()` once, for its leverage row; a hook of the `everlong` shape
  prices tokens 0 and 1 only.
- Proxies: a hook is judged by its own code, whatever its storage holds: a proxy is not routed, nor is a copy of a
  front's build (`front: true`), and a hook the Factory lists as a front (`isListedFront`) is not routed by this
  build.
- `anyHook` adds the builds the gas table measures to a `hookFamilies` list that names others, and lifts no gate: a
  hook of no measured build has no cells, so it is never routed, whatever the list says.
- A pricer of no listed family is never probed, so a pool only costs probes once its hook is a listed build.
- PropAMM fronts: pairs a front prices (the swap hook of a pool whose cheap lane is on, or any pricer the Factory
  lists as a front) are not routed by this build, whatever `skipFrontPriced` says: their listings are reached through
  their own entries.
- State: paused pools and pairs closed in the trade mask (`tradablePairs`, bit `16*i + j`) have no ladder.

## Config

```yaml
dexID: everlong-flamm
chainID: 8453
factories: ["0x029EB597fb328844A807DC7662023244dE698C15"]   # the protocol's Factory, one address on every chain
priceFeeds: ["0xF9f642F747372b70e7473AB359C3C68F44f5605C"]  # the protocol's price feed; none: nothing routed
codeVersions: [1]             # optional; default [1]
hookFamilies:                 # optional; default: the protocol-listed builds (families.go)
  - name: everlong-hook
    codeHash: "0x..."
    immutables: [[offset, length], ...]
    shape: everlong           # how its output settles; none: not routed
    front: false              # true: a front's build, not routed by this build
venueKinds: [0, 1]            # optional; narrows the chain's routed venue kinds, never widens them (see Gas)
skipFrontPriced: true         # optional; does nothing in this build, which routes no front
anyHook: false                # optional; true adds the measured builds to hookFamilies (see Gates)
taker: "0x8f10b468b06c6fd214b65f87778827f7d113f996"  # optional; the executor, where the sampler runs
probeGas: 5000000             # optional; the sampler's cap on each call into a pool, doubled up to 20000000
sampleGas: 45000000           # optional; the sampler call's gas, at least 20717460
disableSampler: false         # optional; true: one eth_call a probe, the plain path (see Pricing)
haircutBps: 1                 # optional
rungs: 20                     # optional; sizes a later refresh seeds past the one under the last first point, 4 to 24
```

On BSC (`chainID: 56`) the source routes venue kinds 1 (Aave V3), 2 (Venus Core) and 3 (Lista Moolah), with the cells
the table measured there, and Lista Moolah's ladders behind the guard on its minimum (see Gas). `factories` and
`priceFeeds` take the same addresses as on Base: the protocol deploys its Factory and price feed at one address on every
chain. With none named, nothing is routed. The default `taker` is the KyberSwap executor's Base address, which holds the
same code on BSC; name the executor there if it differs. A first refresh seeds a direction from 10^(decimals - 3) of its
input, 1e15 wei of BTCB (about $75), which a pool seeded with less refuses past its price band: such a pool's sale stays
unrouted, never quoted over, until its seed grows. A Venus Core preview costs about twice an Aave V3 one, so a full walk
of both directions reaches the sampler's gas gate, which ends it with fewer points: never a failed refresh, and no probe
starves at the default cap on it alone.

## Fork check

`testdata/base-fork.json.gz` is a refresh recorded on a local Base fork with pools deployed by the protocol's own
deploy scripts (an EverlongHook pool on Morpho Blue and one on Aave V3, cbBTC/USDC), the sampler's answers among its
calls, and fills of each pool's `swap` through the adapter at the refresh block: at every probed size, between two,
under the first and past the top, for the first refresh and the next one. `TestForkReplayMatchesThePoolsSwap` replays
the recorded calls through the lister and the tracker, one sampler call of 64 probes a refresh, and checks every fill:
408 quoted, equal to the wei at every probed size, never over the pool; between probed sizes at most 3.09 bp under it
past the haircut on the second refresh, and 151.59 bp on the first, whose buy sizes are about two to one apart (a buy
of $96,354 near the top, where the price falls 4.4 % across the span); the 8 under the first probe the simulator
leaves unquoted, though the pool fills them; the 8 the pool refuses (past its price band) the simulator refuses too.
`TestForkRouteCrossingOnePoolTwice` checks routes that cross one pool twice, as two swaps of the chain in one block:
the simulator refuses a second fill of a direction with no quote, and quotes a fill the other way at no more than the
chain paid. `TestForkCappedPoolPaysItsTopAndReturnsTheRest` checks 40 fills of the two pools with USDC's notional cap lowered
to 30,000: past the top that the cap on a sale's output cut, the swap's one pass pays that top to the wei and returns
the rest, as the simulator says; past the cap on a buy's input the pool fills the cap and returns the rest, and the
simulator quotes nothing.

`testdata/bsc-fork.json.gz` is the same on a local BSC fork, with three BTCB/USDT pools deployed by the protocol's own
deploy scripts, one on each venue kind BSC routes (Aave V3, Venus Core and Lista Moolah), each worth about $100,000, and
`TestForkReplayMatchesThePoolsSwap` replays it the same way: every pool routes both directions on both refreshes, 696
fills quoted, equal to the wei at every probed size, never over the pool; between probed sizes at most 52.34 bp under it
past the haircut on the first refresh and 4.83 bp on the second; the 12 under the first probe the simulator leaves
unquoted, and the 12 the pool refuses (past its price band) the simulator refuses too. Lista lends the sale's output
there, owing nothing, and the window its minimum opens ends under the sale's first point, so its guard keeps the whole
sale.

`testdata/base-fork-lever.json.gz` is the same two pools after three sales, so that the leverage route wins a small
sale and the curve a large one, on a hook build of no listed family, which the fixture's config names. Near the top
of the leverage route's band its answer rounds: over a few sizes the leverage route answers, refuses and answers
again, and where it refuses the pool pays the curve's fill, about 3 % less (the far edge). Beside the two refreshes
the fixture holds fills of each direction through the adapter at 2,000 sizes a constant ratio apart across its
ladders, one unit either side of every point and 300 across each flat span (2,737 sizes a sale, 2,129 a buy), and
fills of each pool's sale at every size across its far edge and 200 either side. `TestForkReplayMatchesThePoolsSwap`
checks every quote of both refreshes there against the fill plus what the pool's rounding allows (`forkTolerance`:
1,539 units on a sale, .001539 USDC, and 4 sats on a buy): with the haircut none passes the fill, and without it, or
by the rule's chord alone, at most 626 units do. `TestForkLeverEdgeIsNeverQuotedOverThePool` replays each refresh's
walk and a sale's walk with two leverage rungs forced into the far edge: every leverage-labelled point published has
room inside the band past its margin and a size the ladder itself prices 1/1024 above it, and no quote passes a fill
by more than that tolerance. With the room test and the probe above off, the forced walk's chord passes the fill by
299.84 bp, which shows the far edge is there.
