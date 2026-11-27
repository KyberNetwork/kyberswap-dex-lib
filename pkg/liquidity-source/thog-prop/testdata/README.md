# Fixtures

Every expected value here was returned by the ThogAMM contract, not computed by
this adapter. A quote fixture holds a `getPoolData` page, the pool's share
holdings (`sharesOf` and `storedMultiplier` per share-based token) and cases: a
request with the contract's return bytes, or a marker that it reverted.

The complete sets and the tools that capture them are published with the
[integration guide](https://github.com/thogard785/thog-amm-integration-guide/tree/main/fixtures).
The files here are excerpts chosen by `select_fixtures.py`: both directions of a
stablecoin pair, a cross-category pair, ordinary/share-based pairs and a pair of
share-based tokens. Each file records the SHA-256 of its source and the original
case indices. Set `THOGAMM_FIXTURES` to a directory holding the complete sets to
run the tests against all 65,592 cases.

| File | Cases | Recorded from |
| --- | ---: | --- |
| `deployed-18.json.gz` | 408 | Mainnet block 125783579. Quotes, marginal prices and limits for ANTx against each of the other 17 tokens in both directions. |
| `deployed-stored-multipliers.json.gz` | 312 | Mainnet block 122916130: 17 tokens, 9 share-based. Read-only quotes, marginal prices and limits. |
| `execution-model3.json.gz` | 1,692 | Settlement on a local Monad fork: exact-input swaps and ERC-7815 buys, at each gas-price threshold, with and without a warm AuctionHandler, for prices updated in the same and in an earlier block, with unit and non-unit share multipliers. |
| `one-block-model3.json.gz` | 320 | Mainnet state of block 117211724, quoted one block later, at the maximum price age and at the first block past it. |
| `deployed-model3.json.gz` | 96 | Mainnet block 117211724. Read-only quotes, marginal prices and limits. |
| `deployed-v8.json.gz` | 96 | A mainnet state with 13 tokens. Read-only quotes, marginal prices and limits. |
| `boundaries-v8.json.gz` | 1,232 | Fourteen fork states at pricing, share-rounding, inventory-cap and price-age boundaries. |
| `lens-stored-multipliers.json.gz` | — | One response of the committed lens at mainnet block 123020553. |
| `lens-age15-monad.json.gz` | 32 | The lens state of block 117733377 quoted at each of the next 16 blocks with a maximum price age of 15. |

`deployed-v8.json.gz` and `boundaries-v8.json.gz` report pricing model 2 in the
state's version field. Its read-only curve is the one this adapter implements, so
the test helper sets that field to 3 before loading them; no expected value changes.

## Recapturing

All capture scripts make read-only calls and send no transaction.

```sh
# Excerpts from the complete sets.
python3 testdata/select_fixtures.py /path/to/thog-amm-integration-guide/fixtures

# The lens response. The script also overrides each share-based token's code so
# that only sharesOf answers, and requires the identical response: the lens asks
# a token for nothing else.
python3 testdata/capture_stored_lens.py --rpc "$MONAD_RPC" \
  --output testdata/lens-stored-multipliers.json.gz

# Lens bytecode and ABI from abi/ThogSnapshotLens.sol.
python3 testdata/build_lens.py --solc /path/to/solc-0.8.25
```

`capture_one_block.py` and `capture_lens_age.py` evaluate a recorded state at
later blocks on a local Anvil fork (`anvil --network monad --fork-url … 
--fork-block-number …`) using `eth_call` block overrides; they refuse non-local
endpoints. `refresh_probe.py` times complete state reads against the endpoints it
is given and can save one lens response (`lens-monad.json.gz` is the input of
`capture_lens_age.py`).

The nested `go.mod` keeps this directory out of downstream module downloads;
tests read fixtures with `os.ReadFile`.
