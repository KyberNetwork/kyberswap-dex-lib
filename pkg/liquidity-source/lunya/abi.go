package lunya

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	poolABI    abi.ABI
	pluginABI  abi.ABI
	factoryABI abi.ABI

	poolCreatedEvent abi.Event
	mintEvent        abi.Event
	burnEvent        abi.Event
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&poolABI, poolABIBytes},
		{&pluginABI, pluginABIBytes},
		{&factoryABI, factoryABIBytes},
	}

	for _, b := range builder {
		var err error
		if *b.ABI, err = abi.JSON(bytes.NewReader(b.data)); err != nil {
			panic(err)
		}
	}

	poolCreatedEvent = factoryABI.Events["PoolCreated"]
	mintEvent = poolABI.Events["Mint"]
	burnEvent = poolABI.Events["Burn"]
}
