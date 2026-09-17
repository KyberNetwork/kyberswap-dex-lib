package lunyafun

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	launchABI  abi.ABI
	factoryABI abi.ABI

	launchCreatedEvent abi.Event
	tradeEvent         abi.Event
)

// LaunchABI is exported for aggregator-encoding's swapdata packer, which calls
// LaunchABI.Pack("buy", ...) / LaunchABI.Pack("sell", ...) to build the on-chain calldata directly
// against the same ABI the tracker parses -- no separate copy to drift.
var LaunchABI abi.ABI

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&launchABI, launchABIBytes},
		{&factoryABI, factoryABIBytes},
	}

	for _, b := range builder {
		var err error
		if *b.ABI, err = abi.JSON(bytes.NewReader(b.data)); err != nil {
			panic(err)
		}
	}

	launchCreatedEvent = factoryABI.Events["LaunchCreated"]
	tradeEvent = launchABI.Events["Trade"]

	LaunchABI = launchABI
}
