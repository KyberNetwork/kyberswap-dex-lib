package main

import (
	"context"
	"os"
	"runtime"

	"github.com/KyberNetwork/kyber-trace-go/pkg/constant"
	"github.com/KyberNetwork/kyber-trace-go/pkg/util/env"
	"github.com/grafana/pyroscope-go"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

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
		log.Ctx(ctx).Error().Msgf("Pyroscope|missing env var %s", envvar.PYROSCOPEHost)
		return func() {}
	}
	serviceName := env.StringFromEnv(constant.EnvKeyOtelServiceName, DefaultServiceName)

	runtime.SetBlockProfileRate(5e4)
	runtime.SetMutexProfileFraction(5e4)
	runtime.SetCPUProfileRate(5e4)
	profiler, err := pyroscope.Start(pyroscope.Config{
		ServerAddress: pyroscopeServer,
		Logger:        logger{&log.Logger},

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
		log.Ctx(ctx).Err(err).Msg("Pyroscope|failed to start")
	}
	return func() {
		if profiler == nil {
			return
		}
		if err := profiler.Stop(); err != nil {
			log.Ctx(ctx).Err(err).Msg("Pyroscope|failed to stop")
		}
	}
}

type logger struct {
	*zerolog.Logger
}

func (l logger) Infof(msg string, args ...any) {
	l.Info().Msgf(msg, args...)
}

func (l logger) Debugf(msg string, args ...any) {
	l.Debug().Msgf(msg, args...)
}

func (l logger) Errorf(msg string, args ...any) {
	l.Error().Msgf(msg, args...)
}
