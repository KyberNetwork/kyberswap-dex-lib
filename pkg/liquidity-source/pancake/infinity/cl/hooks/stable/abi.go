package stable

import "github.com/ethereum/go-ethereum/accounts/abi"

var (
	hookFactoryABI abi.ABI
	hookABI        abi.ABI
)

func init() {
	if err := hookFactoryABI.UnmarshalJSON(hookFactoryABIBytes); err != nil {
		panic(err)
	}
	if err := hookABI.UnmarshalJSON(hookABIBytes); err != nil {
		panic(err)
	}
}
