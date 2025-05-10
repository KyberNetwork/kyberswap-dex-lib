package abis

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	TwammABI            abi.ABI
	BasicDataFetcherABI abi.ABI
	TwammDataFetcherABI abi.ABI
	CoreABI             abi.ABI
)
var (
	OrderUpdatedEvent    abi.Event
	PositionUpdatedEvent abi.Event
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&CoreABI, coreJson},
		{&TwammABI, twammJson},
		{&BasicDataFetcherABI, basicDataFetcherJson},
		{&TwammDataFetcherABI, twammDataFetcherJson},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}

	PositionUpdatedEvent = CoreABI.Events["PositionUpdated"]
	OrderUpdatedEvent = TwammABI.Events["OrderUpdated"]
}
