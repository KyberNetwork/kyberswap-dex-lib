module github.com/KyberNetwork/kyberswap-aggregator

go 1.16

replace (
	github.com/daoleno/uniswap-sdk-core v0.1.5 => github.com/KyberNetwork/uniswap-sdk-core v0.1.5
	github.com/daoleno/uniswapv3-sdk v0.4.0 => github.com/KyberNetwork/uniswapv3-sdk v0.4.1
)

require (
	github.com/DataDog/datadog-go v4.8.2+incompatible
	github.com/KyberNetwork/kyberswap-error v0.0.0-20220630071131-e9b03e456957
	github.com/KyberNetwork/promm-sdk-go v0.5.0
	github.com/KyberNetwork/reload v0.1.1
	github.com/alicebob/gopher-json v0.0.0-20200520072559-a9ecdc9d1d3a // indirect
	github.com/alicebob/miniredis v2.5.0+incompatible
	github.com/cenkalti/backoff/v4 v4.1.3
	github.com/chenyahui/gin-cache v1.8.1
	github.com/daoleno/uniswap-sdk-core v0.1.5
	github.com/daoleno/uniswapv3-sdk v0.4.0
	github.com/envoyproxy/protoc-gen-validate v0.1.0
	github.com/ethereum/go-ethereum v1.10.20
	github.com/getsentry/sentry-go v0.12.0
	github.com/gin-contrib/cors v1.3.1
	github.com/gin-contrib/requestid v0.0.6 // indirect
	github.com/gin-contrib/timeout v0.0.3
	github.com/gin-contrib/zap v0.1.0
	github.com/gin-gonic/gin v1.8.1
	github.com/go-redis/redis/v8 v8.11.5
	github.com/go-resty/resty/v2 v2.7.0
	github.com/golang/mock v1.6.0
	github.com/gomodule/redigo v1.8.8 // indirect
	github.com/json-iterator/go v1.1.12
	github.com/machinebox/graphql v0.2.2
	github.com/matryer/is v1.4.0 // indirect
	github.com/mcuadros/go-defaults v1.2.0
	github.com/oleiade/lane v1.0.1
	github.com/orcaman/concurrent-map v0.0.0-20210501183033-44dafcb38ecc
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/pkg/errors v0.9.1
	github.com/pkg/profile v1.7.0
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/viper v1.8.1
	github.com/stretchr/testify v1.8.0
	github.com/urfave/cli/v2 v2.10.2
	github.com/yuin/gopher-lua v0.0.0-20220504180219-658193537a64 // indirect
	go.uber.org/zap v1.23.0
	golang.org/x/sync v0.1.0
	google.golang.org/genproto v0.0.0-20221027153422-115e99e71e1c // indirect
	google.golang.org/grpc v1.50.1
	google.golang.org/grpc/cmd/protoc-gen-go-grpc v1.2.0
	google.golang.org/protobuf v1.28.1
	gopkg.in/DataDog/dd-trace-go.v1 v1.36.0
	gopkg.in/natefinch/lumberjack.v2 v2.0.0
	gorm.io/gorm v1.21.12
	k8s.io/apimachinery v0.22.2
)
