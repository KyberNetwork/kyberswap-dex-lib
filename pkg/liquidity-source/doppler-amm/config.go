package doppleramm

import (
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

// Config holds every contract address the source needs; none has a default, so a
// deployment is wired by the consumer's yaml only.
type Config struct {
	AMMAddress          string `json:"ammAddress"`
	QuoterAddress       string `json:"quoterAddress"`
	SpotModuleAddress   string `json:"spotModuleAddress"`
	LaunchModuleAddress string `json:"launchModuleAddress"`
	// FeesLockerAddress is the only extension a supported launch market carries.
	FeesLockerAddress string `json:"feesLockerAddress"`
}

// validate checks that every address is set and well formed, and lowercases them.
func (c *Config) validate() error {
	for name, addr := range map[string]*string{
		"ammAddress":          &c.AMMAddress,
		"quoterAddress":       &c.QuoterAddress,
		"spotModuleAddress":   &c.SpotModuleAddress,
		"launchModuleAddress": &c.LaunchModuleAddress,
		"feesLockerAddress":   &c.FeesLockerAddress,
	} {
		if !common.IsHexAddress(*addr) || common.HexToAddress(*addr) == (common.Address{}) {
			return fmt.Errorf("%w: %s=%q", ErrInvalidConfig, name, *addr)
		}
		*addr = strings.ToLower(*addr)
	}
	return nil
}
