# Everlong FLAMM testdata

Every fixture here is replayed by a Go test in this package, to the wei and with the revert class; no parity assertion
carries a tolerance. The few rows a test sets aside are unreachable domains that the test names and counts (int256
casts wrapping at 2^255 in harness-only gate books, private lev helpers driven outside their callers' domain).

## Environment variables

The offline suite needs none. Network tests skip unless their variables are set.

| variable | used by | meaning |
| --- | --- | --- |
| `BASE_RPC_URL` | `live_test.go`, `TestTrackerReplay` when recording | a Base (chain 8453) JSON-RPC endpoint with archive state and an `eth_call` that accepts state overrides and `blockOverrides`, e.g. `https://mainnet.base.org` (the historical blocks, the refresh's forward rounds and the settlement confirmations all need them) |
| `EVERLONG_FLAMM_FORK_RPC` | `fork_*_test.go` | the Base endpoint the anvil fork reads |
| `EVERLONG_ADAPTER_OUT` | `fork_*_test.go` | the `EverlongFlammAdapter.json` forge artifact of the adapter PR (`out/EverlongFlammAdapter.sol/EverlongFlammAdapter.json`) |
| `EVERLONG_FLAMM_C104_OUT` | `fork_multipool_test.go` | the c104-deploy forge output directory (`out/`) that the extra pools' hooks are deployed from; `TestForkMultiPool` skips without it |
| `EVERLONG_FLAMM_FIXTURES` | `fixture_sample_test.go` | `full` replays every fixture row under `-race`; `sampled` samples without it (default: sampled under `-race`, full otherwise) |

Optional, for fork runs and for regenerating fixtures:

| variable | meaning |
| --- | --- |
| `EVERLONG_FLAMM_FORK_BLOCK` | fork block for every fork test (each test has its own default, listed in section 5) |
| `EVERLONG_FLAMM_FORK_SCENARIOS` | comma-separated scenario subset for `TestForkScenarios` and `TestForkVenuesAndFlows` |
| `EVERLONG_FLAMM_FORK_PHASE2` | `1`: `TestForkVenuesAndFlows` refreshes and compares each scenario again after its sequence |
| `EVERLONG_FLAMM_RECORD` | fixture recorders (section 5): `1` / `missing` re-record / extend `tracker_rpc_51302915.json.gz` (with `BASE_RPC_URL`); `tracked` rewrites `tracked_51302915.json`; `armed` records `tracker_rpc_armed_51313004.json.gz`; `sequence` writes `fork_sequence_<block>.json` |

## How to run

From the repository root:

```sh
# offline (what CI runs)
go test -race -count=1 ./pkg/liquidity-source/everlong/flamm/...

# every fixture row under the race detector
EVERLONG_FLAMM_FIXTURES=full go test -race -count=1 ./pkg/liquidity-source/everlong/flamm/...

# live Base
BASE_RPC_URL=https://mainnet.base.org go test -count=1 -run 'Live' -v ./pkg/liquidity-source/everlong/flamm/

# anvil fork of Base through the adapter (anvil on PATH; about an hour, network-dependent)
EVERLONG_FLAMM_FORK_RPC=https://mainnet.base.org \
EVERLONG_ADAPTER_OUT=<adapter PR>/out/EverlongFlammAdapter.sol/EverlongFlammAdapter.json \
EVERLONG_FLAMM_C104_OUT=<c104-deploy>/out \
  go test -count=1 -run 'Fork' -timeout 150m -v ./pkg/liquidity-source/everlong/flamm/
```

`-run 'Live'` also matches a few offline replays of live-state fixtures, and `-run 'Fork'` a few offline replays of
fork-recorded fixtures; both run without the network.

`TestLiveBlocks` compares single `eth_call`s at pinned historical blocks, so it also needs an endpoint that answers
every such call from that block's state. The public gateway does not always: on several runs it answered one
`previewSwap` at a pinned block from some other state (a sell paid about 0.9% less than the port computed, a buy
0.75% more; a different size and block each time), while the same call re-issued at that block, up to ten times
over, answers exactly what the port computed, and an earlier run matched all six blocks. A run that fails that way
on `mainnet.base.org` is the gateway, not the port; the fork suite and the recorded tapes are the deterministic
comparison. The other public endpoints tried (`base-rpc.publicnode.com`, `base.llamarpc.com`) refuse archive
requests.

## Common provenance

- Solidity source of truth: the c104 tree at commit `80abd43` (`80abd43dc4ea53fe612e6267a471f2937fc29f9c`, branch
  `refactor/flamm-rename`), as deployed on Base (chain 8453). Foundry 1.7.1.
- Generators live in `gen/`. To regenerate, copy the listed `gen/` files into `test/kyber/` of a c104 checkout at that
  commit and run the quoted command from the tree root. Fork generators read Base (chain 8453) over RPC, e.g.
  `https://mainnet.base.org`.
- `*.json.gz` / `*.jsonl.gz` files are `gzip -9 -n` of the generator's output; `lev_curve_tape_v1.tar.gz` is a
  deterministic ustar+gzip archive. Every generated fixture above a few tens of kilobytes is stored that way, and
  the tests read them through `readFixture` / the per-area loaders, which decompress by suffix. Where a file is
  compressed, both digests are listed: the stored bytes, which the digest tests pin, and the generator's output,
  which is what a regenerated fixture is compared against.
- Fixtures at the top level come from each area's own generators. `edges/` holds edge fixtures from a second set of
  generators, written independently of the first (own interfaces, dumps and scenarios), that aim at thresholds,
  1-wei neighbours and seeded random states.
- Replays under the race detector (`fixture_sample_test.go`). Without `-race` every row of every fixture is replayed.
  The upstream CI runs `go test -race -cover`, where the detector and the atomic coverage counters make the uint256
  arithmetic some twenty times slower, so under `-race` the dense replays run a sample: in every stream of a fixture
  (a scenario, entry and direction) the first and last row, every stride-th row, every row of an outcome class the
  stream holds few rows of, and along the grids ordered by amount both rows of every change of outcome class;
  scenario sweeps without recorded outcomes run every stride-th case, offset per scenario. Sensitivity runs and
  sequences are never sampled. `EVERLONG_FLAMM_FIXTURES=full` replays every row under `-race` as well, and
  `EVERLONG_FLAMM_FIXTURES=sampled` samples without it. The sampled suite reaches exactly the statements the full
  suite reaches (2,869 covered blocks of 3,311 each, compared with `go test -coverprofile` in both modes) -- which is not
  the same as catching the same faults, so what the sample keeps matters more than what it covers: on a grid
  ordered by amount the rows that separate a correct fill from a wrong one are the pair on either side of a change
  of outcome class, where a gate starts binding, and every amount-ordered grid keeps both. Dropping them from the
  core edge grid alone lets a one-line lever-up mutant (the taker leg taken as the curve's gross output instead of
  the floored net, `lever.go` against `FLAMMLeverLib.sol:103-105`) through the sampled run, while the full run
  fails on one row. With the CI's flags (`GOMAXPROCS=4 go test -race -parallel 128 -cover -vet=off`) the package
  runs in 47-48 s on an Apple M2 Ultra at 90.8% statement coverage, and in 93 s with `-parallel 1` (366 s before
  the sampling and `t.Parallel`).

| area | Go files | tests | fixtures |
| --- | --- | --- | --- |
| Swap hook | `almcurve.go`, `fee.go`, `hook.go` | `almcurve_test.go`, `fee_test.go`, `hook_test.go`, `swap_hook_edges_test.go` | `alm_*`, `fee_*`, `hook_*`, `edges/{alm,fee,hook}_*` |
| Financing | `morpho.go`, `irm.go`, `account.go`, `router.go`, `gate.go` | `account_test.go`, `irm_test.go`, `gate_test.go`, `router_test.go`, `router_fixture_test.go`, `financing_edges_test.go`, `router_settlement_edges_test.go`, `financing_sequences_test.go` | `gate_*`, `mm_*`, `edges/{gate,mm,router,swap_settlement}_*` |
| Leverage venue | `levcurve.go`, `levhook.go` | `levcurve_test.go`, `levhook_test.go`, `leverage_edges_test.go` | `lev_*`, `edges/lev_*` |
| Pool core, end to end | `state.go`, `state_reads.go`, `pricefeed.go`, `swap.go`, `lever.go` | `core_e2e_test.go`, `core_edges_test.go`, `core_edge_sequences_test.go` | `core_e2e_*`, `edges/core_edge_*` |
| Kyber integration | `pools_list_updater.go`, `pool_tracker.go`, `tracker_reads.go`, `attest.go`, `multicall.go`, `hook_registry.go`, `hook_kinds.go`, `pool_simulator.go`, `config.go` | `hook_registry_test.go`, `pools_list_updater_test.go`, `pool_tracker_test.go`, `pool_simulator_test.go`, `policy_binds_test.go`, `simulator_contract_test.go`, `config_test.go`, `msgpack_test.go`, `msgpack_external_test.go`, `export_test.go`; network: `live_test.go`, `fork_multipool_test.go`, `fork_parity_test.go`, `fork_gas_test.go`, `fork_edges_test.go`, `fork_tracker_test.go`, `fork_scenarios_test.go`, `fork_venues_test.go`; harnesses: `rpctape_test.go`, `fork_harness_test.go`, `fixture_sample_test.go`, `race_{on,off}_test.go`, `aliasing_test.go`, `math_common_test.go` | `tracker_rpc_51302915.json.gz`, `tracked_51302915.json`, `tracker_rpc_armed_51313004.json.gz`, `fork_sequence_51330064.json` (section 5) |

## 1. Swap hook: AlmCurve, EverlongStrategy.fillFee, EverlongHook fill

### 1.1 Module fixtures

Generators: `gen/AlmCurveGrid.t.sol`, `gen/FeeFillGrid.t.sol`, `gen/HookFillGrid.t.sol`, `gen/HookLiveSwapTrace.t.sol`.
Command, for each of `AlmCurveGrid`, `FeeFillGrid` and `HookFillGrid`:

```sh
FOUNDRY_SPARSE_MODE=true forge test --match-path test/kyber/<Generator>.t.sol -vv --gas-limit 9223372036854775807
```

Each writes `test/kyber/fixtures/<name>.json`; the grids are stored here gzipped.

- `alm_curve_grid.json.gz` forks Base at block 51310000 and calls the DEPLOYED AlmCurve library
  (`0xf82DdF0A8a50bc2C3F163997766bA1839E527A17`: supportFor, reservesAt, swapExactInX96). The internal yAtX is read
  exactly through reservesAt on a full-domain support with `yHi = 0`, `anchor = Q96`, `kappa = WAD`; priceAtX through
  the live hook's `spot()` with `reservationPriceWad = WAD^2`.
- `fee_fill_grid.json.gz` calls EverlongStrategy's internal fillFee/reductionG/volMultiplier/logRatioAbsWad through a
  thin harness compiled from the c104 source. The library is internal; the deployed-bytecode fee path is covered by
  the hook grid's previewFeeWad rows.
- `hook_fill_grid.json.gz` forks Base at block 51310000 and drives the LIVE EverlongHook
  (`0x65CBD227cBC61248ae77a5fC813A29C54C092134`, codehash `0x63ca8158...3e1d` as in the deployment record) with its
  storage overwritten per state, recording `spot`, `bookFor`, `previewFeeWad`, `previewExactIn` and a pool-pranked
  `executeExactIn` whose committed book is read back from storage.
- `hook_live_swap.json` (same generator, `test_liveSwap`) is the state at block 51302915 and the context of the one
  real Swap (tx `0x46c3cd72a5860b2fe546e5a2130e066314e3777027151661e1e4f19a935901fa`), transcribed from the
  `HookLiveSwapTrace` `-vvvv` trace.

Reverts are recorded as raw revert data.

| file | sha256 (stored) | sha256 (uncompressed) |
| --- | --- | --- |
| `alm_curve_grid.json.gz` | `04b679fb9a16062be73e3af140d65628b88a7b01e13c963189ff79af12c16373` | `ab374ed44844116a557ca41a508ea977b50701a073309db6564739abe26f5327` |
| `fee_fill_grid.json.gz` | `2d704853e4f3c88784646aa8fac663dabb4b1f793ee32f3386b9a4b1046fc5e3` | `d44fc45c2da5a84a99618abda64fbf2cdc8b567e4722863da517e415e6ad2608` |
| `hook_fill_grid.json.gz` | `ecd74e6003afd5cabf5b2a1c73df9e014f7cc5b91dc1c0cfb984767cbbe2b05b` | `892cfeda310f5c16a7dc0b524e0aff52d09941d49ff4528a0df4ebea79c23453` |
| `hook_live_swap.json` | `ab07329ca46620ca9d859747472a98d7603080c70028d8228311d83a6c20b220` | n/a |

### 1.2 Edge fixtures (`swap_hook_edges_test.go`)

Generators: `gen/SwapHookEdgesBase.sol`, `gen/AlmCurveEdges.t.sol`, `gen/FeeEdges.t.sol`, `gen/HookFillEdges.t.sol`.
Each writes `test/kyber/fixtures/*.json`. Command:

```sh
FOUNDRY_SPARSE_MODE=true forge test --match-path 'test/kyber/{AlmCurve,Fee,HookFill}Edges.t.sol' -vv --gas-limit 9223372036854775807
```

Every row is one call `{f, a, ok, r, e}` with raw revert data.

- `alm_curve_edges` (`TestAlmCurveEdges`) runs the internal AlmCurve functions (cWad, yAtX, priceAtX, xAtPrice,
  supportFor, heldAt, swapExactIn) through a source-compiled harness, and reservesAt/swapExactInX96 against the deployed
  library on a Base fork at block 51310000: domain and amplification edges with 1-wei neighbours, the `b = 0` branch
  point, clamp gates, seed-skip gates on `yTarget`, band truncation, malformed supports, and a keyed random grid.
- `fee_edges` (`TestFeeEdges`) runs Solady lnWad (every power-of-two boundary and every top-byte lookup pattern) and the
  EverlongStrategy fee law through a source-compiled harness, including the reductionG/volMultiplier/fillFee overflow
  and zero-denominator panics and the tie, ramp and band thresholds.
- `hook_fill_edges` (`TestHookEdges`) drives the LIVE EverlongHook at block 51310000 with its storage overwritten per
  state (the lazy-rescale branches, retracted and invalid books, invalid `_p.aWad` or support, spot overflow, fee-row
  edges) and records spot, bookFor, previewFeeWad, previewExactIn and a pool-pranked executeExactIn, with the committed
  book read back in slot order (kappa, x, rs, is, rv, iv). Caps are drawn at 1-wei neighbours of the realised net. It is
  `jq -cs add` of the tests' `hook_fill_edges_{s01..s11,r1..r5}.json`, in that order.
- `hook_fill_loan_scale_edges` (`TestHookEdgesLoanScale`; `jq -cs add` of `hook_fill_edges_scale1.json` then
  `hook_fill_edges_scale1e10.json`) repeats two deep sweeps with the hook's three LOAN_SCALE PUSH32 sites patched to 1
  and 1e10.
- `hook_live_tx_edges` (`TestHookEdgesLiveTx`) forks at tx `0x46c3cd72...901fa` and replays it with `vm.transact` for
  the committed book. It then re-drives the same swap from the taker through an etched calldata tap on the hook, to
  capture the exact executeExactIn context (its committed book must equal the transaction's), and records previewFeeWad,
  previewExactIn and executeExactIn at the pre-state.

Single files are compacted with `jq -c .` before compression.

| file | sha256 (stored) | sha256 (uncompressed) |
| --- | --- | --- |
| `edges/alm_curve_edges.json.gz` | `3636618f5913e6aaf430b392736ac18f0e09c056b81b2b95e780530ee098a12b` | `cebb7308e5136e98185b0e92934b72256bd1275a7d3538f5115ec970c4ede4de` |
| `edges/fee_edges.json.gz` | `7959030d1ad231297c9690481882c6129d988454340bbeca27a0454c81d84253` | `b4e060c4a69f598a66b9764feab94590140a605ccc98ba4dfe7d472a75d0f1a9` |
| `edges/hook_fill_edges.json.gz` | `408011fc26d01bd6665dcdbe557419034dbcdfdbac59ad6a890a6279e4220b3b` | `42ac0a7800960c47bc3588c43b11aa3029a06e50f669a8fc318e1038e109b882` |
| `edges/hook_fill_loan_scale_edges.json.gz` | `8850d02792690121ecebb898088727fdc6d514dc122d98028e182d151add38ac` | `770b6944aab551db47235f22b20eccfa3f3906e32bdffe66090fd5fe5f1ded3b` |
| `edges/hook_live_tx_edges.json.gz` | `70562e9bb36f73fd388129f8a938e2189bda2fd205003486c94a61653809a271` | `d28bdcc17d7ea751659602c4ad32289c66b8a7bee97b671a565c15dbe8d15983` |

## 2. Financing: Morpho Blue, AdaptiveCurveIrm, MorphoBlueAccount, MMRouterLib, FLAMMGateLib

### 2.1 Module fixtures

Generators: `gen/GateMathFixture.t.sol`, `gen/MMFixtureBase.sol`, `gen/MMFinancingFixture.t.sol`, with an empty
`kyber-out/` directory at the project root (the generators write there; files were copied verbatim). Commands:

```sh
FOUNDRY_SPARSE_MODE=true forge test --match-path test/kyber/GateMathFixture.t.sol -vv
FOUNDRY_SPARSE_MODE=true forge test --via-ir --match-path test/kyber/MMFinancingFixture.t.sol -vv
```

`MMFinancingFixture` needs `--via-ir`: its JSON writers hit stack-too-deep under legacy codegen.

- `gate_math.json.gz`: FLAMMGateLib compiled from that commit behind a harness with a mock router and feed, 400 seeded
  books.
- The `mm_*.json.gz` fixtures run on Base forks against the DEPLOYED Morpho Blue, AdaptiveCurveIrm `0x4641…2687`, MMRouter
  `0x19A9…6bB4` and MorphoBlueAccount `0x6760…6c48`:
  - `mm_irm_grid.json.gz`: borrowRateView/borrowRate over rateAtTarget x supply x utilisation x elapsed at block 51317000.
  - `mm_live_views.json.gz`: router/account/Morpho/Lens views at 51317000 and warped +1s..+365d, with lowered managed
    figures, binding rate ceilings, and an unreadable IRM inside and beyond the grace.
  - `mm_live_settle.json.gz`: real swaps through the live pool at 51317000, with pre/post state per step.
  - `mm_real_sell.json.gz`: the state at 51302915 and around tx `0x46c3cd72…`, replayed in its own block (`vm.transact`).
  - `mm_multi_venue.json.gz`: the deployed router bytecode etched with fresh storage over three Morpho venues (two USDC
    markets, one seeded with a 10% fee, and a WETH market), driving fund, the cascades, reclaim and the single-venue
    entries.

| file | sha256 (stored) | sha256 (uncompressed) |
| --- | --- | --- |
| `gate_math.json.gz` | `3d2c978672c08d67df2310e34fb9eb936f09562882cfdb366f1d70ca5f14fe3c` | `1451caf859e41753459fb85c35d5468c024546ffdb3df5eefdae5e8a1bae6f94` |
| `mm_irm_grid.json.gz` | `968fdfad25549694a4376a665633cec0f1e9b3cfd6f9301f21c66da98a1ec9c1` | `eeb65d925ac036186cb62f61b6b2ad9f7a94f38de935d2e5a4d912db9af3ef37` |
| `mm_live_settle.json.gz` | `870b1a9fc3b09ab84025292874d9f304ded40ba3c87626d2ec5b703ada3f7f9c` | `6ff776c04df8fcb4d2cf2e12708424d6bfce34612966df4332aa932ed1e48f47` |
| `mm_live_views.json.gz` | `14478ba371bc55b0df5ac5d4427c8e35d66e487796825a7679811ad81c107488` | `167efc2a11d037805ed29d2eddba0548e61da1a17227afca3f36a5a431416998` |
| `mm_multi_venue.json.gz` | `46b96f01244f9a8380918554adc9e662c00f8ca7d7520e8d75b216a32da4c42b` | `44332b44b82794ff51ac3ae98ada677fdf0ba4c50295f72e87e72401dd094289` |
| `mm_real_sell.json.gz` | `d6d6e3ff5e9c84eaab525f570ae31a1c6aff46d9b331e96b93731bfd33330b64` | `04d693f8be8e338a50200c5236800411f5a312a270c950a094dac7a844dd255e` |

### 2.2 Edge fixtures (`financing_edges_test.go`)

Generators: `gen/FinancingEdgesBase.sol` plus `gen/{Morpho,Account,Router,Gate,Settle}Edges.t.sol`. Command:

```sh
FOUNDRY_GAS_LIMIT=9223372036854775807 FOUNDRY_SPARSE_MODE=true forge test --via-ir --match-path "test/kyber/{Morpho,Account,Router,Gate,Settle}Edges.t.sol"
```

They fork Base at block 51317000 and write storage directly into the DEPLOYED Morpho Blue, AdaptiveCurveIrm,
MorphoBlueAccount and MMRouter. The live pool record is extended in place to four venues over two loan assets, one of
them with no rate model; oracles and IRM outages are mocked. The generators record every Blue transition, IRM read,
account view and mutator, Router view, funding ceiling and entry (with full post-state), and the settlement of real
pool swaps. FLAMMGateLib (80abd43) runs behind a harness for adversarial books: domain edges, 1-wei neighbours of the
grace, hysteresis, health, liquidity and monotone-slack thresholds, uint128 / uint256 overflow, and seeded grids. Each
array opens with a `{"header":true}` element. Output goes to `kyber-out/edges/*.json`. Tests: `TestMorphoEdges`,
`TestIrmEdges`, `TestAccountEdges`, `TestRouterEdges`, `TestGateEdges`, `TestSettleEdges`.

| file | sha256 (stored) | sha256 (uncompressed) |
| --- | --- | --- |
| `edges/mm_account_edges.json.gz` | `36f62065833adc4ac9010cd47b67201ee08cb90bfc42fd9afde6ee2a01a5e945` | `62967fe3ecb0738f170aa43e8338b90d619d17428061de571e4aa794e2af36df` |
| `edges/gate_edges.json.gz` | `08b7042989376e9a143f9561bd4bf718ae5306351b1d4e0eabc866ececa3407b` | `0508c918ddae9567915ec80e05c9a60be726f3b8e573552fa3ec03ccc1f76474` |
| `edges/mm_irm_edges.json.gz` | `f89edd49854ef05173d17b0e8e3db076cb2da2565fe058b5b73221b3fd7ad5bc` | `12040f44ac7f12a00808abd2843a9237f4e21238366ac2c3c5f78403c0c44973` |
| `edges/mm_morpho_edges.json.gz` | `e44750b12fb7f2e56374a8d3f5afb511df2aaa8c68b3ee2d2fd7468925222847` | `b0b8893602dc98f8229f4d5c5ebacd9a845580d8720c7e570322f27748a10af7` |
| `edges/mm_router_edges.json.gz` | `a75ca51f72abd6742e623e741a12379faa8dfec2894b297ae3859939a3efef74` | `1e8979d1fb6363d175454f0ec1cb1c88fd14b58a113824244d86d8f28716bd7e` |
| `edges/mm_settle_edges.json.gz` | `bf1aa976d3ab281f6cc64c1f264539bcde480a02a974c3eab70c2fc7fb07b2b1` | `daa4c2a975c4923225452768fe99a9890505b5abe9020a0a8d17c4f8cb45f553` |

### 2.3 Integer edges: int256 and Math.mulDiv (`account_test.go`, `gate_test.go`)

Generator: `gen/GateIntEdges.t.sol`, which extends `gen/GateEdges.t.sol` and `gen/FinancingEdgesBase.sol` (all three in
`test/kyber/`). No fork. Command:

```sh
FOUNDRY_GAS_LIMIT=9223372036854775807 FOUNDRY_SPARSE_MODE=true forge test --via-ir --match-path test/kyber/GateIntEdges.t.sol --match-test test_intEdges
```

- `mm_muldiv_edges` (`TestMMMulDivOZ`) records OpenZeppelin (compat v4) Math.mulDiv, floor and Rounding.Up, over every
  triple of 15 boundary values (0..3, 1e18 +- 1, 5^18, 2^128 +- 1, 2^255 +- 2, 2^256 - 2, 2^256 - 1) plus 400 seeded
  triples. The generator asserts that the rounded-up `+= 1` overflow is Panic(0x11).
- `gate_int_edges` (`TestGateIntEdges`) reuses the `GateEdges` harness and row format for 20 books on FLAMMGateLib's
  int256 edges: netL18 at and across 2^255 (wrapping casts, checked subtraction), netPW's casts landing on or past
  type(int256).min on both sides, roomWad at u = min, `_readableU` and the quarantined entry frame with frozen debt at
  or past 2^255, a mulDivUp whose floor is exactly type(uint256).max (requiredPosted and context), and roomNative with
  an epsilon at and above WAD.

Each array opens with a `{"header":true}` element. The generator writes
`kyber-out/edges/{gate_int_edges,mm_muldiv_edges}.json`.

| file | sha256 (stored) | sha256 (uncompressed) |
| --- | --- | --- |
| `gate_int_edges.json.gz` | `2cf610563f5c38e742ca243b77f9d7c254e5949cad35ae068e33b4d7c0ec123b` | `089f7189127fe806b766f052bde717f4c43eaa07760bbd883e710c5ebc4f2616` |
| `mm_muldiv_edges.json.gz` | `29f59dd8c4989eaedbc63a4fc5817c9de8e79b548e270b1811f67f22ca050c58` | `639678627c3da3f87372826aa4678febf6e0dcbb3e88c84e6bc1967ef9fd471e` |

### 2.4 Settlement edges (`router_settlement_edges_test.go`)

Generators: `gen/RouterSettlementEdgesBase.sol`, `gen/RouterSettlementEdges.t.sol`, `gen/MorphoMarketEdges.t.sol`,
written independently of the generators above. Command:

```sh
FOUNDRY_GAS_LIMIT=9223372036854775807 FOUNDRY_SPARSE_MODE=true forge test --via-ir --match-path "test/kyber/{RouterSettlementEdges,MorphoMarketEdges}.t.sol"
```

They fork Base at block 51317000.

- The settle runs replace the pool proxy's code with a harness that keeps the pool's storage and DELEGATECALLs the
  libraries the deployed implementation links (FLAMMSwapLib `0x89aA5f76765D16c460A5B4a7AC6a385e35d2D405`, FLAMMGateLib
  `0x50417cB978f856b3885AbfA97308e0377FC2844e`). settleSell, the buy branch of execute (anchor, settleBuy,
  assertEntryGate), payLoan, takeLoan, releaseExcess and the gate composites therefore run the shipped bytecode, on
  amounts no preview bounds. The live Router record is extended in place to three USDC venues (`_one_loan`), and to a
  fourth venue on a second, 18-decimal loan asset with mocked feed crosses (`_two_loans`). `TestRouterSettlementEdges`
  replays both. Every Router entry, cascade
  and view is replayed from each written state, with its full post-state.
  - States: 22 hand-built books (managed below actual, rate ceilings, quarantine and grace venues, grace
    3599/3600/3601, dead and zero-answer oracles, tight caps, reversed priorities, pause, excess collateral, a book over
    the pin, an unchecked feed, rate ceilings on grace and quarantined venues, and exposure exactly on the bound with
    physical -1/0/+1), seeded random books, a release sweep across posted == 10 * excess on three venues, and a
    same-venue withdraw-then-borrow priced at exactly the planned rate.
- `mm_market_edges` (`TestMorphoMarketEdges`) drives Morpho Blue, the AdaptiveCurveIrm and MorphoBlueAccount on two
  created markets registered on the live account (a 77% market on the IRM, a 91.5% market with none): oracle answers of
  zero / revert / live on indebted positions, the IRM's zero-speed point with its 1-wei neighbours, both wExp clip
  thresholds, odd adaptations and utilisation above one, the account grace with a fee, borrowRateAfter's fail-closed and
  uint128-saturation edges, repay around the accrued debt and a zero-share burn, Blue's zero-floor, liquidity, health
  and uint128 edges, and a 420-row seeded grid of realistic magnitudes.
- `gen/RouterRepaySnapshotScope.t.sol` writes no fixture. Run with
  `forge test --via-ir --isolate --match-path test/kyber/RouterRepaySnapshotScope.t.sol -vv`, it shows that the
  Router's repay snapshot does not survive into a later transaction (NoRepaySnapshot).

Each array opens with a `{"header":true}` element. Output goes to `kyber-out/edges/*.json`.

| file | sha256 (stored) | sha256 (uncompressed) |
| --- | --- | --- |
| `edges/router_settlement_edges_one_loan.json.gz` | `565816ed7ce9c910d51381025c15e583228e032f8325ceadd87794491b963069` | `fd801edbff89604ac8eb76ab4b12e175aabb38209e0e4eac36b0cfb88275a6de` |
| `edges/router_settlement_edges_two_loans.json.gz` | `aad4bfb10e2cd5b5cf6e754dd918328d795d62813ef38cb016d5b5ea1412f266` | `2d69e492797ca7d2504f7ce84599393e42c59a1ab5a3b5448f7f962dd69f2ff5` |
| `edges/mm_market_edges.json.gz` | `ae283f592a151ace394b26b10b969d732dfa6697be55953b21d8f4e6ab2fdb0d` | `14bad2954178b3131f8da248249be4028b84c749b52b3ae05deb8a98441ffaeb` |

### 2.5 Stateful sequences (`financing_sequences_test.go`)

Generators: `gen/FinancingSequenceBase.sol`, `gen/RouterSequences.t.sol`, `gen/SwapSettlementSequences.t.sol`,
`gen/MorphoAccrualGrid.t.sol`, on a Base fork at block 51317000. Command:

```sh
FOUNDRY_GAS_LIMIT=9223372036854775807 FOUNDRY_SPARSE_MODE=true forge test --via-ir --isolate --match-path "test/kyber/{RouterSequences,SwapSettlementSequences,MorphoAccrualGrid}.t.sol" -vv
```

`--isolate` makes every recorded step its own transaction, so the Router's transient repay snapshot ends with it.
Files are JSON lines, one row per transaction.

- `router_sequence_a` / `_b` (520 steps each) are pseudo-random Router transactions against the DEPLOYED MMRouter,
  MorphoBlueAccount, Morpho Blue and AdaptiveCurveIrm. The live pool record is extended to four venues over two loan
  assets, and a call-bundling proxy is etched over the pool address (repay, then proportional withdraw, in one
  transaction). Between transactions come time warps across the IRM grace, third-party Blue calls (utilisation to
  100%, donations and repays on the account's behalf), liquidations, IRM outages, oracle moves onto the band edges,
  zero / reverting oracles, and changes to caps, rate ceilings (on the live rate +-1), flags, pin, pause, drawn set and
  priorities.
- `router_sequence_liquidation` is a scripted liquidation run (managed collateral above actual, bad debt).
- `swap_settlement_sequence_a`..`_d` (600 steps each) are real `pool.swap` calls through the deployed pool (as shipped,
  with an inflated book, and with two extra USDC venues), with the Chainlink answers re-stamped and moved, lowered /
  inflated physical, and reserve-target and lending-feature changes.
- `swap_settlement_sequence_e` is scripted through releaseExcess's swallowed router revert, quarantine, and a
  proportional withdraw one transaction after a repaying buy (NoRepaySnapshot).
- `mm_accrual_grid` (900 rows) writes random market states on the live market and records the IRM view and stored
  endRateAtTarget, Blue's accrueInterest and the account views.

`TestRouterSequences` and `TestSwapSettlementSequences` REPLAY the sequences (`TestMorphoAccrualGrid` the grid):
markets, positions, rateAtTarget, managed fields and the pool ledger are carried from the port's own transitions and
compared with the chain before and after every step, together with every Router / pool view per step. Output goes to
`kyber-out/edges/*.jsonl`.

| file | sha256 (stored) | sha256 (uncompressed) |
| --- | --- | --- |
| `edges/mm_accrual_grid.jsonl.gz` | `25b2bc16424dd45f1165bec5ba5b6cf668d55b9674ecdb6d11fa251ebc771590` | `d56a5c0f66d79be98a7b863be4c2b0570ae146d4ab6697bbb12c00610db5a113` |
| `edges/router_sequence_a.jsonl.gz` | `4bb4ffac4e4b3cbad94c5114c9f1c0c65b971892501155bcab38ad1cf277b55d` | `0c03c48720ccea2147006316afdfe70d966f3bd0dffa43eaf4a9982d34e90bca` |
| `edges/router_sequence_b.jsonl.gz` | `3adc3e50d26f41a498307bfec81b4735d70c2238ce7575d6e2f737acbce0b646` | `28bafeb5b063e23ea00113a2de7e16eecd14cc70475c52f5bcf07f289e9fe6fa` |
| `edges/router_sequence_liquidation.jsonl.gz` | `2399691adf0b8116dc8adbb4b9caf06490f37577ff96a8d26b085835ec625a54` | `b81c16705d250fa905a1660675beba71910e5245175e12c7abfbe85a6dbbe927` |
| `edges/swap_settlement_sequence_a.jsonl.gz` | `8b1d5d0c0a144097078502c5d74f4b7cd07e96b6e21ff1567c62b0b346826cd8` | `077df643e320650debd00a91e342d6533e5c3b8b3da6aefac1b502236ad39dce` |
| `edges/swap_settlement_sequence_b.jsonl.gz` | `1e1353a7a804097e97ec648a5eec328b12386418a96552f1d560bae7124491ed` | `6d982e6f3f4e4594156ccaab6af160e4d2316323bf9c287ac989fda9b62cec31` |
| `edges/swap_settlement_sequence_c.jsonl.gz` | `6f4fe3b95b44bd336080f380dc7b7eee45bcb7e87e08a3ea89dc64483daa1740` | `964342e49092760c3c21a685da19c4917ee6512a34312b14768f6cb84a4caaeb` |
| `edges/swap_settlement_sequence_d.jsonl.gz` | `04eff241349eb5f1da08d98af84e8172450db92969ae4f56a88458b646a8d0c7` | `41388684858fc7aee5249a2b6f1b8a180bade9af984d02fce259e8f2e2d8b013` |
| `edges/swap_settlement_sequence_e.jsonl.gz` | `efaeb37faac834f3a422b1d12252e9a3fd8c84983a6672214c9b3c21bef094e2` | `d8d590240b6970e75808f0b02ad2d29b9a8d5da13effcf9e7121b89152ea486e` |

## 3. Leverage venue: CollRebalancerMath, EverlongLeverageHook

### 3.1 Module fixtures

`lev_curve_tape_v1.tar.gz` is the unmodified c104 parity tape `test/flamm/lev/tape/levcurve_c1_tape_v1.*.json` (16
blocks + manifest), packed as a deterministic ustar+gzip archive (python `tarfile`, mtime 0). The manifest's sha256 is
`9df07c65ea45563c0f343ed98290df1fb10fa1a3ea870dbc3c9bf51c26a058f6`; it pins each block's sha256, and the test
re-verifies them. `TestLevCurveParityTape` replays all 22,465 rows with the `_expectedOut`/`_assertRow` semantics of
`CollRebalancerMathLevCurveParity.t.sol`.

The other fixtures come from `gen/LevRecorder.sol`, `gen/LevGoldenFixture.t.sol` and `gen/LevForkFixture.t.sol`.
Both generators are deterministic (re-runs reproduce the digests). `TestLevFixtureDigests` pins the four digests below.

```sh
FOUNDRY_SPARSE_MODE=true FOUNDRY_GAS_LIMIT=9223372036854775807 forge test --match-path test/kyber/LevGoldenFixture.t.sol -vv
FOUNDRY_SPARSE_MODE=true FOUNDRY_GAS_LIMIT=9223372036854775807 forge test --match-path test/kyber/LevForkFixture.t.sol -vv
```

- `LevGoldenFixture` runs on the local LevBase stack:
  - `lev_hook_local_fixture.json.gz`: the VenueGolden sequence plus displaced and synthetic grids.
  - `lev_hook_band_fixture.json.gz`: `_assertAnchorAndBand` through a harness.
- `LevForkFixture` runs on a Base fork at block 51317000 against the deployed stack:
  - `lev_hook_fork_fixture.json.gz`: curator unpause and keeper spread posts. The LeverContext is captured by etching a
    calldata-echo probe over the leverage hook for one preview, then frame, previewLever and pool.previewLever are
    recorded with revert data.
  - `lev_curve_fork_fixture.json.gz`: frozenParams() and a leverageQuote/deleverageQuote/anchorAndBase/isStateSafe
    grid plus a keccak-seeded sweep, called on the deployed CollRebalancerMath
    `0xC002d0731E6a2E6e80Be754779bCEf6B01Aff0bb`.

| file | sha256 (stored) | sha256 (uncompressed) |
| --- | --- | --- |
| `lev_curve_tape_v1.tar.gz` | `18fe3e2aa02cce91f8b312f95730b2ef556363e271e82dbbc12d1d107ce29437` | n/a |
| `lev_curve_fork_fixture.json.gz` | `798735119ee4e322ec929a75aa48d8855e630f622fc20aa4d3a27a54c30d4e9f` | `74fce4f93e92ee09200739c5edd44cb6ebbf0bc527863466cbf68357ba45ac46` |
| `lev_hook_fork_fixture.json.gz` | `f5d027dc34dbc37289edbf91312d5adf67217bef1a94e02883bee49319580641` | `fb379cb74733dba31578cb9ed487a03375439993d0d78811814801968f3fdd8d` |
| `lev_hook_local_fixture.json.gz` | `6e8b8d178b2f07c24aa0b4b94021b48a44f50455829786f11f07b92dbeabd59f` | `46d8ea7be0ca6207527cd9d2f263ae39d71ff7213c0928d492dacd10618e5a53` |
| `lev_hook_band_fixture.json.gz` | `52e04fdf28c6224faa48e1a4cf581be3d8d070b5a0a53b65c47cf4651d2cb90d` | `e3f59a449cf5ac2e311066e7cb70f29fb57e86ea96056ea4d41b613605378231` |

### 3.2 Edge fixtures (`leverage_edges_test.go`)

Generators: `gen/LevEdgeRows.sol`, `gen/LevCurveEdgesHarness.sol`, `gen/LevCurveEdges.t.sol`, `gen/LevHookEdges.t.sol`
and `gen/lev_curve_math_copy.py`. Copy them into `test/kyber/` of a c104 checkout at 80abd43, run
`python3 test/kyber/lev_curve_math_copy.py` (it writes `test/kyber/CollRebalancerMathCopy.sol`), then:

```sh
FOUNDRY_SPARSE_MODE=true FOUNDRY_VIA_IR=true FOUNDRY_OPTIMIZER=true FOUNDRY_OPTIMIZER_RUNS=100 FOUNDRY_GAS_LIMIT=9223372036854775807 FOUNDRY_MEMORY_LIMIT=4294967296 forge test --match-path test/kyber/LevCurveEdges.t.sol -vv
FOUNDRY_SPARSE_MODE=true FOUNDRY_VIA_IR=true FOUNDRY_OPTIMIZER=true FOUNDRY_OPTIMIZER_RUNS=100 FOUNDRY_GAS_LIMIT=9223372036854775807 FOUNDRY_MEMORY_LIMIT=4294967296 forge test --match-path test/kyber/LevHookEdges.t.sol -vv
```

Output goes to `test/kyber/fixtures/lev_{curve,hook}_edges.json`; re-runs are byte-identical. Both fixtures fork Base at
block 51318000. Rows are `[op, inputs, [status, words...]]`: status 0 carries every ABI return word, status 1 a revert
as `[len, selector, arg]`.

- `lev_curve_edges` (`TestLevCurveEdges`, 77,002 rows) calls the deployed CollRebalancerMath
  `0xC002d0731E6a2E6e80Be754779bCEf6B01Aff0bb` (`anchorAndBase`, `leverageQuote`, `deleverageQuote`, `isStateSafe`,
  `frozenParams`) over:
  - branch thresholds: half-law/Hermite/wall/recovery debts and their 1-wei neighbours, `3*newDebt` vs anchor, and dust
    anchors 95..106;
  - MAX_INPUT and uint256 edges, ten reservation prices, and an exhaustive dust sweep;
  - exact Mul512 product ties, and rounding ties located off-chain (the recovery y+1 bracket,
    `collAtTarget == collateral`, `newCv == cv`);
  - a keccak-seeded grid.

  Every public row is re-run on a verbatim internal-visibility copy (`gen/lev_curve_math_copy.py` derives it from
  the pinned source) and must be byte-equal. That copy also supplies the private-helper rows (`_cvRequiredOnAnchor`,
  `_debtCapOnAnchor`, `_recoveryState`, `_deleverageProRata`, Bezier/lerp, Mul512, ...).
- `lev_hook_edges` (`TestLevHookEdges`, 33,808 rows) calls the deployed EverlongLeverageHook
  `0xE0A98d8e60035832B8BaD7f7af7B9B0b3A7308F3` `frame`/`previewLever`, with EverlongHook `bookFor`/`reservationPriceWad`
  mocked to arbitrary books: checked and signed overflows, mulDiv reverts, FrameUnquotable, NothingToFill, LevValueLeak,
  realistic CR/spread/amount grids, feed mismatch, dust frames, `in18 == dSBurn` ties, and random books. It checks
  `executeLever == previewLever` on a sample and replays `_assertAnchorAndBand` transcribed over the deployed
  `anchorAndBase`.

| file | sha256 (stored) | sha256 (uncompressed) |
| --- | --- | --- |
| `edges/lev_curve_edges.json.gz` | `12ac1f76ce0eb7b9d03d8440a738b23a18abb531b19866321db12eb01ca66cd4` | `383484cf5b395ad8eaf5fee1a356d9d75eac2e4581b4e813335194384dd55337` |
| `edges/lev_hook_edges.json.gz` | `ef4a9141f4e6704cc84f14ebda69caccbc709fdda30762a76517cf384653f579` | `436373181cdcffeb3ed8eca195c3597277079bf85c48104b90f039e6f545f317` |

## 4. Pool core end to end: FLAMMSwapLib, FLAMMLeverLib, PriceFeed over the composed state

### 4.1 Module fixtures (`core_e2e_test.go`)

Generators: `gen/CoreE2EBase.sol` (imports `gen/MMFixtureBase.sol`), `gen/CoreE2EGrid.t.sol` and
`gen/CoreE2ESeq.t.sol`. Copy all four into `test/kyber/` of a c104 checkout at 80abd43, then from the tree root:

```sh
mkdir -p kyber-out/core_e2e
FOUNDRY_SPARSE_MODE=true forge test --via-ir --match-path test/kyber/CoreE2EGrid.t.sol -vv --gas-limit 9223372036854775807
FOUNDRY_SPARSE_MODE=true forge test --via-ir --match-path test/kyber/CoreE2ESeq.t.sol -vv --gas-limit 9223372036854775807
for b in 51302915 51313000 51324800; do
  gzip -9 -n -c kyber-out/core_e2e/grid_$b.jsonl > core_e2e_grid_$b.jsonl.gz
  gzip -9 -n -c kyber-out/core_e2e/seq_$b.jsonl > core_e2e_seq_$b.jsonl.gz
done
```

Both generators fork Base at blocks 51302915 (the parent of the one real swap), 51313000 and 51324800, and write JSON
lines. Re-runs are byte-identical. `TestCoreE2EFixtureDigests` pins the stored digests below.

Every state row (`"k":"state"`, `"k":"seq"` and each step's `"post"`) is one complete state, read the way the tracker
reads it: view getters only, decoded straight into `flammReads` (`state_reads.go` lists every call), plus three
storage words that no view exposes:

- `FLAMMStore.lastLeverSpreadPpm`: slot `0x5b7e76949cacd5346234367c3806fe494a22f183af782d834d5fc4ee5b0f4516` (ERC-7201
  base + 22), bits 160..191. The layout is confirmed by the executed lever sequences, which store 17500, then 80000.
- MMRouter venue `managedCollateral` / `managedSupplyShares`: `keccak256(keccak256(pool . 1) + 3) + 6i + 4` / `+ 5`,
  checked against `venue(pool, i)` by `MMFixtureBase._managed`.

Each state also carries deployed views the port must reproduce from it: `peekCross`, `pegOk(loan0)`, `loanPosition`,
Router `positions`, `poolAssetPosition().gross` and `totalAssets` (value or revert).

- `core_e2e_grid_<block>.jsonl.gz` (`CoreE2EGrid`): per scenario, one state and `pool.previewSwap` /
  `pool.previewLever` rows (`"k":"sw"` / `"k":"lv"`, `d` the direction, `r` the return words or `e` the revert data).
  Each direction covers a 8-points-per-decade log grid (sells and lever-ups up to 1e12 sats, buys and lever-downs up to
  1e14 native USDC), the amounts 0, 2^64, 2^100, 2^128, 2^160, 2^200, 2^255 and 2^256-1, and, in the scenarios that
  ask for windows, every class boundary the grid brackets: bisected to one unit, then scanned unit by unit over 600
  units on both sides. The accepted set is not an interval near a band edge, so this is what pins the sawtooth.
  The scenarios are:
  - `live`: as deployed (leverage paused);
  - `armed`: curator `setLevPaused(false)`, keeper `setSpread(17500)`;
  - `armed_hi`: spread 90000, clamped to the band;
  - `stale_spread`: unpaused on the stale post (lever-up SpreadUnavailable, lever-down degraded to the 100000 ceiling);
  - `capped`: `maxSwapNotional` 5 USDC (partial sells, NotionalCap buys);
  - `band_tight`: 1% band;
  - `fee_cap` / `fee_floor` / `fee_floor_loan`: pool fee bounds (0, 1e16) and (5e16, 1e17), and a 2e16 loan fee floor;
  - `paused`;
  - `no_sell` / `no_buy` / `no_lev`: feature bits cleared;
  - `stale_feed`: past the cbBTC heartbeat;
  - `seq_grace` / `seq_down`: mocked sequencer rounds;
  - `peg_broken`: USDC mocked at 0.97;
  - `feed_invalid`: cbBTC answer mocked to 0;
  - `btc_down10` / `btc_down25` / `btc_up10`: the cbBTC answer moved (LevBelowFloor);
  - `irm_grace` / `irm_quarantine`: the AdaptiveCurveIrm mocked to revert, 30 min and 2 h later, with fresh feed rounds;
  - `pin_low`: the Router pin stored at 0.2e18 below the pool's ltv (governance keeps them equal), so the sell's
    funding binds on collateral and partial fills take every re-plan pass;
  - `ltv_low`: the pool ltv stored at 0.05e18 (RoomExceeded, RoomExhausted);
  - `accrued`: 30 days later with fresh rounds.
- `core_e2e_seq_<block>.jsonl.gz` (`CoreE2ESeq`): sequences executed from a `deal`-funded account directly on the pool
  (each call with deadline = block.timestamp), with governance moves and warps in between. Each step records the op
  and its arguments, the timestamp, the return words or revert data, the pool's Swap / LeverUp / LeverDown event words,
  and the post-state. Preview rows interleaved in a sequence are checked at the state they saw. The sequences are:
  - `real_sell` (51302915 only): the 15000-sat sell, 11301759 USDC out;
  - `basic`: sells and buys, where a buy repays and lends and the next sell withdraws the supply and then borrows;
    Slippage, Expired, InvalidPair, InvalidAmount, dust and band refusals; the largest quotable sell and buy;
  - `reclaim`: the largest sell, then a buy paying out more poolAsset than is held physically (repay, strict reclaim,
    release of excess collateral);
  - `notional`: a partial sell at a 10 USDC cap, NotionalCap, a reserve target kept liquid, FeeOutOfBounds from a loan
    fee floor, and a fee clipped at a 1e16 pool cap;
  - `lever`: LevPaused, a degraded lever-down at the ceiling, lever-ups and lever-downs at a live 17500 interleaved with
    swaps, Slippage, the largest quotable lever-up and lever-down, a stale post degrading to the stored 17500 (a filled
    degraded lever-down that leaves it unchanged), SpreadUnavailable, a live 90000 clamped to 80000 and stored, Paused;
  - `warp`: 600 s and up to the cbBTC heartbeat, StalePrice past it, then 30 days and 1 day of accrual with fresh
    mocked rounds;
  - `pin_low`: executed multi-pass partial sells and reclaiming buys at the stored low pin;
  - `irm`: the rate model down inside the grace (IrmUnreadable on the repay) and past it (a quarantined venue: the buy
    stays liquid, the sell pays from liquid).

  The replay carries the port's own post-state forward: it compares every return and event word, and the recomputed
  post-state against the chain's field for field (ledger, loan configs, dials, fee bounds, hook book, spread post,
  feed, Router record, Morpho markets and positions, rateAtTarget, managed fields, lastLeverSpreadPpm). Governance
  steps are applied to the Go state by hand. New feed rounds, the IRM outage and the stored pin (`mockFeeds`,
  `irmDown`, `storePin`) are tracker inputs, so they are copied from the post-state.

| file | rows | sha256 (stored) | sha256 (uncompressed) |
| --- | --- | --- | --- |
| `core_e2e_grid_51302915.jsonl.gz` | 34052 | `ae588fcaf6e08b24b5d7e7b3f12c491b89c73f9e155ceb8641090c5a6004ddd5` | `e089ec3efeeafa6668f00bd6a4e6ddc9e49495097c3c81d0fffbf98e382771f7` |
| `core_e2e_grid_51313000.jsonl.gz` | 25070 | `4e21a865486e59b4fc0eae66290ed509fc1afe88254c558b67cad6b7e62258ee` | `1b31022d0a4b0904fb1bfb74f44c01bc825da0e6cf4377636a98a4b49293f0fd` |
| `core_e2e_grid_51324800.jsonl.gz` | 23501 | `2d053f484f935e3e715c8f3e8655ef6a32f08c4255a6d8d8d2622f0d303130b1` | `63c8de9c5663499d6ce96929b8f254e66ff294f34b756ee85cf610d6490b2836` |
| `core_e2e_seq_51302915.jsonl.gz` | 109 | `9ae2f16137c0a744dd730758ad97537bb8533ce6a0868367d1113ff7c35c1116` | `fcd3eb6165c36c68e73b31fdd57a7aee997a6f8218d79f9a7331c9b30fb3c359` |
| `core_e2e_seq_51313000.jsonl.gz` | 108 | `e2d4569b540b7853b49e04507e7ba485439aae3ebb26621c28352b411a92536f` | `44933b2f7b4d1f1be8e8fd0cc214e169d79c58e4cf72c5e2ff671ee90933a07e` |
| `core_e2e_seq_51324800.jsonl.gz` | 108 | `6529b264e751e5026344539a039b45bb90542d56b5f501d5ed3f3f5c05f64d45` | `ae062769d95c666eec3ea03b1ed5f87347d1337e5bfffefce53c624278326b51` |

| generator | sha256 |
| --- | --- |
| `gen/CoreE2EBase.sol` | `e577daaa6e2417722ba30a10511c1493c8a3e954763bf9495a65e2afd24716e7` |
| `gen/CoreE2EGrid.t.sol` | `37415f13b103fbbee629fce208ca290cdfa838a2c9a2783d32a5ea53a57584ee` |
| `gen/CoreE2ESeq.t.sol` | `b9df18f342862aa0823c7686ef73ae95ebf08d9631c1ec28cfcba9c99f851ef3` |

### 4.2 Edge grids and sequences (`core_edges_test.go`, `core_edge_sequences_test.go`)

Generators: `gen/CoreEdgesBase.sol` (imports forge-std only), `gen/CoreEdgeGrid.t.sol`, `gen/CoreEdgeSeq.t.sol`,
written without the CoreE2E generators: own interfaces transcribed from the c104 `src/interfaces`, own state dump, own
scenario logic. Copy the three into `test/kyber/` of a c104 checkout at 80abd43, then from the tree root:

```sh
mkdir -p kyber-out/core_edges
FOUNDRY_SPARSE_MODE=true forge test --match-path 'test/kyber/CoreEdge{Grid,Seq}.t.sol' -vv --gas-limit 9223372036854775807
for k in grid seq; do for b in 51302915 51324800 51326000; do
  gzip -9 -n -c kyber-out/core_edges/${k}_$b.jsonl > edges/core_edge_${k}_$b.jsonl.gz
done; done
```

Both fork Base at blocks 51302915, 51324800 and 51326000 (seeded, so re-runs are byte-identical). The dump decodes into
`flammReads` like the tracker's reads. The two raw-storage layouts are re-derived from the struct declarations and
checked in the generator itself:

- `FLAMMStore.S` base + 22 bits 160..191, with base + 12 required to equal `poolAssetPosition().physical`;
- the Router venue slots, with the account, id and supplyCap words required to equal `venue(pool, i)`.

The Go side rebuilds the revert-selector table from each sentinel's Solidity signature (keccak) and requires the
port's `revertSelectors` to agree. Revert data with no custom error maps as follows:

- the mocks' raw `oracle down` / `down` bytes map to the Morpho oracle / IRM sentinels;
- Morpho `Error(string)` maps by its message.

- `core_edge_grid_<block>` (`"k":"state"` / `"sw"` / `"lv"` / `"fc"`): per scenario one state, then:
  - `previewSwap` / `previewLever` rows: seeded log-random amounts, and geometric sweeps whose every class transition
    is bisected to adjacent units with a +-10..12 unit window;
  - `router.fundingCeiling(pool, 0, collateral, price)` probes at the checked cross, half of it and +1%, over
    collateral from 0 to 1e30 and unit-spaced steps.

  Scenarios, each from a clean snapshot:
  - `live`;
  - `armed_min` / `armed_max`: spread 2500 / 100000;
  - `armed_band5`: band 5%, post 60000 clamped to 50000;
  - `armed_band_tiny`: band 0.15%, ceiling 1500 under the 2500 floor;
  - `armed_band_max`: band 10%;
  - `spread_age_eq` / `spread_age_over`: lastSetTs + 3600 / + 3601;
  - `degrade_zero`: stale post, never filled: lever-down at 100000;
  - `age_zero`: maxSpreadAge 0;
  - `ltv_low_{0,1}`: real `setDials` walks with cooldown warps to ltv 0.25 at phi 1 and 0.12 at phi 0.5;
  - `notional_20` / `notional_1`;
  - fee bounds:
    - `fee_clip`: cap at half the quoted fee;
    - `fee_eq`: floor = cap = the fee;
    - `fee_floor_over`: floor one above the fee;
    - `fee_loanfloor_cap`: a loan floor set under a high cap, then the cap lowered below it;
    - `fee_cap_wad`: cap 1e18;
  - `debtcap_tight`: venue debt cap = debt + 3 USDC;
  - `ratecap_{0,1,1000,1000000,below}`: maxBorrowRateWad at the live borrowRateView plus those offsets, and one below
    it (the non-monotone borrow leg);
  - `borrow_off`: venue borrow flag cleared;
  - `global_paused`: Router `setGlobalPaused` by the protocol safe;
  - feed and sequencer boundaries, each a mocked round:
    - `cb_age_eq` / `cb_age_over`: 3600 / 3601 s;
    - `cb_future`, `cb_round0`, `cb_neg`, `cb_one`;
    - `cb_maxprice_eq` / `cb_maxprice_under`: priceWad exactly 2^200 / one unit below;
    - `usdc_age_eq` / `usdc_age_over`: 90000 / 90001 s;
    - `peg_{lo,hi}_{eq,over}`: USDC at 0.99 / 1.01 and one unit outside;
    - `btc_x3`, `btc_half`;
    - `seq_grace_eq` / `seq_grace_past`: startedAt at now - 3600 / now - 3601;
    - `seq_future`, `seq_down`, `seq_zero_started`;
  - `morpho_fee`: Morpho owner `setFee` 25%, one day of accrual;
  - `irm_dt_{3599,3600,3601}`: the rate model reverting, evaluated at market lastUpdate + dt (IRM_STALE_GRACE edge);
  - `oracle_revert` / `oracle_zero` / `oracle_band_out`: the market oracle reverting, answering 0, or 3% under the feed;
  - `whale_t0` / `whale_dt{0,1}`: a third party borrowing all but 1000 USDC of the market, then 3000 s and 3 days
    later (AdaptiveCurveIrm adaptation at ~100% utilization);
  - `big` / `big_notional`: after a real 25 cbBTC `depositProRata` (gross ~25 BTC, proportional debt);
  - `loancap_debt` / `loan_borrow_off`: Router loan-level caps (debt + 2 USDC; borrowing disabled, supply cap 1
    unit), set by pranking the pool. c104 never calls `setLoanCaps`, so these cover the port's loan-level branches
    rather than a reachable configuration.
- `core_edge_seq_<block>` (`"k":"begin"` / `"pv"` / `"x"` / `"warp"` / `"re"`): executed sequences from a dealt
  taker (`swap`, `leverUp`, `leverDown`), with the rows:
  - `pv`: a preview at the current state;
  - `x`: the call with its `minOut`, lateness against the deadline, return words or revert data, Swap / LeverUp /
    LeverDown event data, and the post-state dump;
  - `warp`: a time move, with a dump that must equal the carried state (nothing but time moved);
  - `re`: a non-core move the tracker re-reads (curator / keeper setters, mocks, third-party Morpho activity, a
    deposit, a snapshot rewind). The carried state must already equal the dump on every field that move cannot touch.

  Sequences:
  - `heartbeat`: cbBTC age 3600 then 3601, USDC age 90000 then 90001;
  - `sequencer`: grace at exactly 3600 s, then up, then down;
  - `spread_degrade`: a live 18000 stored, age 3599 / 3600 / 3601, a degraded lever-down at 18000, a 95000 post
    clamped to 80000 and stored, a degraded lever-down at 80000 under a 0.15% band, maxSpreadAge 0;
  - `ltv_walk`: six `setDials` moves with cooldown warps (some refused) and edge-sized trades at each;
  - `irm_outage`: the rate model down, trades at lastUpdate + 3599 / 3600 / 3601 (a quarantined buy settling into
    liquid);
  - `oracle`: the market oracle reverting or answering 0 (a buy covering the whole debt, lever-downs);
  - `whale`: ~100% utilization, 1800 s and 2 days, then half repaid;
  - `lending`: reserve target 5 USDC, a supply cap at supply + 2 USDC, FEATURE_SUPPLY_LENDING off and on, 7 days,
    venue flags off and on;
  - `morpho_fee`: Morpho fee 25%, 9 days;
  - `same_block`: 14 trades at one timestamp;
  - `donation`: third-party collateral and supply on behalf of the account;
  - `price_move`: the cbBTC feed at 97 / 103 / 90 / 115%, with and without the market oracle following;
  - `pingpong_{a,b,c}`: 60 edge-sized trades each across all four directions at spreads 2500 / 40000 / 80000, with
    Slippage (limit one above the preview), exact-limit and Expired calls and short warps;
  - `settle_hunt`: every preview-accepted size of a sweep executed from the same state and rewound, under a reverting
    or zero oracle, an oracle 4% under the feed and a debt cap at debt + 1. Buys and lever-downs pass the preview and
    revert InsufficientCollateral in settlement;
  - `loancaps`: the loan-level debt cap one unit above the debt and a 1 USDC supply cap (pranked, as above);
  - `big`: 70 trades after the 25 cbBTC deposit, with dial moves;
  - `random_{a,b,c}`: 120 seeded steps mixing all of the above moves.

`TestCoreEdgeGrid` and `TestCoreEdgeSequences` require zero mismatches.

`TestCoreEdgeGridSensitivity` and `TestCoreEdgeSequenceSensitivity` prove the replays can fail. Each small
perturbation must break rows:
- the sequencer grace, a heartbeat or the peg band minus one;
- the spread plus one, the max spread age minus one, a stored degrade value;
- one unit of reserveVolatile;
- the rate cap minus one, the debt cap plus one;
- IrmReadable flipped, a day of Morpho accrual, one unit of totalSupplyAssets;
- 1e15 of room epsilon;
- a carried Morpho accrual stamp one second stale.

| file | rows | sha256 (stored) | sha256 (uncompressed) |
| --- | --- | --- | --- |
| `edges/core_edge_grid_51302915.jsonl.gz` | 13372 | `946d384674b004378198eb5fd023d28fea944ebcf6d626c34994561149d6eba2` | `4ce285898ef6a21e4c07ac7f631d2874711f3ee217c7f998b1360eda9184733e` |
| `edges/core_edge_grid_51324800.jsonl.gz` | 12877 | `8c085a8183d9de9a6edbe7fdab5d08e276141a598739bd4a524baf814fe8f570` | `01c82fd2bda2d25addd8d9faf8abdb819793edbac1703ad62bec510815dca21d` |
| `edges/core_edge_grid_51326000.jsonl.gz` | 12864 | `463868de0f612bdb55be5cad602a5911b6a9d08b3f2904cb578c42f9edc7d3b4` | `ff05306702daf4f71bd67f73e7e58b28b6d0486bc17f3fe1f091f5e7e81478ac` |
| `edges/core_edge_seq_51302915.jsonl.gz` | 2114 | `e6e14db48bcfe28a3f87ff072a5cc2ac70e92e4a27d669c39d3dec9338fca99c` | `3d625c2dd43564c71b20f983f3439c30d276464db28c4721afa8ae15ac4cb328` |
| `edges/core_edge_seq_51324800.jsonl.gz` | 2157 | `53414398f523e1507ed8fdd2c2e4f91642f02dba22a195068af4a5ac948b553d` | `fa2f63a6953ce69f7c861d6fefc28941b845ae815b5f7a49e4bea29657892122` |
| `edges/core_edge_seq_51326000.jsonl.gz` | 2103 | `1906c0dfc12c37b6f3d7dd5b52ffee245cf1107b6aa5da43ea29dc0773ab341e` | `153ca1d81bce1907c1de451173580eeda623582c115831aa03075260f9ca94c4` |

| generator | sha256 |
| --- | --- |
| `gen/CoreEdgesBase.sol` | `5e159ca61ddd51e8c6af9d7e90da40cadd898c9dee83d16abb784c8605d24aa5` |
| `gen/CoreEdgeGrid.t.sol` | `f83a5c6499080e394ba5b06ae08e5f620273718f69470556e4084dde44bdb487` |
| `gen/CoreEdgeSeq.t.sol` | `66558762467d92a309f13acb6834b1c938ce2864cc4549edb9e07338aac02363` |

## 5. Kyber integration: lister, tracker, simulator

These fixtures are recorded by the Go tests themselves, not by a forge generator; the Solidity they describe is still
c104 at `80abd43` as deployed on Base.

- `tracker_rpc_51302915.json.gz` is an RPC tape: every JSON-RPC request the lister and the tracker made, keyed by method
  and canonical params, with Base's answer (`rpctape_test.go`). It holds the listing at the recording head (factory
  `isPool` with the pool's hook set and PriceFeed, the wiring reads, the runtime code of the nine registered
  contracts) and the refresh pinned to block
  51302915, the parent of the one real swap (`0x46c3cd72...`): the view aggregate, the storage words, the probe
  aggregate. Recorded with:

  ```sh
  EVERLONG_FLAMM_RECORD=1 BASE_RPC_URL=https://mainnet.base.org \
    go test ./pkg/liquidity-source/everlong/flamm/ -run 'TestTrackerReplay$' -count=1
  ```

  and then extended twice, keeping every recorded answer the refresh still asks for (the listing keeps its head
  block): with the 17 MMRouter storage words the layout check added (router slot 0, the pool record's eight slots,
  loan 0's first three slots, venue 0's packed flag slot; see `tracker_reads.go`), and with the view aggregate that
  reads the factory's `pendingImplementation()` and `implementationExecutableAt()` (Extra.ScheduledChangeAt), all at
  block 51302915:

  ```sh
  EVERLONG_FLAMM_RECORD=missing BASE_RPC_URL=https://mainnet.base.org \
    go test ./pkg/liquidity-source/everlong/flamm/ -run 'TestTrackerReplay$' -count=1
  ```

  A recording run saves exactly the requests it made, so the superseded view aggregate was dropped.

  The listing aggregate was then rewritten in place, without re-recording, when the lister stopped enumerating
  `FLAMMFactory.pools()` and started confirming each known pool with `FLAMMFactory.isPool(pool)`: the
  recorded `pools()` answer at the listing block 51328889 is exactly `[0xc0fdCB17...D57572]`, and `createPool` sets
  `isPool[pool]` in the same statement that pushes the pool (`FLAMMFactory.sol:252-254`), so `isPool` was true at
  that block. The `pools()` entry was replaced by the `isPool` aggregate carrying that block, that block hash and
  `true`, written by the tape's own writer; re-recording the tape instead would move the listing's head block and
  every read pinned to it. The same rewrite was applied to `tracker_rpc_armed_51313004.json.gz`, whose recorded
  `pools()` answer at 51313004 is the same single pool.

  It was then extended once more, the same way, with the three rounds the dependency set and the oracle window
  added -- the listing's `oracle.BASE_FEED_x()` round, the refresh's `aggregator()` resolve round, and the
  refresh's `oraclePrice` round sent with a `blockOverrides` timestamp -- and the probe aggregate was re-recorded
  under the key that now carries its explicit `gas` (`attest.go` `probeAggregateGas`). All four are pinned to their
  own block, so Base answered them exactly as the original recording would have.

  And once more with the deadline round, the aggregate the refresh sends at the far end of the snapshot window
  (`peekCross`, both `peekUsd`, `pegOk` and `spreadPpm`, with a `blockOverrides` timestamp of the block's own plus
  600 s), and with the factory's two storage words (slots 0 and 1: the beacon implementation and the pending
  implementation packed with its executableAt), which the layout check compares with the views that report them.
  Three keys, nothing dropped or changed. The chain's own deadline is reproducible from the tape's answers: the
  recorded cbBTC round at 51302915 has `updatedAt` 1789394643 against a 3,600 s heartbeat and a block timestamp of
  1789395177, and `PriceFeed.peekCross` on Base answers at `block.timestamp` + 3,066 and stops at + 3,067.

  And once more with the pool's four delayed-governance storage words (`FLAMMStore` slots +30..+33: the pending
  loan-asset hash and executableAt, which no view reports, and the pending venue hash and executableAt, which
  `pendingHookSet()` does report and which the layout check now compares against them). Four keys, nothing dropped
  or changed; all four read `0x00..00` at 51302915, 51313004 and 51330064 on Base, since no ceremony has ever been
  scheduled on this pool. The slot offsets were confirmed on a Base fork at 51376200: the curator's scheduling
  call wrote +32/+33 for `addVenue` and +30/+31 for `addLoanAsset`, and +32/+33 equalled the `(venueHash, venueAt)`
  the view reported.

  And last, the two rounds sent with a `blockOverrides` timestamp -- the oracle window and the deadline round --
  were re-recorded under the key that now carries `Multicall3.getCurrentBlockTimestamp()`, which each of them reads
  its own clock back with (`multicall.go` `clockProof`). Their two superseded keys were dropped; nothing else moved,
  and the entity the replay produces is unchanged. Both endpoints the suite runs against apply the override:
  Base answered `0x6aa80341` at block 51302915, the 1789395777 the round asked for, against the block's own
  1789395177.

  And once more in place, without re-recording, when the lister started walking the hook registry
  (`hook_registry.go`): its first aggregate now reads, beside `FLAMMFactory.isPool(pool)`, the pool's `hooks()` and
  `priceFeed()` (`pools_list_updater.go` `poolHead`). The `isPool`-only entry was replaced by that aggregate at the
  listing block 51328889, assembled from answers the tape already held at that block -- the `isPool` answer, and
  `hooks()` and `priceFeed()` from the listing's view round -- and written by the tape's own writer, which rewrites
  the unchanged tape byte for byte. The assembled answer is byte-identical to Base's own answer to the same request
  at 51328889. The same rewrite was applied to `tracker_rpc_armed_51313004.json.gz` at its listing block 51313004
  (an anvil fork, so assembled only). One key replaced in each; every other request, and the entity the replay
  produces, is unchanged.

  The tape is written as sorted JSON lines and compressed with Go's `gzip.BestCompression` (no timestamp). On replay it
  answers a JSON-RPC batch of more than ten calls the way mainnet.base.org does (one `-32014` error object), so every
  replayed refresh goes through the chunked batches of `multicall.go`.
- `tracked_51302915.json` is the entity that replay produces (its wall-clock `timestamp` zeroed), written by the same
  run or by `EVERLONG_FLAMM_RECORD=tracked go test ... -run 'TestTrackerReplay$'`. Its Extra carries the production
  policy (every margin at its `constant.go` default).
- `tracker_rpc_armed_51313004.json.gz` is a tape of the same shape recorded against an anvil fork of Base at 51313000
  with the leverage venue armed as `TestForkEdges` arms it (curator `setLevPaused(false)`, spread bounds
  0-150,000 ppm, `maxSpreadAge` 3600, keeper `setSpread(13000)`, mined at 51313001-51313004): the listing and the
  refresh at 51313004, whose 21 probes include `previewLever` grids and band edges in both directions. Recorded with:

  ```sh
  EVERLONG_FLAMM_RECORD=armed EVERLONG_FLAMM_FORK_RPC=https://mainnet.base.org \
  EVERLONG_ADAPTER_OUT=<adapter PR>/out/EverlongFlammAdapter.sol/EverlongFlammAdapter.json \
    go test ./pkg/liquidity-source/everlong/flamm/ -run 'TestForkRecordArmedTape$' -count=1 -v
  ```
  The two entities the tape produces (`tracked_51302915.json`) and the two in `fork_sequence_51330064.json` were
  rewritten once more when the Router venue set moved from `StaticExtra` to `Extra`: the set is byte-identical, only
  the field it is published in changed, and `tracked_51302915.json` is reproduced by the replay recorder above.

  The armed tape cannot be extended from Base, because its pool state is the fork's. Its new rounds read only
  contracts or words the arming does not touch -- the market oracle's four feed immutables, three Chainlink
  proxies' `aggregator()`, the factory's slots 0 and 1, and the pool's four delayed-governance slots -- and Base
  has no log on any of those contracts in blocks 51313000-51313006 (the factory's two slots read the same at
  51312990, 51313004 and 51313010; the pool's four are zero at every block checked, and the fork schedules no
  ceremony), so their answers at 51313004 were taken from Base; the probe aggregate was re-keyed in place, with its recorded fork answer
  untouched, under the key that carries the explicit `gas`. It has no oracle-window round and no deadline round:
  `TestTrackerReplayArmed` tracks with the parity policy, whose `maxSnapshotAgeSec` of 0 declares no window. `fork_sequence_51330064.json` needed no edit
  either: an entity that publishes no `oracleAhead` is quoted at its snapshot answer alone.
- `fork_sequence_51330064.json` is `TestForkSequence`'s record: an anvil fork of Base at 51330060 armed with no
  spread staleness window (mined at 51330061-51330064), the refresh the sequence starts from, 90 seeded steps (routed
  and forced fills on both venues, gaps of 2-21 s and some of 60-459 s) each with the adapter's `amountOut` and
  `amountUnused` or its revert data, and the refresh after the last step. Written with:

  ```sh
  EVERLONG_FLAMM_RECORD=sequence EVERLONG_FLAMM_FORK_RPC=https://mainnet.base.org \
  EVERLONG_ADAPTER_OUT=<adapter PR>/out/EverlongFlammAdapter.sol/EverlongFlammAdapter.json \
    go test ./pkg/liquidity-source/everlong/flamm/ -run 'TestForkSequence$' -count=1 -v
  ```

What the tests assert on them:

- `TestTrackerReplay`: the refresh is attested (17 probes: previewSwap grids and band edges in both directions,
  previewLever, fundingCeiling), and its `flammReads` equal field for field the `live` state `CoreE2EGrid.t.sol`
  dumped at the same block (section 4), so the tracker reads exactly what the core replay was validated on. The
  simulator built from it sells 15000 sats for exactly 11301759 USDC.
- `TestTrackerAttestationBinds`: the same probe calls re-evaluated on a state with one priced word changed (hook
  x or reservation price, fee cap, band, physical, borrow shares, rateAtTarget, the cbBTC answer) fail the view
  attestation or a probe; and, from the other side, every view `attestViews` reproduces with one answer of the
  chain's changed -- the cross, the peg and its revert, the Router `positions` and the ledger identity beside them,
  `loanPosition`, `venuePosition`, each field of the account's `tryPosition`, the borrow rate and whether the IRM
  answered, and `totalAssets` both as an answer and as a revert class -- is reported under its own name.
- `TestTrackerDeadlines`, `TestDeadlineClocks`, `TestAttestClockPegRevertClass`: the forward round that binds the
  deadline words. The refresh sends one aggregate at the window's far end and nowhere else, a `lastSetTs` or
  `maxSpreadAge` that would keep the keeper's post alive through the window fails the refresh closed against the
  chain's own `spreadPpm` there, an `updatedAt` that moves the pool asset round's expiry inside the window fails it
  against `peekCross`, each token's `peekUsd` and the loan asset's `pegOk` are compared at the same clock, and a
  policy with `maxSnapshotAgeSec` 0 sends no round at all. `TestDeadlineClocks` pins which clocks are read at
  (the end alone when nothing flips inside the window; the flip second and the one after it when something does).
- `TestTrackerDecodesSpreadWord`: `FLAMMStore.lastLeverSpreadPpm` has no view, so its bits (160..191 of the slot
  that also holds `pendingControllerHook` and `hookSetExecutableAt`) are decoded from four non-zero values.
- `TestTrackerConfigGuards`: a tracker configured for another chain, or for another dex id, refuses the pool.
- `TestTrackerAttestationFailsClosed`, `TestTrackerReplayArmed`: through the whole refresh (`GetNewPoolStateAtBlock`),
  one recorded answer changed -- a previewSwap or previewLever word, a probe whose answer becomes a revert, a probe
  the port already refuses whose revert becomes another of the pool's own errors, a funding ceiling, or a view only
  the attestation binds (`totalAssets`, `positions`, `peekCross`) -- publishes
  `Attested` false with the failure, and the simulator refuses the entity. The armed tape attests with `previewLever`
  probes in both directions and quotes both leverage fills.
- `TestTrackerScheduledChange`, `TestConfigPolicy`: each of the four delayed changes -- a pending implementation
  (executable by anyone, for every pool at once), a pending hook set, a scheduled venue admission and a scheduled
  loan asset -- is stamped as `ScheduledChangeAt`, the earliest of them, and the simulator stops quoting
  `scheduledChangeLeadSec` before it; either word of a pair marks it pending, and an answer that drops a scheduled
  upgrade or a scheduled venue from one side only -- the views alone, or the storage word alone -- is reported as
  drift instead of publishing an attested entity with the guard off; an unset margin takes its default, an explicit zero turns it off, and
  the nested `gas` overrides reach the stamped policy.
- `TestTrackerDrift`, `TestTrackerFailsClosed`, `TestTrackerLayoutBindsRouterConfig`: wiring drift is reported. That
  covers the pair, the loan count, the implementation, the hook set, the router, each hook's bindings as its kind
  reads them (the genesis strategy, the leverage binding), the venue count, a venue id, a venue the registry does
  not accept, an aggregator, the storage layout, and a code override on any registered contract, the Morpho
  singleton or AdaptiveCurveIrm; a storage word that
  disagrees with the views -- one bit of any Router or FLAMMStore configuration word the port reads, flags
  included, or of the factory's beacon and pending-upgrade words -- publishes a
  refusing entity, and so does a required view the chain answers with a revert or undecodable data (drift `unreadable:
  ...` at the block it answered at); only a transport failure returns an error.
- `TestListerProfileChecks`: every check `listPool` and `readVenues` make on what the chain answered, one answer at a
  time. None of them is an error; the pool is simply not listed. The checks are:
  - the pair and both loan counts;
  - each binding and the hook set;
  - each hook's bindings as its kind reads them: the swap and leverage hooks' `POOL`, `LOAN_SCALE` and the leverage
    hook's `HOOK`, the genesis strategy, and the spread hook's `POOL`;
  - the venue count bounds;
  - a venue account the registry does not bind to the pool;
  - a venue market whose pair, IRM or oracle is not the pool's.
- `TestListerReplay`, `TestValidStatic`, `TestReadPlanErrorChain` check:
  - the listing's pinned identity (the venue set is published in `Extra`, not pinned);
  - the cursor: no relisting, a changed digest relists, and a pool the factory no longer owns is forgotten;
  - a contract whose runtime code differs from the registry's, or whose views revert or do not decode, keeps the pool
    unlisted without failing the run;
  - a decoder's own error (ErrInvalidProfile) stays in the error chain;
  - a registered pool the factory does not own (`isPool` false) is not listed and leaves the cursor;
  - `validVenues` refuses a venue set that is empty, too large, or on an unregistered account, IRM or oracle.
- `hook_registry_test.go`, the hook registry and per-pool dispatch:
  - `TestHookRegistry`: the registries hold exactly the c104 deployment record's hooks (kind, codehash, bound pool,
    immutables), shared contracts and financing account, looked up by chain and address; the record's `hookSetHash`
    is the hash of the c104 hook set.
  - `TestResolveHooks`: `resolveHooks` refuses each of these:
    - an unregistered hook in each role;
    - the Base hooks resolved on another chain;
    - a hook bound to another pool;
    - each hook in another role's slot;
    - a kind with no port;
    - each slot rule: slots 1-3 differing from the invariant hook, no swap hook, a loan-swap hook, and a leverage or
      spread hook on its own.

    It accepts a swap-only hook set.
  - `TestValidStaticRegistry`: `validStatic` and `validVenues` refuse another pool's wiring, a swapped pair, and a
    financing account bound to another pool or registered as another kind of contract.
  - `TestListerRegistry`: the lister refuses a pool on its first aggregate, with no further read, when any of these
    holds:
    - `hooks()` names an unregistered hook, a hook of the wrong kind, a swap hook in the spread slot, or a loan-swap
      hook;
    - `priceFeed()` is not registered;
    - `hooks()` does not answer.

    A listed pool the registry no longer accepts keeps its cursor entry while the factory owns it, so a hook set the
    registry accepts later relists it. An unknown factory, or an allow-list that excludes the pool, sends nothing.
  - `TestSimulatorHookRegistry`: a listing the registry does not accept, reads taken for another hook set, and a
    missing kind's reads are refused at construction. A state whose hook addresses, kind tags, emptied roles or
    per-kind states are not the listing's is refused on every quote.
  - `TestSwapOnlyPool`, `TestSimulatorSwapOnly`: a pool with no leverage and spread hooks goes through the lister,
    tracker and simulator. The recorded pool is replayed with both roles emptied, in `hooks()` and in the FLAMMStore
    words.
    - The listing sends no code read for the empty roles, and the refresh sends no read to them.
    - The refresh attests, with fewer probes (no `previewLever`), and its dependency set leaves out the spread hook.
    - Swap quotes equal the full pool's in every recorded scenario.
    - The leverage venue refuses with `LeverageDisabled`.
    - The state survives a msgpack round trip.
- `policy_binds_test.go`: one test per declared refusal of the integration layer, on a state that reaches it -- the
  envelope's unreadable venue (a market oracle that does not answer, an IRM inside the account's grace or
  quarantine), the leverage band margin up and down, the leverage accrual drift under a venue debt cap the fill
  only just fits, `PriceAgeMarginSec` bound by the *loan* asset's round, the spread's liveness and its age margin,
  the leverage venue's half of the oracle-window gate (the swap venue's is `TestSimulatorOracleWindow`, which
  quotes only venue 0 because the deployed pool has leverage paused), the gas terms of a clipped sell, a margin
  that exhausts the whole band, a SwapInfo quoted on another state, and the parameters `CalcAmountOut` refuses
  outright.
- `TestTrackerVenueSetReread`: a refresh with no venue set to start from reads `venueCount` from the Router itself
  before the view round, at the same block -- the path a pool whose curator added a venue heals through, run end to
  end on a fork by `TestForkVenuesAndFlows`.
- `TestTrackerDependencies`: the refresh publishes the resolved aggregators (not the proxies the configuration
  names), the hooks whose kind is a refresh trigger (the swap hook and the spread hook) and the Router, and neither
  Morpho nor the IRM; `SetDependenciesStored`
  round trips through the entity, an unchanged set stays registered, a rotated `aggregator()` clears the flag, and
  a resolve round the node does not answer keeps the previous set without failing the refresh.
- `TestTrackerOracleWindow`, `TestSimulatorOracleWindow`: the refresh reads each venue's market oracle at its own
  block with the timestamp moved to the end of the snapshot window and publishes the answer; at 51302915 that
  answer equals the snapshot's, and a withheld round moved into the window makes the simulator refuse
  (`ErrOracleDrift`) exactly the fills the move changes -- one wei and ten bps leave the 15000-sat sell at
  11301759, three percent (past the Router's 2% oracle band) refuses it, and so does an oracle that stops answering.
  A policy with `maxSnapshotAgeSec` 0 declares no window, so the round is not sent and nothing is published.
- `TestTrackerProbeEmptyRevert`, `TestRevertedClassifies`: a probe the aggregate answers with an empty revert is
  re-run on its own; only a JSON-RPC revert confirms it, while an answer or an out-of-gas fails the refresh closed,
  and the probe aggregate carries its explicit `gas`.
- `TestConfigThroughFactories`, `TestConfigFieldCount`: the configuration decodes through the registered
  `poollist`/`pooltrack` factories with the injected `DexID`, in the tag spelling and the Go field-name spelling,
  and `Config` stays within the sixteen JSON fields that keeps that matching case-insensitive.
- The simulator tests use the tracked entity's listing with the `core_e2e_*` states: every replayed previewSwap /
  previewLever row of every scenario equals the chosen venue's quote, and the simulator refuses a previewed fill only
  for its own spread policy on a leverage fill (`TestSimulatorPreviewGrid` fails on any other refusal). That test
  replays every third row by default -- each row is a full settlement, not a preview, and the three grids hold
  82,545 of them -- a sample of those under `-race`, and all 82,545 under `EVERLONG_FLAMM_FIXTURES=full`
  (20,015 identical fills, 60,532 matched reverts, and 1,998 rows in the two IRM-outage scenarios the envelope
  refuses whole). Every executed on-chain sequence replays through CalcAmountOut and UpdateBalance with identical
  amounts and post-states; plus purity, clone isolation and lineage (`TestPolicyBrokenSimulator`: a second
  simulator built from the same entity is the same state and its SwapInfo is adopted, while one from a state that
  has moved on, one from another state at the same block, and anything that is not a SwapInfo are refused),
  venue selection and the tie rule
  (`TestSimulatorPickVenue`), leverage gates (the inclusive spread deadline must quote), clock, margins and freshness,
  partial fills, the clipped sell's gas terms (`TestSimulatorClippedSellGas`), refusals (a donated venue position
  by default, `TestSimulatorDonation`), and the msgpack round trip.
- The core sequence fixtures chain at most four fills at one timestamp between refreshes. `TestSimulatorForkSequence`
  replays `fork_sequence_51330064.json` offline with no refresh after the first: 48 fills adopted over 3388 s (9
  after a gap of a minute or more), each quote at its mined timestamp with the adapter's amounts to the wei, 42
  mined reverts by class, and after the last step the state equal to the fork's refresh.
- `simulator_contract_test.go` (offline, over the `core_e2e_*` states): concurrent purity with every margin on,
  UpdateBalance adopting `SwapInfo.next` verbatim, deep clones, the result shape (output, unused input, gas and fee of
  every venue and direction), a monotone clock, the input contract, margins that only refuse (a margined quote is the
  unmargined one), UpdateBalance across a msgpack hop, and how far one unseen cbBTC round moves quotes.
  `msgpack_external_test.go` round-trips through the production encoder of `pkg/msgpack` from an external test
  package (`export_test.go` exposes what it needs).

| file | sha256 (stored) | sha256 (uncompressed) |
| --- | --- | --- |
| `tracker_rpc_51302915.json.gz` | `1622fd4df79c7e027fe93bea579ce47bee641ad0aa54159ed4d51a36d159f8ad` | `6194b03075b0b6648997f48d7b5b930025c7005adb55b4b82faf0d6bc27cd6de` |
| `tracked_51302915.json` | `4281d97c4c7af0c159265b4ab4c0d35644e06f711eda5b3e89d065938d5e4cb8` | |
| `tracker_rpc_armed_51313004.json.gz` | `42413a226967a887e070b8fe42cc5ee5acad8db84f0c3231495fdfb690e1c1cf` | `dd9be3feb87b4bd9d650083f209da6c6fb71828520540a737aa53296391e5eb1` |
| `fork_sequence_51330064.json` | `9b1ef76bc47701646c7f2721cf573ede806ff48a9260c883e03699d0614621f5` | |

Network tests (not run in CI, skipped without their environment; see "Environment variables"). Every fork test needs
`EVERLONG_FLAMM_FORK_RPC` and `EVERLONG_ADAPTER_OUT` and takes `EVERLONG_FLAMM_FORK_BLOCK` over its default block; the
live tests need `BASE_RPC_URL`. The parity tests refresh with every margin set to zero (`parityConfig`), so band edges
and clock deadlines are compared, not refused. Every fork environment freezes each venue's Morpho market oracle at
the answer it gives at the fork block (`fork_harness_test.go` `pinOracles`, a constant-`price()` stub carrying that
same answer): its BTC/USD feed is a Chainlink SVR DualAggregator, and a fork mines its fills at the timestamps the
simulator quoted them at, so an upstream reveal could otherwise land inside a sequence and move `oraclePrice` with
no transaction in between.

- `live_test.go` (`BASE_RPC_URL`): `TestLiveHead` runs lister -> tracker -> simulator at the head and compares the
  simulator's swap and leverage venues with the pool's previews over a 24-point grid per direction at the tracked
  block; `TestLiveRealSell` refreshes at 51302915 and sells 15000 sats for 11301759 USDC; `TestLiveBatchLimit`
  refreshes against the endpoint's 10-call batch cap; `TestLiveBlocks` compares both venues with the pool's previews
  at blocks 51298500, 51302915, 51310000, 51318000, 51326000 and 51333700. The comparison counts five outcomes
  apart (`liveTally`): identical fills, sizes both sides refuse with the same error, sizes the chain's *preview*
  accepts while the port refuses at settlement, and sizes the port refuses by a declared policy of its own (the
  lapsed spread post), counted by whether the chain reverts too or fills. Every one of the third
  kind is confirmed by an `eth_call` of the pool's own `swap` / `leverUp` / `leverDown` at the same block from an
  account the call's overrides fund and approve, which has to revert too -- `TestLiveSettlement` pins that the
  confirmation can tell the two apart, by settling the one real swap (15,000 sats for 11,301,759 USDC at 51302915)
  and a size the band refuses through the same path.
- `fork_multipool_test.go` (default block 51313000; also needs `EVERLONG_FLAMM_C104_OUT`): `TestForkMultiPool` checks
  several pools on one fork.
  - **Setup.** Two more pools, B and C, are created through the live `FLAMMFactory.createPool`, each on a Morpho
    market of its own. Each gets its own EverlongHook, EverlongLeverageHook and LeverageSpreadHook, built from the
    c104 artifacts and linked to the deployed libraries. The creation code and constructor arguments are first shown
    to be exactly what the c104 deployment sent.
  - **Registration.** The new hooks and accounts are registered in a test-only extension of the registries. The real
    lister lists exactly the pools whose hooks are all registered, of their slot's kind and bound to them. It refuses
    a registry line of the wrong kind, pool, codehash, genesis strategy or loan scale, and an account that is
    unregistered or bound to another pool.
  - **Parity.** Every listed pool attests under the parity and production policies. Its quotes equal its own
    previews and the adapter's fills on both venues, including a mined sequence interleaved across c104 and B that
    ends with each state equal to a refresh.
  - **Dispatch.** B's and C's quotes differ, and B quotes exactly as C once given C's hook states. A keeper retune of
    B moves B and leaves c104 untouched.
  - **Fail closed.** Unregistering C's hooks makes C fail closed. Static data naming another pool's hook is refused,
    and so is a hook slot rewritten on chain to another pool's hook.
- `fork_parity_test.go` (default block 51313000): `TestForkParity` runs an anvil fork with the adapter
  runtime. Single fills compare the quote with `executeEverlongFlamm` eth_calls (adapter code and balance as state
  overrides) on the pool as deployed, on the armed leverage venue (adapter venue 1) and under a notional cap that
  clips sells; sequential fills are mined at the timestamp the simulator quoted, the simulator advanced only by
  UpdateBalance, and a fresh refresh after each sequence must equal its state. Every mined fill's receipt gas must be
  within the simulator's estimate.
- `fork_gas_test.go`: `TestForkGas` measures the receipt gas behind the defaults in `constant.go`. The leverage venue is
  armed and a seeded walk of mined fills moves the pool's leverage frame between the curve's half-law and Hermite
  pieces; at every other step, the smallest and largest fill of each venue and direction, and for the leverage venue
  the smallest and largest fill that starts and ends on the Hermite piece (every anchor solve bisects), are mined
  from an anvil snapshot. Amounts must match and every receipt must be within the estimate. Run at blocks 51313000
  and 51330060 (default 51313000).
  `TestForkGasSwapSellClip` (default block 51333700) measures the clipped swap sell's two input-dependent terms: after
  arming leverage and moving the book, a notional cap, a venue debt cap, Morpho liquidity borrowed away, and funding
  ceilings bound by a lowered Router pin each clip sells of 2^9 to 2^72 sats, mined from a snapshot (645 fills, 1-4
  funding passes, 24-256 cap-bisection solves). Amounts must match, every receipt must be within the estimate, and the
  measured slopes and intercept plus 25% must be within `defaultGasSwapSellCapEval`, `defaultGasSwapSellPass` and
  `defaultGasSwapSell`:

  ```sh
  EVERLONG_FLAMM_FORK_RPC=https://mainnet.base.org \
  EVERLONG_ADAPTER_OUT=<adapter PR>/out/EverlongFlammAdapter.sol/EverlongFlammAdapter.json \
    go test ./pkg/liquidity-source/everlong/flamm/ -run 'TestForkGasSwapSellClip$' -count=1 -v
  ```
- `fork_edges_test.go` (default block 51330060): `TestForkEdges` compares unit windows around every edge the port finds
  (band edges, the dust floor, the partial-fill boundary, the leverage curve's edges) on the pool as deployed, armed at
  four spreads and staleness windows, with the clock moved to and past the feed and spread deadlines, and under three
  notional caps; `TestForkSequence` mines 90 seeded routed and forced fills with irregular gaps adopted only through
  UpdateBalance, then requires a refresh equal to the state (`EVERLONG_FLAMM_RECORD=sequence` writes
  `fork_sequence_<block>.json`); `TestForkGasGrid` mines a size grid per venue from snapshots against the gas estimate.
  A spread refusal is a mismatch whenever the pool's spread hook still answers at the quote's clock.
- `fork_tracker_test.go` (default block 51330060): `TestForkTrackerTamper` changes, through a proxy, every word of every
  view answer and every storage word of an armed refresh, and requires any refresh that still attests with a changed
  published entity to quote a grid on both venues exactly as the adapter fills. Every scheduled tamper must have
  edited an answer: each hook counts the responses it rewrote and the test fails if a tamper applied to none, so a
  tamper that silently stopped matching is a failure rather than a "changed nothing" row; `TestForkTrackerPinning` mines a swap in the middle
  of a refresh and requires every later read pinned to the first round's block; `TestForkTrackerDrift` changes the
  wiring on the fork; `TestForkDonation` donates one unit of collateral and of supply on behalf of the venue account:
  the refresh attests, the production policy refuses the pool, and with `QuoteDonatedVenues` every swap quote equals
  the adapter's fill.
- `fork_scenarios_test.go` (default block 51333700; `EVERLONG_FLAMM_FORK_SCENARIOS` runs a subset): `TestForkScenarios`
  applies, from one armed snapshot, each of 30 changes a curator, guardian, keeper, the protocol Safe or a third party
  can make (pauses, features, venue flags and caps, band, fee bounds, dials, `maintainCollateral` after the ltv dial is
  lowered by its largest step, poke, recenter, skim, donations, a repay on behalf, token transfers, Morpho liquidity
  squeezes), refreshes, compares edges, drift clocks and a seeded sequence with clone and msgpack hops; a scenario
  that cannot be applied fails the test, and a donation the production policy refuses is compared with
  `QuoteDonatedVenues`. `TestForkGasHunt` and `TestForkGasLarge` hunt for receipts above the gas estimate.
- `fork_venues_test.go` (default block 51336630; `EVERLONG_FLAMM_FORK_SCENARIOS` runs a subset,
  `EVERLONG_FLAMM_FORK_PHASE2=1` compares again after each scenario's sequence): `TestForkVenuesAndFlows` admits a
  second Router venue (relisting, drift of the old listing, priority orders, a thin market, caps, refinance and supply
  migration), LP flows, the emergency lane and maintenance between refreshes, failing on a scenario that cannot be
  applied; `TestForkSameBlock` mines several fills in one block (ten rounds) adopted only through UpdateBalance.
- Recorders: `EVERLONG_FLAMM_RECORD` = `1` or `missing` (with `BASE_RPC_URL`) or `tracked` for `TestTrackerReplay`,
  `armed` for `TestForkRecordArmedTape`, `sequence` for `TestForkSequence` (section 5 above).
