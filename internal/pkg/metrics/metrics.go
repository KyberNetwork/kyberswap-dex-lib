package metrics

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"golang.org/x/exp/maps"

	kybermetric "github.com/KyberNetwork/kyber-trace-go/pkg/metric"
	"github.com/samber/lo"

	"github.com/KyberNetwork/router-service/pkg/logger"
)

const (
	DexHitRateMetricsName              = "dex_hit_rate.count"
	PoolTypeHitRateMetricsName         = "pool_hit_rate.count"
	RequestPairCountMetricsName        = "request_pair.count"
	FindRouteCacheCountMetricsName     = "find_route_cache.count"
	RequestCountMetricsName            = "request.count"
	InvalidSynthetixVolumeMetricsName  = "invalid_synthetix_volume.count"
	FindRoutePregenHitRateMetricsName  = "find_route_pregen.count"
	EstimateGasStatusMetricsName       = "estimate_gas.count"
	EstimateGasWithSlippageMetricsName = "estimate_gas_slippage"
)

var (
	dexHitRateCounter             metric.Float64Counter
	poolTypeHitRateCounter        metric.Float64Counter
	requestPairCountCounter       metric.Float64Counter
	findRouteCacheCounter         metric.Float64Counter
	requestCountCounter           metric.Float64Counter
	invalidSynthetixVolumeCounter metric.Float64Counter
	findRoutePregenHitRateCounter metric.Float64Counter
	estimateGasStatusCounter      metric.Float64Counter
	mapMetricNameToCounter        map[string]metric.Float64Counter

	// histogram metrics
	estimateGasSlippageHistogram metric.Int64Histogram
	mapMetricNameToHistogram     map[string]metric.Int64Histogram
)

func init() {
	dexHitRateCounter, _ = kybermetric.Meter().Float64Counter(formatMetricName(DexHitRateMetricsName))
	poolTypeHitRateCounter, _ = kybermetric.Meter().Float64Counter(formatMetricName(PoolTypeHitRateMetricsName))
	requestPairCountCounter, _ = kybermetric.Meter().Float64Counter(formatMetricName(RequestPairCountMetricsName))
	findRouteCacheCounter, _ = kybermetric.Meter().Float64Counter(formatMetricName(FindRouteCacheCountMetricsName))
	requestCountCounter, _ = kybermetric.Meter().Float64Counter(formatMetricName(RequestCountMetricsName))
	invalidSynthetixVolumeCounter, _ = kybermetric.Meter().Float64Counter(formatMetricName(InvalidSynthetixVolumeMetricsName))
	findRoutePregenHitRateCounter, _ = kybermetric.Meter().Float64Counter(formatMetricName(FindRoutePregenHitRateMetricsName))
	estimateGasStatusCounter, _ = kybermetric.Meter().Float64Counter(formatMetricName(EstimateGasStatusMetricsName))
	estimateGasSlippageHistogram, _ = kybermetric.Meter().Int64Histogram(formatMetricName(EstimateGasWithSlippageMetricsName))
	mapMetricNameToCounter = map[string]metric.Float64Counter{
		DexHitRateMetricsName:             dexHitRateCounter,
		PoolTypeHitRateMetricsName:        poolTypeHitRateCounter,
		RequestPairCountMetricsName:       requestPairCountCounter,
		FindRouteCacheCountMetricsName:    findRouteCacheCounter,
		RequestCountMetricsName:           requestCountCounter,
		InvalidSynthetixVolumeMetricsName: invalidSynthetixVolumeCounter,
		FindRoutePregenHitRateMetricsName: findRoutePregenHitRateCounter,
		EstimateGasStatusMetricsName:      estimateGasStatusCounter,
	}
	mapMetricNameToHistogram = map[string]metric.Int64Histogram{
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

func IncrRequestPairCount(ctx context.Context, tokenInAddress, tokenOutAddress string) {
	tags := map[string]string{
		"token0": tokenInAddress,
		"token1": tokenOutAddress,
	}

	incr(ctx, RequestPairCountMetricsName, tags, 0.5)
}

func IncrFindRoutePregenCount(ctx context.Context, pregenHit bool, otherTags map[string]string) {
	tags := map[string]string{
		"hit": strconv.FormatBool(pregenHit),
	}

	maps.Copy(tags, otherTags)

	incr(ctx, FindRoutePregenHitRateMetricsName, tags, 1)
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
	histogram(ctx, EstimateGasWithSlippageMetricsName, slippage, tags, 1)
}

func Flush() {
	// Flush VanPT
	if err := kybermetric.Flush(context.Background()); err != nil {
		logger.WithFieldsNonContext(logger.Fields{
			"error": err,
		}).Warn("failed to flush VanPT metrics")
	}

	// Flush DataDog
	if client == nil {
		return
	}

	if err := client.Flush(); err != nil {
		logger.WithFieldsNonContext(logger.Fields{
			"error": err,
		}).Warn("failed to flush metrics")
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

	// Incr DataDog
	if client == nil {
		return
	}

	ddTags := lo.MapToSlice(tags, func(k, v string) string {
		return fmt.Sprintf("%s:%s", k, v)
	})
	if err := client.Incr(name, ddTags, rate); err != nil {
		logger.WithFields(ctx, logger.Fields{
			"error": err,
		}).Warnf("failed to push %s metrics", name)
	}
}

// NOTE: Still keep this unused function in case we further need to use gauge metrics
// nolint:golint,unused
func gauge(ctx context.Context, name string, value float64, tags map[string]string, rate float64) {
	if client == nil {
		return
	}

	ddTags := lo.MapToSlice(tags, func(k, v string) string {
		return fmt.Sprintf("%s:%s", k, v)
	})
	if err := client.Gauge(name, value, ddTags, rate); err != nil {
		logger.WithFields(ctx, logger.Fields{
			"error": err,
		}).Warnf("failed to push %s metrics", name)
	}
}

// NOTE: Still keep this unused function in case we further need to use histogram metrics
// nolint:golint,unused
func histogram(ctx context.Context, name string, value float64, tags map[string]string, rate int64) {
	if histogramMetric, exist := mapMetricNameToHistogram[name]; histogramMetric != nil && exist {
		attributes := make([]attribute.KeyValue, 0, len(tags))
		for key, value := range tags {
			attributes = append(attributes, attribute.String(key, value))
		}
		histogramMetric.Record(context.Background(), rate, metric.WithAttributes(attributes...))
	} else {
		logger.Warnf(ctx, "histogram for %s metrics not found", name)
	}

	if client == nil {
		return
	}

	ddTags := lo.MapToSlice(tags, func(k, v string) string {
		return fmt.Sprintf("%s:%s", k, v)
	})
	if err := client.Histogram(name, value, ddTags, float64(rate)); err != nil {
		logger.WithFields(ctx, logger.Fields{
			"error": err,
		}).Warnf("failed to push %s metrics", name)
	}
}

func formatMetricName(name string) string {
	// VanPT doesn't accept "." in the metric name,
	// so replace all the current "." to "_".
	return strings.Replace(name, ".", "_", -1)
}
