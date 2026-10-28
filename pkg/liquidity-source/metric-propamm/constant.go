package metricpropamm

import (
	"math/big"
	"time"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

const (
	DexType          = "metric-propamm"
	metadataPageSize = 50

	// baseGas is the swap gas of a pool before extensions. Mean of 27 Tenderly pool-call
	// frames (base+bsc, both directions, POD included) of frame gas - extensions*gasPerExtension.
	baseGas = 159433
	// gasPerExtension is the swap gas of one extension. Mean of 23 Tenderly hook frames
	// (base+bsc: priceVelocity, stopLoss, third extension).
	gasPerExtension = 18650

	// extensionsTTL is how long a fetched extension count is reused; retry delay after a failed fetch.
	extensionsTTL   = 10 * time.Minute
	extensionsRetry = 30 * time.Second

	// batchSize is Metric's max pools per bid_ask/batch request. retryAfter* bound the pause after a 429.
	batchSize         = 100
	retryAfterDefault = 5 * time.Second
	retryAfterMax     = time.Minute

	// statusFeedDown is bid_ask.priceProviderStatus while Metric has no price feed (quotes are empty).
	statusFeedDown = "feed_down"
)

var Q64BI = new(big.Int).Lsh(bignumber.One, 64)
