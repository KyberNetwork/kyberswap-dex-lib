package xsolvbtc

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	PoolABI     abi.ABI
	xsolvBTCABI abi.ABI
	OracleABI   abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{
			&PoolABI, poolABIJson,
		},
		{
			&xsolvBTCABI, xsolvBTCABIJson,
		},
		{
			&OracleABI, []byte(`[{"inputs":[{"internalType":"address","name":"erc20_","type":"address"}],"name":"getNav","outputs":[{"internalType":"uint256","name":"nav","type":"uint256"}],"stateMutability":"view","type":"function"}]`),
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
