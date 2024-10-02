# Preloaded ERC20 balance slots

First we need to probe balance slots of ERC20 tokens in our Redis database (per chain and environment). The database dump can be downloaded from our Google Cloud storage bucket (please contact SRE team for the bucket link). We use the `./cmd/erc20probebalanceslot` command to probe balance slots. See [README.md](../../../../../cmd/erc20probebalanceslot/README.md) for details of the command. For an example:

```
go run ./cmd/erc20probebalanceslot/main.go \
    --config internal/pkg/config/files/dev/avalanche.yaml \
    probe-balance-slot \
    --jsonrpcurl-override "<RPC endpoint>" \
    --wallet "<wallet corresponding to the probed balance slot>"
```

Then we make the preloaded using the same command. A peloaded is a `gob`-encoded `entity.TokenBalanceSlots` which is used to embed in router-service. These files in this directory are created using `convert-to-preloaded` command. The command reads ERC20 balance slots from Redis then convert to a preloaded. For an example:

```
go run ./cmd/erc20probebalanceslot/main.go \
    --config internal/pkg/config/files/dev/avalanche.yaml \
    convert-to-preloaded \
    --output internal/pkg/usecase/erc20balanceslot/preloaded/avalanche 
```