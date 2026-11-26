package syntropiaprop

import (
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

// Config has no defaults: the consumer's per-dex properties name both contracts.
type Config struct {
	DexID string `json:"dexID"`
	// Pool is the one contract every pair trades through: getPairs() lists them, and the
	// executor swaps on it.
	Pool string `json:"pool"`
	// Lens is the read-only helper whose getOrderbook(base, quote) quotes a pair's depth.
	Lens string `json:"lens"`
	// Buffer, in bps, is the share of each quoted output that is kept: 9990 quotes 0.1% under
	// the pool. The pool reprices about once a second, so the snapshot always trails it.
	Buffer int64 `json:"buffer"`
}

// validate checks both addresses and the buffer, and lowercases the addresses.
func (c *Config) validate() error {
	for name, addr := range map[string]*string{"pool": &c.Pool, "lens": &c.Lens} {
		if !common.IsHexAddress(*addr) || common.HexToAddress(*addr) == (common.Address{}) {
			return fmt.Errorf("%w: %s=%q", ErrInvalidConfig, name, *addr)
		}
		*addr = strings.ToLower(*addr)
	}
	if c.Buffer <= 0 || c.Buffer > 10_000 {
		return fmt.Errorf("%w: buffer=%d, want bps in (0, 10000]", ErrInvalidConfig, c.Buffer)
	}
	return nil
}
