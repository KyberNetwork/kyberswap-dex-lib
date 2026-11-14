# Zero-seed launch pools

This adapter targets the packed-metadata/tick-rate revision of
[Ekubo's zero-seed launch extension](https://github.com/EkuboProtocol/evm-contracts/blob/1d74ed414a5a91100440b128ef07fbe15acf37a5/docs/zero-seed-launch.md).
The reviewed source is `1d74ed414a5a91100440b128ef07fbe15acf37a5`; canonical main
`fac9631e8fad77271f241733177060fc98be3545` has the same tree.

Configure `chainId: 4663` and `zeroSeedLaunch:
0x517621Ea1bf51F31826819b118fa4dfEb4D198D2`. The final ABI is accepted only for
that pair. Core is `0x00000000000014aA86C5d3c41765bb24e11bd701`; index from block
106030313. Historical launch addresses are not aliases for this implementation.

Discovery includes the extension in the existing Core initialization subgraph
query. The tracker reads Core state/ticks and `getLaunch(poolId)` at one header's
block number, stores that header's timestamp, and verifies its hash again after
the reads. Launch refreshes use RPC even when Core logs are supplied: fee decay
must not depend on the last swap or wall-clock time.

The creator fee is rounded up on actual output for exact input; exact output
grosses up actual input. The returned fee asset is the calculated token. Core's
pool fee is zero. The swap metadata requests `Core.forward(extension,
abi.encode(poolKey, swapParameters))`, skip-ahead 1, and the installed range limit.
Incomplete fills retain their unconsumed specified amount. Native currency uses
the existing wrapped-native discovery and native execution conventions.

## Checks and delivery boundary

`CI=true go test ./pkg/liquidity-source/ekubo/v3/...` covers all twelve sequential
final-source trade vectors (both orders, 18/18 and 18/6), five independent
fee-curve digests, maximum-fee/overflow/partial-fill boundaries, discovery,
serialization, and a pinned/reorg RPC fixture. Fixtures are local contract
execution, not live-chain observations. The Core compact-price rounding fix is
shared with #1843.

The 1,900,000 execution-gas estimate is a conservative frozen-source bound;
provider executor calibration remains necessary. This library supplies forward
metadata, not the deployed Kyber executor. Provider routing-service configuration,
subgraph readiness, executor compatibility, release, and a pinned Robinhood fork
comparison remain activation requirements. No live-routing availability is
established by this source change.
