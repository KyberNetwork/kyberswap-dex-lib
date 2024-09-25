package usd0pp

const (
	DexType = "usd0pp"
	USD0PP  = "0x35d8949372d46b7a3d5a56006ae77b215fc69bc0"
	USD0    = "0x73a15fed60bf67631dc6cd7bc5b6e8da8190acf5"
)

var (
	defaultGas = Gas{
		Mint: 200000,
	}
)

const (
	// number of seconds from beginning to end of the bond period (4 years)
	totalBondTimes = 126230400
)

const (
	usd0ppMethodPaused       = "paused"
	usd0ppMethodGetEndTime   = "getEndTime"
	usd0ppMethodGetStartTime = "getStartTime"
	usd0ppMethodTotalSupply  = "totalSupply"
)
