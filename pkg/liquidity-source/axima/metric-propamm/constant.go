package metricpropamm

const (
	DexType          = "metric-propamm"
	metadataPageSize = 50

	// statusFeedDown is bid_ask.priceProviderStatus while Metric has no price feed (quotes are empty).
	statusFeedDown = "feed_down"

	priceProviderMethodPrice = "price"
)

// priceFeedKey is the feed selector the quoter extension passes to
// priceProvider.price(key, pool). Observed as 2 in the on-chain swap path on
// Ethereum; Metric has not documented whether it ever varies per pool.
var priceFeedKey = [32]byte{31: 2}
