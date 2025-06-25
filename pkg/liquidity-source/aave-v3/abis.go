package aavev3

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	aaveV3PoolABI        abi.ABI
	atokenABI            abi.ABI
	variableDebtTokenABI abi.ABI
	stableDebtTokenABI   abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&aaveV3PoolABI, aaveV3PoolJSON},
		{&atokenABI, atokenJSON},
		{&variableDebtTokenABI, variableDebtTokenJSON},
		{&stableDebtTokenABI, stableDebtTokenJSON},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
