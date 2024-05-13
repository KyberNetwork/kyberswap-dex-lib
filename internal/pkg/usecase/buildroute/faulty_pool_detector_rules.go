package buildroute

import (
	"strings"
)

func isErrReturnAmountIsNotEnough(err error) bool {
	return err != nil && strings.Contains(err.Error(), ErrReturnAmountIsNotEnough.Error())
}

func slippageIsAboveMinThreshold(slippageTolerance int64, config FaultyPoolsConfig) bool {
	return slippageTolerance > config.MinSlippageThreshold
}
