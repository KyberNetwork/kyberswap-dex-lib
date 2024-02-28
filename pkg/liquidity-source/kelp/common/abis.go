package common

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	LRTDepositPoolABI abi.ABI
	LRTConfigABI      abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{
			&LRTConfigABI, lrtConfigABIJson,
		},
		{
			&LRTDepositPoolABI, lrtDepositPoolABIJson,
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
