package ponsv2

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

var (
	curveABI   abi.ABI
	factoryABI abi.ABI

	// tokenLaunchedEventHash is the topic0 of Factory.TokenLaunched, used to
	// filter logs in pool_list_updater.go.
	tokenLaunchedEventHash common.Hash
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&curveABI, curveABIBytes},
		{&factoryABI, factoryABIBytes},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}

	tokenLaunchedEventHash = factoryABI.Events["TokenLaunched"].ID
}
