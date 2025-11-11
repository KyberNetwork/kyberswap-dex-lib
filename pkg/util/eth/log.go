package eth

import (
	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

func GetBlockNumberFromLogs(logs []ethtypes.Log) uint64 {
	if len(logs) == 0 {
		return 0
	}

	// We check both logs order direction
	firstLog := logs[0]
	lastLog := logs[len(logs)-1]
	if firstLog.BlockNumber < lastLog.BlockNumber {
		return lastLog.BlockNumber
	}

	return firstLog.BlockNumber
}

func GetLatestBlockNumberFromLogs(logs []ethtypes.Log) uint64 {
	if len(logs) == 0 {
		return 0
	}

	for i := len(logs) - 1; i >= 0; i-- {
		if logs[i].Removed {
			continue
		}
		return logs[i].BlockNumber
	}

	return 0
}

func HasRevertedLog(logs []ethtypes.Log) bool {
	for _, log := range logs {
		if log.Removed {
			return true
		}
	}

	return false
}
