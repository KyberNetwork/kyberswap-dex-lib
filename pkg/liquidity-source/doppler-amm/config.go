package doppleramm

type Config struct {
	DexID             string `json:"dexID"`
	ChainID           int    `json:"chainId"`
	AMMAddress        string `json:"ammAddress"`
	QuoterAddress     string `json:"quoterAddress"`
	SpotModuleAddress string `json:"spotModuleAddress"`
	// MarketKind picks which markets this source lists: "spot" (default) or "launch".
	MarketKind          string `json:"marketKind"`
	LaunchModuleAddress string `json:"launchModuleAddress"`
	FeesLockerAddress   string `json:"feesLockerAddress"`
	StartBlock          uint64 `json:"startBlock"`
	MaxBlockRange       uint64 `json:"maxBlockRange"`
}

const (
	defaultMaxBlockRange = 10_000
	MarketKindSpot       = "spot"
	MarketKindLaunch     = "launch"
)

// withDefaults fills every field the deployment set can supply, so pool-service
// only has to configure dexID and chainId.
func (c *Config) withDefaults() error {
	d, ok := deployments[c.ChainID]
	if !ok {
		return ErrUnsupportedChain
	}
	if c.AMMAddress == "" {
		c.AMMAddress = d.AMM
	}
	if c.QuoterAddress == "" {
		c.QuoterAddress = d.Quoter
	}
	if c.SpotModuleAddress == "" {
		c.SpotModuleAddress = d.SpotModule
	}
	if c.LaunchModuleAddress == "" {
		c.LaunchModuleAddress = d.LaunchModule
	}
	if c.FeesLockerAddress == "" {
		c.FeesLockerAddress = d.FeesLocker
	}
	if c.StartBlock == 0 {
		c.StartBlock = d.StartBlock
	}
	if c.MaxBlockRange == 0 {
		c.MaxBlockRange = defaultMaxBlockRange
	}
	return nil
}
