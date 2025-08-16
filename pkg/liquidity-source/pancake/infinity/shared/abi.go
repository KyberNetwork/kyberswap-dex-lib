package shared

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var quoterABI abi.ABI
var BinPoolManagerABI abi.ABI
var CLPoolManagerABI abi.ABI

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&quoterABI, quoterABIJson},
		{&BinPoolManagerABI, binPoolManagerABIJson},
		{&CLPoolManagerABI, clPoolManagerABIJson},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
