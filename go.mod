module github.com/KyberNetwork/router-service

go 1.22.7

replace (
	github.com/daoleno/uniswapv3-sdk v0.4.0 => github.com/KyberNetwork/uniswapv3-sdk v0.5.2
	github.com/ethereum/go-ethereum => github.com/ethereum/go-ethereum v1.13.14
)

require (
	github.com/ALTree/bigfloat v0.2.0
	github.com/KyberNetwork/aevm v1.1.18
	github.com/KyberNetwork/aggregator-encoding v0.32.3
	github.com/KyberNetwork/blackjack v0.3.0
	github.com/KyberNetwork/blockchain-toolkit v0.8.1
	github.com/KyberNetwork/elastic-go-sdk/v2 v2.0.4
	github.com/KyberNetwork/ethrpc v0.7.3
	github.com/KyberNetwork/grpc-service v0.3.2-0.20240705221303-511311ba0545
	github.com/KyberNetwork/kutils v0.2.3
	github.com/KyberNetwork/kyber-trace-go v0.1.2
	github.com/KyberNetwork/kyberswap-dex-lib v0.75.4
	github.com/KyberNetwork/kyberswap-dex-lib-private v0.3.0
	github.com/KyberNetwork/logger v0.2.0
	github.com/KyberNetwork/msgpack/v5 v5.4.2
	github.com/KyberNetwork/pathfinder-lib v0.0.13
	github.com/KyberNetwork/pool-service v0.67.0
	github.com/KyberNetwork/reload v0.1.1
	github.com/KyberNetwork/service-framework v0.5.4
	github.com/alicebob/miniredis/v2 v2.31.1
	github.com/cenkalti/backoff/v4 v4.3.0
	github.com/cespare/xxhash/v2 v2.3.0
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc
	github.com/deckarep/golang-set/v2 v2.6.0
	github.com/dgraph-io/ristretto v0.1.1
	github.com/dranikpg/gtrs v0.6.0
	github.com/dustin/go-humanize v1.0.1
	github.com/envoyproxy/protoc-gen-validate v1.0.4
	github.com/ethereum/go-ethereum v1.13.15
	github.com/getsentry/sentry-go v0.18.0
	github.com/gin-contrib/cors v1.5.0
	github.com/gin-contrib/pprof v1.4.0
	github.com/gin-contrib/requestid v0.0.6
	github.com/gin-gonic/gin v1.10.0
	github.com/go-resty/resty/v2 v2.12.0
	github.com/golang/mock v1.6.0
	github.com/google/uuid v1.6.0
	github.com/grafana/pyroscope-go v1.2.0
	github.com/hashicorp/golang-lru/v2 v2.0.3
	github.com/holiman/uint256 v1.3.1
	github.com/huandu/go-clone v1.7.2
	github.com/izumiFinance/iZiSwap-SDK-go v1.0.0
	github.com/json-iterator/go v1.1.12
	github.com/machinebox/graphql v0.2.2
	github.com/mcuadros/go-defaults v1.2.0
	github.com/mitchellh/mapstructure v1.5.0
	github.com/oleiade/lane v1.0.1
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/pkg/errors v0.9.1
	github.com/redis/go-redis/v9 v9.5.1
	github.com/samber/lo v1.46.0
	github.com/sirupsen/logrus v1.9.3
	github.com/sourcegraph/conc v0.3.0
	github.com/spf13/viper v1.18.2
	github.com/stretchr/testify v1.9.0
	github.com/tdewolff/minify/v2 v2.12.4
	github.com/urfave/cli/v2 v2.27.3
	go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin v0.52.0
	go.opentelemetry.io/otel v1.27.0
	go.opentelemetry.io/otel/metric v1.27.0
	go.opentelemetry.io/otel/trace v1.27.0
	go.uber.org/automaxprocs v1.5.2
	go.uber.org/zap v1.27.0
	golang.org/x/exp v0.0.0-20240909161429-701f63a606c0
	golang.org/x/sync v0.8.0
	google.golang.org/grpc v1.64.0
	google.golang.org/grpc/cmd/protoc-gen-go-grpc v1.2.0
	google.golang.org/protobuf v1.34.1
	gopkg.in/natefinch/lumberjack.v2 v2.2.1
	k8s.io/apimachinery v0.23.17
)

require (
	github.com/KyberNetwork/iZiSwap-SDK-go v1.1.0 // indirect
	github.com/KyberNetwork/int256 v0.1.4 // indirect
	github.com/KyberNetwork/pancake-v3-sdk v0.2.2 // indirect
	github.com/KyberNetwork/uniswapv3-sdk-uint256 v0.5.2 // indirect
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/alicebob/gopher-json v0.0.0-20230218143504-906a9b012302 // indirect
	github.com/bits-and-blooms/bitset v1.14.3 // indirect
	github.com/btcsuite/btcd/btcec/v2 v2.3.4 // indirect
	github.com/bytedance/sonic v1.11.6 // indirect
	github.com/bytedance/sonic/loader v0.1.1 // indirect
	github.com/cloudwego/base64x v0.1.4 // indirect
	github.com/cloudwego/iasm v0.2.0 // indirect
	github.com/consensys/bavard v0.1.15 // indirect
	github.com/consensys/gnark-crypto v0.12.1 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.4 // indirect
	github.com/crate-crypto/go-kzg-4844 v0.7.0 // indirect
	github.com/daoleno/uniswap-sdk-core v0.1.7 // indirect
	github.com/daoleno/uniswapv3-sdk v0.4.0 // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/ethereum/c-kzg-4844 v1.0.3 // indirect
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	github.com/gabriel-vasile/mimetype v1.4.3 // indirect
	github.com/gin-contrib/sse v0.1.0 // indirect
	github.com/go-logr/logr v1.4.1 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-ole/go-ole v1.3.0 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator/v10 v10.20.0 // indirect
	github.com/goccy/go-json v0.10.2 // indirect
	github.com/golang/glog v1.2.0 // indirect
	github.com/golang/snappy v0.0.5-0.20220116011046-fa5810519dcb // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/grafana/pyroscope-go/godeltaprof v0.1.8 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware/v2 v2.0.1 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.20.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-retryablehttp v0.7.5 // indirect
	github.com/hashicorp/hcl v1.0.1-vault-5 // indirect
	github.com/huin/goupnp v1.3.0 // indirect
	github.com/jackpal/go-nat-pmp v1.0.2 // indirect
	github.com/klauspost/compress v1.17.9 // indirect
	github.com/klauspost/cpuid/v2 v2.2.7 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/magiconair/properties v1.8.7 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mmcloughlin/addchain v0.4.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/oleiade/lane/v2 v2.0.0 // indirect
	github.com/orcaman/concurrent-map v1.0.0 // indirect
	github.com/pelletier/go-toml/v2 v2.2.2 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/sagikazarmark/locafero v0.4.0 // indirect
	github.com/sagikazarmark/slog-shim v0.1.0 // indirect
	github.com/shirou/gopsutil v3.21.11+incompatible // indirect
	github.com/shopspring/decimal v1.3.1 // indirect
	github.com/spf13/afero v1.11.0 // indirect
	github.com/spf13/cast v1.6.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/supranational/blst v0.3.13 // indirect
	github.com/syndtr/goleveldb v1.0.1-0.20220721030215-126854af5e6d // indirect
	github.com/tdewolff/parse/v2 v2.6.6 // indirect
	github.com/tklauser/go-sysconf v0.3.14 // indirect
	github.com/tklauser/numcpus v0.8.0 // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	github.com/ugorji/go/codec v1.2.12 // indirect
	github.com/vmihailenco/tagparser/v2 v2.0.0 // indirect
	github.com/xrash/smetrics v0.0.0-20240521201337-686a1a2994c1 // indirect
	github.com/yuin/gopher-lua v1.1.1 // indirect
	github.com/yusufpapurcu/wmi v1.2.4 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.48.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc v1.27.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp v1.27.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.27.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.27.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.27.0 // indirect
	go.opentelemetry.io/otel/sdk v1.27.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.27.0 // indirect
	go.opentelemetry.io/proto/otlp v1.2.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/arch v0.8.0 // indirect
	golang.org/x/crypto v0.27.0 // indirect
	golang.org/x/net v0.29.0 // indirect
	golang.org/x/sys v0.25.0 // indirect
	golang.org/x/text v0.18.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20240520151616-dc85e6b867a5 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240515191416-fc5f0ca64291 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	rsc.io/tmplfunc v0.0.3 // indirect
)
