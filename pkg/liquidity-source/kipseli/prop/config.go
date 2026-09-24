package prop

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/titan"
)

// Config is a superset of the two on-chain variants this package speaks:
// the EIP-712-signed prop venue (Verifier/Quoter/Buffer), and the
// Titan-quoted pAMM venue (PositionCapAddress/Multicall3Address/Titan) — see
// isPamm. Each chain's yaml only ever populates the fields its own variant
// needs.
type Config struct {
	DexID         string `json:"dexID"`
	ChainID       int    `json:"chainId"`
	LensAddress   string `json:"lensAddress"`
	RouterAddress string `json:"routerAddress"`

	// prop variant only
	Verifier common.Address `json:"verifier,omitempty"`
	Quoter   common.Hash    `json:"quoter,omitempty"`
	Buffer   int64          `json:"buffer,omitempty"`

	// pamm variant only
	PositionCapAddress string       `json:"positionCapAddress,omitempty"`
	Multicall3Address  string       `json:"multicall3Address,omitempty"`
	Titan              titan.Config `json:"titan,omitempty"`
}

// isPamm reports whether this chain's config speaks the Titan-quoted pAMM
// venue rather than the EIP-712-signed prop venue. Verifier is mandatory for
// the prop venue's EIP-712 domain and never zero in a real config, so its
// absence reliably signals the pamm variant — config-driven rather than
// chain-ID-gated, so a future chain running either variant works without a
// code change here.
func (c *Config) isPamm() bool {
	return c.Verifier == (common.Address{})
}
