package etherfivampire

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	curvePlainABI    abi.ABI
	eETHABI          abi.ABI
	liquidityPoolABI abi.ABI
	stETHABI         abi.ABI
	vampireABI       abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&curvePlainABI, curvePlainABIJson},
		{&eETHABI, eETHABIJson},
		{&liquidityPoolABI, liquidityPoolABIJson},
		{&stETHABI, stETHABIJson},
		{&vampireABI, vampireABIJson},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
