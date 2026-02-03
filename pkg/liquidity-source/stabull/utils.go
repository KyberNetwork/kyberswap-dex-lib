package stabull

import (
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

// isTradeEvent checks if a log is a Trade event (swap)
func isTradeEvent(log types.Log) bool {
	if len(log.Topics) == 0 {
		return false
	}
	return log.Topics[0] == tradeEventTopic
}

// isParametersSetEvent checks if a log is a ParametersSet event
func isParametersSetEvent(log types.Log) bool {
	if len(log.Topics) == 0 {
		return false
	}
	return log.Topics[0] == parametersSetEventTopic
}

// Helper to check if a log address matches the pool address
func isLogFromPool(log types.Log, poolAddress string) bool {
	return hexutil.Encode(log.Address[:]) == poolAddress
}

// isAnswerUpdatedEvent checks if a log is a Chainlink AnswerUpdated event
func isAnswerUpdatedEvent(log types.Log) bool {
	if len(log.Topics) == 0 {
		return false
	}
	return log.Topics[0] == answerUpdatedEventTopic
}

// isNewTransmissionEvent checks if a log is a Chainlink NewTransmission event (OCR2)
func isNewTransmissionEvent(log types.Log) bool {
	if len(log.Topics) == 0 {
		return false
	}
	return log.Topics[0] == newTransmissionEventTopic
}
