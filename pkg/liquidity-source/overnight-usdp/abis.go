package overnightusdp

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	exchangeABI abi.ABI
	erc20ABI    abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&exchangeABI, exchangeABIJson},
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
