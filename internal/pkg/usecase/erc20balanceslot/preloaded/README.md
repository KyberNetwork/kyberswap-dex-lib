# Preloaded ERC20 balance slots

A peloaded is a `gob`-encoded `entity.TokenBalanceSlots` which is used to embed in router-service. These files in this directory are created using `convert-to-preloaded` command. The command reads ERC20 balance slots from Redis then convert to a preloaded. For an example:

```
go run ./cmd/erc20probebalanceslot/main.go \
    --config internal/pkg/config/files/dev/avalanche.yaml \
    convert-to-preloaded \
    --output internal/pkg/usecase/erc20balanceslot/preloaded/avalanche 
```