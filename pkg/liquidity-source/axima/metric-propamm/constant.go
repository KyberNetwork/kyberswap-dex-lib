package metricpropamm

import "time"

const (
	DexType          = "metric-propamm"
	metadataPageSize = 50

	// extensionsTTL is how long a fetched extension count is reused; retry delay after a failed fetch.
	extensionsTTL   = 10 * time.Minute
	extensionsRetry = 30 * time.Second

	// statusFeedDown is bid_ask.priceProviderStatus while Metric has no price feed (quotes are empty).
	statusFeedDown = "feed_down"
)
