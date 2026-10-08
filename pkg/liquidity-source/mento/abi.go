package mento

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	factoryABI            abi.ABI
	poolABI               abi.ABI
	oracleAdapterABI      abi.ABI
	sortedOraclesABI      abi.ABI
	breakerBoxABI         abi.ABI
	marketHoursBreakerABI abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&factoryABI, factoryABIData},
		{&poolABI, poolABIData},
		{&oracleAdapterABI, oracleAdapterABIData},
		{&sortedOraclesABI, sortedOraclesABIData},
		{&breakerBoxABI, breakerBoxABIData},
		{&marketHoursBreakerABI, marketHoursBreakerABIData},
	}

	for _, b := range builder {
		parsed, err := abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
		*b.ABI = parsed
	}
}
