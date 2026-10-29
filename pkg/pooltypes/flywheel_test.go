package pooltypes

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	v4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// Importing pooltypes alone must register the hook, independently of the
// msgpack cache's generated imports. Do not import the hook directly here.
func TestFlywheelHookRegistered(t *testing.T) {
	hook, ok := v4.GetHook(common.HexToAddress("0x992118de23b4ebe9f527dacbbb7d8d4bbbf8e044"),
		&v4.HookParam{Cfg: &v4.Config{ChainID: valueobject.ChainIDRobinhood}})
	require.True(t, ok)
	require.Equal(t, valueobject.ExchangeUniswapV4Flywheel, hook.GetExchange())
}
