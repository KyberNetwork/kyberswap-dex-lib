package abis

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	TwammABI                  abi.ABI
	QuoteDataFetcherABI       abi.ABI
	TwammDataFetcherABI       abi.ABI
	CoreABI                   abi.ABI
	MevCaptureRouterABI       abi.ABI
	BoostedFeesABI            abi.ABI
	BoostedFeesDataFetcherABI abi.ABI
	Ve33ABI                   abi.ABI
	Ve33DataFetcherABI        abi.ABI

	OrderUpdatedEvent      abi.Event
	PositionUpdatedEvent   abi.Event
	PoolBoostedEvent       abi.Event
	PoolInitializedEvent   abi.Event
	VoteWeightAppliedEvent abi.Event
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
		{&MevCaptureRouterABI, mevCaptureRouterJson},
		{&BoostedFeesABI, boostedFeesJson},
		{&BoostedFeesDataFetcherABI, boostedFeesDataFetcherJson},
		{&Ve33ABI, ve33Json},
		{&Ve33DataFetcherABI, ve33DataFetcherJson},
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
	PoolBoostedEvent = BoostedFeesABI.Events["PoolBoosted"]
	PoolInitializedEvent = CoreABI.Events["PoolInitialized"]
	VoteWeightAppliedEvent = Ve33ABI.Events["VoteWeightApplied"]
}
