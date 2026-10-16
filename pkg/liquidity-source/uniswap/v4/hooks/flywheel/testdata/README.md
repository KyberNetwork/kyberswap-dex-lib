# Recorded execution fixtures

`fork-fixtures.json` records actual local-fork UniversalRouter executions of the
deployed October 13 Flywheel contracts, alongside canonical V4Quoter outputs,
pool snapshots, fee events and post-trade snapshots. `mainnetTransactions` is zero.
Transaction hashes inside the fixtures belong to the local fork, not mainnet.

The block number/hash and verified runtime hashes are recorded in the fixture.
USDG uses 6 decimals and a token0 launch; WETH and CASHCAT use 18 decimals and
token1 launches. `first-buy` starts at a one-sided position boundary. Graduation
occurs inside `single-buy-graduation` with the same pool ID before and after.

To regenerate, install Node.js 22+, `ethers` v6, and Foundry's `anvil`. From this
directory, run `node generate.cjs`. The script pins a recent mainnet block,
verifies deployed code hashes, starts a local fork, impersonates a funded holder
**only on that local fork**, and records ten trades. It never loads a private key.
`ANVIL` can name the anvil executable; `ANVIL_PORT` defaults to 18647.
`FORK_BLOCK` can pin an explicit block, but the public endpoint may not retain old
state. Live holder balances must still cover the tiny USDG/CASHCAT test buys.

The upstream proxy only allows read methods, caps new reads at 2,000, and caches
them in the ignored `.fork-proof/` directory. It uses the public Robinhood RPC;
there is no Alchemy fallback. Native WETH funding and new token deployment occur
on the local fork. No contracts, approvals, balances or transactions on mainnet
are changed.

The first successful capture used 288 uncached public reads. Ordinary Go tests
load the recorded JSON and do not need Node, Anvil, credentials, or a network.
This directory is a separate Go module so the fixtures are excluded from the
published dependency module zip.
