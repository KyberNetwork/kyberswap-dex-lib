package valueobject

import "github.com/ethereum/go-ethereum/common"

const (
	NativeAddress = "0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE"
	ZeroAddress   = "0x0000000000000000000000000000000000000000"
	Multicall3    = "0xcA11bde05977b3631167028862bE2a173976CA11"
)

var (
	AddrZero       common.Address
	AddrNative     = common.HexToAddress(NativeAddress)
	AddrMulticall3 = common.HexToAddress(Multicall3)

	HashZero common.Hash
)

func IsZeroAddress(address common.Address) bool {
	return address == AddrZero
}
