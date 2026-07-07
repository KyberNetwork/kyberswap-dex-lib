package common

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	SWETHABI                abi.ABI
	RSWETHABI               abi.ABI
	AccessControlManagerABI abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{
			&SWETHABI, swETHABIJson,
		},
		{
			&RSWETHABI, rswETHABIJson,
		},
		{
			&AccessControlManagerABI, accessControlManagerABIJson,
		},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
