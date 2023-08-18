# ERC20 balance slot commands

This executable contains the following commands 
* find the storage slot where the ERC20 token balance of a wallet is stored
* convert balance slots to a serialized format for embedding in router-service

By default, all commands require a configuration file which is the same as what router-service uses.

## Options

| Option name | Required | Description                                     |
| ----------- | -------- | ----------------------------------------------- |
| `--config`  | yes      | The configuration file which router-server uses |

## Finding balance slots command

The command reads all entries from Redis hash key `<prefix>:tokens` then find the storage slot where each ERC20 token balance of the specified wallet is stored.

### Options

| Option name                | Required | Description                                                                                   |
| -------------------------- | -------- | --------------------------------------------------------------------------------------------- |
| `--jsonrpcurl-override`    | no       | If set, use this URL instead of common.rpc in the configuration file                          |
| `--retry-not-found-tokens` | no       | If set, retry probing tokens that its balance slot is failed to be found                      |
| `--skip-existing-tokens`   | no       | If set, don't probe tokens that already exist in Redis (whether balance slot is found or not) |
| `--tokens`                 | no       | If any, use these tokens instead of loading from Redis                                        |
| `--wallet`                 | no       | The wallet address to be probed its balance slot. If not set, a randomized address is used    |

## Convert balance slots to a serialized format for embedding in router-service

Each balance slot probing takes time. To save time while running router-service, we can run the finding balance slots command above beforehand then make router-service use this pre-calculated balance slots. Of course router-service will find balance slot for new token if needed.

To make router-service use pre-calculated balance slots, we need to embed balance slots to router-service executable. This command reads all balance slots from Redis then convert to a serialized format for embedding into router-service.

### Options

| Option name | Required | Description     |
| ----------- | -------- | --------------- |
| `--output`  | yes      | The output file |

### Embed into router-service

To embed a pre-calculated balance slots into router-service, we first
* copy the output file above to `internal/pkg/usecase/erc20balanceslot/preloaded`
* add a corresponding entry to the `preloadedByPrefix` map in `internal/pkg/usecase/erc20balanceslot/preloaded.go`, for an example
```
//go:embed preloaded/newchain
var newchain []byte

// ERC20 balance slots calculated beforehand. This make bootstrapping router-service more convinent.
var preloadedByPrefix = map[string][]byte{
	"avalanche": avalanche,
	"ethereum":  ethereum,
    "newchain":  newchain,
}
```