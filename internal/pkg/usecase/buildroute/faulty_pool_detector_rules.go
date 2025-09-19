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
	nativev1 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/native/v1/client"
	swaapv2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/swaap-v2/client"

	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

// isSlippageAboveMinThreshold checks if the estimated slippage exceeds the configured threshold for a token group
// Returns true if slippage is too high, indicating a potentially faulty pool
func isSlippageAboveMinThreshold(estimatedSlippage float64, tokenGroup string, slippageCfg map[string]SlippageGroupConfig) bool {
	var threshold float64
	if slippageConfig, ok := slippageCfg[strings.ToLower(tokenGroup)]; ok {
		threshold = slippageConfig.MinThreshold
	}

	return estimatedSlippage > threshold
}

// requests to be tracked should only involve tokens that have been whitelisted or native token
func isTokenWhiteList(token string, config FaultyPoolsConfig, chainID valueobject.ChainID) bool {
	if valueobject.IsNative(token) || valueobject.IsWrappedNative(token, chainID) {
		return true
	}

	if ok := config.WhitelistedTokenSet[token]; ok {
		return true
	}

	return false
}

// requests to be tracked should only involve tokens that have been indentified as non-fot token or non-honeypot
func isInvalid(tokenInfo *entity.TokenInfo) bool {
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
		errors.Is(err, clipper.ErrQuoteSignFailed) ||
		errors.Is(err, dexalot.ErrRFQFailed) ||
		errors.Is(err, mxtrading.ErrOrderIsTooSmall) ||
		errors.Is(err, mxtrading.ErrRFQFailed) {
		return true
	}

	return false
}

func isSwapSinglePoolFailed(err error) bool {
	return err != nil && strings.Contains(err.Error(), "swapSinglePool failed")
}
