package navjit

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	hookABI     abi.ABI
	navGuardABI abi.ABI
	erc20ABI    abi.ABI
	quoterABI   abi.ABI
)

func init() {
	for _, item := range []struct {
		abi  *abi.ABI
		data []byte
	}{
		{&hookABI, hookABIJson},
		{&navGuardABI, navGuardABIJson},
		{&erc20ABI, erc20ABIJson},
		{&quoterABI, quoterABIJson},
	} {
		var err error
		if *item.abi, err = abi.JSON(bytes.NewReader(item.data)); err != nil {
			panic(err)
		}
	}
}
