package common

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	EETHABI          abi.ABI
	LiquidityPoolABI abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{
			&EETHABI, eETHABIJson,
		},
		{
			&LiquidityPoolABI, liquidityPoolABIJson,
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
