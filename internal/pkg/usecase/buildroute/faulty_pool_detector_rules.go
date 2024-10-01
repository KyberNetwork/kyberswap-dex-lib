package buildroute

import (
	"errors"
	"strings"

	hashflowv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/hashflow-v3/client"
	nativev1 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/native-v1/client"
	swaapv2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/swaap-v2/client"
	kyberpmm "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/kyber-pmm/client"
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
func isTokenValid(token string, config FaultyPoolsConfig, chainID valueobject.ChainID) bool {
	if eth.IsEther(token) || eth.IsWETH(token, chainID) {
		return true
	}

	if ok := config.WhitelistedTokenSet[token]; ok {
		return true
	}

	return false
}

func isPMMFaultyPoolError(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, kyberpmm.ErrFirmQuoteFailed) {
		return true
	}

	if errors.Is(err, hashflowv3.ErrRFQMarketsTooVolatile) {
		return true
	}

	if errors.Is(err, nativev1.ErrRFQAllPricerFailed) {
		return true
	}

	if errors.Is(err, swaapv2.ErrQuoteFailed) {
		return true
	}

	return false
}
