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

	// Ambient uses the zero address for the native token inside pair metadata.
	// kyberswap-dex-lib stores the wrapped native address in pool tokens instead.
	NativeTokenAddress string `json:"nativeTokenAddress"`

	// The singleton CrocSwapDex contract that executes swaps and stores pair state.
	SwapDexContractAddress string `json:"swapDexContractAddress"`

	// The multicall contract used to read ERC20 and native balances.
	MulticallContractAddress string `json:"multicallContractAddress"`

	// The pool discriminator for a token pair. We currently integrate one poolIdx
	// per source configuration.
	PoolIdx *big.Int `json:"poolIdx"`
}

func (c *Config) Validate() error {
	if c.DexID == "" {
		return fmt.Errorf("expected DexID")
	}
	if c.NativeTokenAddress == valueobject.ZeroAddress {
		return fmt.Errorf("expected NativeTokenAddress")
	}
	if c.SwapDexContractAddress == valueobject.ZeroAddress {
		return fmt.Errorf("expected SwapDexContractAddress")
	}
	if c.MulticallContractAddress == valueobject.ZeroAddress {
		return fmt.Errorf("expected MulticallContractAddress")
	}
	if c.PoolIdx == nil {
		return fmt.Errorf("expected PoolIdx")
	}
	return nil
}
