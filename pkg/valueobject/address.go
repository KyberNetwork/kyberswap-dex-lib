package valueobject

import "github.com/ethereum/go-ethereum/common"

const (
	EtherAddress    = NativeAddress // deprecated: use the correctly named NativeAddress
	NativeAddress   = "0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE"
	ZeroAddress     = "0x0000000000000000000000000000000000000000"
	MKRTokenAddress = "0x9f8f72aa9304c8b593d555f12ef6589cc3a579a2"
	DAITokenAddress = "0x6b175474e89094c44da98b954eedeac495271d0f"
)

func IsZeroAddress(address common.Address) bool {
	return address.Cmp(common.Address{}) == 0
}
