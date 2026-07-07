package clanker

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	clankerABI        abi.ABI
	dynamicFeeHookABI abi.ABI
	staticFeeHookABI  abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&clankerABI, clankerABIJson},
		{&dynamicFeeHookABI, dynamicFeeHookABIJson},
		{&staticFeeHookABI, staticFeeHookABIJson},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
