package common

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	LRTDepositPoolABI abi.ABI
	LRTConfigABI      abi.ABI
	LRTOracleABI      abi.ABI
	Erc20ABI          abi.ABI
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
		{
			&LRTOracleABI, lrtOracleABIJson,
		},
		{
			&Erc20ABI, erc20ABIJson,
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
