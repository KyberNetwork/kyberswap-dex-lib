package buildroute

import (
	"errors"
	"strings"

	kyberpmm "github.com/KyberNetwork/kyberswap-dex-lib-private/pkg/liquidity-source/kyber-pmm/client"
	mxtrading "github.com/KyberNetwork/kyberswap-dex-lib-private/pkg/liquidity-source/mx-trading/client"
	bebopclient "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/bebop/client"
	clipper "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/clipper/client"
	dexalot "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/dexalot/client"
	hashflowv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/hashflow-v3/client"
	nativev1 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/native-v1/client"
	swaapv2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/swaap-v2/client"

	routerEntity "github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/eth"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

func isErrReturnAmountIsNotEnough(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), ErrReturnAmountIsNotEnough.Error())
}

func slippageIsAboveMinThreshold(slippageTolerance float64, config FaultyPoolsConfig) bool {
	return slippageTolerance > config.MinSlippageThreshold
}

// requests to be tracked should only involve tokens that have been whitelisted or native token
func isTokenWhiteList(token string, config FaultyPoolsConfig, chainID valueobject.ChainID) bool {
	if eth.IsEther(token) || eth.IsWETH(token, chainID) {
		return true
	}

	if ok := config.WhitelistedTokenSet[token]; ok {
		return true
	}

	return false
}

// requests to be tracked should only involve tokens that have been indentified as non-fot token or non-honeypot
func isInvalid(tokenInfo *routerEntity.TokenInfo) bool {
	return tokenInfo.IsFOT || tokenInfo.IsHoneypot
}

func isPMMFaultyPoolError(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, kyberpmm.ErrFirmQuoteFailed) ||
		errors.Is(err, hashflowv3.ErrRFQMarketsTooVolatile) ||
		errors.Is(err, nativev1.ErrRFQAllPricerFailed) ||
		errors.Is(err, swaapv2.ErrQuoteFailed) ||
		errors.Is(err, bebopclient.ErrRFQFailed) ||
		errors.Is(err, clipper.ErrQuoteFailed) ||
		errors.Is(err, clipper.ErrSignFailed) ||
		errors.Is(err, dexalot.ErrRFQFailed) ||
		errors.Is(err, mxtrading.ErrOrderIsTooSmall) ||
		errors.Is(err, mxtrading.ErrRFQFailed) {
		return true
	}

	return false
}
