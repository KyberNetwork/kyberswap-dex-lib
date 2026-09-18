# Premium MemeCurve (`prm-fun`)

Extension of [KyberSwap PR #1656](https://github.com/KyberNetwork/kyberswap-dex-lib/pull/1656).
Premium is a meme launchpad on Robinhood mainnet (chain 4663). This source discovers,
tracks and simulates the pre-graduation constant-product bonding curve for **ETH,
PRM and existing stock pairs, including pHODO**. It builds on the curve implementation
by @minhnhathoang in that PR. The existing `uniswap-v4-prm` hook integration handles
the graduated pools separately.

[Protocol documentation](https://www.prm.market/docs/markets#meme-curve) and
[contract interfaces](https://www.prm.market/docs/interfaces).

## Pair coverage and contracts

| Pair | Quote token in pool discovery | Settlement through PremiumRouter |
| --- | --- | --- |
| ETH | WETH `0x0bd7d308f8e1639fab988df18a8011f41eacad73` | Native ETH input/output |
| PRM | `0xf24f8f6b08fe87cf062e833a732ad7f636064bc8` | PRM ERC-20 input/output |
| pHODO | `0x64e8bee350c0ba7c7601f7eca7f61ab965148b51` | pHODO ERC-20 input/output |

Other registered stock pairs use their own actual `deskToken`; this is not an
address allowlist. ETH has subject ID zero; PRM uses `keccak256("premium.pair.PRM")`.
Both PRM and stock subject IDs are nonzero and must remain discoverable.

- [MemeFactory](https://robinhoodchain.blockscout.com/address/0x34fb85af7588db97fd6db6508aa387d0f088564c?tab=contract): `0x34fb85af7588db97fd6db6508aa387d0f088564c`
- [PremiumRouter](https://robinhoodchain.blockscout.com/address/0x08a59435c8359a45f4f5dc8d91df893cc33daf29?tab=contract): `0x08a59435c8359a45f4f5dc8d91df893cc33daf29`
- [Existing pHODO-paired curve](https://robinhoodchain.blockscout.com/address/0xb178d3a7c1d765f9a8aca632f6b7ae3e7a98af73?tab=contract): `0xb178d3a7c1d765f9a8aca632f6b7ae3e7a98af73`
- [Existing ETH-paired curve](https://robinhoodchain.blockscout.com/address/0xaea1eaf948e97581fbe9d8dea2951c327669af57?tab=contract): `0xaea1eaf948e97581fbe9d8dea2951c327669af57`

Configuration uses `dexId: prm-fun`, `chainId: 4663`, the factory/router above, and
`newPoolLimit: 100`. Discovery reads `memeCount`, `memeTokens`, `getMeme`, then each
curve's actual pair and native flag. Version-2 cursor metadata deliberately replays
an older ETH-only cursor, backfilling previously skipped PRM/stock markets. Deduplicate
by curve address. An incomplete required-read batch retains its original cursor.

## Pricing and state

All amounts use 18-decimal integer units. There are 800 million sale tokens and
200 million graduation-liquidity tokens. Initial virtual reserves are
`4 * saleSupply / 3` meme tokens and `G / 3` pair tokens, with integer truncation.
Read the frozen target `G` from each factory market; do not reuse the ETH target
for PRM or pHODO. New ETH markets use 4.2 ETH, excluding fees.

Let `vM`, `vQ`, `sold`, and `raised` be the tracked reserves and inventory:

- Buy: `fee = ceil(input * 100 / 10000)`, `net = input - fee`, then
  `out = min(vM - ceil(vM * vQ / (vQ + net)), saleSupply - sold)`.
- Final buy: if net reaches `G - raised`, accept
  `used = ceil((G - raised) * 10000 / 9900)`, fee `used - (G - raised)`, and deliver
  all remaining sale inventory. Return `input - used` as remaining input.
- Sell: `gross = vQ - ceil(vM * vQ / (vM + input))`,
  `fee = ceil(gross * 100 / 10000)`, output `gross - fee`.
  Recorded principal and sold inventory bound sells.

The trader fee is 1% in the pair currency. Creator/holder reward destinations do
not change these equations. Quotes do not mutate state; `UpdateBalance` consumes
the captured post-trade state, including actual accepted final-buy input.
Non-Trading curves cannot quote; final buys atomically graduate and retire the
curve from further quoting. Graduated v4 swap math remains in the existing hook source.

All tracker/discovery calls use a pinned **L2 RPC block number**. Robinhood's
Multicall EVM `block.number` can be the L1 parent number and must not replace the
L2 snapshot identity returned in `PoolMeta.blockNumber`.

## Execution handoff

Use the router's existing bot entry points; output is delivered to the caller:

- `buyExactIn(address meme,uint256 input,uint256 minimumOut)` — `0x4af58d22`.
  ETH pairs send `msg.value == input`; ERC-20 pairs approve the router and send no
  value. The router pulls the offered ERC-20 input and refunds unused pair tokens
  on a final fill, so the caller must initially hold the whole offered amount.
- `sellExactIn(address meme,uint256 input,uint256 minimumOut)` — `0x30a2aa20`.
  Approve the meme token to the router; receive native ETH for ETH pairs and the
  pair ERC-20 for PRM/stock pairs.

Both return one output amount, not an `(output, used)` tuple. The executor must
preserve native/ERC-20 refunds, approvals and balance accounting. The `nQ` flag in
SwapInfo and `isNativeQuote` metadata identify the native path. Token index 0 is
always the pair and index 1 the meme. `ApprovalAddress` is the router.

Kyber must enable discovery/configuration and ensure its encoder/executor handles
both paths. Library tests and direct router fork settlement do not prove deployment
of Kyber's private encoder or execution through a live Kyber route.

Gas allowances cover router execution: 330,000 buy, 250,000 sell, with an additional
1,500,000 for atomic graduation. At fork block **65190613**, measured buy/sell/final-buy gas was pHODO
216106/200886/913379, ETH 223033/199376/944111, and PRM 282138/198780/910796.
Fork receipt measurements are recorded in
[testdata/fork-settlement.json](testdata/fork-settlement.json); these allowances
include headroom and are not a measurement of the outer Kyber executor.

## Verification

```sh
CI=true GOTOOLCHAIN=go1.25.10 go test ./pkg/liquidity-source/prm-fun
CI=true GOTOOLCHAIN=go1.25.10 go test -race ./pkg/liquidity-source/prm-fun
CI=true GOTOOLCHAIN=go1.25.10 go test ./pkg/liquidity-source/uniswap/v4/hooks/premium ./pkg/pooltypes ./pkg/msgpack ./pkg/source/limitorder
```

Deterministic tests cover all three pair types, discovery/backfill, coherent block
snapshots, required-read failures, exact fee/rounding, buy/sell conservation, partial
fills, clone/state isolation, invalid/overflow inputs and lifecycle boundaries.

`testdata/mainnet-quotes.json` contains six read-only mainnet comparisons for pHODO
and ETH at L2 blocks 65179322 and 65179337. No active PRM curve was available in
that sample. `testdata/fork-quotes.json` adds buys, sells and final buys for each of
pHODO, ETH and PRM, with post-settlement state replayed by the Go simulator.
The PRM fixture market was created only on the disposable fork using the deployed
factory and actual PRM price oracle. All nine settlement checks compare actual
wallet token/native deltas and final-fill refunds with the contract quote.

Read-only live quotes are opt-in:

```sh
PREMIUM_RPC_URL=https://rpc.mainnet.chain.robinhood.com \
  go test ./pkg/liquidity-source/prm-fun -run '^TestLiveMainnetQuoteParity$' -v
```

To repeat settlement checks, start a fresh local Anvil fork (chain ID 4663) with
a dedicated locally funded test account. Pin `--fork-block-number` to the block in the evidence
when using an archive provider. The public RPC prunes old state; omit that option
to verify the current deployment instead. The script records the chosen fork block
and restores its initial local snapshot. It rejects non-localhost URLs and non-Anvil
clients, uses a dedicated locally impersonated account, and needs no production signing key.

```sh
anvil --fork-url https://rpc.mainnet.chain.robinhood.com --chain-id 4663 \
  --host 127.0.0.1 --port 18547 --silent
# In another terminal, install the verification dependency outside the repository:
npm install --prefix /tmp/prm-fork-check viem@2.56.1
VIEM_MODULE=/tmp/prm-fork-check/node_modules/viem/_esm/index.js \
  node pkg/liquidity-source/prm-fun/testdata/fork-settlement.mjs
```

The settlement script impersonates an existing pHODO holder only on the local
fork and acquires PRM through the deployed router. These are local test transactions,
not explorer/mainnet transaction claims. Fixtures can be replayed without Node,
Anvil, network access or credentials by the ordinary Go tests.
