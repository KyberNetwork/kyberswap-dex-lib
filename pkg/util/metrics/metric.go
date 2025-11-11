package metrics

import "context"

const (
	MetricNameInvalidPoolTicks      = "invalid_pool_ticks_count"
	MetricNameMissingTrieNode       = "missing_trie_node_count"
	MetricNameUnprocessedEventTopic = "unprocessed_event_topic_count"
)

type IncrFnType func(ctx context.Context, name string, tags map[string]string, value float64)

var (
	IncrFn IncrFnType = func(context.Context, string, map[string]string, float64) {}
)

func SetIncrFn(fn IncrFnType) {
	if fn != nil {
		IncrFn = fn
	}
}

func IncrInvalidPoolTicks(exchange string) {
	if IncrFn != nil {
		IncrFn(context.Background(), MetricNameInvalidPoolTicks, map[string]string{
			"exchange": exchange,
		}, 1)
	}
}

func IncrMissingTrieNode() {
	if IncrFn != nil {
		IncrFn(context.Background(), MetricNameMissingTrieNode, nil, 1)
	}
}

func IncrUnprocessedEventTopic(poolType string, topic string) {
	if IncrFn != nil {
		IncrFn(context.Background(), MetricNameUnprocessedEventTopic, map[string]string{
			"poolType": poolType,
			"topic":    topic,
		}, 0.1)
	}
}
