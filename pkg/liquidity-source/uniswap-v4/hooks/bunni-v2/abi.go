package bunniv2

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	legacyBunniHubABI abi.ABI
	bunniHubABI       abi.ABI
	bunniHookABI      abi.ABI
	erc4626ABI        abi.ABI
	erc20ABI          abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&legacyBunniHubABI, legacyBunniHubABIJson},
		{&bunniHubABI, bunniHubABIJson},
		{&bunniHookABI, bunniHookABIJson},
		{&erc4626ABI, erc4626ABIJson},
		{&erc20ABI, erc20ABIJson},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
