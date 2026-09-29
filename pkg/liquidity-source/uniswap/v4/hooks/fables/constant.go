package fables

import (
	"github.com/ethereum/go-ethereum/common"
)

// gasBeforeSwap covers the beforeSwap fee-override callback. Fables never touches swap
// deltas and takes no hook fee, so afterSwap is not permissioned and costs nothing extra.
const gasBeforeSwap = 60000

// HookAddresses lists every known Fables dynamic-fee hook on Robinhood Chain (chainId 4663).
//
// Fables deploys a NEW immutable hook per pool. Every pool — present and future — is recorded
// on-chain in the FablesPoolRegistry at 0x159A113E012593D9B3cC63ad45E30F0467e13Ef3, whose
// activePools()/allPools() views expose each pool's full PoolKey (hook included). This list is
// a snapshot of the registry's current membership and must be regenerated from it whenever
// Fables launches a new pool (procedure in doc.go).
//
// Current snapshot: one hook per registered pool, in registration order. Each entry is labeled
// with the pool's currencies in PoolKey order (currency0/currency1, address(0) shown as ETH) and
// the hook's verified contract name. TestHookAddresses_RegistrySnapshot pins the expected
// length; bump it together with this list.
var HookAddresses = []common.Address{
	common.HexToAddress("0x66622f77B797D506e5376F7798b67ab288966080"), // USDG/NVDA (FablesRWA)
	common.HexToAddress("0xA0E8fBFf13E24Af2b5e61A72800E08a161bDe080"), // SPY/USDG (FablesRWA)
	common.HexToAddress("0x79576FBAD6e83915630BBB5D5658483F05532080"), // SPY/NVDA (FablesRWA)
	common.HexToAddress("0x70a9A88402989226847Ec122043CE5e7FF462080"), // USDG/AAPL (FablesRWA)
	common.HexToAddress("0x67D86050d22D574Df046F3D90F722045F714e080"), // TSLA/USDG (FablesRWA)
	common.HexToAddress("0x06a889870C8f83640D6816319f72e2aA579b6080"), // ETH/USDG (FablesRampETH)
	common.HexToAddress("0xB608a78761f179f7C56f15E7D13921B92F00a080"), // USDG/GLD (FablesRWA)
	common.HexToAddress("0xA4570C37590E45f0b06898123D4de16307A32080"), // SPY/GLD (FablesRWA)
	common.HexToAddress("0x8AF95932eC4484fb10C641a4cBcf19a798cB2080"), // USDG/META (FablesRWA)
	common.HexToAddress("0x5Eb87F69bE00Df39981622Fd60A8De4b7837e080"), // AMC/USDG (FablesRWA)
	common.HexToAddress("0x08E52564Bad99E05a694b4809F397edcA417A080"), // CASHCAT/USDG (FablesRamp)
	common.HexToAddress("0xcA89f079AF00f752bfD3C345358Dc38d4d73e080"), // ETH/AMC (FablesRWAETH)
	common.HexToAddress("0x594e8e6281eDf2d363a0293a50004Cf868E7a080"), // ETH/PONS (FablesRampETH)
}
