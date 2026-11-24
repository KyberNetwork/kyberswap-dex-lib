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

The config names each front and its door factory. A front has no listing count: ids run from 1 in order, so the
lister reads `front.doorView(id)` and `factory.doorOf(id)` for id = 1, 2, ... in batches of 64, until a listing the
front never reserved (a zero pool), at most `maxListings` (4,096) a poll. An id once taken keeps its pool in
`doorView` for good: a retired or delisted listing only loses its live flag (`flags` bit 0), so there is no gap
before the last id. The lister keeps the live listings, scanning on past the others, and of them the doors whose
immutables name that front, that listing and its pair (a door not yet opened has no code and answers none). Every read
of a front is pinned to one block, and the factory must answer the configured front and a door codehash
(`DOOR_CODEHASH()`). Doors open at
any time, so the whole list is read on every poll; a front that reaches `maxListings` is logged. The scan decides
rather than the factory's `ListingDoorOpened` log, since a door is listed only while its listing is live, which only
the front's state says.

One pool per door: the pool address is the door, the tokens are `TOKEN0` and `TOKEN1`, and `StaticExtra` holds the
front, the listing id and the listing's pool (`custody`).

**Every maker's listings.** Listing ids are the front's own and span every desk, so the scan finds each maker's
listings with nothing in the config naming a desk, a maker or a door. A fill goes to the listing's own door, or in the
front's mode to the front with the listing id; a desk's maker door (`PropAMMMakerDoor`, one desk's venue for routers
that list a venue by address) is never read or called. What makes a door genuine is checked, never trusted: it is the
CREATE2 child of the configured factory under the factory's own door code hash, the factory answers the configured
front, and the door's immutables name that front, that listing and its pair. A listing is still quoted only where:

- its price source is approved (Pricing, How to approve a source). Anyone registers a source of any code, so this is
  the one row a new pair needs: its price contract's address in `approvedSources`. A listing bound to a source
  already approved needs none;
- on Ethereum, its desk contract holds the code the front names (`CELL_CODEHASH`) and its desk's update for the
  next block is in a `/state` answer (Ethereum). A desk whose updates the configured urls do not carry is not quoted;
- its gas class has a measured figure in the front's fill mode (Execution and gas). A listing with a credit mode
  (`creditListings`) and a stock listing (`listingGas`) need their own figures, and are not quoted without them.

Its tokens are not screened here (Pricing): the router's own token rules decide.

## Pricing

The source runs none of the front's math. Each refresh reads the listing (`doorView`), its pool's balances of both
tokens, the front's `clock()` and each side's top (the door's `quoteFillable` of the most the front takes) at one
block, then asks the door's own `quote` at a ladder of sizes in both directions in one multicall pinned to that block,
with the block's time and the listing's controls word beside it:

- The calls carry no gas price, so the quote is at a tip of 0: the door takes the tip from the call
  (`tx.gasprice - block.basefee`). The front prices no level by the tip, but a price source a listing reads runs in
  the fill's own frame (below).
- A listing that reads a price source (fence kind PRINT or SESSION) is quoted only where every source it is bound to
  is one the front's config approves (`approvedSources`). Each refresh reads the listing's bound source ids (record
  word 10: the primary at bits 0-31, and the backup at 32-63 where it has one) and each source's entry (its address,
  bits 0-159, and its guard, bit 247; an entry is written once and never edited) from the front's storage at the
  refresh block, so a listing its operator rebinds to another source is dropped at the next refresh. A listing bound
  to no source, or whose binding the node does not answer, is not quoted. The list is empty by default: until an
  operator of this integration names the protocol's own sources (Base's stored price lanes), only the listings that
  read no source (WORD and NONE) are quoted. On those, and on a listing whose every source is approved, the door's
  quote is what a fill pays at the same state.
- **How to approve a source.** A source is any contract registered on the front, which anyone may register, and the
  listing's operator binds it. The source's code runs inside the quote and again inside the fill, and it can tell the
  two apart: by what the call reads of its transaction and block, and by state the fill has already changed when the
  source runs. The front holds its lock during a fill, and in the front's fill mode the input is already at the pool.
  A source that tells them apart can fill a real transaction for less than the `eth_call` quoted, down to the route's
  minimum output. So approve a source only where its code cannot change and its answer comes from its own stored
  prices and the clock alone. Approval is by address: every entry anyone registers on an approved contract is
  approved, guarded or not. The same holds for a listing's two tokens: their code runs inside the fill and can tell it
  from a quote in the same ways, and this source screens no token, so it lists a listing whatever its two tokens are.
  Route through a listing only where its tokens' code cannot change and moves exactly the amounts asked.
- Sizes come from the shared `ladder` grid: a sweep by the input token's decimals at a listing's first refresh, then
  a grid dense at the last ladder's knee, plus twice and four times the last ladder's top, and the side's top itself,
  so the depth above the last size of the grid is quoted up to the most the listing fills.
- A size the door refuses is no point, never a zero price. A direction's ladder is the run of passing sizes from the
  smallest up to the first refusal (or a falling output) after it, at most 64 points.
- A listing with a credit mode is quoted only on Ethereum, where its front's config names it with its gas (Ethereum,
  Credit-mode listings): past its custody its fill goes to its pool's own swap, at another price and several times
  the gas, which the ladder marks size by size. Anywhere else it is not quoted.
- A SESSION listing with a closed phase is not quoted. Its session word (record word 13, read from the front's storage
  at the refresh block) gives a closed-phase delay, `closedAfter`, bits 0-15. Once the print is that old, the listing
  fills again under the closed phase's settings, each level clamped to a band around the stale print that eases as
  the print ages while the word's own levels tighten. So a quote in that phase can fall below every sample of its
  window, and the refused gap before it can be shorter than a second. A SESSION listing whose session word cannot be
  read is not quoted either. A SESSION listing with no closed phase (`closedAfter` 0) is quoted as a PRINT listing.
- The ladder holds for its whole window. A listing's levels may drift and widen with its word's age past the controls'
  grace, and a premium lowers fills for a window after the maker's latest post, so the door pays less at a later second
  of the same state. The state is quoted for fills landing up to its age bound (10 s) plus the inclusion allowance after
  its block, and never after its word ends. Over that window the front's discount on the listings this source quotes
  grows with age but for the premium's end and a guarded source's hold (below; the listings whose discount can fall with
  age in other ways are not quoted, see Listings not quoted), so the refresh samples every size again in the window's
  last block and, where a premium ends inside the window, in the premium's last block and in the first block at or after
  its end, each by an `eth_call` with a block override on the same pinned block, and keeps the least output of each
  size. A size the door refuses at any of them is no point: a side bound by its output (its desk's bucket, its pool's
  room) pays the most just after its premium lifts, and may refuse a size there that it fills before and after. A
  premium that ends by the moment the front's clock reads in the state's own block is over for every fill, and is not
  sampled: the refresh reads that moment beside the block's time (on a block-count clock the anchor's milliseconds where
  they fall in the block's second, else BaseTime's `timestampMs()`, as the front does).
- Where each sample is placed follows the front's clock as it reads at the refresh block. On a seconds clock (mode 0)
  the override sets the second. On a block-count clock (`clock().clockMode` 2, Base's) the front reads its count, 200
  ms a block from its anchor, in a block where the count falls in the block's second, and BaseTime's `timestampMs()`
  in any other. On Base the count sits on BaseTime's reading, and a block cadence other than 200 ms after the anchor
  moves it off. So:
  - The count on BaseTime's reading (or BaseTime not answering in the block's second): the override sets the block's
    number as well as its second, so the front's clock reads that block's count: the last 200 ms block of the window's
    last second, and the last block before the premium ends.
  - The count off BaseTime by less than a second: the front reads the count in the blocks where it falls in the
    block's second and BaseTime in the others, so its clock may read earlier at a later block, and no few samples bound
    what the door pays over the window. No listing of the front is quoted while that lasts.
  - The count a second or more off BaseTime: the front reads BaseTime in every block. A block override does not move
    the sub-second BaseTime stores (the chain writes it each block), so each sample also sets it (BaseTime's slot 0)
    by a state override, to the moment sampled: the window's end at the last millisecond of its last second, which no
    fill in that second is older than, the premium's last millisecond and the premium's end itself. Where the block
    such a moment needs reads the count instead (the count a second or so ahead of BaseTime), the moment is not
    placed.
  - Neither the count nor BaseTime in the block's second (the front reads the second's start): no sample is placed.
- A listing that reads a price print (fence kind PRINT or SESSION, `scopes` bits 1-2) is quoted only as far as that
  print lives. The front refuses every size once the print is past its source's window or the listing's max age, and the
  source assumes no later print: the refresh asks the door, in the last block of each second of the window on the pinned
  state, whether it still fills the least size it filled at the block, and the window ends at the last second of the run
  from the block that fills, before its first refusal. The end sample is taken there, and `validUntil` follows it. Route
  finding stops the inclusion allowance before `validUntil`, so after a post a listing on a source whose window is W
  seconds is quoted for about W - 1 - allowance seconds: on a 4 s source at Base's default allowance of 1 s, about 2 s
  of each print, and on a source whose window is no longer than 1 s plus the allowance, not at all. Its state is
  refreshed on its lane's post (see Refresh), so such a listing is quoted only while its lane keeps posting, and stops
  being quoted within those seconds once the lane stops. Where the print's life cannot be read (the node refuses
  overrides, or the clock is one no override places), the listing is quoted no later than its block's own second, that
  is not at all. No run reaches a closed phase: a SESSION listing with one is not quoted (above).
- A listing bound to a guarded source (its entry's bit 247) is quoted with its source's hold sampled too. A fill reading
  a guarded source asks its stored price lane for the fill's side's worst price over the prints the lane still holds: a
  post that replaces a print holds the replaced one for the lane's `HOLD_MS` (a public immutable of the lane, 400 ms on
  Base's hold lane, at most 2,000 ms), and once that has passed with no later post the side is priced on the kept print
  alone. So within a state's window the price is not one-way: when a hold ends it moves in the taker's favour, and a
  side bound by its output can refuse a size there for a few blocks. In the window's batch, pinned to the same block,
  the refresh also samples every size at each block from the state's to the first at or after `HOLD_MS` past the state's
  moment (on a block-count clock each 200 ms block, on a seconds clock each second), keeps the least output of each
  size, and drops a size refused at any of them. `HOLD_MS` is read from each guarded entry's lane in the pinned
  multicall; where it cannot be read the hold is taken as 2,000 ms. Every other sample of a print-reading listing (the
  window's and the print's ends, the premium's edges) is taken as on any other. Where the node refuses or ignores an
  override, or a block of the hold cannot be placed, a guarded listing's direction is not quoted at all. Nor is a
  guarded listing quoted where the front's block-count clock reads BaseTime (its count a second or more off): a block
  there may store any sub-second, so a moment just before the hold ends is no 200 ms step from the state's, and the hold
  cannot be sampled at every block. On Base the count sits on BaseTime's reading, so this does not arise while its
  blocks stay 200 ms apart.
- Each sample reads its block's time and number beside its quotes, and BaseTime where it set it, so a node that takes
  an override and ignores it is caught. Where the node refuses or ignores an override, or no sample is placed (above,
  and a clock that reads BaseTime alone or a precompile, modes 1 and 3), a side whose drift or widening is nonzero is
  quoted no later than the last second of its grace, and a side whose premium ends inside the window no later than
  the last second wholly before that end; so is a side whose premium edge cannot be placed. The controls word is read
  from the front's storage (its listing record's word 7), as no view returns it.
- The front's implementation is checked at every refresh. The source reads the front's storage by its layout (the
  controls, source and session words, the source entries), mirrors its clock rule and relies on the way its prices
  age, all of which an upgrade of the front's proxy may change. So each refresh reads the proxy's ERC-1967
  implementation slot at the refresh block, and a front whose implementation is not in its config's
  `implementations` (required, no default) quotes nothing until an operator of this integration lists it.

**Listings not quoted.** This source quotes no:

- listing with a credit mode, but on Ethereum where its front's `creditListings` names it (Ethereum, Credit-mode
  listings);
- SESSION listing with a closed phase, or whose session word cannot be read (above);
- listing bound to a price source its front's config does not approve;
- listing of an asset-module pair (a tokenized stock's, the pair word's bit 15) that its front's `listingGas` does
  not name: its module's checks are in no class figure (Stock listings, below);
- tied listing (scopes bits 1-4 read as 8): its fills walk its desk's levels from its print, so their age runs from
  the print's time;
- listing whose nonzero tip cap is under `minTipCapWei` (one whose cap is 0 on the front's storage build fills
  nothing, and the door refuses every size);
- listing of a front whose block-count clock may read earlier at a later block, or whose implementation is not
  listed;
- listing bound to a guarded source where the front's block-count clock reads BaseTime (above).

**Pool shapes.** A listing is read the same whatever its pool holds, and this source reads no lender:

- A pool of two tokens with no lender (credit mode 0, both tokens in the pool's custody) is a plain listing. Its
  state is the door's reads and the pool's two balances, and it is charged its class's figure or its own
  (`listingGas`), never a credit listing's: its fill reads no lender and never routes.
- A pool that lends its custody out (credit mode 0) is read the same: a size its custody cannot pay is refused by the
  door, so the ladder ends there.
- A pool whose listing has a credit mode is the credit listing above (Ethereum, Credit-mode listings).

**Stock listings.** A tokenized stock's pair carries an asset module, which every fill asks first: the listing
fills only in a session it allows, from a print inside that session's age ceiling and past its quiet window, with no
halt, latch, issuer pause or multiplier switch standing. The door's `quote` runs the same checks, so a listing its
module refuses has no point at the refresh block and is dark, and each second of its window is sampled as any
print-reading listing's is: a print that goes stale, or a check that starts refusing with time, ends the window. No
price is ever taken from a print the front would not fill on. A stock listing bound SESSION with a closed phase is
not quoted at any hour (above). Such a listing's gas is its own (`listingGas`, or `creditListings` where it has a
credit mode), since the module's checks are in no class figure: until its front's config names it, it is not quoted.

For every other listing a price only worsens with age within a window but where its premium ends and, on a guarded
source, where its lane's hold ends, and the samples above bound it. That is not true of the listings left out.

**The taker's tip cap, a limit this source cannot enforce.** A listing caps the taker's tip, the priority fee above the
base fee: a fill sent at a higher tip reverts (`TipCapped`), and with it the whole route. The front prices no level by
the tip (a price source might, which is why only approved sources are quoted). On the front's storage build, Base's, a
desk's word stands only while its listing's cap is set, so a listing that fills has a cap; one whose cap is 0 fills
nothing, and the door refuses every size. On a front that is not a storage build, a cap of 0 means no cap.
Nothing in a route says what the taker will tip, so neither the simulator nor the router can check it. The source keeps
each listing's cap in its state and in the pool meta (`tipCapWei`, wei per gas), and leaves out every listing whose
nonzero cap is under the config's `minTipCapWei` (default 0: none left out). A route through this source must be sent at
a tip no higher than the cap of each listing it fills.

The simulator never pays more than the door at the refresh block. The front walks a listing's levels from the side's
fill memory, each level no better than the one before (a post with a better later level is refused), and floors the
output once, so the output is a floored concave function of the input:

- At a sampled size it pays exactly the door's answer.
- Between two sizes it pays the floored chord. The points are checked too: a span whose neighbours are not
  consistent with a concave output (up to the floors of the outputs) pays the point below it.
- Under the smallest sampled size it quotes nothing (the listing's minimum fill), and past the largest nothing.
- It pays no more than the listing's pool holds of the output token. Listings of one pool share a swap limit
  (`RegisterUseSwapLimit`), keyed by pool and token: what the ladder's last point proves the pool pays out of that
  token, no more than it holds, the larger of its listings' standing. A pool's room for a token (its balance less any
  fee owed, its outflow bucket, its cap) is shared by its listings and may be under its balance; each ladder's top is
  a fill that room took at the refresh, so a route split across two listings of one pool takes no more than one of
  them could fill.
- A direction takes one fill per route: a fill moves the side's fill memory, rounded up to the listing's board unit,
  so a second fill would walk from further up than the ladder says. The other direction keeps its own levels.
- `StaleCheck`: route finding refuses a ladder older than 10 s, and quotes no fill that would land after its
  `validUntil`, the end of the window its ladder was sampled over: the state's age bound plus the inclusion allowance
  after its block, never after the last unix second in which a fill certainly finds the sampled word standing and the
  sampled sides' depth unexpired. So a simulator built from the state and kept prices no fill the samples did not cover.
  A fill lands some time after its route is built, so route finding stops `inclusionAllowance` seconds before
  `validUntil` (config; default 1 s on a chain whose blocks are under a second apart, Base, and 3 s on any other). A
  shorter allowance quotes a state for longer, which a listing whose print or word lives a few seconds needs; it relies,
  for a fill that lands later than the allowance, on the lane's or the desk's next post (a refresh) and on the route's
  own minimum output, as it does for a post between refreshes. The front fills from the desk's word while its age clock
  reads at most the word's maker time (`word` bits 72-119, ms) plus its life (bits 120-135, s) capped at
  `clock().maxTtlMs` (30 s on Base). Every clock mode reads the block's own second, so in a block of unix second T the
  age clock reads at most T x 1000 + max(999, `ageSlackMs`); a side's depth (`depth[0]` bid, `depth[2]` ask) fills while
  the block's second is under its expiry (bits 60-91). Once the word has expired the door refuses every size and a
  refresh has no ladder.
- A new word moves the price at once and restarts the word's life, so `validUntil` does not bound a post's change
  within it: until the next refresh the ladder prices the old word (see Refresh). The 10 s age bound and the route's
  own minimum output protect the taker there.

## Refresh

A listing's price moves with every post of its desk's word or book and every fill of it, through any entry, so the
source asks for refreshes two ways:

- **Schedule.** The tracker implements `NextUpdateAt`: a refresh one second after the state's block time, event or
  none, which is never after a quoted state's `validUntil` (route finding stops the inclusion allowance before it). On
  Ethereum, a second from now or at `validUntil` where that is sooner (see Ethereum).
- **Logs.** The event parser (`EventParser`, a `poolfactory` decoder) maps each log that moves a listing to its door:
  - the front's logs naming a listing: `PoolFill` (a fill through any door or entry), `DepthPosted`,
    `ControlsPosted`, `PoolStatus`, `SettingsSet`, `MakerKeySet`, `ListingBound`, `SourceDarkened`, `LeaseSet`,
    `AssetModuleSet`, `PoolReserved`, `PoolRegistered`, `PoolRetired`, `ListingDelisted` and `ListingRelisted`. The
    door is computed from the listing id, the configured factory and the door codehash the factory answers
    (`DOOR_CODEHASH()`, read once from the deployed factory and kept; CREATE2), as the factory's `doorOf` does. The
    config holds no door codehash: a door's address moves with its creation code, so no hash is taken from config.
  - the front's logs naming a desk, `DeskStopped`, `DeskRevived` and `DeskKeySet`: the doors of every listing bound
    to that desk.
  - the front's `AllPaused` and `AllUnpaused`, `PairRetired`, an `AssetModuleSet` of listing 0 (a pair's module
    bound or removed for good, or the protocol's) and a `LeaseSet` of listing 0 (every lease cut, or that cut lifted):
    every door of the front. The parser does not learn a listing's pair, so a pair's log refreshes them all.
  - a desk contract's `DeskPush` and `LaneFill`: the doors of every listing bound to that desk contract, however many
    (the logs name no pair). A desk contract is not a configured address: the parser learns each listing's from the
    front by the slot rule (`doorView`'s desk, then `deskCellOf`, which must be the CREATE2 address of the desk) and
    learns again after a `ListingBound`, a source change, a `DeskOpened` or a `PoolReserved`, or when a cell or desk
    it does not know logs (at most once every 10 s). It learns every listing the front reserved, live or not, so a
    listing relisted later is mapped as it is.
  - a stored price lane's `Posted` (a post of its prints; `Posted(uint256 stored)`, the lane the log's address), where
    the lane is in a front's `approvedSources`: the doors of every listing that reads a print (PRINT or SESSION, not
    tied) bound to a source on that lane, as its primary or its backup. The parser learns each such listing's binding
    from the front's storage (record word 10 and the entries it names) when it learns the desks' cells, and again
    after a `ListingBound` or a source change (a `SettingsSet` of kind 34, which the front's `setSource` logs), in
    JSON-RPC batches of 64 storage reads. Where the node refuses a batch, the front's lanes stay unmapped, which is
    logged, until the next learning, which any log asks for once the last is 10 s old; the desks' cells read in the
    same pass are kept. A post that stored nothing (`stored` 0) maps to nothing, and a lane not approved
    is not followed. So a listing's state is refreshed on its print's own post, not up to a second later by the
    schedule.
  - a fill's `Swapped` (the standard event, once per settled fill). Its emitter follows the front's build. On Base
    the door called logs it: that door. On Ethereum the filling listing's quote contract logs it and no door does,
    naming the front's caller as `sender`: where the sender is a door of a listing that contract holds, that door,
    and else (a fill through the front's own entries) every listing of the contract, as for `LaneFill`. Other
    venues log the same event, so a `Swapped` from an address the parser does not know maps to nothing and prompts
    no learning.

**Node load.** One refresh, on Base at the default allowance, makes:

- a listing that reads no source (WORD or NONE): 3 `eth_call` and 2 `eth_getStorageAt`;
- a listing that reads a print (PRINT or SESSION) on approved sources: 14 `eth_call` and 4 `eth_getStorageAt`. Eleven
  of the calls are the print's life, one a second of the window (its 10 s age bound plus the allowance);
- a listing bound to a guarded source: 16 `eth_call` and 5 `eth_getStorageAt`, the extra calls being the hold's
  blocks after the state's (two at a 400 ms hold) and the extra read its live source word.

Each `eth_call` is one multicall carrying the sizes it samples. The schedule refreshes a listing once a second, a
refresh starting the next second's count, and a log that moves it (its desk's or lane's post, a fill) brings one
sooner. So at one refresh a second a guarded listing costs a node about 1.8 million calls a day, a print listing about
1.6 million and a WORD listing about 430,000. A listing whose print is past its source's window, or that is not quoted
for another reason, costs less.

A listing that reads a print is also refreshed at each post of its lane, at most once a block where the service
refreshes once per block of logs, so these figures scale with the lane's posting rate. A lane's post needs no
particular sender, only a signed update newer than the print it stores, so a lane can be posted every block: on Base,
five blocks a second, a guarded listing then costs a node about 9 million calls a day and a print listing about 7.8
million. Bounding that (at most one mapped refresh per listing per block of logs) is the service's matter; the parser
makes no call per post once it has learned the lanes, and a post of a lane not approved costs nothing.

The limit: the parser maps only the logs the service hands it. The fronts and the approved lanes are configured
addresses; the desks' quote contracts are not, and a desk opened later has a new one. Where the service passes only
the logs of the fronts, no `DeskPush` or `Posted` reaches the parser, and a post is picked up by the one-second
schedule instead; a listing on a short source window is then quoted for up to a second less of each print (Pricing).

## Execution and gas

Each front's config names its fill mode, `front` or `door`, with no default. Each pool carries it with the door, the
front and the listing id (`StaticExtra` and `PoolMeta`: `door`, `front`, `listing`, `mode`), and the consumer builds
the adapter's data from them, `abi.encode(door, front, listing, mode)`:

- Mode 1, `front`: the adapter reads `front.payPoolOf(listing)`. Where it names a pool, as it does for every listing
  this source quotes in this mode (one with no credit mode), the adapter pays the input to that pool and calls the
  front's paid entry, `swapExactInPaid(tokenIn, tokenOut, amountIn, 1, recipient, block.timestamp, listing)`. Where
  it names none, the adapter approves the front and calls its named entry, `swapSender`, with the same arguments.
- Mode 0, `door`: the adapter approves the door and calls its pull entry, `swapFrom(tokenIn, tokenOut, amountIn, 1,
  recipient, block.timestamp)`, which takes the input from the adapter straight to the pool (on a listing with a
  credit mode, into the door, which fills through the front's named entry).

Each settled fill logs one `Swapped`, in either mode. On Base the door called logs it. On Ethereum the listing's
quote contract logs it, with the door as `sender`, and the door logs nothing: a block builder that places the maker's
update finds the fill at the contract the update writes.

Quotes come from the door's `quote` in both modes, and the pool's address is the door. `PoolMeta.approvalAddress` is
the one contract the adapter may approve: the door in the door's mode, the front in the front's. The route's own
minimum output protects the fill against a post landing between the refresh and the fill. `amountUnused` is 0: a
fill is all or nothing.

Gas is one figure a mode, a side (the base in or the quote in) and a class of listing, each what the fill adds to
its transaction's receipt: the adapter call frame's gas less the refund the fill earns. The basis of every figure:

- the counted clock: the fork's blocks 200 ms apart on the front's count, each block's second its own and BaseTime
  holding the matching sub-second, as on live Base, so the front reads its count and calls no millisecond predeploy;
- the first fill of that side in its block;
- the recipient holding the output token (one holding none costs a fill more, for the new balance; that is not
  charged);
- the adapter call frame (`executeEfficientProp`) of a real transaction, less the refund the frame earns (below);
- the tokens' own transfer cost as measured on WETH and USDC (the hook pair) and cbBTC and USDC (past it), Base's
  deployed token code; a pair of other tokens costs what its tokens' transfers cost.

Each figure is the least first fill measured of its class; the first fills of a class lay within about 1,050 gas of
it in the front's mode and 800 in the door's.

| Class | `front`, base in | `front`, quote in | `door`, base in | `door`, quote in |
|---|---|---|---|---|
| WORD or NONE, on the pool's hook pair | 101,664 | 93,570 | 114,072 | 109,146 |
| WORD or NONE, past the hook pair | 129,338 | 129,007 | 145,654 | 144,523 |
| reads a price source, on the hook pair | 114,226 | 106,376 | 126,634 | 121,952 |
| reads a price source, past the hook pair | 141,900 | 141,813 | 158,216 | 157,329 |
| reads a guarded source, on the hook pair | 120,205 | 112,108 | 132,360 | 127,678 |
| reads a guarded source, past the hook pair | 147,626 | 147,292 | 163,689 | 162,802 |

- On the hook pair means the listing's base is its pool's token 0 and its quote token 1 (`doorView`'s base index 0
  and quote index 1); any other place is past it. The hook pair was measured on a WORD WETH/USDC listing, past it on a
  WORD cbBTC/USDC listing (cbBTC the base) of a pool of WETH, USDC and cbBTC.
- A listing that reads a price source (PRINT or SESSION) adds the print difference measured on a PRINT WETH/USDC
  listing bound to the protocol's stored price lane against the WORD listing: 12,562 with the base in and 12,806 with
  the quote in, the same in both modes (its source word, the source's check and its two slots). On the hook pair the
  sum is that listing's own measured figure. Past the hook pair it is the WORD figure plus that difference: a sum, not
  a measurement.
- A listing that reads a guarded source (PRINT or SESSION) is charged its own figure, measured on a PRINT WETH/USDC
  listing bound to a guarded entry of Base's hold lane with no print held at the fill. The class follows the source the
  listing reads now, the guard (bit 247) of its record word 2, which a fill reads alone: a listing whose primary and
  backup differ in guard is charged by the one it stands on (its backup once its primary is darkened), and either
  entry's guard is taken where that word is not read. Its hold is sampled where either entry is guarded. Its difference
  over the WORD listing is 18,288 and 18,285 in the front's mode, 18,035 and 18,279 in the door's (base in, quote in).
  Past the hook pair it is the WORD figure plus that difference: a sum, not a measurement. A fill in the block of a post
  or the next, while the lane holds the replaced print, reads the held print too and costs about 5,000 more; that is not
  charged.
- A listing of an asset-module pair (the pair word's bit 15), whose fill also runs its module's checks, was not
  measured, and no class figure holds those checks: it is quoted only where its front's `listingGas` names it with
  figures measured on its own pool.

The refund is netted in every figure: 19,900 in the front's mode (the adapter's input balance, set and cleared within
the transaction), 39,800 in the door's (that and the door's allowance). On Ethereum's deployed build it was read as
each measured transaction's refund counter; the Base figures net the same two words. A custody fill of a listing with
a credit mode earns 19,900 more in either mode, and its route fill 104,300 to 139,300 (Ethereum, Credit-mode
listings). A refund is paid up to one fifth of its transaction's gas. A fill alone in a transaction of a bare test hop
was paid all of it on every class but one: the credit listing's custody fill in the door's mode, paid 50,265 to
51,599 of its 59,700. A router heavier than that hop is paid more of it, so the figure nets all of it and is
short by up to 9,435 for a fill alone behind the lightest hop. A gas limit is set from the gas before refunds (Gas
limits). The front's named entry, which a listing
with a credit mode takes, has no figure here: such a listing is quoted only on Ethereum, at the figures its config
names (Ethereum, Credit-mode listings).

**Ethereum.** On chain 1 a front fills in either mode, and the front's is the cheaper and the one the config below names
(Ethereum, Entry mode). A listing's gas is one figure per input token and class: on the pool's hook pair; past it with
the pool's asset (its token 0) one of the pair's tokens; or past it with neither token the asset (`doorView`'s base and
quote index both not 0). The figure follows the input token, not its side: past the hook pair with the asset, a fill
with the asset in is charged the asset-in figure whether the asset is the pair's base or its quote. Only pairs whose
base is the asset were measured: a pair whose quote is the asset is charged the same figures swapped, without a
measurement of its own, and that shape is measured before such a pair is listed. On the hook pair the base is always the
asset. With neither token the asset, the figures stay by base and quote. Each figure is the least first fill of its side
seen across runs through the adapter, less its refund, with the recipient holding the output token, the fill landing
behind the quote contracts' updates in their block and the front's clock the block's second. Same-size first fills vary
by up to about 500 gas from run to run, so a later run may see a fill a little under its figure.

| Class (door mode) | Input | Reads a price source | WORD or NONE |
|---|---|---|---|
| on the pool's hook pair | the base, the pool's asset | 117,708 | 107,621 |
| on the pool's hook pair | the quote | 116,339 | 106,052 |
| past the hook pair, the pool's asset one of its tokens | the asset, base or quote | 144,744 | 132,553 |
| past the hook pair, the pool's asset one of its tokens | the other token | 136,333 | 123,212 |
| past the hook pair, neither token the pool's asset | the base | 150,796 | 138,621 |
| past the hook pair, neither token the pool's asset | the quote | 154,012 | 141,125 |

The four price-source figures past the hook pair were measured on a fork of Ethereum holding the deployed build (front
`0x0000000000403aeB66691063CC2fD42E71d6AcD6`) and a pool of wstETH, WETH, USDC and USDT: the asset in on wstETH/USDC
(144,744; wstETH/USDT spends 145,430), the other token in on wstETH/USDT (136,333; wstETH/USDC spends 138,338), and both
sides of USDT/USDC. A class's figure is the least of its listings. The two hook-pair figures were measured on the same
build on a pool of two tokens, wstETH and WETH, whose listing has no credit mode: 117,708 with wstETH in and 116,339
with WETH in. That pool had no lender, and the same pool with a lender bound and unused spent the same within 16 gas:
the class follows the listing's credit mode, not its pool's lenders. A hook-pair listing with a credit mode is charged
its configured figures instead (`creditListings`), which are higher, since its fill reads its pool's room and its input
passes through the door. A pair with USDT or USDC in it spends a few thousand more than these two tokens do, so name
such a listing in `listingGas`. The WORD or NONE column is what a listing that reads no price source is charged, a NONE
listing included. None of it was measured on the deployed build (the measured pools list no pair that reads no price
source): measure it before such a listing is served. Past the hook pair it is the earlier build's. On the hook pair it
is a sum: the deployed build's price-source figure less the earlier build's difference below. The earlier build's own
figures there were 115,769 and 109,098, and the deployed build's price-source figures on that pair came out 8,148 and
3,046 under the earlier build's; a charge must not stand above a fill's gas. On the earlier build it was lower than the
price-source column by 10,087 with the base or the pool's asset in, and by 10,287 with the other token in, on every
class. A carried fill (a quote sampled in block N and filled in N+1) costs about 2,400 more than the same quote filled
in N, on either input; that is not charged. A listing bound to a guarded entry has no figure there: Ethereum's sources
are price contracts, never guarded, and such a listing is not quoted. The frame's refund, 39,800, is netted in each. A
fill that lands in a block without its quote contract's update reverts `NoWord` after about 83,500 to 87,100 gas in the
frame, and a revert earns no refund. The update for the block is paid by whoever lands it, never by the taker, and is in
none of these figures.

| Class (front mode) | Input | Reads a price source |
|---|---|---|
| on the pool's hook pair | the base, the pool's asset | 106,000 |
| on the pool's hook pair | the quote | 106,561 |
| past the hook pair, the pool's asset one of its tokens | the asset, base or quote | 133,036 |
| past the hook pair, the pool's asset one of its tokens | the other token | 125,307 |
| past the hook pair, neither token the pool's asset | the base | 140,911 |
| past the hook pair, neither token the pool's asset | the quote | 141,041 |

The front's mode was measured on the same fork and pool, through the adapter's frame, on the same fills: wstETH/USDC
133,036 with wstETH in and 125,307 with USDC in, wstETH/USDT 133,722 and 126,448, USDT/USDC 140,911 with USDT in and
141,041 with USDC in, each 9,885 to 13,031 under the door's mode (29,785 to 32,931 before
refunds: the door's mode earns one refund more). A class's figure is again the least of its listings.
The hook pair's two were measured on the two-token pool above, the same fills, 11,708 and 9,778 under the door's mode.
No WORD or NONE class was measured in the front's mode: a listing of one is quoted there only where `listingGas` names
it with figures of its own. The frame's refund in this mode, 19,900, is netted in each.

**A listing's own gas.** Two listings of one class differ by their tokens' own transfer costs: by up to 2,005 gas in
the door's mode (USDC in on wstETH/USDC spends 138,338 against its class's 136,333) and up to 1,141 in the front's
(USDT in on wstETH/USDT, 126,448 against 125,307), so the class's figure is short for the dearer listing by that much.
`listingGas` names a listing with its own two figures, which then stand in place of its class's; the Ethereum config
below names each launch listing so (Config). A listing with a credit mode takes its figures from `creditListings`
alone.

**A pool holding none of the input.** A fill whose pool holds none of the input token writes the pool's balance of it
from zero and costs 17,044 to 17,068 more: measured on the wstETH/WETH listing with WETH in, 145,957 against 128,913
through the door and 134,132 against 117,064 through the front (the EVM's own difference between a storage word set
from zero and one changed is 17,100). The state marks each token its pool held none of at its block (`noneHeld`, from
the balance read itself, since a side with no ladder zeroes its output's reserve), and the simulator charges a fill
with that token in 17,044 more, on every chain and in either mode, until a fill of the route pays the pool some. A
recipient holding none of the output token costs 17,100 more by the same rule; the source does not know the recipient
and charges nothing for it.

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
  builder's data host that publishes the same map. Each refresh asks every url, with an RPC client per url dialled as
  Kyber's `titan` package dials them, and waits as its client does (300 ms after the first answer holding an entry, at
  most the timeout). An answer names its target block (`blockNumber`), its beacon slot (`slot`, whose second is
  1,606,824,023 + 12 x `slot`) and, per quote contract, the slots that block's update writes to it and to its price
  contract. A url whose read fails (an error, a refusal or the timeout) is logged at Warn, at most once a minute per
  url, with its userinfo and query left out of the url and of the error's text, where an HTTP error names them; so is a
  url that does not dial, once, as the source starts, and it is left out of the reads; a url still answering when the
  300 ms wait ends is cut off, not failed, and is not logged.
- **One answer per listing.** The answers are never merged. A listing is read from one answer: of the answers for the
  highest block any url names, the first in the configured url order that holds the listing's own update in its quote
  contract (below, What is quoted) and, on a listing that reads a print, its price contract's entry. The api's map and
  a builder's are therefore combined by the highest block, then the url order, and a quote contract's slots are never
  taken from two answers in one read. Where two entries of one answer give a slot two different values, that slot is
  left out, and a listing whose own slots include it is not read from that answer; a listing whose slots do not
  include it still is. Where they give an account two balances or nonces, both are left out (the head's stand), which
  darkens no desk contract's listing (it owns slots, not the account). A price contract with any value split is left out
  whole.
- **The slot rule.** Where a listing's update sits is one rule of the source (`slot_rule.go`), the per-desk rule of the
  builder code. A maker has one desk contract, with a pair slot for each of its listings (`doorView`'s `wordIndex`,
  0 to 255).
  - *Account.* The listing's desk contract is the front's `deskCellOf(desk)` of `doorView`'s desk (an unbound
    listing reads desk 0 and is not quoted). It must be the CREATE2 child of the front at salt `(1 << 252) | desk`
    under the front's `CELL_CODEHASH()`, which the source reads from the front once, and which must be the
    `cellCodehash` the config names: a front answering another is refused, and a `deskCellOf` naming any other
    account leaves the listing unquoted.
  - *Words.* Pair slot `i`'s quote word is at slot `0x100 + i`, today's single word bit for bit: stamp at bits
    0-31, the desk epoch it was written under at 32-62, bit 63 always 0, carry at 201, firm at 205. Its firm word is at
    `0x200 + i` and counts only where the quote word's bit 205 is set. Slot 1 is the desk word: pusher 0-159, the desk
    epoch 160-190, stopped at bit 224, the stop's block 225-255; slots 2 and 3 are the registry and the word key. Only
    the front's own ops write slots 1 to 3.
  - *Coverage.* An answer covers a listing only where all hold: it writes the listing's quote word with the target
    block's stamp (or the block before's where the listing and the word both carry, and the head already holds that
    very word, and its firm word where the answer writes one, since only the head's block could have landed them); the
    head's slot 1 has bit 224 clear; and the quote word's bits 32-62 equal the head's slot 1 bits 160-190, all 31
    bits. The desk word is read from the desk contract's slot 1 at the head's block, in the head batch, and is never
    taken from an answer; a pair's word never shows a stop, and a word of another epoch is empty. A desk word, or for
    a block-before stamp the head's own quote word, that cannot be read covers nothing. A firm quote for which the
    answer gives no firm word takes the head's, and counts only where the head's quote word is of the desk's epoch and
    firm too. That is the whole of this source's part in firm quotes: an update that carries the desk's firm state is
    accepted, and no fill quoted here reads it. Signed firm quotes (a maker's signature filled through the front's
    `fillQuote`) are not served by this integration.
  - *Malformed entries.* An answer with any entry that cannot be read (a numeric nonce or balance, a `stateOverride`
    that is not an object) covers no listing for its block, since what that entry writes is unknown; a later clean
    answer for the block can still cover.
  - *Foreign writes.* An answer that writes any slot of the desk contract other than a `0x100 + i` or `0x200 + i`
    (slots 0 to 3 among them), written or split by two entries, covers none of that desk's listings.
  - *Several listings on one desk.* Each is covered by its own words alone: an answer that stamps only pair A serves A
    and not B, a stop at the head or a foreign write darkens both, and a split of A's words leaves B served (the
    split slot is left out of B's override). A listing's read carries the desk contract whole from its answer, the other
    pairs' words with it. A split balance or nonce is left out and does not darken a desk contract's listing, which has
    no value and creates nothing.
  - *Not modelled.* The source does not model registry lanes and does not read the front's `deskOf` record (its stop
    sits at bit 192, where the desk contract's slot 1 has it at bit 224; the two are never decoded with each other's
    masks), so neither has a decoder here. A lane copy shows up as a `LaneFill`, which maps to every listing of
    the desk.
- **Only the accounts read.** Of that answer the listing's calls carry the overrides of its quote contract and its
  bound price contracts (its primary and its backup, each the front's CREATE3 child of its source id, the contract the
  front calls) alone. Any other account of the answer is dropped.
- **Sampling.** Every call a refresh's state is made of (the listing's `doorView`, its pool's balances, the front's
  `clock()`, each side's top and every quote) is pinned to the head's block and runs with the target block's number
  and second as block overrides and those accounts as state overrides. The front's own storage reads (the controls,
  binding and session words, the live source word, the implementation slot) stay plain reads at the head: an update
  writes only its quote contract and its price contract. The front's clock on Ethereum is the block's second
  (`clockMode` 0) plus its age slack, so a time override places every sample. A front on another clock is not quoted,
  and the chain's millisecond-time contract is never read.
- **What is quoted.** A listing is quoted only where the answer's block is the head's next, its `slot` is the head's
  slot plus one (its second the head's plus 12 s), the node ran the calls in that block, and the answer covers the
  listing by the slot rule above: its quote word at the target block's stamp, or the block before's where the listing
  and the word both carry (the carry rule), of the head's desk epoch, with the desk not stopped and no foreign slot
  written. Another listing's update on the same desk contract does not count, nor does an older or a later stamp: the
  door of such a listing is not asked. A PRINT or SESSION listing also needs
  the entry of the price contract it reads in that block: its live source (record word 2), the primary, or the backup
  once the primary is darkened. An answer for a block further ahead (a node a block or more behind the api), one whose
  `slot` is later than the head's next (a missed slot, or an answer ahead of the wall clock) and one with no `slot`
  quote nothing for that refresh. None of them backs off or stops the next poll: the next refresh, a second later, reads
  the next answer as usual. The api must send `slot`. A listing whose maker has no update in the next block quotes
  nothing. A listing bound to a guarded entry is not quoted on chain 1: Ethereum's sources are price contracts, which
  are never guarded. Every other rule of Pricing holds.
- **Validity.** A state is good for its target block alone. Its `validUntil` is that block's second less the inclusion
  allowance, the last second a route may be built from it, and route finding takes no allowance off it again. The
  allowance (`inclusionAllowance`) is 1 s by default on a front with a `titan` block, and stays config. It trades
  quoting time against reach: a longer allowance gives up more of each block's 12 s, and a shorter one quotes routes
  that reach the builder after it has built the block, which then land in a block without the update (below).
- **The carry rule.** The front lets a fill read a word in the block after its target where the listing's binding
  carries (`doorView`'s scopes bit 3), the word carries (any of its bits 4-7) and the listing's fence kind is not NONE.
  Where both carry, the refresh also samples every size in the next block (a slot later, on the same slots), keeps the
  lower output of each size, drops a size refused there, and is valid to that block's second less the allowance. Where
  that sample cannot be read, or refuses every size of a direction (a word whose life ends before that block), the
  direction keeps the target block's ladder, valid to that block's second less the allowance. No later block reads the
  word. An update in the answer already stamped with the block before the target's is quoted in the target block
  alone, where both carry. A carried quote stands only until the first refresh after its target block lands: that
  refresh reads the answer for the next block, which in the carry case holds no update of the maker's quote contract,
  so the listing quotes nothing then, though the door at the head would still fill the carried word in that block. The
  carry thus covers the time between the target block's second and that refresh, about a second, and never quotes
  above the door. The source
  reads the listing's carry life from scopes bit 3 alone: the front's life is two bits, of which a one-block life sets
  bit 3, and a longer life would be read as none, which only shortens the quote's validity.
- **Refresh.** The map moves with every block, and nothing on chain logs a coming update: no quote contract logs
  `DeskPush` there and no stored lane posts, so the parser maps neither and reads no price binding. The tracker asks
  for a refresh at the earlier of the state's `validUntil` and a second from now, so a new map version, a higher block,
  is quoted within about a second of its publication. The front's logs map to doors as elsewhere, and a quote
  contract's `LaneFill` and `Swapped` to the doors of its listings. A slot with no block moves the target block to a
  later second than the map's, where the front's clock reads that later second; the route's own minimum output covers
  that.
- **Entry mode.** Both modes fill on chain 1, and the front's is the one to configure. A listing's door is a forwarder
  with no storage: its `quote` is the front's `quoteFrom` for the listing, an answer that does not depend on the
  caller, and its `swapFrom` pays the listing's pool and calls the front's paid entry (`swapExactInPaid`), or on a
  listing with a credit mode pulls the input and calls the front's named entry (`swapSender`). The adapter's front mode
  calls those same two entries itself, by the front's `payPoolOf`. So a fill pays the same output in either mode and a
  refusal is the same refusal, and the door's mode adds only the door's own check of the pair and of a zero quote, and
  its gas: 9,885 to 13,031 a fill on the price-source listings and 11,849 to 17,244 on the credit listing at the
  receipt (29,785 to 37,144 before refunds, of which the door's mode earns 19,900 back), with a
  `NoWord` revert about 32,000 dearer too (about 100,000 against 68,000 on the credit listing). The contracts a fill
  reads and writes are the same in either mode but for the door itself, and so is its `Swapped`, which a builder
  matches to place the maker's update: the listing's quote contract logs it once a fill whichever entry fills, naming
  the front's caller as its sender (the door in the door's mode, the executor in the front's). The door's mode stays a
  config option for an executor that will only approve a contract that cannot change: a door is immutable, where the
  front is a proxy whose implementation its owner can replace after a delay (this source quotes only the
  implementations its config lists). In the front's mode the adapter approves nothing on a listing with no credit
  mode (it pays the pool and calls the paid entry), and approves the front for exactly the fill's input on a listing
  with one. Both modes are tested on a fork of the deployed build (Tests).
- **The `NoWord` exposure.** A route that lands in a block without the maker's update (one built by a builder that does
  not carry it, or a later block than the state's, past any carry) reverts at the front (`NoWord`), and the taker pays
  the gas of the reverted transaction. The route's minimum output does not prevent that. The quote's one-block life
  and the allowance bound how often it happens.
- **Tip cap.** Ethereum's front is not a storage build: a listing's tip cap of 0 means no cap, and such a listing passes
  any `minTipCapWei`.
- **Node load.** One refresh makes one `/state` read of each url and, for a quoted listing that reads no source, 4
  `eth_call` and 7 `eth_getStorageAt` (three of them the desk word and the listing's own quote and firm words, read at
  the head). A carried block adds one `eth_call`, and a print listing its binding's 2 or 3 storage reads. The `/state`
  read is per refresh, so an endpoint that limits each caller's rate bounds how many listings one consumer refreshes
  every second. A credit-mode listing adds no request: its plan reads ride in the same `eth_call`s (below).

### Credit-mode listings

A listing with a credit mode (`doorView`'s `creditMode`, 1 to 3) fills inside its pool's custody where that fills it
whole, and past it, where the pool's switches let it, through the pool's own swap: its credit route, which may borrow
the token it pays out. Its door's `quote` already answers what either fill pays (the cheap fill inside custody, the
pool's own preview on the route, every refusal of the route's caps a `NoFill`), and the adapter fills both, in either
mode, through the front's named entry: its front mode calls it (the front names no pool to pay for such a listing), and
its door mode calls `swapFrom`, which pulls the input into the door and calls it. What differs is the gas, several times
higher on a route, and the output, which need not continue across the step from custody to the route (a pool fee, a pool
minimum). So on chain 1 such a listing is quoted only where its front's `creditListings` names it with four gas figures,
measured on its own pool, since a route's gas is its pool's (its venue, its tokens, its caps), and no figure of this
source's own can stand for it:

- **Each size's class.** Beside each door `quote`, in every block sampled, the refresh asks the front's tip-taking
  `quoteFrom(tokenIn, tokenOut, size, listing, 1 << 253, door)` (`PLAN_ONLY`, no tip, the door the taker). The front's
  plan answers a size it routes tagged with bit 255 (`ROUTED`), and a size it fills inside custody untagged, the door's
  own output. A size of neither (the door refused it, the plan refused it or answered another amount) is no point, and
  so is one whose class in the carried block is not the target block's.
- **One step a side.** A side's ladder is its custody points, then its routed points: a custody point after a routed one
  ends the run, as a falling output does. The state keeps such a listing's ladders in `credit.l`, not in `l`, and marks
  each side's first routed point (`credit.from`): a simulator that predates credit data finds no ladder and quotes
  nothing, where it would draw a chord across the step and charge a route the custody's gas.
- **Nothing across the step.** The simulator quotes nothing strictly between a side's last custody point and its first
  routed point (`ErrCreditStep`). At or below the last custody point it charges the custody figure, at or above the
  first routed point the route's.
- **Depth past custody.** The pool's reserves are its balances, as any listing's. A routed size pays past them (the
  route may borrow the token it pays), so the simulator checks a route against its own ladder alone, sampled on the
  custody as the refresh found it; the route's room (its venue's caps, the pair's debt cap, the outflow bucket) is what
  the door refused past the top. A custody fill shares the pool's limit with its other listings as any fill does, the
  limit its last custody point, never the route's reach. The sampling grid is not rescaled by the paid token's balance.
- **The pool after a fill.** A route borrows or repays, which moves what every listing of the pool fills (a cheap fill
  of a token the pool owes is refused, and a mode-2 bid routes, until a flush). Every listing of a pool shares a route
  key (`<pool>:route`) beside its token limits: a route is quoted only on a pool no fill of the route has touched, and
  once taken no listing of the pool is quoted again in that route. After one fill of such a listing, its other side is
  not quoted either (`ErrSecondFill`).
- **What moves after the sample.** The route reads state the target block's sample cannot pin: another fill's borrow
  that lifts the venue's rate over the pair's rate ceiling, a feed update that moves the pool's band check, the venue's
  own accrual. A fill that meets one reverts at the taker's gas, as a `NoWord` does, and the route's minimum output does
  not prevent it.
- **Gas.** Each figure is the least first fill of its class and input on that pool through the adapter in the front's
  configured mode, the frame less the refund the fill earns, as every figure here. A custody fill of such a listing
  takes the front's named entry, not the paid one, so it is measured too, not taken from the tables above. On a fork of
  Ethereum holding the deployed build, the wstETH/WETH listing of a pool in credit mode 2, whose WETH is borrowed at
  Morpho Blue, cost through the door and through the front: a route fill of wstETH in 841,583 and 824,339, the least
  measured, where the pool's custody pays part of the WETH and the pool borrows the rest (a fill that borrows all of it
  843,526 and 826,282 on a pool that already owes, 49 more on its first borrow; before refunds a route fill spent up to
  982,875 and 945,731, and a limit is set from those), a custody fill of wstETH in 135,485 and 118,241, and a custody
  fill of WETH in 128,913 and 117,064 where the pool held WETH (where it held none, 17,044 to 17,068 more, which the
  state marks and the simulator adds: Execution and gas). A fill of WETH in past the custody was refused at every size
  tried, by the door's quote (`NoFill`) and by the fill (`NotEnoughDepth`) alike, so that side had no routed point and
  has no route figure. A route figure of 0 says so: that side is quoted inside its custody alone, and no routed size of
  it is quoted until its figure is measured. The two custody figures are required. A route's gas is its pool's settings
  as much as its venue's: on that pool the borrowing fill moved by up to 82,591 gas before refunds with the pool's own
  switches (collateral posted ahead of the fill, the pair's debt cap, the venue's rate ceiling, the router's second
  check), so a listing's figures are measured again on its live pool after any change of them. One route figure a side
  stands for every path the route takes (a borrow, a repay, a supply, a withdraw, the pool's first borrow): it is the
  least first fill of the side's routed sizes, so a dearer path is under-charged, which the figure's measurement must
  state. The same holds across lending venues: the front's plan says a size routes and nothing of the venue the pool
  will draw at, so where the pair may borrow at more than one, the figure is the cheaper venue's and a route through the
  dearer is under-charged by the difference between the two. The gas limit must still cover the dearer:
  `routeLimitBaseIn` and `routeLimitQuoteIn` state it, and a routed quote carries it as `SwapInfo.GasLimit`. In credit
  mode 3 a size inside custody routes too and draws nothing, so it is charged the route's figure, more than it spends.
- **The front's implementation.** The plan's `PLAN_ONLY` answer is the front's from 7093a435e on (in f1f8d82c8). On an
  earlier front, bit 253 is part of the tip: a listing with a tip cap refuses every plan read (`TipCapped`), and one
  with none answers a routed size the pool's preview untagged, the door's own output, so it would be classed custody
  and charged the custody's gas. A front's config names no credit listing until its live implementation has it.
- **The reads' gas.** The door's quote of a routed size runs the pool's preview (its venue reads), and its plan read
  the front's walk, both inside the refresh's `eth_call`s, which carry no gas budget of their own: a node's per-call gas
  cap bounds how many sizes one refresh samples. A call over the cap fails the refresh, as any failed read does:
  liveness, not a wrong quote.

## Config

```json
{
  "dexID": "efficient-prop",
  "chainID": 8453,
  "minTipCapWei": 0,
  "inclusionAllowance": 1,
  "fronts": [
    {
      "address": "<the front's proxy>",
      "doorFactory": "<the front's listing-door factory>",
      "fillMode": "front",
      "implementations": ["<a front implementation its proxy may run>"],
      "approvedSources": [],
      "baseTime": "<the chain's millisecond-time predeploy>"
    }
  ]
}
```

On Ethereum, each front adds its `titan` block, its listings' own gas and its credit listings:

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

No front is configured on Base: this build is not deployed there. Its front and listing-door factory are added here
once it is.

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

`layout` and `cellCodehash` are required on a builder chain, where the per-desk coverage rule runs (Ethereum alone
today: BNB Chain's front is a builder build too, but this source has no next-block read for it yet, so no BNB front is
configured), and refused on any other chain: on Base the head is read and no quote contract is, so neither applies and a
front that sets either is refused at load. `layout` names the layout of the front's quote contract; its only value is
`builder`, the desk contract with a pair slot for each listing. (Base's quote contract has another layout that this
source does not read yet.) `cellCodehash` is the hash of the desk contracts' creation code, the front's
`CELL_CODEHASH()`, fixed when the front is created. On a builder chain the lister refuses a front that answers another
(it lists nothing of it), so does the tracker on its next-block path (the refresh fails), and the parser leaves that
front out of its learning, as it does a front whose factory answers no `DOOR_CODEHASH()`; the other fronts carry on. A
hash that failed to read, or did not match, is not read again for 5 s. The config names no door code hash: the source
reads `DOOR_CODEHASH()` from the deployed factory, keeps it, and takes each door from the factory's `doorOf`, which the
lister checks is the CREATE2 door of the listing id under that hash (a listing whose door is another is left out).
`fillMode` is required, `front` or `door` (see Execution and gas); a pool keeps the mode it was listed with.
`implementations` lists the front implementations its proxy may run while its listings are quoted (at least one, no
default; see Pricing): list a new one only after checking that its storage layout, clock and pricing rules are the ones
this source reads. `approvedSources` lists the price sources a listing of the front may read and be quoted, each the
address in its entry with the front (on Base a stored price lane); none by default, so only listings that read no source
are quoted (see Pricing, How to approve a source). `baseTime` is the address of the chain's millisecond-time predeploy
(BaseTime, on Base `0x4200000000000000000000000000000000000030`), which the front's block-count clock reads; it is
required where the front's clock is the block-count mode (a refresh of such a front's listings with none configured
fails), and unused on a front whose clock reads the block's second. `creditListings` (Ethereum alone, refused on any
other chain) names each listing with a credit mode the front quotes, with its four gas figures in the front's fill mode,
each at most 3,000,000: `{"listing": <id>, "custodyBaseIn": <gas>, "custodyQuoteIn": <gas>, "routeBaseIn": <gas>,
"routeQuoteIn": <gas>}`. The two custody figures are above 0; a route figure of 0 leaves that side's routed sizes
unquoted. A route figure is the least route fill of its side, the cheaper lending venue's where the pair may borrow at
more than one; `routeLimitBaseIn` and `routeLimitQuoteIn` (optional, each at least its side's route figure, at most
6,000,000) state the gas limit the dearest route fill of the side needs, from its gas before refunds, which a routed
quote carries as `SwapInfo.GasLimit`. None by default, so no listing with a credit mode is quoted (see Ethereum,
Credit-mode listings). `listingGas` (any chain) names a listing with no credit mode and its own two figures in the
front's fill mode, each above 0 and at most 3,000,000: `{"listing": <id>, "baseIn": <gas>, "quoteIn": <gas>}`; they
stand in place of its class's (see Execution and gas). A listing named twice, or in both lists, is refused at load.
`inclusionAllowance` is the time a routed fill takes to land, seconds: 1 by default on Base, whose blocks are 200 ms
apart, 1 on a front with a `titan` block (see Ethereum), and 3 on any other chain (see Pricing for what a shorter one
trades).

On Ethereum the approved sources are every price contract a listing binds, each pair's primary and its backup: a listing
with any bound source not approved is not quoted. `titan` is required on chain 1 and refused on any other chain: `urls`
at least one, in the order a listing's answer is chosen by (see Ethereum), and `timeout` in nanoseconds, a Go duration
as Kyber's `titan` client reads it: `2000000000` is 2 s, and a bare `2` is 2 ns, which fails every read. Unset, it is 10
s; keep it well under the 12 s block.

## Tests

```sh
go test ./pkg/liquidity-source/efficient-prop
EFFICIENT_PROP_RPC=<rpc> EFFICIENT_PROP_FRONT=<front> EFFICIENT_PROP_FACTORY=<door factory> \
  EFFICIENT_PROP_CELL_CODEHASH=<the front's CELL_CODEHASH> EFFICIENT_PROP_SOURCES=<approved sources> \
  go test ./pkg/liquidity-source/efficient-prop -run TestFork -v
```

The unit tests run the lister and the tracker against a fake chain whose doors walk concave levels, and check the
simulator against the door at random sizes. On the fake chain they cover the approved sources (a listing bound to a
source not approved, or whose binding is not read, is not quoted; a guarded one approved is; a tied listing is not;
the binding is read at every refresh; ids are 32 bits), the implementation list and the credit mode on WORD, PRINT and
SESSION listings, each of the three clocks a block-count front may read (its count, BaseTime off the count by under a
second, and BaseTime a second or more off) and the configured BaseTime, a side bound by its output whose premium
starts at its maker time, its side's depth nonce or its controls nonce and ends inside the window, on both clocks, and
a guarded listing whose side is bound by its output while its lane holds a print: every block of the hold sampled, on
a block-count and on a seconds clock, a hold that ends in the window's last second, a top refused only in the hold's
first block, the longer hold of two lanes, the hold read from its lane or taken as 2,000 ms, and the listing not
quoted where the node refuses or ignores an override or the front's clock reads BaseTime. They check the event
parser's map, a lane's post included, and the gas of each class, a guarded listing's by the source it reads now.
On Ethereum they run the tracker against fake `/state` servers and a fake node of a builder chain, whose door fills only
in a block whose state carries its quote contract's update: every call of a refresh carries the answer's block and
state, the ladder pays the door in the target block, and nothing is quoted on a map not for the head's next block (the
head's own, an older one, one two blocks ahead even beside one for the next block, one at the head's second, one whose
slot is later than the head's next, one with no block, entry or slot), for a quote contract with no entry, for a print
listing whose answer lacks the price contract it reads (its primary, or its backup once on it), on a guarded entry, or
where `doorView`'s desk moved between the reads. They check that a listing's accounts come from one answer, the first
url's of the highest block holding them all, even when it answers last, and never one url's quote contract beside
another's price contract; that a failed url is logged once a minute at most, and a url the wait cuts off is not; that an
empty answer from a fast url does not start the 300 ms wait, so a slower url's answer with the entries is read; that an
answer giving a desk contract two values of a slot drops the listing that owns it, one giving its balance or nonce
darkens none, and a foreign slot written darkens every listing of the desk; that two
listings sharing one quote contract are each served only by their own stamp, a split slot or a stale stamp of one
leaving the other served; that an update stamped with the block before the target is quoted there alone where both
carry; that only the quote contract
and its price contracts ride on the calls; the carry rule in both blocks, on any of the word's carry bits (and a refusal
of some sizes, of every size, or an unread sample in the carried one); a higher map version picked up on the schedule; a
front off the block's second and a node ignoring an override quoting nothing; the price contract as the front's CREATE3
child; the config refusing the front's mode; the parser mapping no quote contract's or lane's post; a tip cap of 0; and
Ethereum's gas classes, each side charged by its input token (a pair whose quote is the pool's asset included).

On Ethereum, `TestForkBuilderFillsTheSampledQuote` runs on a local fork that a stepper drives a block at a time, with
a `/state` server answering for the stepper's next block (`EFFICIENT_PROP_TITAN_URLS`, a comma list), beside the fork's
variables below:

```sh
EFFICIENT_PROP_RPC=<fork> EFFICIENT_PROP_FRONT=<front> EFFICIENT_PROP_FACTORY=<door factory> \
  EFFICIENT_PROP_CELL_CODEHASH=<the front's CELL_CODEHASH> EFFICIENT_PROP_SOURCES=<approved sources> \
  EFFICIENT_PROP_TITAN_URLS=<state urls> EFFICIENT_PROP_STEP=<stepper> EFFICIENT_PROP_TAKER=<taker> \
  EFFICIENT_PROP_PRINTS=<listings that read a print> \
  [EFFICIENT_PROP_FILL_MODE=front] [EFFICIENT_PROP_LISTING_GAS=<listing:baseIn:quoteIn,...>] \
  [EFFICIENT_PROP_CREDIT_LISTINGS=<listing:custodyBaseIn:custodyQuoteIn:routeBaseIn:routeQuoteIn,...>] \
  go test ./pkg/liquidity-source/efficient-prop -run TestForkBuilder -v
```

`EFFICIENT_PROP_FILL_MODE` is the front's fill mode under test, `door` where unset: in `front` the taker's fill goes to
the front's own entry, as the adapter's mode 1 sends it, and the taker approves the front too.
`EFFICIENT_PROP_LISTING_GAS` and `EFFICIENT_PROP_CREDIT_LISTINGS` are the config's `listingGas` and `creditListings`. A
listing with a credit mode that the second does not name must come back unquoted and is left out of the rest; one it
names is filled at its first custody size and, where a side steps from custody to the route, at its first routed size
too, each paying the state's output to the wei. A side of such a listing that fills no size in the block (its custody
empty, its route off or refused) must be unquoted, and is logged.

The stepper is a command run with `set` (the standing options of the blocks to come, which the `/state` server follows:
`--skip` and `--carry`, comma lists of quote contracts whose updates are left out or whose words carry, and `--life
<quote contract>=<s>`), `send` (the next block's updates, unmined) or `mine` (that block, 12 s after the head). The
taker is an account the node signs for, holding both tokens of every listing and approving every door. For each listing
and direction the test refreshes through `/state`, checks the simulator against `door.quote` under the same overrides in
the target block (never over it, never a size it refuses, its answer at every sampled size), lands the updates and fills
the ladder's first size through the door in that block: the fill pays the state's output to the wei. With a listing's
update left out of the next block, a refresh quotes nothing and the same fill reverts `NoWord`, moving none of the
taker's tokens. A map whose entries hold the quote contracts alone, served by a filter in front of the `/state` server,
quotes no listing that reads a print, though the head holds every pair's update and the door still prices the stable
pair under that map at the head's print; on a fork whose listings read no print (WORD), that case does not apply and is
logged. `EFFICIENT_PROP_PRINTS`, required, is how many of the fork's listings read a print (3 on the wstETH pool's PRINT
fork, 0 on its WORD fork): exactly that many are read as reading one and checked on that map, and each fill's class,
which picks its gas, says the same. A map replayed after its block is mined quotes nothing, and so does a map whose
`slot` a filter moves a slot later, a carrying listing's word set to a 16 s life included, while the next poll of the
true map quotes again. A listing whose binding carries, its word set to carry with a 16 s life, is sampled in the next
block too, is valid to that block's second less the allowance, and a fill there, in a block without its update, pays the
state's output; with an 11 s life the next block refuses every size, and the target block's ladder stands, valid to that
block's second less the allowance. A side that a fill emptied is waited for, a block of updates at a time.

`TestForkBuilderMapsAFillsSwappedToItsDoor` needs no `/state` server and no stepper: with the fork's variables,
`EFFICIENT_PROP_CELL_CODEHASH` and `EFFICIENT_PROP_FILL_BLOCK`, a block in which a route filled a listing through its
door, it hands that block's logs to the event parser and checks that the fill's `Swapped` is the quote contract's and
not a door's, that it names the door as `sender`, and that it maps to that door alone.

The fork tests below read a listing at the head, which is Base's path. On a node whose chain is a builder chain they
are skipped, each saying so: at Ethereum's head no quote stands, no lane's post is mapped, and a tip cap of 0 means no
cap.

The fork test (`EFFICIENT_PROP_FILL_MODE`, default `front`; `EFFICIENT_PROP_SOURCES`, a comma list of the price sources
to approve, default none; `EFFICIENT_PROP_IMPLEMENTATIONS`, a comma list, default the implementation the front's proxy
runs at the head; `EFFICIENT_PROP_BASE_TIME`, default Base's BaseTime) lists a live front's doors, checks each pool's
meta against its door and, in the front's mode, that the front names the listing's pool, refreshes each and checks the
simulator against `door.quote` at the refresh block at every sampled size, between them, outside them and at a sweep of
sizes by the input token's decimals. `TestForkServesAPrintListingOnlyOnApprovedSources` refreshes each listing with no
source approved and with those named, reading each listing's binding from the front's storage: a listing bound to no
source is quoted either way, one bound to a source not named, or a tied listing, is never quoted, and one bound only to
named sources is quoted once they are named, and its simulator never pays more than the door, except one bound to a
guarded entry where the front's clock reads BaseTime, which is not quoted (Pricing).
`TestForkQuotesNothingOffTheImplementationList` refreshes every listing with the implementation the proxy runs listed,
and with another: no listing is quoted off the list. `EFFICIENT_PROP_STATE=<file>` also prices the pools an earlier run
saved there against the door now and logs the gap. On a local fork only, `EFFICIENT_PROP_EXPIRE=1` then moves the fork's
clock, from a snapshot it reverts to after each listing, to each listing's `validUntil`, where the door must still fill
and route finding must quote no later fill, and on to the word's end, where the door must refuse and a refresh must
quote nothing.

On a local fork only, `EFFICIENT_PROP_AGING=1` with `-run TestForkLadderHoldsOnAnAgingListing` sets each listing's
controls to widen with age and its two sides' depth to drift at different rates (20 and 50 ppm/s), refreshes it, and
checks the simulator against the door in every block of the state's window, by block overrides on the refresh block
and, where the front's clock reads BaseTime, BaseTime's sub-second set by a state override, the last millisecond of each
second included. With it, `TestForkSidesFollowThePairWord` ends each side's depth in turn a few seconds after the head
and checks that both directions keep a ladder, that the state is valid no later than the side's end, and that the door
refuses the direction selling into that side from there; a listing whose base is the pair's higher address shows a side
paired with the other's depth. `TestForkSamplesNoPremiumThatEndedBeforeTheBlock` sets a premium that ends 100 ms before
the head's moment, in its second, with no widening or drift, and checks that the ladder pays the door's quote at the
head, and `TestForkQuotesNothingOnAZeroTipCap` sets each listing's tip cap to 0 as its pool's owner, then checks that
the door refuses every size the state quoted and that a refresh quotes none of them. A listing bound to a source not
approved is logged and skipped in these. The fork tests other than `EFFICIENT_PROP_EXPIRE` (which mines blocks of its
own) read at the head and place their samples by block overrides, so they run on a fork kept on the counted clock as
well as on one whose count has left BaseTime. There a guarded listing has no ladder; and where BaseTime runs ahead of
the count, a print posted on the count is that much older on the front's clock, while `TestForkSidesFollowThePairWord`
needs a print listing's print to outlive the shortened side by a second, so run that test there with no source approved.
`TestForkQuotesNothingOnAZeroTipCap` sends a transaction and waits for its receipt, so it needs a node that mines while
it runs (automine, or a block every 200 ms on the counted clock); on a fork where nothing mines it fails and leaves its
transaction pending, to land in the next block mined.

With `EFFICIENT_PROP_SOURCES` set, `TestForkMapsALanesPostToItsListings` takes the latest of the last 20 blocks that
holds a post of an approved source, hands that block's logs to the event parser and refreshes every listing: each
print listing that is quoted gets its lane's post, and no listing that reads no print gets one.

On a local fork kept on the counted clock only, `EFFICIENT_PROP_HOLD=1` with `-run
TestForkGuardedListingHoldsAcrossItsHold`, run right after a print post on a guarded source's lane that moves the price,
refreshes each listing bound to a guarded entry of an approved source at the head, then mines the fork's blocks on the
counted clock, from a snapshot it reverts to, up to the state's `validUntil`, and checks at each block that the
simulator never pays more than the door at any sampled size, between them and at a sweep of sizes, and never quotes a
size the door refuses. `EFFICIENT_PROP_HOLD_OUT=<file>` writes the checks. `EFFICIENT_PROP_HOLD_DRIFT=<ppm/s>` and
`EFFICIENT_PROP_HOLD_BUCKET=<whole quote tokens/s>` first write, from that snapshot, each such listing's bid to drift at
that rate and its outflow bucket to refill at that rate, so the bid is bound by its output: run one block after a post
that moves the price up (600 ppm/s and 1,000 USDC/s on a WETH/USDC listing on Base's hold lane), the top the bid fills
in the state's block is refused in the first block after the hold ends and filled again later, so a refresh that does
not sample the hold's blocks quotes a size the door refuses there.
