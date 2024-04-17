package unieth

const (
	DexType = "bedrock-unieth"
)

var (
	WETH    = "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"
	UNIETH  = "0xf1376bcef0f78459c0ed0ba5ddce976f1ddf51f4"
	Staking = "0x4befa2aa9c305238aa3e0b5d17eb20c045269e9d"
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
