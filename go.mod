module github.com/KyberNetwork/router-service

go 1.21.0

toolchain go1.21.1

replace (
	github.com/daoleno/uniswap-sdk-core v0.1.5 => github.com/KyberNetwork/uniswap-sdk-core v0.1.5
	github.com/daoleno/uniswapv3-sdk v0.4.0 => github.com/KyberNetwork/uniswapv3-sdk v0.4.3
)

require (
	github.com/DataDog/datadog-go v4.8.2+incompatible
	github.com/KyberNetwork/aevm v1.0.6
	github.com/KyberNetwork/ethrpc v0.3.0
	github.com/KyberNetwork/kyber-trace-go v0.1.0
	github.com/KyberNetwork/kyberswap-dex-lib v0.18.0
	github.com/KyberNetwork/logger v0.1.0
	github.com/KyberNetwork/reload v0.1.1
	github.com/alicebob/miniredis v2.5.0+incompatible
	github.com/cenkalti/backoff/v4 v4.2.1
	github.com/cespare/xxhash/v2 v2.2.0
	github.com/dgraph-io/ristretto v0.1.1
	github.com/envoyproxy/protoc-gen-validate v0.10.1
	github.com/ethereum/go-ethereum v1.12.0
	github.com/getsentry/sentry-go v0.18.0
	github.com/gin-contrib/cors v1.3.1
	github.com/gin-contrib/pprof v1.4.0
	github.com/gin-contrib/requestid v0.0.6
	github.com/gin-gonic/gin v1.9.1
	github.com/go-redis/cache/v9 v9.0.0
	github.com/go-resty/resty/v2 v2.7.0
	github.com/golang/mock v1.6.0
	github.com/google/uuid v1.3.0
	github.com/hashicorp/golang-lru/v2 v2.0.2
	github.com/holiman/uint256 v1.2.2
	github.com/huandu/go-clone v1.6.0
	github.com/json-iterator/go v1.1.12
	github.com/mcuadros/go-defaults v1.2.0
	github.com/oleiade/lane v1.0.1
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/pkg/errors v0.9.1
	github.com/redis/go-redis/v9 v9.0.3
	github.com/samber/lo v1.38.1
	github.com/sirupsen/logrus v1.9.0
	github.com/sourcegraph/conc v0.3.0
	github.com/spf13/viper v1.16.0
	github.com/stretchr/testify v1.8.4
	github.com/tdewolff/minify/v2 v2.12.4
	github.com/urfave/cli/v2 v2.17.2-0.20221006022127-8f469abc00aa
	go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin v0.43.0
	go.opentelemetry.io/otel v1.17.0
	go.opentelemetry.io/otel/metric v1.17.0
	go.opentelemetry.io/otel/trace v1.17.0
	go.uber.org/automaxprocs v1.5.2
	go.uber.org/zap v1.24.0
	golang.org/x/sync v0.2.0
	google.golang.org/grpc v1.57.0
	google.golang.org/grpc/cmd/protoc-gen-go-grpc v1.2.0
	google.golang.org/protobuf v1.31.0
	gopkg.in/DataDog/dd-trace-go.v1 v1.51.0
	gopkg.in/natefinch/lumberjack.v2 v2.2.1
	k8s.io/apimachinery v0.23.17
)

require github.com/tdewolff/parse/v2 v2.6.6 // indirect

require (
	github.com/DataDog/appsec-internal-go v1.0.0 // indirect
	github.com/DataDog/datadog-agent/pkg/obfuscate v0.45.0-rc.1 // indirect
	github.com/DataDog/datadog-agent/pkg/remoteconfig/state v0.45.0-rc.1 // indirect
	github.com/DataDog/datadog-go/v5 v5.1.1 // indirect
	github.com/DataDog/go-libddwaf v1.2.0 // indirect
	github.com/DataDog/go-tuf v0.3.0--fix-localmeta-fork // indirect
	github.com/DataDog/gostackparse v0.5.0 // indirect
	github.com/DataDog/sketches-go v1.2.1 // indirect
	github.com/KyberNetwork/blockchain-toolkit v0.2.0 // indirect
	github.com/KyberNetwork/elastic-go-sdk/v2 v2.0.2 // indirect
	github.com/KyberNetwork/pancake-v3-sdk v0.1.0 // indirect
	github.com/Microsoft/go-winio v0.5.1 // indirect
	github.com/alicebob/gopher-json v0.0.0-20200520072559-a9ecdc9d1d3a // indirect
	github.com/btcsuite/btcd/btcec/v2 v2.2.0 // indirect
	github.com/bytedance/sonic v1.9.1 // indirect
	github.com/chenzhuoyu/base64x v0.0.0-20221115062448-fe3a3abad311 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.2 // indirect
	github.com/daoleno/uniswap-sdk-core v0.1.7 // indirect
	github.com/daoleno/uniswapv3-sdk v0.4.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/deckarep/golang-set/v2 v2.1.0 // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.0.1 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/dustin/go-humanize v1.0.0 // indirect
	github.com/fsnotify/fsnotify v1.6.0 // indirect
	github.com/gabriel-vasile/mimetype v1.4.2 // indirect
	github.com/gin-contrib/sse v0.1.0 // indirect
	github.com/go-logr/logr v1.2.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator/v10 v10.14.0 // indirect
	github.com/go-stack/stack v1.8.1 // indirect
	github.com/goccy/go-json v0.10.2 // indirect
	github.com/golang/glog v1.1.0 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/gomodule/redigo v1.8.9 // indirect
	github.com/google/pprof v0.0.0-20211214055906-6f57359322fd // indirect
	github.com/gorilla/websocket v1.5.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.7.0 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/izumiFinance/iZiSwap-SDK-go v1.0.0 // indirect
	github.com/klauspost/compress v1.15.15 // indirect
	github.com/klauspost/cpuid/v2 v2.2.4 // indirect
	github.com/leodido/go-urn v1.2.4 // indirect
	github.com/machinebox/graphql v0.2.2 // indirect
	github.com/magiconair/properties v1.8.7 // indirect
	github.com/mattn/go-isatty v0.0.19 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/orcaman/concurrent-map v1.0.0 // indirect
	github.com/outcaste-io/ristretto v0.2.1 // indirect
	github.com/pelletier/go-toml/v2 v2.0.8 // indirect
	github.com/philhofer/fwd v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/richardartoul/molecule v1.0.1-0.20221107223329-32cfee06a052 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/secure-systems-lab/go-securesystemslib v0.5.0 // indirect
	github.com/shirou/gopsutil v3.21.11+incompatible // indirect
	github.com/shopspring/decimal v1.3.1 // indirect
	github.com/spaolacci/murmur3 v1.1.0 // indirect
	github.com/spf13/afero v1.9.5 // indirect
	github.com/spf13/cast v1.5.1 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/subosito/gotenv v1.4.2 // indirect
	github.com/tinylib/msgp v1.1.6 // indirect
	github.com/tklauser/go-sysconf v0.3.10 // indirect
	github.com/tklauser/numcpus v0.5.0 // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	github.com/ugorji/go/codec v1.2.11 // indirect
	github.com/vmihailenco/go-tinylfu v0.2.2 // indirect
	github.com/vmihailenco/msgpack/v5 v5.3.5 // indirect
	github.com/vmihailenco/tagparser/v2 v2.0.0 // indirect
	github.com/xrash/smetrics v0.0.0-20201216005158-039620a65673 // indirect
	github.com/yuin/gopher-lua v0.0.0-20220504180219-658193537a64 // indirect
	github.com/yusufpapurcu/wmi v1.2.2 // indirect
	go.opentelemetry.io/otel/exporters/otlp/internal/retry v1.16.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric v0.39.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc v0.39.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp v0.39.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.16.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.16.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.16.0 // indirect
	go.opentelemetry.io/otel/sdk v1.16.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v0.39.0 // indirect
	go.opentelemetry.io/proto/otlp v0.19.0 // indirect
	go.uber.org/atomic v1.10.0 // indirect
	go.uber.org/multierr v1.9.0 // indirect
	go4.org/intern v0.0.0-20211027215823-ae77deb06f29 // indirect
	go4.org/unsafe/assume-no-moving-gc v0.0.0-20220617031537-928513b29760 // indirect
	golang.org/x/arch v0.3.0 // indirect
	golang.org/x/crypto v0.12.0 // indirect
	golang.org/x/exp v0.0.0-20230206171751-46f607a40771 // indirect
	golang.org/x/net v0.14.0 // indirect
	golang.org/x/sys v0.11.0 // indirect
	golang.org/x/text v0.12.0 // indirect
	golang.org/x/time v0.1.0 // indirect
	golang.org/x/xerrors v0.0.0-20220907171357-04be3eba64a2 // indirect
	google.golang.org/genproto v0.0.0-20230526161137-0005af68ea54 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20230525234035-dd9d682886f9 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20230525234030-28d5490b6b19 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/natefinch/npipe.v2 v2.0.0-20160621034901-c1b8fa8bdcce // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	inet.af/netaddr v0.0.0-20220811202034-502d2d690317 // indirect
)
