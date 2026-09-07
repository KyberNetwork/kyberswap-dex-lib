# Spire proprietary AMM (`spire-prop`)

Spire combines maker books into a compressed on-chain curve. This integration supports exact-input swaps between configured base ERC20s and a Spire entrypoint's quote token. WETH–USDC on Base (chain ID 8453) is the deployment verified below. It has no RFQ HTTP dependency, native ETH support, partial fills, or separate taker fee.

Protocol documentation: [overview](https://docs.baibai.cx/takers/overview), [quoting and swapping](https://docs.baibai.cx/takers/quoting-and-swapping), [deployments](https://docs.baibai.cx/takers/deployments).

| Contract | Base mainnet |
|---|---|
| Entrypoint / approval target | [0x98c1D9E102Eb2806D902b13186BDc7892aC4fFBa](https://basescan.org/address/0x98c1D9E102Eb2806D902b13186BDc7892aC4fFBa#code) |
| Curve book | [0x604d9b9eB1e1571C78661a6C1088427EC9c8c6E5](https://basescan.org/address/0x604d9b9eB1e1571C78661a6C1088427EC9c8c6E5#code) |
| Shared custodian | [0xAaC48FEB93c5C97E0fb3c7C57E1633922A4ACDa3](https://basescan.org/address/0xAaC48FEB93c5C97E0fb3c7C57E1633922A4ACDa3#code) |

Example discovery configuration:

```json
{
  "dexID": "spire-prop",
  "entrypoint": "0x98c1d9e102eb2806d902b13186bdc7892ac4ffba",
  "bases": ["0x4200000000000000000000000000000000000006"]
}
```

Add another listed base to `bases` to discover another pair; the existing simulator and adapter do not require pair-specific code. One entrypoint has one quote token; a different quote token requires a separate deployment/source configuration. On-chain `qUnit`, `cUnit` and midpoint values carry token scaling, so the simulator does not assume 18-decimal WETH or 6-decimal USDC. A regression test exercises an eight-decimal base and prevents two bases from spending the same shared quote inventory. Fee-on-transfer and rebasing tokens are unsupported.

There is no enumerable factory. Discovery uses configured base tokens and reads the entrypoint's immutable curve, custodian and quote token. The pool identifier is the low 20 bytes of `keccak256(entrypoint || base)` with both addresses packed as 20 bytes. This identifier is not a deployed contract. Execution uses `meta.entrypoint` and `meta.base`; balances are held in the custodian. Discovery emits zero reserve placeholders; tracking supplies `availableLiquidity`, excluding pending claims.

Tracking reads a block header, pins both multicalls to its hash, and checks the canonical hash again before returning. It verifies contract wiring and stores the actual block number in pool state and metadata. Routing should enable `FactoryOpts.StaleCheck`: expiry is checked again on every quote. Historical fixture replay leaves that option disabled but still validates expiry at the snapshot timestamp. Expiry is strict `timestamp > min(validUntil, lastUpdateAt + ttl)`; `ttl = 0` disables only the second bound. Refresh immediately before routing because fills, curve replacement and expiry can invalidate a snapshot.

Pricing ports the curve's checked uint256 operations, including operation order. Signed spread adjusts the midpoint. Cumulative quote impact is linearly interpolated with ceiling rounding. Buys walk ask segments from the consumed cursor, round segment costs upward, and use full-width floor multiplication/division for the last partial segment. Sells subtract the change in cumulative bid impact from the floor midpoint value. Depth limits and available custody are checked before returning a quote. Spread and curve impact are already in the output; the additional fee is zero.

`UpdateBalance` consumes returned swap information, advances the appropriate cursor/fill sequence, and adjusts reserves. Clones own their mutable values. Shared route inventory is keyed by custodian and token so two configured bases cannot independently spend the same quote balance. Gas uses a conservative 200,000 baseline plus 5,000 per knot on the larger side; fork measurements for this deployment were below 174,000 for the adapter call, excluding enclosing router overhead.

The execution adapter receives `abi.encode(meta.entrypoint, meta.base)`. The enclosing router must enforce final minimum output and deadline; the entrypoint has no caller deadline argument.

Validation:

```sh
go test ./pkg/liquidity-source/spire-prop ./pkg/msgpack/... ./pkg/pooltypes
go test -race ./pkg/liquidity-source/spire-prop
SPIRE_RPC_URL=<Base RPC> go test -tags integration ./pkg/liquidity-source/spire-prop -v
```

The committed fixture records 34 comparisons at [Base block 50979793](https://basescan.org/block/50979793): 17 executable results match the deployed curve exactly, and 17 are rejected for zero output or insufficient available custody. Additional tests cover signed spread, multi-knot integer rounding, consumed cursors, clone/purity, shared inventory, expiry, malformed state and overflow. The explicit integration test also exercises discovery, metadata replay, tracking and same-block contract comparisons. `SPIRE_FIXTURE_PATH` optionally writes a refreshed fixture.
