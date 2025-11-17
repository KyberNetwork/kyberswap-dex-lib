package metrics

import "context"

const (
	MetricNameInvalidPoolTicks      = "invalid_pool_ticks_count"
	MetricNameMissingTrieNode       = "missing_trie_node_count"
	MetricNameUnprocessedEventTopic = "unprocessed_event_topic_count"
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
