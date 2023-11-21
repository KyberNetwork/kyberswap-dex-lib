package envvar

const (
	OTELEnabled        = "OTEL_ENABLED"
	OTELService        = "OTEL_SERVICE_NAME"
	OTELEnv            = "OTEL_ENV"
	OTELServiceVersion = "OTEL_SERVICE_VERSION"
	OTELAgentHost      = "OTEL_AGENT_HOST"

	PYROSCOPEEnabled = "PYROSCOPE_ENABLED"
	PYROSCOPEHost    = "PYROSCOPE_HOST"

	DDProfilerEnabled = "DD_PROFILER_ENABLED"
	DDEnabled         = "DD_ENABLED"
	DDAgentHost       = "DD_AGENT_HOST"
	DDEnv             = "DD_ENV"
	DDService         = "DD_SERVICE"
	DDVersion         = "DD_VERSION"
	DDSamplerRate     = "DD_SAMPLER_RATE"
)
