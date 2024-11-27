# kyberswap-dex-lib

## Marshal/unmarshal pool simulator

When implementing a new pool simulator, to make it marshal-able and unmarshal-able, we have to notice the following:

* rerun `go generate ./...` to register the new pool simulator struct
    * Because we might marshal/unmarshal a pool simulator under the `IPoolSimulator` interface. We have to register the underlying struct so we can unmarshal it as `IPoolSimulator`.

* pointer aliases
    * If the pool simulator struct contains pointer aliases, we must use `msgpack:"-"` tag to ignore the aliases and set them inside the `AfterMsgpackUnmarshal()` method. For an example:
        ```
        type PoolSimulator struct {
            vault *Vault
            vaultUtils *VaultUtils
        }
        
        type VaultUtils struct {
            vault *Vault `msgpack:"-"`
        }

        func (p *PoolSimulator) AfterMsgpackUnmarshal() error {
            if p.vaultUtils != nil {
                p.vaultUtils.vault = p.vault
            }
            return nil
        }
        ```
