# Contract fixture provenance

These compressed excerpts preserve contract-observed results; selection does not
calculate or edit expected outputs. Each archive records the full source SHA-256
and the original case indices for each snapshot. `select_fixtures.py` chooses
both directions of representative stablecoin, cross-category, ordinary/xStock,
and xStock/xStock pairs, plus maximum observed execution gas cases. The one-block
corpus retains all live pair directions and eight age-boundary cases.

The historical and execution originals and capture tools are published in the
[ThogAMM integration guide](https://github.com/thogard785/thog-amm-integration-guide/tree/main/fixtures).
`execution-model3.json.gz` records local compiled-source hashes. Its six worlds
cover fresh/aged prices with unit and two non-unit issuer multipliers, and each
world tests the three strict gas-price thresholds and warm AuctionHandler.

`deployed-model3.json.gz` was captured at mainnet block **117211724** with the
public guide's `tools/capture_quotes.py`. It records the canonical block hash,
actual implementation address/code SHA-256, raw page, and issuer shares/multipliers.
To reproduce this historical source, use the guide at revision `d4e93c6`
(the current capture tool targets the upgraded stored-multiplier ABI). Archive
RPC access is required:

```sh
python3 /path/to/thog-amm-integration-guide/tools/capture_quotes.py \
  --rpc "$MONAD_ARCHIVE_RPC" --blocks 117211724 \
  --output /path/to/full/fixtures/deployed-model3.json.gz
```

`one-block-model3.json.gz` uses the same actual deployed bytecode on a local Anvil
1.8.6 Monad fork at block 117211724. `eth_call`'s block-number override evaluates
the same page at source block + 1, maximum accepted price age, and the first stale
block. Inventory, issuer state, valuations, base fee and bytecode remain fixed.
No storage override or local implementation upgrade is used for this corpus.
Reproduce with `capture_one_block.py` after starting that fork:

```sh
anvil --network monad --fork-url "$MONAD_ARCHIVE_RPC" \
  --fork-block-number 117211724 --code-size-limit 131072 --port 18546
python3 testdata/capture_one_block.py \
  /path/to/full/fixtures/deployed-model3.json.gz http://127.0.0.1:18546 \
  /path/to/full/fixtures/one-block-model3.json.gz
python3 testdata/select_fixtures.py /path/to/full/fixtures
```

The historical v8 sources still identify model 2. The Go test helper explicitly
promotes only the model identity to exercise the unchanged read-only curve under
model-3 validation. Production reads strictly require schema/wire/model 8/8/3.

The nested `go.mod` keeps these files out of downstream module archives; tests
read them from a checkout rather than embedding them.

`lens-monad.json.gz` records one creation `eth_call` of the committed lens on
October 29 at block **117733377**, with nonzero simulation gas price and no state
overrides. The payload includes the page and all five listed xStock issuers.
The captured policy has maximum age 5. The fixture records the lens bytecode hash.
Rebuild the lens using `build_lens.py`; capture a new observation using:

```sh
python3 pkg/liquidity-source/thog-prop/testdata/refresh_probe.py \
  --rpc https://rpc.monad.xyz --count 1 --capture /tmp/lens-monad.json.gz \
  --output /tmp/thog-refresh-probe.json
```

`lens-age15-monad.json.gz` is an additional 32-case age regression. It uses
deployed bytecode and the lens state from that block on a local Anvil Monad fork.
Only the control word's maximum-age field is changed from 5 to 15; all other
control bits, inventory, issuer data, valuations, and bytecode are preserved.
Evaluation blocks range from snapshot+1 through snapshot+16. USDC→WMON and
USDC→xStock produce 30 exact successes and two stale rejections. These are
controlled execution-age tests, not a historical price-movement replay.

```sh
anvil --network monad --fork-url https://rpc.monad.xyz \
  --fork-block-number 117733377 --no-mining --port 18549
python3 pkg/liquidity-source/thog-prop/testdata/capture_lens_age.py \
  pkg/liquidity-source/thog-prop/testdata/lens-monad.json.gz \
  http://127.0.0.1:18549 /tmp/lens-age15-monad.json.gz
```

The capture tool refuses non-local endpoints. It uses read-only `eth_call` state
and block overrides, and broadcasts no transactions.


`deployed-stored-multipliers.json.gz` selects **312 of 3,264** contract observations
from block **122916130**, with the current implementation
`0xb108a1daf3c9ba52778b97a15ebdfa3f6f59b506` (17 tokens, 14 categories, nine xStocks).
The full public file has SHA-256
`62f526e3cef3402d43f2ff8c69f6c4b567a510cd8248bb54e9fd23ae88439a4c`.
It captures issuer `sharesOf(proxy)` and **pool `storedMultiplier(token)`** in the
same state. The excerpt covers USDC against all nine stocks in both directions,
new-stock crosses, and ordinary-token controls. Recreate only this excerpt with:

```sh
python3 pkg/liquidity-source/thog-prop/testdata/select_fixtures.py \
  /path/to/thog-amm-integration-guide/fixtures deployed-stored-multipliers
```

`lens-stored-multipliers.json.gz` captures the rebuilt lens at block **122921791**.
The capture also repeats the same call at the same canonical block hash with
all nine stock contracts overridden to return their observed shares and revert
on every other selector. The updated lens returns identical bytes; the original
lens at `a6db4e4a3e4edb36ca4e74a4408faf5cbbe23067` fails with `ReadFailed` at the
issuer `getCurrentMultiplier()` selector. The archive records the original and
isolated responses, both lens hashes, and the exact overrides. Reproduce against
a current deployment (or an archive endpoint by adapting the selected header):

```sh
python3 pkg/liquidity-source/thog-prop/testdata/capture_stored_lens.py \
  --rpc "$MONAD_RPC" \
  --legacy-ref a6db4e4a3e4edb36ca4e74a4408faf5cbbe23067 \
  --output /tmp/lens-stored-multipliers.json.gz
```

These overrides exist only inside read-only `eth_call`; no transaction is sent.
