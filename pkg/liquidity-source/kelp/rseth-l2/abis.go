package rsethl2

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	LRTDepositPoolABI abi.ABI
	LRTOracleABI      abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&LRTDepositPoolABI, LRTDepositPoolABIData},
		{&LRTOracleABI, LRTOracleABIData},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
