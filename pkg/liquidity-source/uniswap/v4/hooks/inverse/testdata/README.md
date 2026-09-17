# Execution fixtures

`solidity.json` contains 232 local EVM executions: 178 successful trades and 54
reverts. Every successful case includes the full before/after snapshot. Coverage:

- Both token currency orderings; buys, prepaid sells and postpaid sells.
- LP fee 3000; directional protocol fees 0, 333, 666, 1000 and 998..1000 pips.
- Sequential trades, bitmap-word crossings and large rebase factors.
- Zero/dust/oversized input, reserve/index boundaries, exhausted rounding buffer,
  unavailable protocol collection and native rounding failures.

Generator: `test/RoutedKyberVectors.t.sol` and
`integrations/kyber/export_vectors.py` in the candidate source reference branch:
https://github.com/calmdentist/inversecoin/tree/ec44bc670be90057188fcd6d02f14848e6010163

Source identity is independently checked against `candidate-manifest.json` before
export. That file pins the three production sources, compiler settings, dependency
lock and creation/runtime artifacts; hashes with immutable placeholders are NOT
deployed bytecode hashes. Foundry 1.4.3, solc 0.8.26, Cancun, optimizer one run.

From that source checkout, install its pinned dependencies and run:

```
bash scripts/bootstrap.sh
python3 integrations/kyber/export_vectors.py --output \
  /path/to/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/inverse/testdata/solidity.json
```

No RPC, funded wallet or mainnet transaction is used. Local mock quote token and
fee controller reproduce the exact-transfer WETH / permissionless-controller
interfaces; live candidate tracker/adapter fixtures remain an activation gate.
