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
To reproduce the full source (archive RPC access required):

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
