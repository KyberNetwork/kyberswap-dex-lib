package erc4626

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	ABI abi.ABI

	uint256Type, _ = abi.NewType("uint256", "", nil)
)

// Uint256GetterABI builds an ABI holding a single no-arg view method returning uint256.
func Uint256GetterABI(method string) abi.ABI {
	return abi.ABI{Methods: map[string]abi.Method{
		method: abi.NewMethod(method, method, abi.Function, "view", false, false, nil,
			abi.Arguments{{Type: uint256Type}}),
	}}
}

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&ABI, ERC626Json},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
