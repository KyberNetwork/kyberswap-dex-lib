package primeeth

import (
	"bytes"
	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	lrtDepositPoolABI abi.ABI
	lrtConfigABI      abi.ABI
	lrtOracleABI      abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&lrtDepositPoolABI, lrtDepositPoolABIJson},
		{&lrtConfigABI, lrtConfigABIJson},
		{&lrtOracleABI, lrtOracleABIJson},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
