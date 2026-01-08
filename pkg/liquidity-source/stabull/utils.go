package stabull

import (
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/core/types"
)

// isTradeEvent checks if a log is a Trade event (swap)
func isTradeEvent(log types.Log) bool {
	if len(log.Topics) == 0 {
		return false
	}
	return log.Topics[0].Hex() == tradeEventTopic
}

// isParametersSetEvent checks if a log is a ParametersSet event
func isParametersSetEvent(log types.Log) bool {
	if len(log.Topics) == 0 {
		return false
	}
	return log.Topics[0].Hex() == parametersSetEventTopic
}

// normalizeAddress converts an address to lowercase for comparison
func normalizeAddress(addr string) string {
	return strings.ToLower(addr)
}

// reserveString converts a big.Int to string, handling nil
func reserveString(reserve *big.Int) string {
	if reserve == nil {
		return reserveZero
	}
	return reserve.String()
}

// Helper to check if a log address matches the pool address
func isLogFromPool(log types.Log, poolAddress string) bool {
	return normalizeAddress(log.Address.Hex()) == normalizeAddress(poolAddress)
}

// Helper to check if a log address matches an oracle address
func isLogFromOracle(log types.Log, oracleAddress string) bool {
	return normalizeAddress(log.Address.Hex()) == normalizeAddress(oracleAddress)
}

// isAnswerUpdatedEvent checks if a log is a Chainlink AnswerUpdated event
func isAnswerUpdatedEvent(log types.Log) bool {
	if len(log.Topics) == 0 {
		return false
	}
	return log.Topics[0].Hex() == answerUpdatedEventTopic
}

// isNewTransmissionEvent checks if a log is a Chainlink NewTransmission event (OCR2)
func isNewTransmissionEvent(log types.Log) bool {
	if len(log.Topics) == 0 {
		return false
	}
	return log.Topics[0].Hex() == newTransmissionEventTopic
}
