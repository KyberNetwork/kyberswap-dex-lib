package slyngfun

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	launchpadABI abi.ABI

	tokenCreatedEvent abi.Event
	boughtEvent       abi.Event
	soldEvent         abi.Event
	graduatedEvent    abi.Event
)

// LaunchpadABI is exported for aggregator-encoding: buy(token, amountIn, minTokensOut) and
// sell(token, tokensIn, minQuoteOut) are packed against the same ABI the tracker reads with.
var LaunchpadABI abi.ABI

func init() {
	var err error
	if launchpadABI, err = abi.JSON(bytes.NewReader(launchpadABIBytes)); err != nil {
		panic(err)
	}

	tokenCreatedEvent = launchpadABI.Events[launchpadEventTokenCreated]
	boughtEvent = launchpadABI.Events[launchpadEventBought]
	soldEvent = launchpadABI.Events[launchpadEventSold]
	graduatedEvent = launchpadABI.Events[launchpadEventGraduated]

	LaunchpadABI = launchpadABI
}
