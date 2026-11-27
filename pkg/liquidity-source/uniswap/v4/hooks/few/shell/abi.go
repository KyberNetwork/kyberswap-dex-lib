package fewshell

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

// shellABI holds the FewV2ShellHook views Track reads and the two FewV2 pair views (token0, getReserves) it reads next.
var shellABI abi.ABI

func init() {
	var err error
	if shellABI, err = abi.JSON(bytes.NewReader(shellABIJson)); err != nil {
		panic(err)
	}
}
