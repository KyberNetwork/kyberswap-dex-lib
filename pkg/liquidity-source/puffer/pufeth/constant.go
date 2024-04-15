package pufeth

var (
	DexType = "puffer-pufeth"
)

var (
	PufferDepositor = "0x4aa799c5dfc01ee7d790e3bf1a7c2257ce1dceff"
	PUFETH          = "0xd9a442856c234a39a81a089c06451ebaa4306a72"
	STETH           = "0xae7ab96520de3a18e5e111b5eaab095312d7fe84"
	WSTETH          = "0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0"
)

var (
	PufferVaultMethodTotalSupply  = "totalSupply"
	PufferVaultMethodTotalAssets  = "totalAssets"
	LidoMethodGetTotalPooledEther = "getTotalPooledEther"
	LidoMethodGetTotalShares      = "getTotalShares"
)

var defaultGas = Gas{
	depositStETH:  250000,
	depositWstETH: 280000,
}

const (
	// unlimited reserve
	reserves = "10000000000000000000"
)
