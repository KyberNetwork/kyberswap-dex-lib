# Premium MemeCurve (`prm-fun`)

This source discovers and quotes Premium's pre-graduation meme curves on Robinhood
(chain 4663). Pair support is generic: each market supplies its actual `deskToken()`,
`isNativeQuote()` and frozen graduation target. No stock-token address allowlist is
used. ETH, PRM, legacy pHODO and V2 pHYPD are verification examples; future registered
stock pairs use the same path.

This restores the all-pair extension submitted in
[PR #1688](https://github.com/KyberNetwork/kyberswap-dex-lib/pull/1688), which was absent
from the merged tree. The existing `uniswap-v4-prm` source handles graduated meme
pools. The stock token's own ETH market is a separate routing leg, not modeled here.

[Protocol](https://www.prm.market/docs/markets#meme-curve) ·
[Contract interfaces](https://www.prm.market/docs/interfaces) ·
[Original extension](https://github.com/prm-market/kyberswap-dex-lib/tree/5d7b13d2023881b5a4e379fa901541ad48f94ec2/pkg/liquidity-source/prm-fun)

## Discovery and rollout

Configure `dexId: prm-fun`, `chainId: 4663`, `newPoolLimit: 100` with:

| Component | Address |
| --- | --- |
| MemeFactory | `0x34fb85af7588db97fd6db6508aa387d0f088564c` |
| PremiumRouter | `0x08a59435c8359a45f4f5dc8d91df893cc33daf29` |
| WETH identity for native ETH pairs | `0x0bd7d308f8e1639fab988df18a8011f41eacad73` |

ETH subject ID is zero. Stock subjects and PRM have nonzero IDs and must not be
filtered out. Discovery reads the factory and per-curve getters at one L2 RPC block;
token index 0 is the pair and index 1 the meme. Required-read failures retain the cursor.

Version 2 cursor metadata **replays an older ETH-only offset from zero** to recover
previously omitted stock/PRM markets. Deduplicate/upsert by curve address. Tracker
refreshes actual pair/native metadata, including older ETH rows. A persisted ETH row
without `nQ` retains its native behavior until refreshed; new ERC-20 rows explicitly
serialize `nQ: false`. The simulator's native flag also survives msgpack persistence.

Do not reuse Multicall's EVM `block.number` as a Robinhood L2 snapshot: it can be the
L1 parent number. `Pool.BlockNumber` and `PoolMeta.blockNumber` use the pinned RPC head.

## Quotes and state

All protocol amounts have 18 decimals and use integer arithmetic. Initial virtual
reserves are `4 * 800M / 3` meme and `G / 3` pair tokens. `G` is the individual
market's frozen target, not a universal 4.2 ETH value. Both directions charge a
ceiling-rounded 1% fee in the pair token.

Quotes are immutable. `UpdateBalance` applies the captured post-trade state from
`SwapInfo`, using the accepted input when a final buy refunds excess. Clones own
separate mutable balances. Final buys graduate atomically; subsequent curve quotes
are disabled. The simulator rejects invalid, zero-output and overflowing inputs.

## Execution handoff

Use the existing PremiumRouter bot methods; output is delivered to the caller:

- `buyExactIn(address meme,uint256 input,uint256 minimumOut)` (`0x4af58d22`): native
  pairs require `msg.value == input`. ERC-20 pairs approve the actual pair token to
  PremiumRouter and send no native value. The caller must hold the offered amount;
  final-fill excess returns in that same pair currency.
- `sellExactIn(address meme,uint256 input,uint256 minimumOut)` (`0x30a2aa20`): approve
  the meme token; receive native ETH for ETH pairs, or the actual pair ERC-20 otherwise.

Both return one output amount. `SwapInfo.nQ` and `PoolMeta.isNativeQuote` identify
native settlement; `ApprovalAddress` is PremiumRouter. Preserve approvals, partial
fills, refunds and native/ERC-20 balance accounting in the Kyber encoder/executor.

Kyber must deploy discovery/backfill and validate its encoder/executor. The merge
review on 28 September reported missing `prm-fun` execution wiring; its current private
state is not visible here. **These tests use PremiumRouter on a local fork, not
Kyber's private encoder or a live funded Kyber route.** Enable production only after
both direct pair↔meme and ETH↔pair↔meme routes pass in both directions.

## Fresh verification — 29 September 2026

- Eleven read-only onchain quote comparisons match exactly for ETH, PRM, pHODO and
  pHYPD at L2 blocks 75791192–75791238. PRM had no sold inventory in that sampled curve;
  its sell is covered by the fork settlement below.
- Twelve settlements pass on a disposable mainnet fork at block 75791163: buy,
  sell and final buy for ETH, PRM, pHODO and a newly created pHYPD Reserve-mode meme.
  Actual token/native deltas, partial-fill refunds and resulting curve state match.
  PRM and pHYPD test memes exist only on that fork; no production key is used.
- Deterministic tests cover discovery/backfill, arbitrary new stock addresses,
  same-block RPC reads, failed reads, integer fees, final-fill conservation, native
  compatibility, clone isolation and msgpack persistence.

Measured PremiumRouter gas (excludes Kyber's outer executor):

| Pair | Buy | Sell | Final buy |
| --- | ---: | ---: | ---: |
| ETH | 223172 | 199546 | 944250 |
| PRM | 282462 | 199121 | 911116 |
| pHODO | 216292 | 201083 | 913569 |
| pHYPD-Reserve | 283064 | 199553 | 1239537 |

Allowances are 330,000 for buys and 250,000 for sells, plus 3,000,000 for graduation. The latter
conservatively covers v4 seeding/locking and RangeMemeCurve's Reserve credit update
plus its bounded 1,500,000-gas auto-settlement attempt, without a stock-address special
case. The sample does not exhaust every future Reserve state.

Fixtures in `testdata/` are excluded from downstream module downloads by their nested
`go.mod`; tests read them from a checkout using `os.ReadFile`.

```sh
CI=true GOTOOLCHAIN=go1.25.10 go test -race ./pkg/liquidity-source/prm-fun
CI=true GOTOOLCHAIN=go1.25.10 go vet ./pkg/liquidity-source/prm-fun
CI=true GOTOOLCHAIN=go1.25.10 go test ./pkg/liquidity-source/uniswap/v4/hooks/premium ./pkg/pooltypes ./pkg/msgpack
```

Opt-in read-only parity (fixture path is relative to the package test directory):

```sh
PREMIUM_RPC_URL=https://rpc.mainnet.chain.robinhood.com \
  PREMIUM_FIXTURE_OUTPUT=testdata/mainnet-quotes.json \
  go test ./pkg/liquidity-source/prm-fun -run '^TestLiveMainnetQuoteParity$' -count=1 -v
```

To repeat settlement, run a fresh localhost Anvil fork on chain 4663, then execute
`testdata/fork-settlement.mjs` with Node 24 and viem 2.56.1. Set
`PREMIUM_FORK_RPC_URL` (default `http://127.0.0.1:18547`) and optionally `VIEM_MODULE`
to the installed viem module. The script checks localhost/Anvil/chain identity,
uses disposable impersonated/funded accounts, and restores its snapshot in `finally`.
It records the actual fork block and local-only receipts. The public RPC prunes old
state; an archive provider is needed to reproduce a historical block exactly.
