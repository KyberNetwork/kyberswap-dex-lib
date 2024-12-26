package ambient

import (
	"fmt"
	"math/big"
	"net/http"

	"github.com/KyberNetwork/blockchain-toolkit/time/durationjson"
	"github.com/ethereum/go-ethereum/common"
)

type Config struct {
	DexID                  string                `json:"dexID"`
	SubgraphAPI            string                `json:"subgraphAPI"`
	SubgraphHeaders        http.Header           `json:"subgraphHeaders"`
	SubgraphRequestTimeout durationjson.Duration `json:"subgraphRequestTimeout"`
	SubgraphLimit          uint64                `json:"subgraphLimit"`

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
	if common.HexToAddress(c.NativeTokenAddress) == (common.Address{}) {
		return fmt.Errorf("expected NativeTokenAddress")
	}
	if common.HexToAddress(c.SwapDexContractAddress) == (common.Address{}) {
		return fmt.Errorf("expected SwapDexContractAddress")
	}
	if common.HexToAddress(c.QueryContractAddress) == (common.Address{}) {
		return fmt.Errorf("expected QueryContractAddress")
	}
	if common.HexToAddress(c.MulticallContractAddress) == (common.Address{}) {
		return fmt.Errorf("expected MulticallContractAddress")
	}
	if c.PoolIdx == nil {
		return fmt.Errorf("expected PoolIdx")
	}
	return nil
}
