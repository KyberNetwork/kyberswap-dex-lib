package metrics

import (
	"context"
	"math"
	"strconv"
	"time"

	"github.com/KyberNetwork/kutils"
	"github.com/KyberNetwork/kutils/klog"
	kybermetric "github.com/KyberNetwork/kyber-trace-go/pkg/metric"
	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var (
	meter = kybermetric.Meter()

	counterDexHitRate, _             = meter.Int64Counter("dex_hit_rate_count")
	counterPoolTypeHitRate, _        = meter.Int64Counter("pool_hit_rate_count")
	counterRequestCount, _           = meter.Int64Counter("request_count")
	counterInvalidSynthetixVolume, _ = meter.Int64Counter("invalid_synthetix_volume_count")
	counterEstimateGasStatus, _      = meter.Int64Counter("estimate_gas_count")
	counterIndexPoolsDelay, _        = meter.Int64Counter("index_pools_count")
	counterClonePoolPanic, _         = meter.Int64Counter("clone_pool_panic_count")
	counterResourceHit, _            = meter.Int64Counter("resource_hit_counter")

	histogramEstimateGasSlippage, _ = meter.Float64Histogram("estimate_gas_slippage")
	histogramIndexPoolsDelay, _     = meter.Int64Histogram("index_job_pools_delay",
		metric.WithExplicitBucketBoundaries(0, 50, 300, 1200, 2500, 5000, 10e3, 30e3, 90e3, 300e3, 1200e3, 3600e3))
	histogramCalcAmountOutCountPerRequest, _ = meter.Int64Histogram("calc_amount_out_count_per_request",
		metric.WithExplicitBucketBoundaries(1, 10, 100, 1000, math.Inf(1)))
	histogramCalcAmountOutDuration, _    = meter.Float64Histogram("calc_amount_out_ms")
	histogramAEVMMultipleCallDuration, _ = meter.Float64Histogram("aevm_multiple_call_duration_ms")
	histogramClonePoolDuration, _        = meter.Float64Histogram("clone_pool_ms")
)

func CountDexHit(ctx context.Context, dex string) {
	add(ctx, counterDexHitRate, 1, "dex", dex)
}

func CountPoolTypeHit(ctx context.Context, poolType string) {
	add(ctx, counterPoolTypeHitRate, 1, "pool_type", poolType)
}

func CountIndexPools(ctx context.Context, jobName string, isSuccess bool, counter int) {
	add(ctx, counterIndexPoolsDelay, int64(counter),
		"job_name", jobName, "state", lo.Ternary(isSuccess, "success", "failed"))
}

func CountClonePoolPanic(ctx context.Context) {
	add(ctx, counterClonePoolPanic, 1)
}

func CountFindRouteCache(ctx context.Context, cacheHit bool, kvTags ...string) {
	countResourceHit(ctx, "route", "redis", 1, cacheHit, kvTags...)
}

func CountRequest(ctx context.Context, clientID, ja4 string, responseStatus int) {
	add(ctx, counterRequestCount, 1, "client_id", clientID, "ja4", ja4,
		"http_status", kutils.Itoa(responseStatus))
}

func CountInvalidSynthetixVolume(ctx context.Context) {
	add(ctx, counterInvalidSynthetixVolume, 1)
}

func CountEstimateGas(ctx context.Context, isSuccess bool, dexID string, clientId string) {
	add(ctx, counterEstimateGasStatus, 1,
		"dex_id", dexID, "state", lo.Ternary(isSuccess, "success", "failed"), "client_id", clientId)
}

func RecordEstimateGasWithSlippage(ctx context.Context, slippage float64, isSuccess bool) {
	record(ctx, histogramEstimateGasSlippage, slippage, "state", lo.Ternary(isSuccess, "success", "failed"))
}

func RecordIndexPoolsDelay(ctx context.Context, jobName string, delay time.Duration, isSuccess bool) {
	record(ctx, histogramIndexPoolsDelay, delay.Milliseconds(),
		"job_name", jobName, "state", lo.Ternary(isSuccess, "success", "failed"))
}

func RecordCalcAmountOutCountPerRequest(ctx context.Context, count int64, dexType string) {
	record(ctx, histogramCalcAmountOutCountPerRequest, count, "dexType", dexType)
}

func RecordCalcAmountOutDuration(ctx context.Context, duration time.Duration, dexUseAEVM bool, dex string) {
	record(ctx, histogramCalcAmountOutDuration, float64(duration.Nanoseconds())/1e6,
		"dexUseAEVM", strconv.FormatBool(dexUseAEVM), "dex", dex)
}

func RecordAEVMMultipleCallDuration(ctx context.Context, duration time.Duration) {
	record(ctx, histogramAEVMMultipleCallDuration, float64(duration.Nanoseconds())/1e6)
}

func RecordClonePoolDuration(ctx context.Context, duration time.Duration, dex string) {
	record(ctx, histogramClonePoolDuration, float64(duration.Nanoseconds())/1e6, "dex", dex)
}

func countResourceHit(ctx context.Context, resourceType, scope string, count int64, isHit bool, kvTags ...string) {
	kvTags = append(kvTags, "hit", lo.Ternary(isHit, "true", "false"), "type", resourceType, "scope", scope)
	add(ctx, counterResourceHit, count, kvTags...)
}

func CountTokenHitLocalCache(ctx context.Context, count int64, isHit bool) {
	countResourceHit(ctx, "token", "localCache", count, isHit)
}

func CountPriceHitLocalCache(ctx context.Context, count int64, isHit bool) {
	countResourceHit(ctx, "price", "localCache", count, isHit)
}

func CountTokenInfoHitLocalCache(ctx context.Context, count int64, isHit bool) {
	countResourceHit(ctx, "tokenInfo", "localCache", count, isHit)
}

func Flush() {
	if err := kybermetric.Flush(context.Background()); err != nil {
		klog.WithFields(context.Background(), klog.Fields{
			"error": err,
		}).Warn("failed to flush VanPT metrics")
	}
}

func add[T int64 | float64](ctx context.Context, counter interface {
	Add(ctx context.Context, incr T, options ...metric.AddOption)
}, rate T, tagKvs ...string) {
	if counter == nil {
		return
	}
	attributes := make([]attribute.KeyValue, len(tagKvs)/2)
	for i := range attributes {
		attributes[i] = attribute.String(tagKvs[i*2], tagKvs[i*2+1])
	}
	counter.Add(ctx, rate, metric.WithAttributeSet(attribute.NewSet(attributes...)))
}

func record[T int64 | float64](ctx context.Context, histogram interface {
	Record(ctx context.Context, incr T, options ...metric.RecordOption)
}, incr T, tagKvs ...string) {
	if histogram == nil {
		return
	}
	attributes := make([]attribute.KeyValue, len(tagKvs)/2)
	for i := range attributes {
		attributes[i] = attribute.String(tagKvs[i*2], tagKvs[i*2+1])
	}
	histogram.Record(ctx, incr, metric.WithAttributes(attributes...))
}
