package bunniv2

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	bunniHubABI           abi.ABI
	bunniHookABI          abi.ABI
	feeOverrideHookletABI abi.ABI
	erc4626ABI            abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&bunniHubABI, bunniHubABIJson},
		{&bunniHookABI, bunniHookABIJson},
		{&feeOverrideHookletABI, feeOverrideHookletABIJson},
		{&erc4626ABI, erc4626ABIJson},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
