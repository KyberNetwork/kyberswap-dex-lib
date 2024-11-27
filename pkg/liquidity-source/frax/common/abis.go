package frax_common

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	FrxETHMinterABI abi.ABI
	SfrxETHABI      abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&FrxETHMinterABI, frxETHMinterJson},
		{&SfrxETHABI, sfrxETHJson},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
