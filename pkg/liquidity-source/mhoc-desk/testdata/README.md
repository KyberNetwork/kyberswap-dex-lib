# mhoc-desk test vectors

`vectors.json.gz` is produced by the pool's own bytecode, not by a re-implementation. The generator deploys
`PropAMM.sol` (v0.5) on a local anvil chain in four pool shapes (WETH/USDC and cbBTC/USDC with their `PRICE_SCALE`
and `K_SCALE`, an 18/18 pair, a 6-decimal base against an 18-decimal quote), sets random but seeded limits, floors,
budgets and price cards with `kBid != kAsk` (including cards that are stale by time, dead by block number or pinned
to one block, paused, oracle-halted and seq-reset pools, and a per-minute cap of 0, which means no trading), and
records per scenario:

- the state a tracker reads at the end of a block, through the pool's public views, including `counters()`;
- `quoteDetailed` for about 30 sizes chosen to hit every branch and edge, in that block (`quotes`) and in the
  next, empty block (`nextQuotes`), where the per-block counter has restarted and a card pinned to the previous
  block no longer trades;
- a sequence of real transactions mined in later blocks, several per block: swaps with each one's fill or revert,
  and price updates with the state word the pool stored (newer cards, cards that expire a few blocks later, the
  cancel card). Fills include second and later swaps on a card (the size slippage is charged on the cumulative
  fill), a newer card in the same block as earlier fills (the per-block counter carries over to it), and sequences
  that cross into the next calendar minute.

The simulator must reproduce every quote from the snapshot alone, and every step of the sequence by applying its
own `UpdateBalance` after each fill (and the stored state word after each update), to the last raw unit. The
generator is in the desk's repository (`trading/onchain/integrations/vectors/gen_vectors_anvil.py`). The file's
`meta.contractSha256` names the exact contract source it was built from.

`anvil_live_test.go` holds two live tests, skipped by default: `TestAnvilLive` (driven by the desk's
`anvil_live.py` against a local chain, where the pool is found through a registry) and `TestBaseMainnetLive` (set
`MHOCDESK_BASE_RPC` and `MHOCDESK_BASE_POOL`, optionally `MHOCDESK_BASE_REGISTRY`, to read a deployed pool on Base,
read-only).
