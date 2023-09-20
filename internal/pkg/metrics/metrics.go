package metrics

import (
	"context"
	"fmt"
	"strings"

	"github.com/KyberNetwork/kyber-trace-go/pkg/metric"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

const (
	DexHitRateMetricsName             = "dex_hit_rate.count"
	PoolTypeHitRateMetricsName        = "pool_hit_rate.count"
	RequestPairCountMetricsName       = "request_pair.count"
	FindRouteCacheCountMetricsName    = "find_route_cache.count"
	ClientIDMetricsName               = "client_id.count"
	InvalidSynthetixVolumeMetricsName = "invalid_synthetix_volume.count"
)

func IncrDexHitRate(dex string) {
	tags := []string{
		fmt.Sprintf("dex:%s", dex),
	}

	incr(DexHitRateMetricsName, tags, 0.1)
}

func IncrPoolTypeHitRate(poolType string) {
	tags := []string{
		fmt.Sprintf("pool_type:%s", poolType),
	}

	incr(PoolTypeHitRateMetricsName, tags, 0.1)
}

func IncrRequestPairCount(tokenInAddress, tokenOutAddress string) {
	tags := []string{
		fmt.Sprintf("token0:%s", tokenInAddress),
		fmt.Sprintf("token1:%s", tokenOutAddress),
	}

	incr(RequestPairCountMetricsName, tags, 0.5)
}

func IncrFindRouteCacheCount(cacheHit bool, otherTags []string) {
	tags := []string{
		fmt.Sprintf("hit:%t", cacheHit),
	}

	if len(otherTags) > 0 {
		tags = append(tags, otherTags...)
	}

	incr(FindRouteCacheCountMetricsName, tags, 1)
}

func IncrClientIDCount(clientID string, responseStatus int) {
	tags := []string{
		fmt.Sprintf("client_id:%s", clientID),
		fmt.Sprintf("http_status:%d", responseStatus),
	}

	incr(ClientIDMetricsName, tags, 1)
}

func IncrInvalidSynthetixVolume() {
	incr(InvalidSynthetixVolumeMetricsName, nil, 1)
}

func Flush() {
	// Flush VanPT
	if err := metric.Flush(context.Background()); err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Warn("failed to flush VanPT metrics")
	}

	// Flush DataDog
	if client == nil {
		return
	}

	if err := client.Flush(); err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Warn("failed to flush metrics")
	}
}

func incr(name string, tags []string, rate float64) {
	// Incr VanPT
	// VanPT doesn't accept "." in the counter name,
	// so replace all the current "." to "_".
	name = strings.Replace(name, ".", "_", -1)
	counter, err := metric.Meter().Float64Counter(name)
	if err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Warnf("failed to push %s metrics to VanPT", name)
	}
	ctx := context.Background()
	counter.Add(ctx, rate)
	metric.Flush(ctx)

	// Incr DataDog
	if client == nil {
		return
	}

	if err := client.Incr(name, tags, rate); err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Warnf("failed to push %s metrics", name)
	}
}

// NOTE: Still keep this unused function in case we further need to use gauge metrics
// nolint:golint,unused
func gauge(name string, value float64, tags []string, rate float64) {
	if client == nil {
		return
	}

	if err := client.Gauge(name, value, tags, rate); err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Warnf("failed to push %s metrics", name)
	}
}

// NOTE: Still keep this unused function in case we further need to use histogram metrics
// nolint:golint,unused
func histogram(name string, value float64, tags []string, rate float64) {
	if client == nil {
		return
	}

	if err := client.Histogram(name, value, tags, rate); err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Warnf("failed to push %s metrics", name)
	}
}
