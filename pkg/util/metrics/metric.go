package metrics

import "context"

const (
	MetricNameInvalidPoolTicks      = "invalid_pool_ticks_count"
	MetricNameMissingTrieNode       = "missing_trie_node_count"
	MetricNameUnprocessedEventTopic = "unprocessed_event_topic_count"

	MetricNameDiscardedStrategy     = "discarded_strategy_count"
	MetricNameTokenUnprobed         = "token_unprobed"
	MetricNameTokenUnprobedFillable = "token_unprobed_fillable"
	MetricNameSkippedBlocks         = "skipped_unservable_blocks"
	MetricNamePortalFallback        = "portal_fallback"
)

type incrFnType func(ctx context.Context, name string, tags map[string]string, value float64)

var (
	incrFn incrFnType = func(context.Context, string, map[string]string, float64) {}
)

func SetIncrFn(fn incrFnType) {
	if fn != nil {
		incrFn = fn
	}
}

func IncrInvalidPoolTicks(exchange string) {
	if incrFn != nil {
		incrFn(context.Background(), MetricNameInvalidPoolTicks, map[string]string{
			"exchange": exchange,
		}, 1)
	}
}

func IncrMissingTrieNode() {
	if incrFn != nil {
		incrFn(context.Background(), MetricNameMissingTrieNode, nil, 1)
	}
}

func IncrUnprocessedEventTopic(poolType string, topic string) {
	if incrFn != nil {
		incrFn(context.Background(), MetricNameUnprocessedEventTopic, map[string]string{
			"poolType": poolType,
			"topic":    topic,
		}, 0.1)
	}
}

// IncrDiscardedStrategy counts the strategies a lister refused to turn into a pool, by discovery source
// ("logs" or "portal") and reason.
func IncrDiscardedStrategy(exchange, source, reason string, count float64) {
	if incrFn != nil && count > 0 {
		incrFn(context.Background(), MetricNameDiscardedStrategy, map[string]string{
			"exchange": exchange,
			"source":   source,
			"reason":   reason,
		}, count)
	}
}

// IncrPortalFallback counts a lister run that fell back from the Portal to the on-chain log scan, by reason.
func IncrPortalFallback(exchange, reason string) {
	if incrFn != nil {
		incrFn(context.Background(), MetricNamePortalFallback, map[string]string{
			"exchange": exchange,
			"reason":   reason,
		}, 1)
	}
}

// IncrTokenUnprobed counts a strategy left out because a token is not on the probed list, and sums the
// fillable amount (in the token's smallest unit) that this leaves out.
func IncrTokenUnprobed(exchange, token string, fillable float64) {
	if incrFn != nil {
		tags := map[string]string{"exchange": exchange, "token": token}
		incrFn(context.Background(), MetricNameTokenUnprobed, tags, 1)
		if fillable > 0 {
			incrFn(context.Background(), MetricNameTokenUnprobedFillable, tags, fillable)
		}
	}
}

// IncrSkippedBlocks counts blocks a log scan skipped because the node could not serve them.
func IncrSkippedBlocks(exchange string, blocks float64) {
	if incrFn != nil {
		incrFn(context.Background(), MetricNameSkippedBlocks, map[string]string{"exchange": exchange}, blocks)
	}
}
