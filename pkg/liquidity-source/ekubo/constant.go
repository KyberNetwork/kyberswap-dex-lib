package ekubo

import (
	"errors"

	ekubo "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/hooks"
)

const (
	DexType = "ekubo"
)

const (
	getPoolKeysEndpoint = "/v1/poolKeys"
)

const (
	Base ExtensionType = iota
	Oracle
)

var ExtensionConfigs = map[ExtensionType]ekubo.HooksConfig{
	Base:   {ShouldCallBeforeSwap: false, ShouldCallAfterSwap: false},
	Oracle: {ShouldCallBeforeSwap: true, ShouldCallAfterSwap: false},
}

var (
	ErrGetPoolKeysFailed = errors.New("get pool keys failed")
	ErrZeroSwapAmount    = errors.New("zero swap amount")
)
