package dai_usds

const (
	DexType = "dai-usds"

	defaultTokenWeight = 1
	reserveZero        = "0"
	defaultReserves    = "10000000000000000000"

	// Hop gas incl. the executor per-edge overhead (15,674), measured on Ethereum on 2026-11-28 via Tenderly
	// traces of pool 0x3225737a9bbb6473cb4a45b7244aca2befdb276a. Gas does not depend on the swap size.
	GasDaiToUsds int64 = 144433 // DAI (0x6b175474e89094c44da98b954eedeac495271d0f) -> USDS (0xdc035d45d973e3ec169d2276ddab16f1e407384f)
	GasUsdsToDai int64 = 163472 // USDS -> DAI

	// DefaultGas is only the value of the unused PoolSimulator.gas field, between the two directions.
	DefaultGas int64 = (GasDaiToUsds + GasUsdsToDai) / 2
)
