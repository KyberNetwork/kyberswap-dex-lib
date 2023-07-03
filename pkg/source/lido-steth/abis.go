package lido_steth

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	stEthABI abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{
			&stEthABI, stEthABIData,
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
