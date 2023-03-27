package metrics

import (
	"fmt"
	"time"

	"github.com/KyberNetwork/kyberswap-aggregator/pkg/logger"
)

const (
	DexHitRateMetricsName                = "dex_hit_rate.count"
	PoolTypeHitRateMetricsName           = "pool_hit_rate.count"
	RequestPairCountMetricsName          = "request_pair.count"
	FindRouteCacheCountMetricsName       = "find_route_cache.count"
	AggregatorScanLatestBlockMetricsName = "aggregator_scan_latest_block"
	InvalidSynthetixVolumeMetricsName    = "invalid_synthetix_volume.count"

	HistogramScannerUpdateReservesDurationMetricsName = "scanner_update_reserves_duration"
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

func IncrRequestPairCount(tokenInAddress, tokenOutAddress, amountIn string) {
	tags := []string{
		fmt.Sprintf("pair:%s-%s", tokenInAddress, tokenOutAddress),
		fmt.Sprintf("token0:%s", tokenInAddress),
		fmt.Sprintf("token1:%s", tokenOutAddress),
		//fmt.Sprintf("amountIn:%s", amountIn),
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

func IncrInvalidSynthetixVolume() {
	incr(InvalidSynthetixVolumeMetricsName, nil, 1)
}

func GaugeAggregatorScanLatestBlock(maxBlocks uint64) {
	gauge(AggregatorScanLatestBlockMetricsName, float64(maxBlocks), nil, 1)
}

func HistogramScannerUpdateReservesDuration(duration time.Duration, dex string, poolCount int) {
	tags := []string{
		fmt.Sprintf("dex:%s", dex),
		fmt.Sprintf("poolCount:%d", poolCount),
	}

	histogram(HistogramScannerUpdateReservesDurationMetricsName, float64(duration.Milliseconds()), tags, 1)
}

func Flush() {
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
	if client == nil {
		return
	}

	if err := client.Incr(name, tags, rate); err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Warnf("failed to push %s metrics", name)
	}
}

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
