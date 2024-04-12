package unieth

var (
	DexType = "bedrock-unieth"
)

var (
	WETH    = "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"
	UNIETH  = "0xf1376bcef0f78459c0ed0ba5ddce976f1ddf51f4"
	Staking = "0xd968495636c4cf7435b36a6a8135c1b528ff31b1"
)

var (
	UniETHMethodTotalSupply     = "totalSupply"
	StakingMethodCurrentReserve = "currentReserve"
	StakingMethodPaused         = "paused"
)

var (
	defaultGas = Gas{
		Mint: 100000,
	}
)

const (
	// unlimited reserve
	reserves = "10000000000000000000"
)
