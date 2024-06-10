package buildroute

import (
	"strings"

	"github.com/KyberNetwork/router-service/internal/pkg/utils/eth"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

func isErrReturnAmountIsNotEnough(err error) bool {
	return err != nil && strings.Contains(err.Error(), ErrReturnAmountIsNotEnough.Error())
}

func slippageIsAboveMinThreshold(slippageTolerance int64, config FaultyPoolsConfig) bool {
	return slippageTolerance > config.MinSlippageThreshold
}

// requests to be tracked should only involve tokens that have been whitelisted or native token
func IsTokenValid(token string, config FaultyPoolsConfig, chainID valueobject.ChainID) bool {
	if eth.IsEther(token) || eth.IsWETH(token, chainID) {
		return true
	}

	if ok := config.WhitelistedTokenSet[token]; ok {
		return true
	}

	return false
}
