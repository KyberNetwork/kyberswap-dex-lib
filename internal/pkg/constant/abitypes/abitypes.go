package abitypes

import "github.com/ethereum/go-ethereum/accounts/abi"

var (
	Uint256, _    = abi.NewType("uint256", "", nil)
	Uint160, _    = abi.NewType("uint160", "", nil)
	Uint32, _     = abi.NewType("uint32", "", nil)
	Uint16, _     = abi.NewType("uint16", "", nil)
	Uint8, _      = abi.NewType("uint8", "", nil)
	String, _     = abi.NewType("string", "", nil)
	Bool, _       = abi.NewType("bool", "", nil)
	Bytes, _      = abi.NewType("bytes", "", nil)
	Bytes32, _    = abi.NewType("bytes32", "", nil)
	Address, _    = abi.NewType("address", "", nil)
	Uint64Arr, _  = abi.NewType("uint64[]", "", nil)
	Uint256Arr, _ = abi.NewType("uint256[]", "", nil)
	AddressArr, _ = abi.NewType("address[]", "", nil)
	BytesArr, _   = abi.NewType("bytes[]", "", nil)
	Int8, _       = abi.NewType("int8", "", nil)
	Int128, _     = abi.NewType("int128", "", nil)
)
