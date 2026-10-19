package metricpropamm

import "time"

const (
	DexType          = "metric-propamm"
	metadataPageSize = 50

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
