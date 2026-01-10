package abis

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	TwammABI            abi.ABI
	QuoteDataFetcherABI abi.ABI
	TwammDataFetcherABI abi.ABI
	CoreABI             abi.ABI

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
		{&QuoteDataFetcherABI, quoteDataFetcherJson},
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
