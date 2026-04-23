package ambient

import (
	"math/big"

	"github.com/KyberNetwork/blockchain-toolkit/time/durationjson"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type Config struct {
	DexId   valueobject.Exchange `json:"dexId"`
	ChainId valueobject.ChainID  `json:"chainId"`

	HTTPConfig     HTTPConfig `json:"httpConfig"`
	IndexerChainId string     `json:"indexerChainId"`

	SwapDex    string `json:"swapDex"`
	Multicall3 string `json:"multicall3"`

	PoolIdx *big.Int `json:"poolIdx"`

	// TickRange limits the number of ticks fetched around the current price.
	// 0 means fetch all ticks (full int24 range). A positive value N fetches
	// ticks in [currentTick-N, currentTick+N], reducing cold-load RPC calls
	// at the cost of rejecting swaps that move the price beyond the window.
	TickRange int32 `json:"tickRange"`
}

type HTTPConfig struct {
	BaseURL    string                `json:"baseUrl"`
	Timeout    durationjson.Duration `json:"timeout"`
	RetryCount int                   `json:"retryCount"`
}
