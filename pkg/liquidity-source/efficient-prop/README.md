# efficient-prop

Efficient PropAMM listings. A PropAMM front holds any number of listings. Each listing is one pair on one pool:
its operator posts signed words and depth levels on chain, and the front fills a taker from them out of the pool's
custody. Every listing is reached through its own listing door, an `IPropAMM` contract with one pair:

| Entry | What it does |
|---|---|
| `quote(tokenIn, tokenOut, amountIn)` | View: what a fill of `amountIn` pays now, where the listing reads no price source or only approved ones (see Pricing); reverts where the listing does not fill it (`NoFill`, `UnsupportedPair`, or the front's own error) |
| `swapFrom(tokenIn, tokenOut, amountIn, minOut, recipient, deadline)` | Pull entry: takes `amountIn` from the caller by allowance and fills it |
| `swap(...)` | Push entry: the caller sends `amountIn` to the door first |
| `FRONT`, `LISTING`, `TOKEN0`, `TOKEN1` | Immutables: the front, the listing id and the pair |

A fill is all or nothing: a door never fills part of `amountIn`. Doors are created by the front's door factory
(`PropAMMListingDoors.open(listing)`, permissionless) at the CREATE2 address of the listing id, `doorOf(listing)`.

## Discovery

The config names each front and its door factory. Listing ids run from 1 in order, so the lister reads
`front.doorView(id)` and `factory.doorOf(id)` in batches of 64, pinned to one block, until a listing the front never
reserved, at most `maxListings` (4,096) a poll. A retired or delisted listing only loses its live flag, so there is no
gap before the last id. A door is listed only while its listing is live, which only the front's state says, so the
whole list is read on every poll and no creation log decides.

One pool per door: the pool address is the door, the tokens are `TOKEN0` and `TOKEN1`, and `StaticExtra` holds the
front, the listing id and the listing's pool (`custody`).

**Every maker's listings.** Listing ids are the front's own and span every desk, so the scan finds each maker's
listings with nothing in the config naming a desk, a maker or a door. What makes a door genuine is checked, never
trusted: it is the CREATE2 child of the configured factory under the factory's own door code hash
(`DOOR_CODEHASH()`), the factory answers the configured front, and the door's immutables name that front, that listing
and its pair. A listing is still quoted only where:

- its price source is approved (Pricing, How to approve a source). Anyone registers a source of any code, so this is
  the one row a new pair needs: its price contract's address in `approvedSources`;
- on Ethereum, its desk contract holds the code the front names (`CELL_CODEHASH`) and its desk's update for the
  next block is in a `/state` answer (Ethereum);
- its gas class has a measured figure in the front's fill mode (Execution and gas). A listing with a credit mode
  (`creditListings`) and a stock listing (`listingGas`) need their own figures, and are not quoted without them.

## Pricing

The source runs none of the front's math. Each refresh reads the listing (`doorView`), its pool's balances of both
tokens, the front's `clock()` and each side's top at one block, then asks the door's own `quote` at a ladder of sizes
in both directions in one multicall pinned to that block, at a tip of 0 (the front prices no level by the tip):

- A listing that reads a price source (fence kind PRINT or SESSION) is quoted only where every source it is bound to
  is one the front's config approves (`approvedSources`). Each refresh reads the listing's binding from the front's
  storage at the refresh block, so a listing its operator rebinds to another source is dropped at the next refresh. A
  listing whose binding the node does not answer is not quoted. The list is empty by default: until an operator of
  this integration names the protocol's own sources, only the listings that read no source (WORD and NONE) are
  quoted. On those, and on a listing whose every source is approved, the door's quote is what a fill pays at the same
  state.
- **How to approve a source.** A source is any contract registered on the front, which anyone may register, and the
  listing's operator binds it. The source's code runs inside the quote and again inside the fill, and it can tell the
  two apart: by what the call reads of its transaction and block, and by state the fill has already changed when the
  source runs. A source that tells them apart can fill a real transaction for less than the `eth_call` quoted, down to
  the route's minimum output. So approve a source only where its code cannot change and its answer comes from its own
  stored prices and the clock alone. Approval is by address: every entry anyone registers on an approved contract is
  approved, guarded or not. The same holds for a listing's two tokens: their code runs inside the fill, and this
  source screens no token. Route through a listing only where its tokens' code cannot change and moves exactly the
  amounts asked.
- Sizes come from the shared `ladder` grid: a sweep by the input token's decimals at a listing's first refresh, then
  a grid dense at the last ladder's knee, plus twice and four times the last ladder's top, and the side's top itself.
- A size the door refuses is no point, never a zero price. A direction's ladder is the run of passing sizes from the
  smallest up to the first refusal (or a falling output) after it, at most 64 points.
- The ladder holds for its whole window. A listing's levels may drift and widen with its word's age, and a premium
  lowers fills for a window after the maker's latest post, so the door pays less at a later second of the same state.
  The state is quoted for fills landing up to its age bound (10 s) plus the inclusion allowance after its block, and
  never after its word ends. The refresh samples every size again in the window's last block and, where a premium
  ends inside the window, in the premium's last block and in the first block at or after its end, each by an
  `eth_call` with a block override on the same pinned block, and keeps the least output of each size. A size the door
  refuses at any of them is no point.
- A listing that reads a price print is quoted only as far as that print lives: the refresh asks the door, in the
  last block of each second of the window, whether it still fills the least size it filled at the block, and
  `validUntil` follows the last second that fills. Its state is refreshed on its lane's post (Refresh).
- A listing bound to a guarded source (its entry's bit 247) is quoted with its source's hold sampled too. A guarded
  source prices a fill at the side's worst price over the prints its lane still holds, and a lane holds a replaced
  print for its `HOLD_MS` (at most 2,000 ms). The refresh samples every size at each block of the hold, keeps the
  least output and drops a size refused at any of them.
- Each sample reads its block's time and number beside its quotes, so a node that takes an override and ignores it is
  caught: a side whose sample cannot be placed is quoted no later than the last second its price cannot have moved.
- The front's implementation is checked at every refresh. The source reads the front's storage by its layout and
  mirrors its clock rule, which an upgrade of the front's proxy may change. So each refresh reads the proxy's ERC-1967
  implementation slot, and a front whose implementation is not in its config's `implementations` (required, no
  default) quotes nothing until an operator of this integration lists it.

**Listings not quoted.** This source quotes no:

- listing with a credit mode, but on Ethereum where its front's `creditListings` names it (Ethereum, Credit-mode
  listings);
- SESSION listing with a closed phase, or whose session word cannot be read: in that phase its levels ease as the
  print ages, so a quote can fall below every sample of its window;
- listing bound to a price source its front's config does not approve;
- listing of an asset-module pair (a tokenized stock's, the pair word's bit 15) that its front's `listingGas` does
  not name: its module's checks are in no class figure;
- tied listing (scopes bits 1-4 read as 8): its fills walk its desk's levels from its print, so their age runs from
  the print's time;
- listing whose nonzero tip cap is under `minTipCapWei`;
- listing of a front whose block-count clock may read earlier at a later block, or whose implementation is not
  listed.

**The taker's tip cap, a limit this source cannot enforce.** A listing caps the taker's tip, the priority fee above the
base fee: a fill sent at a higher tip reverts (`TipCapped`), and with it the whole route. Nothing in a route says what
the taker will tip, so neither the simulator nor the router can check it. The source keeps each listing's cap in its
state and in the pool meta (`tipCapWei`, wei per gas), and leaves out every listing whose nonzero cap is under the
config's `minTipCapWei` (default 0: none left out). On Ethereum's front a cap of 0 means no cap.

The simulator never pays more than the door at the refresh block. The front walks a listing's levels from the side's
fill memory, each level no better than the one before, and floors the output once, so the output is a floored concave
function of the input:

- At a sampled size it pays exactly the door's answer.
- Between two sizes it pays the floored chord. The points are checked too: a span whose neighbours are not
  consistent with a concave output (up to the floors of the outputs) pays the point below it.
- Under the smallest sampled size it quotes nothing (the listing's minimum fill), and past the largest nothing.
- It pays no more than the listing's pool holds of the output token. Listings of one pool share a swap limit
  (`RegisterUseSwapLimit`), keyed by pool and token, so a route split across two listings of one pool takes no more
  than one of them could fill.
- A direction takes one fill per route: a fill moves the side's fill memory, so a second fill would walk from further
  up than the ladder says. The other direction keeps its own levels.
- `StaleCheck`: route finding refuses a ladder older than 10 s, and quotes no fill that would land after its
  `validUntil`, the end of the window its ladder was sampled over. A fill lands some time after its route is built, so
  route finding stops `inclusionAllowance` seconds before `validUntil` (config; default 1 s on a chain whose blocks
  are under a second apart and 3 s on any other). With `StaleCheck` off, no age gate applies.
- A new word moves the price at once: until the next refresh the ladder prices the old word, and the 10 s age bound
  and the route's own minimum output protect the taker there.

## Refresh

A listing's price moves with every post of its desk's word or book and every fill of it, through any entry, so the
source asks for refreshes two ways:

- **Schedule.** The tracker implements `NextUpdateAt`: a refresh one second after the state's block time, event or
  none. On Ethereum, a second from now or at `validUntil` where that is sooner (Ethereum).
- **Logs.** The event parser (`EventParser`, a `poolfactory` decoder) maps each log that moves a listing to its door:
  - the front's logs naming a listing (a fill, a post of depth or controls, a status, setting, binding, key or lease
    change): its door, computed as the factory's `doorOf` does. A log naming a desk maps to the doors of every listing
    bound to it, and a pause or a pair's log to every door of the front;
  - a desk contract's `DeskPush` and `LaneFill`: the doors of every listing bound to that desk contract. A desk
    contract is not a configured address: the parser learns each listing's from the front (`doorView`'s desk, then
    `deskCellOf`, which must be the CREATE2 address of the desk) and learns again when a binding changes or a contract
    it does not know logs, at most once every 10 s;
  - a stored price lane's `Posted`, where the lane is in a front's `approvedSources`: the doors of every listing that
    reads a print bound to a source on that lane. A lane not approved is not followed;
  - a fill's `Swapped` (the standard event, once per settled fill). On a head chain the door called logs it. On
    Ethereum the filling listing's quote contract logs it, naming the front's caller as `sender`: where the sender is
    a door of a listing that contract holds, that door, and else every listing of the contract. Other venues log the
    same event, so a `Swapped` from an address the parser does not know maps to nothing and prompts no learning.

**Node load.** One refresh of a listing read at the head makes: for a listing that reads no source (WORD or NONE), 3
`eth_call` and 2 `eth_getStorageAt`; for one that reads a print on approved sources, 14 `eth_call` and 4
`eth_getStorageAt`; for one bound to a guarded source, 16 `eth_call` and 5 `eth_getStorageAt`. Each `eth_call` is one
multicall carrying the sizes it samples. On Ethereum one refresh makes one `/state` read of each url and, for a quoted
listing that reads no source, 4 `eth_call` and 7 `eth_getStorageAt`; a credit-mode listing adds no request. The
schedule refreshes a listing once a second, and a listing that reads a print is also refreshed at each post of its
lane; bounding that (at most one mapped refresh per listing per block of logs) is the service's matter. The parser
maps only the logs the service hands it: where it passes only the fronts' logs, the one-second schedule picks a post up.

## Execution and gas

Each front's config names its fill mode, `front` or `door`, with no default. Each pool carries it with the door, the
front and the listing id (`StaticExtra` and `PoolMeta`: `door`, `front`, `listing`, `mode`), and the consumer builds
the adapter's data from them, `abi.encode(door, front, listing, mode)`:

- Mode 1, `front`: the adapter reads `front.payPoolOf(listing)`. Where it names a pool (a listing with no credit
  mode), the adapter pays the input to that pool and calls the front's paid entry, `swapExactInPaid(tokenIn, tokenOut,
  amountIn, 1, recipient, block.timestamp, listing)`. Where it names none, the adapter approves the front and calls
  its named entry, `swapSender`, with the same arguments.
- Mode 0, `door`: the adapter approves the door and calls its pull entry, `swapFrom(tokenIn, tokenOut, amountIn, 1,
  recipient, block.timestamp)`.

Quotes come from the door's `quote` in both modes, and the pool's address is the door. `PoolMeta.approvalAddress` is
the one contract the adapter may approve: the door in the door's mode, the front in the front's. `amountUnused` is 0:
a fill is all or nothing, and the route's own minimum output protects it against a post landing before it.

Gas is one figure a mode, a side (the base in or the quote in) and a class of listing, each what the fill adds to
its transaction's receipt: the adapter call frame's gas (`executeEfficientProp`, in a real transaction) less the
refund the fill earns. Each figure is the least first fill measured of its class in its block, with the recipient
already holding the output token. The figures of a front read at the head, measured on a fork on the counted clock:

| Class | `front`, base in | `front`, quote in | `door`, base in | `door`, quote in |
|---|---|---|---|---|
| WORD or NONE, on the pool's hook pair | 101,664 | 93,570 | 114,072 | 109,146 |
| WORD or NONE, past the hook pair | 129,338 | 129,007 | 145,654 | 144,523 |
| reads a price source, on the hook pair | 114,226 | 106,376 | 126,634 | 121,952 |
| reads a price source, past the hook pair | 141,900 | 141,813 | 158,216 | 157,329 |
| reads a guarded source, on the hook pair | 120,205 | 112,108 | 132,360 | 127,678 |
| reads a guarded source, past the hook pair | 147,626 | 147,292 | 163,689 | 162,802 |

On the hook pair means the listing's base is its pool's token 0 and its quote token 1 (`doorView`'s base index 0 and
quote index 1); any other place is past it. `constant.go` gives each figure's measurement; past the hook pair the
price-source and guarded figures are the WORD figure plus a measured difference, a sum and not a measurement.

The refund is netted in every figure: 19,900 in the front's mode (the adapter's input balance, set and cleared within
the transaction), 39,800 in the door's (that and the door's allowance). A custody fill of a listing with a credit mode
earns 19,900 more in either mode, and its route fill 104,300 to 139,300. A refund is paid up to one fifth of its
transaction's gas. A fill alone in a transaction of a bare test hop was paid all of it on every class but one: the
credit listing's custody fill in the door's mode, paid 50,265 to 51,599 of its 59,700. A router heavier than that hop
is paid more of it, so the figure nets all of it and is short by up to 9,435 for a fill alone behind the lightest hop.

**Ethereum.** On chain 1 a front fills in either mode, and the front's is the cheaper and the one the config below
names (Ethereum, Entry mode). A listing's gas is one figure per input token and class: on the pool's hook pair; past
it with the pool's asset (its token 0) one of the pair's tokens; or past it with neither token the asset. The figure
follows the input token, not its side. Only pairs whose base is the asset were measured: a pair whose quote is the
asset is charged the same figures swapped, and that shape is measured before such a pair is listed. Same-size first
fills vary by up to about 500 gas from run to run, so a later run may see a fill a little under its figure.

| Class (door mode) | Input | Reads a price source | WORD or NONE |
|---|---|---|---|
| on the pool's hook pair | the base, the pool's asset | 117,708 | 107,621 |
| on the pool's hook pair | the quote | 116,339 | 106,052 |
| past the hook pair, the pool's asset one of its tokens | the asset, base or quote | 144,744 | 132,553 |
| past the hook pair, the pool's asset one of its tokens | the other token | 136,333 | 123,212 |
| past the hook pair, neither token the pool's asset | the base | 150,796 | 138,621 |
| past the hook pair, neither token the pool's asset | the quote | 154,012 | 141,125 |

| Class (front mode) | Input | Reads a price source |
|---|---|---|
| on the pool's hook pair | the base, the pool's asset | 106,000 |
| on the pool's hook pair | the quote | 106,561 |
| past the hook pair, the pool's asset one of its tokens | the asset, base or quote | 133,036 |
| past the hook pair, the pool's asset one of its tokens | the other token | 125,307 |
| past the hook pair, neither token the pool's asset | the base | 140,911 |
| past the hook pair, neither token the pool's asset | the quote | 141,041 |

The price-source figures past the hook pair were measured on a fork of Ethereum holding the deployed build (front
`0x0000000000403aeB66691063CC2fD42E71d6AcD6`) and a pool of wstETH, WETH, USDC and USDT: wstETH/USDC, wstETH/USDT and
both sides of USDT/USDC. A class's figure is the least of its listings. The two hook-pair figures were measured on the
same build on a pool of two tokens, wstETH and WETH, whose listing has no credit mode; the same pool with a lender
bound and unused spent the same within 16 gas. The front's mode came out 9,885 to 13,031 under the door's on the same
fills (29,785 to 32,931 before refunds: the door's mode earns one refund more). The WORD or NONE column was not
measured on the deployed build, whose measured pools list no pair that reads no price source: it is the price-source
column less a difference measured on an earlier build (10,087 with the base or the pool's asset in, 10,287 with the
other token in), and is measured before such a listing is served. No WORD or NONE class has a figure in the front's
mode: a listing of one is quoted there only where `listingGas` names it.

**A listing's own gas.** Two listings of one class differ by their tokens' own transfer costs: by up to 2,005 gas in
the door's mode and up to 1,141 in the front's. `listingGas` names a listing with its own two figures, which then
stand in place of its class's; the Ethereum config below names each launch listing so. A listing with a credit mode
takes its figures from `creditListings` alone.

**A pool holding none of the input.** A fill whose pool holds none of the input token writes the pool's balance of it
from zero and costs 17,044 to 17,068 more, as measured. The state marks each token its pool held none of at its block
(`noneHeld`), and the simulator charges a fill with that token in 17,044 more until a fill of the route pays the pool
some. The source does not know the recipient and charges nothing for one that holds none of the output token.

**Gas limits.** Every figure here is what a fill adds to a receipt, never a limit to give it: a fill must hold the gas
it spends before any refund, its figure plus the refund above, and more. The least gas limit that let a fill
through the adapter was 21% to 29% above the gas its receipt reports: 21% to 23% on the three price-source listings,
29% on the credit listing's custody fill and 24% on its borrowing fill, since each call down to the pool and its
lending venue forwards at most 63/64 of the gas left. A limit set from these figures for a route through a listing is
at least 1.3 times the gas the route is expected to use.

## Ethereum

Ethereum's blocks are built by builders, who sell a block's placement. A maker's update there (its quote contract's
quote for one block and, on a PRINT listing, the print its price contract reads) lands only in a block a builder built
with it ahead of a fill, and stands for that block alone. At the chain's head a listing's word is an older block's, and
its door refuses every size (`NoFill`). So on chain 1 the source reads the state the next block will hold:

- **The read path.** Each front names a `titan` block: the urls answering `titan_getPammStateOverrides` (JSON-RPC 2.0,
  no params) and an optional `timeout`. The urls are the protocol's api, `https://<api host>/v1/1/state`, and any
  builder's data host that publishes the same map. Each refresh asks every url, dialled as Kyber's `titan` package
  dials them, and waits as its client does (300 ms after the first answer holding an entry, at most the timeout). An
  answer names its target block (`blockNumber`), its beacon slot (`slot`) and, per quote contract, the slots that
  block's update writes to it and to its price contract. A url whose read fails is logged at Warn, at most once a
  minute per url, with its userinfo and query left out.
- **One answer per listing.** The answers are never merged. A listing is read from one answer: of the answers for the
  highest block any url names, the first in the configured url order that holds the listing's own update in its quote
  contract and, on a listing that reads a print, its price contract's entry. Where two entries of one answer give a
  slot two different values, that slot is left out, and a listing whose own slots include it is not read from that
  answer.
- **The slot rule.** Where a listing's update sits is one rule of the source (`slot_rule.go`). A maker has one desk
  contract, with a pair slot for each of its listings (`doorView`'s `wordIndex`). The desk contract is the front's
  `deskCellOf(desk)` and must be the CREATE2 child of the front under the front's `CELL_CODEHASH()`, which must be the
  `cellCodehash` the config names. An answer covers a listing only where it writes the listing's quote word (slot
  `0x100 + i`) with the target block's stamp, of the desk epoch the head's desk word (slot 1) holds, with the desk not
  stopped there. An answer that writes any other slot of the desk contract than a quote word or a firm word
  (`0x200 + i`) covers none of that desk's listings, and an answer with any entry that cannot be read covers no
  listing for its block. Several listings on one desk are each covered by their own words alone. An update that
  carries the desk's firm state is accepted, and no fill quoted here reads it: signed firm quotes (a maker's signature
  filled through the front's `fillQuote`) are not served by this integration.
- **Sampling.** Every call a refresh's state is made of is pinned to the head's block and runs with the target block's
  number and second as block overrides and, as state overrides, the listing's quote contract and its bound price
  contracts alone; any other account of the answer is dropped. The front's own storage reads stay plain reads at the
  head. The front's clock on Ethereum is the block's second (`clockMode` 0), so a time override places every sample.
- **What is quoted.** A listing is quoted only where the answer's block is the head's next, its `slot` is the head's
  slot plus one, the node ran the calls in that block, and the answer covers the listing by the slot rule. A PRINT or
  SESSION listing also needs the entry of the price contract it reads in that block. An answer for a block further
  ahead, one whose `slot` is later than the head's next and one with no `slot` quote nothing for that refresh; the
  next refresh, a second later, reads the next answer as usual. A listing whose maker has no update in the next block
  quotes nothing. Every other rule of Pricing holds.
- **Validity.** A state is good for its target block alone. Its `validUntil` is that block's second less the inclusion
  allowance (`inclusionAllowance`, 1 s by default on a front with a `titan` block). A longer allowance gives up more
  of each block's 12 s, and a shorter one quotes routes that land in a block without the update (below).
- **The carry rule.** The front lets a fill read a word in the block after its target where the listing's binding
  carries (`doorView`'s scopes bit 3), the word carries and the listing's fence kind is not NONE. Where both carry,
  the refresh also samples every size in the next block, keeps the lower output of each size, drops a size refused
  there, and is valid to that block's second less the allowance. No later block reads the word.
- **Refresh.** Nothing on chain logs a coming update. The tracker asks for a refresh at the earlier of the state's
  `validUntil` and a second from now, so a new map version is quoted within about a second of its publication.
- **Entry mode.** Both modes fill on chain 1, and the front's is the one to configure. A listing's door is a forwarder
  with no storage: its `quote` is the front's own for the listing, and its `swapFrom` calls the same two entries the
  adapter's front mode calls itself. So a fill pays the same output in either mode and a refusal is the same refusal,
  and the door's mode adds only its gas: 9,885 to 13,031 a fill on the price-source listings and 11,849 to 17,244 on
  the credit listing at the receipt, with a `NoWord` revert about 32,000 dearer too (about 100,000 against 68,000 on
  the credit listing). The door's mode stays a config option for an executor that will only approve a contract that
  cannot change: a door is immutable, where the front is a proxy whose owner can replace its implementation.
- **The `NoWord` exposure.** A route that lands in a block without the maker's update (one built by a builder that does
  not carry it, or a later block than the state's, past any carry) reverts at the front (`NoWord`), and the taker pays
  the gas of the reverted transaction: about 83,500 to 87,100 gas in the adapter's frame through the door on the
  price-source listings. The route's minimum output does not prevent that; the allowance bounds how often it happens.

### Credit-mode listings

A listing with a credit mode (`doorView`'s `creditMode`, 1 to 3) fills inside its pool's custody where that fills it
whole, and past it, where the pool's switches let it, through the pool's own swap: its credit route, which may borrow
the token it pays out. Its door's `quote` already answers what either fill pays, and the adapter fills both, in either
mode, through the front's named entry. What differs is the gas, several times higher on a route, and the output, which
need not continue across the step from custody to the route. So such a listing is quoted only on chain 1 and only
where its front's `creditListings` names it with gas figures measured on its own pool:

- **Each size's class.** Beside each door `quote`, the refresh asks the front's plan for the size (`quoteFrom` with
  `PLAN_ONLY`, bit 253). The plan answers a size it routes tagged with bit 255 (`ROUTED`), and a size it fills inside
  custody untagged, the door's own output. A size of neither is no point. An implementation that predates the plan
  read would class a routed size as custody: a front's config names no credit listing until its implementation has it.
- **One step a side, and nothing across it.** A side's ladder is its custody points, then its routed points. The
  simulator quotes nothing strictly between a side's last custody point and its first routed point (`ErrCreditStep`).
  At or below the last custody point it charges the custody figure, at or above the first routed point the route's.
- **The pool after a fill.** A route borrows or repays, which moves what every listing of the pool fills. Every
  listing of a pool shares a route key beside its token limits: a route is quoted only on a pool no fill of the route
  has touched, and once taken no listing of the pool is quoted again in that route (`ErrSecondFill` on its other side).
- **What moves after the sample.** The route reads state the target block's sample cannot pin (another fill's borrow
  that lifts the venue's rate, a feed update, the venue's own accrual). A fill that meets one reverts at the taker's
  gas, as a `NoWord` does, and the route's minimum output does not prevent it.
- **Gas.** Each figure is the least first fill of its class and input on that pool through the adapter in the front's
  configured mode, the frame less the refund the fill earns. On a fork of Ethereum holding the deployed build, the
  wstETH/WETH listing of a pool in credit mode 2, whose WETH is borrowed at Morpho Blue, cost through the door and
  through the front: a route fill of wstETH in 841,583 and 824,339, a custody fill of wstETH in 135,485 and 118,241,
  and a custody fill of WETH in 128,913 and 117,064. Before refunds a route fill spent up to 982,875 and 945,731, and
  a limit is set from those. A fill of WETH in past the custody was refused at every size tried, so that side has no
  route figure: a route figure of 0 says so, and that side is quoted inside its custody alone. A route's gas is its
  pool's settings as much as its venue's, so its figures are measured again on its live pool after any change of
  them. One route figure a side stands for every path and every lending venue the route may take: it is the least, so
  a dearer one is under-charged, and `routeLimitBaseIn` and `routeLimitQuoteIn` state the gas limit the dearest needs.

## Config

Every address is a config field with no default. A front on Ethereum:

```json
{
  "dexID": "efficient-prop",
  "chainID": 1,
  "minTipCapWei": 0,
  "inclusionAllowance": 1,
  "fronts": [
    {
      "address": "0x0000000000403aeB66691063CC2fD42E71d6AcD6",
      "doorFactory": "0x39eD9993449C96503f6a629cCD2BB5F132D7cc93",
      "layout": "builder",
      "cellCodehash": "0xb4f7512e65eee793ac5939d62b8383d8544eb1ca850c276fd706e97a0b7938de",
      "fillMode": "front",
      "implementations": ["0x48C6efab497EeC8501483Ceb23B6623f626d8371"],
      "approvedSources": ["0xF166e802fb1994929b522f1ca5410a5f5BD1A366", "0xCa143657998DbeFb3AC6bb78342B8Cf3Db04dee8", "0xb9e048a763b0A6e57c8dEA7F53559b39B6c58949", "0x8f8F0f5c14F11024DBa87D3737Dd1c403F7d8D9B"],
      "titan": {"urls": ["https://<api host>/v1/1/state", "<a builder's data host>"], "timeout": 2000000000},
      "listingGas": [
        {"listing": 2, "baseIn": 133036, "quoteIn": 125307},
        {"listing": 3, "baseIn": 133722, "quoteIn": 126448},
        {"listing": 4, "baseIn": 140911, "quoteIn": 141041}
      ],
      "creditListings": [
        {"listing": 1, "custodyBaseIn": 118241, "custodyQuoteIn": 117064, "routeBaseIn": 824339, "routeQuoteIn": 0}
      ]
    }
  ]
}
```

On Ethereum the front proxy is `0x0000000000403aeB66691063CC2fD42E71d6AcD6` (implementation
`0x48C6efab497EeC8501483Ceb23B6623f626d8371`), its listing-door factory `0x39eD9993449C96503f6a629cCD2BB5F132D7cc93`
(`DOOR_CODEHASH()` `0x0716310d0f4dac4c4f088ccf08c4fe5bfdfea57e9073fcd90a26feefc7c0d6d9`), and its `CELL_CODEHASH()`
`0xb4f7512e65eee793ac5939d62b8383d8544eb1ca850c276fd706e97a0b7938de`. Its price contracts 1 to 4 are the four
`approvedSources` above, and its first maker's quote contract (desk 1) is
`0x4f8dd270acE370dE2f6Ac5EBf3230cBE099a0753`, which the source derives and no config names. No listing is registered
on this front yet. Once the first is, this section names its pool and each listing's door, the factory's
`doorOf(listing)`. The listing ids and gas figures in the example are the launch rehearsal's, measured on a fork of the
deployed build in the front's mode: listing 1 wstETH/WETH (credit mode 2), 2 wstETH/USDC, 3 wstETH/USDT, 4 USDT/USDC.
They are checked against the live pool's ids, and the credit listing's figures measured on the live pool, before the
config is used. In the door's mode the same listings take `"fillMode": "door"` and the door's figures: `listingGas`
2: 144,744 and 138,338; 3: 145,430 and 136,333; 4: 150,796 and 154,012; `creditListings` 1: 135,485, 128,913, 841,583
and 0.

Per front: `address` and `doorFactory` are required, and no door code hash is configured (the source reads
`DOOR_CODEHASH()` from the deployed factory). `fillMode` is required, `front` or `door`; a pool keeps the mode it was
listed with. `implementations` lists the front implementations its proxy may run while its listings are quoted (at
least one, no default): list a new one only after checking that its storage layout, clock and pricing rules are the
ones this source reads. `approvedSources` lists the price sources a listing of the front may read and be quoted; none
by default (Pricing, How to approve a source). `listingGas` (any chain) names a listing with no credit mode and its
own two figures in the front's fill mode, each above 0 and at most 3,000,000. A listing named twice, or in both lists,
is refused at load. `inclusionAllowance` is the time a routed fill takes to land, in seconds.

On a builder chain (Ethereum alone today) `layout` and `cellCodehash` are required, and refused on any other chain.
`layout` names the layout of the front's quote contract; its only value is `builder`. `cellCodehash` is the front's
`CELL_CODEHASH()`: a front that answers another is refused. `titan` is required on chain 1 and refused on any other
chain: `urls` at least one, in the order a listing's answer is chosen by, and `timeout` in nanoseconds, a Go duration
as Kyber's `titan` client reads it (`2000000000` is 2 s; unset, it is 10 s; keep it well under the 12 s block).
`creditListings` (Ethereum alone) names each listing with a credit mode the front quotes, with its gas figures in the
front's fill mode, each at most 3,000,000: `{"listing": <id>, "custodyBaseIn": <gas>, "custodyQuoteIn": <gas>,
"routeBaseIn": <gas>, "routeQuoteIn": <gas>}`. The two custody figures are above 0; a route figure of 0 leaves that
side's routed sizes unquoted. `routeLimitBaseIn` and `routeLimitQuoteIn` (optional, each at least its side's route
figure, at most 6,000,000) state the gas limit the dearest route fill of the side needs. BNB Chain's front is a
builder build too, but this source has no next-block read for it, so no BNB front is configured.

No front is configured on Base: this build is not deployed there. A front on a chain read at the head sets no
`layout`, `cellCodehash`, `titan` or `creditListings`, and names `baseTime`, the chain's millisecond-time predeploy
(on Base `0x4200000000000000000000000000000000000030`), where its clock is the block-count mode.

## Tests

```sh
go test ./pkg/liquidity-source/efficient-prop
```

The unit tests run offline: the lister and the tracker against a fake chain whose doors walk concave levels, the
simulator against the door at random sizes, and on Ethereum the tracker against fake `/state` servers and a fake node
of a builder chain. The fork tests are skipped unless their variables are set:

```sh
EFFICIENT_PROP_RPC=<fork> EFFICIENT_PROP_FRONT=<front> EFFICIENT_PROP_FACTORY=<door factory> \
  EFFICIENT_PROP_CELL_CODEHASH=<the front's CELL_CODEHASH> EFFICIENT_PROP_SOURCES=<approved sources> \
  EFFICIENT_PROP_TITAN_URLS=<state urls> EFFICIENT_PROP_STEP=<stepper> EFFICIENT_PROP_TAKER=<taker> \
  EFFICIENT_PROP_PRINTS=<listings that read a print> \
  [EFFICIENT_PROP_FILL_MODE=front] [EFFICIENT_PROP_LISTING_GAS=<listing:baseIn:quoteIn,...>] \
  [EFFICIENT_PROP_CREDIT_LISTINGS=<listing:custodyBaseIn:custodyQuoteIn:routeBaseIn:routeQuoteIn,...>] \
  go test ./pkg/liquidity-source/efficient-prop -run TestForkBuilder -v
```

On Ethereum, `TestForkBuilderFillsTheSampledQuote` runs on a local fork that a stepper drives a block at a time, with
a `/state` server answering for the stepper's next block. For each listing and direction it refreshes through
`/state`, checks the simulator against `door.quote` under the same overrides, lands the updates and fills the ladder's
first size in that block in the fill mode under test: the fill pays the state's output to the wei, a credit listing's
first routed size included. With a listing's update left out of the next block, a refresh quotes nothing and the same
fill reverts `NoWord`. The other fork tests (`-run TestFork`) read a listing at the head and are skipped on a builder
chain, each saying so.
