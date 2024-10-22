package meth

const (
	DexType = "meth"

	MantleLSPStaking = "0xe3cbd06d7dadb3f4e6557bab7edd924cd1489e8f"
	MantlePauser     = "0x29Ab878aEd032e2e2c86FF4A9a9B05e3276cf1f8"
	METH             = "0xd5f7838f5c461feff7fe49ea5ebaf7728bb0adfa"
	WETH             = "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"

	BASIS_POINTS_DENOMINATOR uint16 = 10000

	defaultReserves = "1000000000000000000000000"
)

const (
	mantlePauserMethodIsStakingPaused = "isStakingPaused"

	mantleLSPStakingMethodTotalControlled        = "totalControlled"
	mantleLSPStakingMethodExchangeAdjustmentRate = "exchangeAdjustmentRate"
	mantleLSPStakingMethodMaximumDepositAmount   = "maximumDepositAmount"
	mantleLSPStakingMethodMinimumStakeBound      = "minimumStakeBound"
	mantleLSPStakingMethodMaximumMETHSupply      = "maximumMETHSupply"

	mETHMethodTotalSupply = "totalSupply"
)

var (
	defaultGas = Gas{
		Stake: 100000,
	}
)
