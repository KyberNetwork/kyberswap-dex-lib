package ambient

import (
	"fmt"
	"math/big"
	"net/http"

	"github.com/KyberNetwork/blockchain-toolkit/time/durationjson"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type Config struct {
	DexID           string                `json:"dexID"`
	SubgraphAPI     string                `json:"subgraphAPI"`
	SubgraphHeaders http.Header           `json:"subgraphHeaders"`
	SubgraphTimeout durationjson.Duration `json:"subgraphTimeout"`
	SubgraphLimit   uint64                `json:"subgraphLimit"`

	// Ambient doesn't use ERC20 wrapped native token when swapping with native token, it uses 0x0 address instead.
	// kyberswap-dex-lib uses ERC20 wrapped native token to store pool's tokens that are native.
	// So we have to fill in the ERC20 wrapped native address when fetching Ambient pools.
	NativeTokenAddress string `json:"nativeTokenAddress"`

	// The deployed address of CrocSwapDex.sol contract. kyberswap-dex-lib uses it to get pool's reserves.
	SwapDexContractAddress string `json:"swapDexContractAddress"`

	// The deployed address of CrocQuery.sol contract. kyberswap-dex-lib uses it to get pool's liquidity and price.
	QueryContractAddress string `json:"queryContractAddress"`

	// The deployed address of multicall contract. kyberswap-dex-lib uses its `getEthBalance` function to get native balance of pools.
	MulticallContractAddress string `json:"multicallContractAddress"`

	// The discriminator for pools of the same token pair. We assume that there is at most 1 pool for a token pair.
	PoolIdx *big.Int `json:"poolIdx"`
}

func (c *Config) Validate() error {
	if c.NativeTokenAddress == valueobject.ZeroAddress {
		return fmt.Errorf("expected NativeTokenAddress")
	}
	if c.SwapDexContractAddress == valueobject.ZeroAddress {
		return fmt.Errorf("expected SwapDexContractAddress")
	}
	if c.QueryContractAddress == valueobject.ZeroAddress {
		return fmt.Errorf("expected QueryContractAddress")
	}
	if c.MulticallContractAddress == valueobject.ZeroAddress {
		return fmt.Errorf("expected MulticallContractAddress")
	}
	if c.PoolIdx == nil {
		return fmt.Errorf("expected PoolIdx")
	}
	return nil
}
