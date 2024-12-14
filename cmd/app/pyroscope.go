package main

import (
	"context"
	"os"
	"runtime"

	"github.com/KyberNetwork/kutils/klog"
	"github.com/KyberNetwork/kyber-trace-go/pkg/constant"
	"github.com/KyberNetwork/kyber-trace-go/pkg/util/env"
	"github.com/grafana/pyroscope-go"

	"github.com/KyberNetwork/router-service/internal/pkg/utils/envvar"
)

const (
	DefaultServiceName = "router-service"
)

func Pyroscope(ctx context.Context) (cleanUpFn func()) {
	if os.Getenv(envvar.PYROSCOPEEnabled) == "" {
		return func() {}
	}
	pyroscopeServer := os.Getenv(envvar.PYROSCOPEHost)
	if pyroscopeServer == "" {
		klog.Errorf(ctx, "Pyroscope|missing env var %s", envvar.PYROSCOPEHost)
		return func() {}
	}
	log := klog.DefaultLogger()
	_ = log.SetLogLevel("info")
	serviceName := env.StringFromEnv(constant.EnvKeyOtelServiceName, DefaultServiceName)

	runtime.SetBlockProfileRate(5e4)
	runtime.SetMutexProfileFraction(5e4)
	runtime.SetCPUProfileRate(5e4)
	profiler, err := pyroscope.Start(pyroscope.Config{
		ServerAddress: pyroscopeServer,
		Logger:        log,

		ApplicationName: serviceName,
		Tags: map[string]string{
			"hostname": serviceName,
			"env":      env.StringFromEnv(envvar.OTELEnv, os.Getenv("ENV")),
			"version":  env.StringFromEnv(constant.EnvKeyOtelServiceVersion, ""),
		},

		ProfileTypes: []pyroscope.ProfileType{
			// these profile types are enabled by default:
			pyroscope.ProfileCPU,
			// pyroscope.ProfileAllocObjects,
			pyroscope.ProfileAllocSpace,
			// pyroscope.ProfileInuseObjects,
			pyroscope.ProfileInuseSpace,

			// these profile types are optional:
			pyroscope.ProfileGoroutines,
			pyroscope.ProfileMutexCount,
			pyroscope.ProfileMutexDuration,
			pyroscope.ProfileBlockCount,
			pyroscope.ProfileBlockDuration,
		},
	})
	if err != nil {
		klog.Errorf(ctx, "Pyroscope|failed to start: %v", err)
	}
	return func() {
		if profiler == nil {
			return
		}
		if err := profiler.Stop(); err != nil {
			klog.Errorf(ctx, "Pyroscope|failed to stop: %v", err)
		}
	}
}
