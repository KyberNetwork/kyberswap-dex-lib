package eth

import (
	"slices"

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

	for _, log := range slices.Backward(logs) {
		if log.Removed {
			continue
		}
		return log.BlockNumber
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

func SortLogs(logs []ethtypes.Log) {
	slices.SortFunc(logs, func(l, r ethtypes.Log) int {
		if l.BlockNumber == r.BlockNumber {
			return int(l.Index - r.Index)
		}

		return int(l.BlockNumber - r.BlockNumber)
	})
}
