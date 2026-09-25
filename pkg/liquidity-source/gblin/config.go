package gblin

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type Config struct {
	DexID   string              `json:"dexID"`
	ChainID valueobject.ChainID `json:"chainID"`
	// Vault is the GBLIN vault: it is both the ERC-20 share token and the contract that mints it.
	Vault string `json:"vault"`
	// Lens is the read-only GBLINLens the vault's fee, accrual and sequencer settings are read from.
	Lens string `json:"lens"`
	// MulticallAddress is Multicall3, used to read the vault's native ETH balance and the block
	// timestamp in the same aggregate as the rest of the state.
	MulticallAddress string `json:"multicallAddress"`
}
