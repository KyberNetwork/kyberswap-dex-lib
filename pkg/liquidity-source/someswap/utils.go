package someswap

import (
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"
)

type syncEvent struct {
	Reserve0 *big.Int `abi:"reserve0"`
	Reserve1 *big.Int `abi:"reserve1"`
}

func isSyncEvent(log types.Log) bool {
	if len(log.Topics) == 0 {
		return false
	}
	return log.Topics[0] == pairABI.Events["Sync"].ID
}

func decodeSyncEvent(log types.Log) (ReserveData, error) {
	var evt syncEvent
	if err := pairABI.UnpackIntoInterface(&evt, "Sync", log.Data); err != nil {
		return ReserveData{}, err
	}
	return ReserveData{
		Reserve0: evt.Reserve0,
		Reserve1: evt.Reserve1,
	}, nil
}

func findLatestSyncEvent(logs []types.Log) *types.Log {
	var (
		found       bool
		latestEvent types.Log
	)
	for _, log := range logs {
		if log.Removed || !isSyncEvent(log) {
			continue
		}
		if !found || latestEvent.BlockNumber < log.BlockNumber ||
			(latestEvent.BlockNumber == log.BlockNumber && latestEvent.Index < log.Index) {
			found = true
			latestEvent = log
		}
	}
	if !found {
		return nil
	}
	return &latestEvent
}
