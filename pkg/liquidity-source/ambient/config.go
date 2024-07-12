package ambient

import (
	"github.com/KyberNetwork/blockchain-toolkit/time/durationjson"
)

type Config struct {
	DexID                  string                `json:"dexID"`
	SubgraphURL            string                `json:"subgraphUrl"`
	SubgraphRequestTimeout durationjson.Duration `json:"subgraphRequestTimeout"`

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
}
