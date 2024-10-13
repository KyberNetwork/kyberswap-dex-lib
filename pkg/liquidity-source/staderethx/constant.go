package staderethx

const (
	DexType = "staderethx"

	staderStakePoolsManager = "0xcf5ea1b38380f6af39068375516daf40ed70d299"
	staderOracle            = "0xf64bae65f6f2a5277571143a24faafdfc0c2a737"

	WETH = "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"
	ETHx = "0xa35b1b31ce002fbf2058d22f30f95d405200a15b"

	defaultReserves = "1000000000000000000000000"
)

const (
	staderStakePoolsManagerMethodPaused     = "paused"
	staderStakePoolsManagerMethodMinDeposit = "minDeposit"
	staderStakePoolsManagerMethodMaxDeposit = "maxDeposit"
	// staderStakePoolsManagerMethodGetExchangeRate = "getExchangeRate"

	staderOracleMethodExchangeRate = "exchangeRate"
)

var (
	defaultGas = Gas{
		Deposit: 250000,
	}
)
