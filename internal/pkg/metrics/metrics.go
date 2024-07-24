package metrics

import (
	"context"
	"math"
	"strconv"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"golang.org/x/exp/maps"

	kybermetric "github.com/KyberNetwork/kyber-trace-go/pkg/metric"

	"github.com/KyberNetwork/router-service/pkg/logger"
)

const (
	DexHitRateMetricsName              = "dex_hit_rate_count"
	PoolTypeHitRateMetricsName         = "pool_hit_rate_count"
	FindRouteCacheCountMetricsName     = "find_route_cache_count"
	RequestCountMetricsName            = "request_count"
	InvalidSynthetixVolumeMetricsName  = "invalid_synthetix_volume_count"
	EstimateGasStatusMetricsName       = "estimate_gas_count"
	EstimateGasWithSlippageMetricsName = "estimate_gas_slippage"
	IndexPoolsMetricsCounterName       = "index_pools_count"
	ClonePoolPanicMetricsName          = "clone_pool_panic_count"
	IndexPoolsDelayMetricsName         = "index_job_pools_delay"

	CalcAmountOutCountPerRequest = "calc_amount_out_count_per_request"
)

var (
	dexHitRateCounter             metric.Float64Counter
	poolTypeHitRateCounter        metric.Float64Counter
	findRouteCacheCounter         metric.Float64Counter
	requestCountCounter           metric.Float64Counter
	invalidSynthetixVolumeCounter metric.Float64Counter
	estimateGasStatusCounter      metric.Float64Counter
	mapMetricNameToCounter        map[string]metric.Float64Counter
	indexPoolsDelayCounter        metric.Float64Counter
	clonePoolPanicCounter         metric.Float64Counter

	// histogram metrics
	indexPoolsDelayHistogram     metric.Int64Histogram
	estimateGasSlippageHistogram metric.Float64Histogram

	calcAmountOutCountPerRequestHistogram metric.Int64Histogram

	mapMetricNameToHistogram        map[string]metric.Int64Histogram
	mapMetricNameToFloat64Histogram map[string]metric.Float64Histogram
)

func init() {
	dexHitRateCounter, _ = kybermetric.Meter().Float64Counter(DexHitRateMetricsName)
	poolTypeHitRateCounter, _ = kybermetric.Meter().Float64Counter(PoolTypeHitRateMetricsName)
	findRouteCacheCounter, _ = kybermetric.Meter().Float64Counter(FindRouteCacheCountMetricsName)
	requestCountCounter, _ = kybermetric.Meter().Float64Counter(RequestCountMetricsName)
	invalidSynthetixVolumeCounter, _ = kybermetric.Meter().Float64Counter(InvalidSynthetixVolumeMetricsName)
	estimateGasStatusCounter, _ = kybermetric.Meter().Float64Counter(EstimateGasStatusMetricsName)
	estimateGasSlippageHistogram, _ = kybermetric.Meter().Float64Histogram(EstimateGasWithSlippageMetricsName)
	indexPoolsDelayHistogram, _ = kybermetric.Meter().Int64Histogram(IndexPoolsDelayMetricsName,
		metric.WithExplicitBucketBoundaries(0, 50, 300, 1200, 2500, 5000, 10e3, 30e3, 90e3, 300e3, 1200e3, 3600e3))
	indexPoolsDelayCounter, _ = kybermetric.Meter().Float64Counter(IndexPoolsMetricsCounterName)
	clonePoolPanicCounter, _ = kybermetric.Meter().Float64Counter(ClonePoolPanicMetricsName)
	calcAmountOutCountPerRequestHistogram, _ = kybermetric.Meter().Int64Histogram(CalcAmountOutCountPerRequest)
	metric.WithExplicitBucketBoundaries(1, 10, 100, 1000, math.Inf(1))

	mapMetricNameToCounter = map[string]metric.Float64Counter{
		DexHitRateMetricsName:             dexHitRateCounter,
		PoolTypeHitRateMetricsName:        poolTypeHitRateCounter,
		FindRouteCacheCountMetricsName:    findRouteCacheCounter,
		RequestCountMetricsName:           requestCountCounter,
		InvalidSynthetixVolumeMetricsName: invalidSynthetixVolumeCounter,
		EstimateGasStatusMetricsName:      estimateGasStatusCounter,
		IndexPoolsMetricsCounterName:      indexPoolsDelayCounter,
		ClonePoolPanicMetricsName:         clonePoolPanicCounter,
	}
	mapMetricNameToHistogram = map[string]metric.Int64Histogram{
		IndexPoolsDelayMetricsName: indexPoolsDelayHistogram,
	}
	mapMetricNameToFloat64Histogram = map[string]metric.Float64Histogram{
		EstimateGasWithSlippageMetricsName: estimateGasSlippageHistogram,
	}

}

func IncrDexHitRate(ctx context.Context, dex string) {
	tags := map[string]string{
		"dex": dex,
	}

	incr(ctx, DexHitRateMetricsName, tags, 0.1)
}

func IncrPoolTypeHitRate(ctx context.Context, poolType string) {
	tags := map[string]string{
		"pool_type": poolType,
	}

	incr(ctx, PoolTypeHitRateMetricsName, tags, 0.1)
}

func IncrIndexPoolsCounter(ctx context.Context, jobName string, isSuccess bool, counter int) {
	state := "failed"
	if isSuccess {
		state = "success"
	}
	incr(ctx, IndexPoolsMetricsCounterName, map[string]string{
		"job_name": jobName,
		"state":    state,
	}, float64(counter))
}

func IncrClonePoolPanicCounter(ctx context.Context) {
	incr(ctx, ClonePoolPanicMetricsName, nil, 1)
}

func IncrFindRouteCacheCount(ctx context.Context, cacheHit bool, otherTags map[string]string) {
	tags := map[string]string{
		"hit": strconv.FormatBool(cacheHit),
	}

	maps.Copy(tags, otherTags)

	incr(ctx, FindRouteCacheCountMetricsName, tags, 1)
}

func IncrRequestCount(ctx context.Context, clientID string, responseStatus int) {
	tags := map[string]string{
		"client_id":   clientID,
		"http_status": strconv.FormatInt(int64(responseStatus), 10),
	}

	incr(ctx, RequestCountMetricsName, tags, 1)
}

func IncrInvalidSynthetixVolume(ctx context.Context) {
	incr(ctx, InvalidSynthetixVolumeMetricsName, nil, 1)
}

func IncrEstimateGas(ctx context.Context, isSuccess bool, dexID string, clientId string) {
	state := "success"
	if !isSuccess {
		state = "failed"
	}
	tags := map[string]string{
		"dex_id":    dexID,
		"state":     state,
		"client_id": clientId,
	}

	incr(ctx, EstimateGasStatusMetricsName, tags, 1)
}

func HistogramEstimateGasWithSlippage(ctx context.Context, slippage float64, isSuccess bool) {
	state := "success"
	if !isSuccess {
		state = "failed"
	}
	tags := map[string]string{
		"state": state,
	}
	histogram(ctx, EstimateGasWithSlippageMetricsName, slippage, tags)
}

func HistogramIndexPoolsDelay(ctx context.Context, jobName string, delay time.Duration, isSuccess bool) {
	state := "failed"
	if isSuccess {
		state = "success"
	}
	delayMs := delay.Milliseconds()
	histogram(ctx, IndexPoolsDelayMetricsName, delayMs, map[string]string{
		"job_name": jobName,
		"state":    state,
	})
}

func HistogramCalcAmountOutCountPerRequest(ctx context.Context, count int64, dexType string) {
	calcAmountOutCountPerRequestHistogram.Record(ctx, count, metric.WithAttributes(attribute.String("dexType", dexType)))
}

func Flush() {
	// Flush VanPT
	if err := kybermetric.Flush(context.Background()); err != nil {
		logger.WithFieldsNonContext(logger.Fields{
			"error": err,
		}).Warn("failed to flush VanPT metrics")
	}
}

func incr(ctx context.Context, name string, tags map[string]string, rate float64) {
	// Incr VanPT
	if counter, exist := mapMetricNameToCounter[name]; counter != nil && exist {
		attributes := make([]attribute.KeyValue, 0, len(tags))
		for key, value := range tags {
			attributes = append(attributes, attribute.String(key, value))
		}
		counter.Add(context.Background(), rate, metric.WithAttributes(attributes...))
	} else {
		logger.Warnf(ctx, "counter for %s metrics not found", name)
	}
}

func histogram[T int64 | float64](ctx context.Context, name string, value T, tags map[string]string) {
	attributes := make([]attribute.KeyValue, 0, len(tags))
	for key, value := range tags {
		attributes = append(attributes, attribute.String(key, value))
	}

	switch val := any(value).(type) {
	case int64:
		if histogramMetric, exist := mapMetricNameToHistogram[name]; histogramMetric != nil && exist {
			histogramMetric.Record(context.Background(), val, metric.WithAttributes(attributes...))
		} else {
			logger.Warnf(ctx, "int64histogram for %s metrics not found", name)
		}
	case float64:
		if histogramMetric, exist := mapMetricNameToFloat64Histogram[name]; histogramMetric != nil && exist {
			histogramMetric.Record(context.Background(), val, metric.WithAttributes(attributes...))
		} else {
			logger.Warnf(ctx, "float64histogram for %s metrics not found", name)
		}
	}

}
