package uniswap

import (
	"github.com/ethereum/go-ethereum/core/types"
)

func isSyncEvent(log types.Log) bool {
	if len(log.Topics) == 0 {
		return false
	}

	return log.Topics[0] == uniswapV2PairABI.Events["Sync"].ID
}

func decodeSyncEvent(log types.Log) (Reserves, error) {
	filterer, err := NewUniswapFilterer(log.Address, nil)
	if err != nil {
		return Reserves{}, err
	}

	syncEvent, err := filterer.ParseSync(log)
	if err != nil {
		return Reserves{}, err
	}

	return Reserves{
		Reserve0: syncEvent.Reserve0,
		Reserve1: syncEvent.Reserve1,
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
