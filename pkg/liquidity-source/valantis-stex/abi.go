package valantisstex

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	sovereignPoolABI    abi.ABI
	swapFeeModuleABI    abi.ABI
	stexAMMABI          abi.ABI
	withdrawalModuleABI abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&sovereignPoolABI, sovereignPoolBytes},
		{&swapFeeModuleABI, swapFeeModuleBytes},
		{&stexAMMABI, stexAMMBytes},
		{&withdrawalModuleABI, withdrawalModuleBytes},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
