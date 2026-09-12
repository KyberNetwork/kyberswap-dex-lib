# LunarBase snapshot modes

The default tracker reads a complete snapshot at a selected block hash and
checks that block's canonical membership after the read. It reuses a cached
snapshot only after a fresh latest-header observation verifies its identity.
This mode retains the existing MessagePack v2 format.

`singleCallSnapshot: true` enables one `tryBlockAndAggregate` call at `latest`
for ordinary `GetNewPoolState` refreshes. All ten getters execute in one EVM
state. The tracker retains strict result validation, pricing-model selection,
token identity checks, and the minimum block required by the previous entity,
relevant logs, and supplied header heights. It does not merge partial events.
Discovery and `GetNewPoolStateWithOverrides` retain the strict path.

The single-call mode records the aggregate's block number and a separate
`SnapshotComplete` marker, with no block hash. It does not detect a branch
replacement during the call. Later replay at the same number can select a
different branch. Neither mode guarantees that a future transaction will
execute against the quoted state.

## Reader upgrade before enabling

The completeness marker preserves known-zero freshness and model metadata;
it must not be discarded or replaced with a fabricated block hash.
Simulators carrying this marker use MessagePack v3. Other simulators continue
to use v2, so upgrading readers with the option disabled does not switch
default writers to v3.

Upgrade every consumer of LunarBase entity JSON and simulator MessagePack
before enabling `singleCallSnapshot`. The v2 decoder rejects a v3 entry, and
the enclosing pool-map decoder then rejects the entire map. Older generic
and JSON decoders may ignore new fields instead, which can misinterpret
known-zero metadata. Merely bumping the version does not make such readers
safe. Use a coordinated cutover or isolated cache namespace where necessary.

Disabling the option stops newly refreshed entities from using the marker;
it does not rewrite v3 snapshots already in storage. Do not roll readers back
until those entries have been refreshed, replaced, or isolated. Re-encoding
legacy ambiguous state must preserve its per-pool RPC-refresh requirement.
