package liquidityparty

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

var (
	partyPlannerABI abi.ABI
	partyPoolABI    abi.ABI
	partyInfoABI    abi.ABI
)

// Method / event names used across the factory decoder, lister and tracker.
const (
	plannerMethodPoolCount   = "poolCount"
	plannerMethodGetAllPools = "getAllPools"
	plannerEventPartyStarted = "PartyStarted"

	poolMethodAllTokens = "allTokens"
	poolMethodKilled    = "killed"

	infoMethodFetchPoolState            = "fetchPoolState"
	infoMethodSwapAmounts               = "swapAmounts"
	infoMethodSwapAmountsForExactOutput = "swapAmountsForExactOutput"
)

// Event topic0 hashes, resolved in init once the ABIs are parsed.
var (
	// partyStartedEventTopic is PartyPlanner's `PartyStarted(pool,name,symbol,tokens[])` event —
	// the primary pool-discovery signal (see pool_factory.go).
	partyStartedEventTopic common.Hash

	// killedEventTopic is PartyPool's `Killed()` event — the primary, authoritative kill signal the
	// tracker acts on (kill is irreversible), avoiding a per-refresh killed() RPC poll.
	killedEventTopic common.Hash
)

func init() {
	builder := []struct {
		abiVal *abi.ABI
		json   []byte
	}{
		{&partyPlannerABI, partyPlannerABIJson},
		{&partyPoolABI, partyPoolABIJson},
		{&partyInfoABI, partyInfoABIJson},
	}

	for _, item := range builder {
		parsed, err := abi.JSON(bytes.NewReader(item.json))
		if err != nil {
			panic(err)
		}
		*item.abiVal = parsed
	}

	partyStartedEventTopic = partyPlannerABI.Events[plannerEventPartyStarted].ID
	killedEventTopic = partyPoolABI.Events["Killed"].ID
}
