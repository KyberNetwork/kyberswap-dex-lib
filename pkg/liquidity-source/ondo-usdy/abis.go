package ondo_usdy

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	mUSDABI             abi.ABI
	rwaDynamicOracleABI abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&mUSDABI, mUSDABIJSON},
		{&rwaDynamicOracleABI, rwaDynamicOracleABIJSON},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
