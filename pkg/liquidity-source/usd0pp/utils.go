package usd0pp

import "github.com/ethereum/go-ethereum/core/types"

func isSpecificEvent(log types.Log, eventName string) bool {
	if len(log.Topics) == 0 {
		return false
	}

	if _, ok := usd0ppABI.Events[eventName]; !ok {
		return false
	}

	return log.Topics[0] == usd0ppABI.Events[eventName].ID
}

func findLatestPausedOrUnpausedEvent(logs []types.Log) (bool, bool, types.Log) {
	var (
		found       bool
		isPaused    bool
		latestEvent types.Log
	)
	for _, log := range logs {
		if log.Removed {
			continue
		}

		isPausedEvent := isSpecificEvent(log, "Paused")
		isUnpausedEvent := isSpecificEvent(log, "Unpaused")

		if !isPausedEvent && !isUnpausedEvent {
			continue
		}

		if latestEvent.BlockNumber < log.BlockNumber ||
			(latestEvent.BlockNumber == log.BlockNumber && latestEvent.TxIndex < log.TxIndex) ||
			(latestEvent.BlockNumber == log.BlockNumber && latestEvent.TxIndex == log.TxIndex && latestEvent.Index < log.Index) {

			latestEvent = log
			found = true
			isPaused = isPausedEvent
		}
	}

	return found, isPaused, latestEvent
}
