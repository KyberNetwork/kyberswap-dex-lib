package netstaking

type Config struct {
	DexId          string `json:"dexId"`
	StakingAddress string `json:"stakingAddress"`
	// WrapAddress is optional. Empty disables wrapping entirely: only the base<->staked
	// 1:1 leg (Stake/Unstake) is offered; Wrap/Unwrap/StakeAndWrap never appear.
	WrapAddress string `json:"wrapAddress,omitempty"`

	// BaseTokenMethod/StakedTokenMethod name the no-arg, address-returning view method on
	// the staking contract to call for token discovery - different staking forks name
	// these getters differently (net()/sNet() vs nuke()/stakedNuke()). The 4-byte selector
	// is derived from the name the same way solc does. Empty (the yaml default) falls back
	// to "net"/"sNet", so the existing net-staking pool config needs no changes.
	BaseTokenMethod   string `json:"baseTokenMethod,omitempty"`
	StakedTokenMethod string `json:"stakedTokenMethod,omitempty"`
}
