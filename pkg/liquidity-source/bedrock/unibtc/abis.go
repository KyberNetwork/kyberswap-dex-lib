package unibtc

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	VaultUniBTCABI abi.ABI
	VaultBrBTCABI  abi.ABI
	PausedABI      abi.ABI
	TotalSupplyABI abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{
			&VaultUniBTCABI, vaultUniBTCABIJson,
		},
		{
			&VaultBrBTCABI, vaultBrBTCABIJson,
		},
		{
			&PausedABI, []byte(`[{"inputs":[],"name":"paused","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"}]`),
		},
		{
			&TotalSupplyABI, []byte(`[{"inputs":[{"internalType":"address","name":"_leadingToken","type":"address"}],"name":"totalSupply","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"}]`),
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
