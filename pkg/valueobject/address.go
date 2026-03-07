package valueobject

import "github.com/ethereum/go-ethereum/common"

const (
	NativeAddress = "0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE"
	ZeroAddress   = "0x0000000000000000000000000000000000000000"
)

var (
	AddrZero   common.Address
	AddrNative = common.HexToAddress(NativeAddress)

	HashZero common.Hash
)

func IsZeroAddress(address common.Address) bool {
	return address == AddrZero
}
