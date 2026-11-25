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
that desk is live on the front. No listing is registered on Ethereum's front yet.

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
- its tip cap is at least `minTipCapWei`, or 0, which on Ethereum means no cap. A fill sent at a higher priority fee
  than its listing's cap reverts (`TipCapped`), and nothing in a route says what the taker will tip, so the cap is
  also in the pool meta (`tipCapWei`);
- its gas class has a measured figure in the front's fill mode (Execution and gas);
- on Ethereum, its maker's update for the next block is in a `/state` answer (Ethereum).

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
  direction's ladder is the run of passing sizes up to the first refusal or falling output, at most 64 points.
- The ladder holds for its whole window. A listing's levels may widen with its quote's age, so every size is sampled
  again at the window's last block (and around a premium's end, and at each block of a guarded source's hold) by an
  `eth_call` with a block override, and the least output of each size is kept. Each sample reads its block's time and
  number beside its quotes, so a node that ignores the override is caught.
- The simulator never pays more than the door at the refresh block. The front's output is a floored concave function
  of the input: at a sampled size the simulator pays the door's answer, between two sizes the floored chord, and under
  the smallest or past the largest nothing.
- It pays no more than the listing's pool holds of the output token. Listings of one pool share a swap limit keyed by
  pool and token, and a direction takes one fill a route, since a fill moves the levels the ladder was sampled from.
- `StaleCheck`: route finding refuses a ladder older than 10 s and stops `inclusionAllowance` seconds before the
  ladder's `validUntil`. With `StaleCheck` off, no age gate applies.

## Refresh

- **Schedule.** The tracker implements `NextUpdateAt`: one second after the state's block time; on Ethereum a second
  from now, or at `validUntil` where that is sooner. A listing not quoted for a standing reason (its desk not
  approved, its fill mode, its tip cap, no gas figure, not live) is read every 5 s.
- **Logs.** The event parser (a `poolfactory` decoder) maps each log that moves a listing to its door: the front's
  logs naming a listing, a desk or a pair; a desk contract's `DeskPush` and `LaneFill`; an approved price lane's
  `Posted`; and a fill's `Swapped`, which on Ethereum the listing's quote contract logs. A `Swapped` from an address
  the parser does not know maps to nothing.
- **Node load.** One refresh of a quoted listing that reads a price source (a primary and a backup), on Ethereum, makes
  5 `eth_call` (6 where the listing and its word carry into the next block) and 9
  `eth_getStorageAt` (`TestBuilderRefreshNodeLoad`), and the listings of a front refreshed within 250 ms share one
  `/state` read of each url. At the head a refresh makes 3 to 16 `eth_call` by the listing's kind.

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
is the adapter call frame's gas less the refund the fill earns, the least first fill measured of its class. On
Ethereum, for a listing that reads a price source:

| Class | Input | Door mode | Front mode |
|---|---|---|---|
| on the pool's hook pair | the base, the pool's asset | 117,708 | 106,000 |
| on the pool's hook pair | the quote | 116,339 | 106,561 |
| past the hook pair, the base the pool's asset | the asset | 144,744 | 133,036 |
| past the hook pair, the base the pool's asset | the quote | 136,333 | 125,307 |
| past the hook pair, neither token the pool's asset | the base | 150,796 | 140,911 |
| past the hook pair, neither token the pool's asset | the quote | 154,012 | 141,041 |

Each was measured on a fork of Ethereum holding the deployed build, over wstETH, WETH, USDC and USDT pairs; a class's
figure is the least of its listings, so a new listing of an approved desk in one of these classes is quoted with no
config change. Same-size first fills vary by up to about 500 gas from run to run, and a listing whose token transfers
cheaper than these may fill a little under its class figure. Two listings of one class differ by their tokens'
transfer costs, by up to 2,005 gas, and `listingGas` gives a listing its own figures. A class never measured on the
deployed build has no figure and is quoted only where `listingGas` names the listing with its own two figures: a
listing that reads no price source, a pair past the hook pair whose quote is the pool's asset, and an asset-module
pair. The figures of a front read at the head are in `constant.go`.

- **A pool holding none of the input.** Such a fill writes the pool's balance from zero and costs 17,044 more. The
  state marks it (`noneHeld`), and the simulator charges it until a fill of the route, through any listing of the
  pool, pays the pool some.
- **Refunds.** A refund is paid up to one fifth of its transaction's gas, and every figure nets all of it. A fill
  alone behind a bare test hop was paid all of it on every class but the credit listing's custody fill in the door's
  mode, whose figure is short by up to 9,435 there.
- **Gas limits.** Every figure is what a fill adds to a receipt, never a limit: the least gas limit that let a fill
  through the adapter was 21% to 29% above its receipt's gas, since each call forwards at most 63/64 of the gas left.
  Set a route's limit at 1.3 times its expected gas or more.

## Ethereum

A maker's update on Ethereum (its quote contract's quote for one block and, where the listing reads one, its price
print) lands only in a block a builder built with it ahead of a fill, and stands for that block alone. At the head a
door refuses every size. So on chain 1 the source reads the state the next block will hold:

- **The read path.** Each front names a `titan` block: the urls answering `titan_getPammStateOverrides`, dialled and
  awaited as Kyber's `titan` package does. An answer names its target block, its beacon slot and, per quote contract,
  the slots that block's update writes.
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
  so route finding quotes a fill only while now plus the allowance is not past the end. `validUntil` stays beside them in
  whole seconds, the end less the allowance, rounded down. Where both the listing and the quote carry, the next block is sampled
  too and the lower output of each size kept.
- **The `NoWord` exposure.** A route that lands in a block without the maker's update reverts at the front (`NoWord`),
  and the taker pays the reverted transaction's gas: about 83,500 to 87,100 in the adapter's frame through the door on
  the price-source listings, and about 32,000 less through the front. The route's minimum output does not prevent it;
  the allowance bounds how often it happens.

### Credit-mode listings

A listing with a credit mode fills inside its pool's custody where that fills it whole and, past it, through the
pool's credit route, which may borrow the token it pays out. The door's `quote` answers what either fill pays and the
adapter fills both; the gas differs several times over. So such a listing is quoted only on chain 1 and only where
`creditListings` names it with figures measured on its own pool:

- Beside each `quote`, the refresh asks the front's plan for the size (`quoteFrom` with `PLAN_ONLY`), which says
  whether the size is routed. A side's ladder is its custody points, then its routed points, and the simulator quotes
  nothing strictly between the two (`ErrCreditStep`).
- A route borrows or repays, which moves what every listing of the pool fills: a route is quoted only on a pool no
  fill of the route has touched, and once taken no listing of that pool is quoted again in that route.
- The route reads state the sample cannot pin (the lending venue's rate, a feed update). A fill that meets a change
  reverts at the taker's gas, as a `NoWord` does.
- **Measured** on a fork of Ethereum holding the deployed build, on the wstETH/WETH listing of a pool whose WETH is
  borrowed at Morpho Blue, through the door and through the front: a custody fill of wstETH in 135,485 and 118,241, of
  WETH in 128,913 and 117,064, and a route fill of wstETH in 841,583 and 824,339 (982,875 and 945,731 before refunds).
  A fill of WETH in just past the custody is a route too. With no loan open it filled from 1.008 to 1.028 times the
  custody (to 1.036 with the pool holding some WETH) and reverted `NotEnoughDepth()` at the next size tried; with the
  pool owing 0.37 WETH it filled to 1.5 times the custody and reverted at 3. It cost 948,309 and 936,460 with the pool
  holding some WETH (1,072,509 and 1,040,760 before refunds), 966,765 and 954,916 with it holding none, and 1,061,009
  and 1,049,160 where it repays the loan (1,190,009 and 1,158,260 before refunds). One route figure a side stands for
  every state and path the route may take: it is the least, so a dearer fill is under-charged (the repaying one by
  112,700), and `routeLimitBaseIn` and `routeLimitQuoteIn` state the gas limit the dearest needs. Measure the figures
  again on the live pool after any change of its settings.

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
      "titan": {"urls": ["https://<api host>/v1/1/state", "<a builder's data host>"], "timeout": 2000000000},
      "creditListings": [
        {"listing": 1, "custodyBaseIn": 118241, "custodyQuoteIn": 117064,
         "routeBaseIn": 824339, "routeQuoteIn": 936460}
      ]
    }
  ]
}
```

The sample leaves `approvedDesksURL` out: it is added when the api serves the path, and until then `approvedDesks`
rules.

These are Ethereum's deployed front (proxy and implementation), its door factory, its `CELL_CODEHASH()`, its four
price contracts and its first maker's desk contract (desk 1). The `creditListings` entry is the fork measurement above
for a wstETH/WETH listing with id 1; ids and figures are checked against the live pool before the config is used.

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
| `layout`, `cellCodehash`, `titan` | Required on Ethereum, refused elsewhere. `layout` is `builder`; `titan.timeout` is in nanoseconds (unset: 1 s, at most 10 s) |
| `creditListings` | Ethereum alone: `listing`, `custodyBaseIn`, `custodyQuoteIn`, `routeBaseIn`, `routeQuoteIn`, optional `routeLimitBaseIn`, `routeLimitQuoteIn`. A route figure of 0 leaves that side's routed sizes unquoted |
| `listingGas` | A listing with no credit mode and its own `baseIn` and `quoteIn` figures, in place of its class's |
| `minTipCapWei` | The least nonzero tip cap listed. Unset, 2 gwei on Ethereum and none elsewhere; `1` lists every listing |
| `inclusionAllowance` | Seconds a routed fill takes to land: 1 on Ethereum and on chains with sub-second blocks, 3 elsewhere |
| `baseTime` | Head chains with a block-count clock: the chain's millisecond-time predeploy |

The 2 gwei default: over the day to block 26,486,699 the 90th-percentile priority fee of a block was at most 2.0 gwei
in 90% of blocks. A config for BNB Chain is refused at load: its front is a builder build and this source has no
next-block read for it. No front is configured on Base: this build is not deployed there.

## Tests

```sh
go test ./pkg/liquidity-source/efficient-prop
```

The unit tests run offline: the lister and the tracker against a fake chain whose doors walk concave levels, the
simulator against the door at random sizes, and the Ethereum path against fake `/state` servers and a fake node. The
fork tests (`TestFork*`) are skipped unless their `EFFICIENT_PROP_*` variables are set.
`TestForkBuilderFillsTheSampledQuote` runs on a local fork of Ethereum that a stepper drives a block at a time with a
`/state` server answering for its next block: for each listing and direction it refreshes, checks the simulator
against `door.quote` under the same overrides, lands the update and fills the ladder's first size, which pays the
state's output to the wei, a credit listing's first routed size included.
