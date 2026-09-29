package prop

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/titan"
)

// Config covers both on-chain variants. Everything else (swapImpl, wallet,
// quoter, caps) is resolved live from RouterAddress by KipseliPropLens on
// every call, so venue migrations need no config change.
type Config struct {
	DexID         string `json:"dexID"`
	ChainID       int    `json:"chainId"`
	RouterAddress string `json:"routerAddress"`

	// Dest is what the swap path passes to SwapImpl.swap as `dest`, which
	// selects the venue's per-taker quoter. Defaults to the KyberSwap executor.
	Dest string `json:"dest,omitempty"`

	// Buffer, in bps, haircuts every probed amountOut when set.
	Buffer int64 `json:"buffer,omitempty"`

	// Titan is set only for the pAMM variant, whose quoter prices off a
	// Titan-pushed registry state that must be overridden into each call.
	Titan titan.Config `json:"titan,omitempty"`
}

func (c *Config) usesTitan() bool {
	return len(c.Titan.URLs) > 0
}

func (c *Config) dest() common.Address {
	if c.Dest == "" {
		return defaultDest
	}
	return common.HexToAddress(c.Dest)
}
