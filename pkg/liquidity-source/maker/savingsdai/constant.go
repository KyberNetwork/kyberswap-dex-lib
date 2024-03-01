package savingsdai

import "github.com/KyberNetwork/blockchain-toolkit/number"

const (
	DexType = "maker-savingsdai"

	dai        = "0x6b175474e89094c44da98b954eedeac495271d0f"
	pot        = "0x197e90f9fad81970ba7976f33cbd77088e5d7cf7"
	savingsdai = "0x83f20f44975d03b1b09e64809b757c47f942beea"

	potMethodDSR                = "dsr"
	potMethodRHO                = "rho"
	potMethodCHI                = "chi"
	savingsdaiMethodTotalAssets = "totalAssets"
	savingsdaiMethodTotalSupply = "totalSupply"

	blocktime = 12
)

var (
	one = number.TenPow(27)
	ray = one
)
