package envvar

const (
	Env = "ENV"

	LogLevel = "LOG_LEVEL"

	HttpProxy  = "HTTP_PROXY"
	HttpsProxy = "HTTPS_PROXY"

	RedisHost         = "REDIS_HOST"
	RedisPort         = "REDIS_PORT"
	RedisDBNumber     = "REDIS_DBNUMBER"
	RedisPassword     = "REDIS_PASSWORD"
	RedisPrefix       = "REDIS_PREFIX"
	RedisReadTimeout  = "REDIS_READ_TIMEOUT"
	RedisWriteTimeout = "REDIS_WRITE_TIMEOUT"

	RedisReplicaHost         = "REDIS_REPLICA_HOST"
	RedisReplicaPort         = "REDIS_REPLICA_PORT"
	RedisReplicaDBNumber     = "REDIS_REPLICA_DBNUMBER"
	RedisReplicaPassword     = "REDIS_REPLICA_PASSWORD"
	RedisReplicaPrefix       = "REDIS_REPLICA_PREFIX"
	RedisReplicaReadTimeout  = "REDIS_REPLICA_READ_TIMEOUT"
	RedisReplicaWriteTimeout = "REDIS_REPLICA_WRITE_TIMEOUT"

	RedisSentinelMasterName   = "REDIS_SENTINEL_MASTERNAME"
	RedisSentinelPort         = "REDIS_SENTINEL_PORT"
	RedisSentinelDBNumber     = "REDIS_SENTINEL_DBNUMBER"
	RedisSentinelPassword     = "REDIS_SENTINEL_PASSWORD"
	RedisSentinelPrefix       = "REDIS_SENTINEL_PREFIX"
	RedisSentinelReadTimeout  = "REDIS_SENTINEL_READ_TIMEOUT"
	RedisSentinelWriteTimeout = "REDIS_SENTINEL_WRITE_TIMEOUT"

	PublicRPC = "PUBLIC_RPC"
	RPCs      = "RPCS"

	LogSentryDSN = "LOG_SENTRY_DSN"

	DDEnabled     = "DD_ENABLED"
	DDAgentHost   = "DD_AGENT_HOST"
	DDEnv         = "DD_ENV"
	DDService     = "DD_SERVICE"
	DDVersion     = "DD_VERSION"
	DDSamplerRate = "DD_SAMPLER_RATE"

	RouterAddress   = "ROUTER_ADDRESS"
	ExecutorAddress = "EXECUTOR_ADDRESS"

	MinLiquidityUsd = "MIN_LIQUIDITY_USD"

	AllowSubgraphError = "ALLOW_SUBGRAPH_ERROR"

	BlacklistedPools = "BLACKLISTED_POOLS"

	KeyPairStorageFilePath           = "KEY_PAIR_STORAGE_FILE_PATH"
	KeyPairKeyIDForSealingClientData = "KEY_PAIR_KEY_ID_FOR_SEALING_DATA_CLIENT_DATA"

	TokenCatalogHTTPURL = "TOKEN_CATALOG_HTTP_URL"

	ReloadConfigHTTPURL = "RELOAD_CONFIG__HTTP_URL"
)
