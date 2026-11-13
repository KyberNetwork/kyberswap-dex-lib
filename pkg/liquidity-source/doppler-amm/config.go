package doppleramm

type Config struct {
	ChainID             int    `json:"chainId"`
	AMMAddress          string `json:"ammAddress"`
	QuoterAddress       string `json:"quoterAddress"`
	SpotModuleAddress   string `json:"spotModuleAddress"`
	LaunchModuleAddress string `json:"launchModuleAddress"`
	FeesLockerAddress   string `json:"feesLockerAddress"`
}

// withDefaults fills every field the deployment set can supply, so pool-service
// only has to configure chainId.
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
	return nil
}
