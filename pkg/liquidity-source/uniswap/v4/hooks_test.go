package uniswapv4

import (
	"errors"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake/infinity/shared"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

func TestHasSwapPermissions(t *testing.T) {
	hook := "0xd73339564ac99f3e09b0ebc80603ff8b796500c0"
	t.Log(HasSwapPermissions(common.HexToAddress(hook)))
}

// deniedHookPool is a prod ethereum pool snapshot using a denied hook.
const deniedHookPool = `{"address":"0xff2441552bc87bf82d0647f28f51e8a218218ff6a0fd7bee865359276fea3ef2","swapFee":3040,"exchange":"uniswap-v4-fee","type":"uniswap-v4","timestamp":1795536395,"reserves":["417066561925500","999999"],"tokens":[{"address":"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2","symbol":"WETH","decimals":18,"swappable":true},{"address":"0xdac17f958d2ee523a2206206994597c13d831ec7","symbol":"USDT","decimals":6,"swappable":true}],"extra":"{\"liquidity\":20422207567,\"sqrtPriceX96\":3879510197480991806181244,\"tickSpacing\":1,\"tick\":-198498,\"ticks\":[{\"index\":-887272,\"liquidityGross\":20422207567,\"liquidityNet\":20422207567},{\"index\":887272,\"liquidityGross\":20422207567,\"liquidityNet\":-20422207567}],\"sB\":26497620,\"pF\":50331811880}","staticExtra":"{\"0x0\":[false,false],\"fee\":3000,\"tS\":1,\"hooks\":\"0xb39aa0fea3d8940c1106059460934ccfb0212088\",\"uR\":\"0x66a9893cc07d91d95644aedd05d03f95e1dba8af\",\"pm2\":\"0x000000000022d473030f116ddee9f6b43ac78ba3\",\"mc3\":\"0xca11bde05977b3631167028862be2a173976ca11\"}","blockNumber":26486105}`

// TestGetHook_Denied: denied hooks must not be quoted even when the fallback factory
// would quote them, while a non-denied hook with the same permission bits still is.
func TestGetHook_Denied(t *testing.T) {
	prev := fallbackHookFactory
	t.Cleanup(func() { fallbackHookFactory = prev })
	SetFallbackHookFactory(func(*HookParam) Hook { return &BaseHook{Exchange: "uniswap-v4-fee"} })

	var p entity.Pool
	require.NoError(t, json.Unmarshal([]byte(deniedHookPool), &p))
	poolHook := "0xb39aa0fea3d8940c1106059460934ccfb0212088" // hook in deniedHookPool
	staticExtra := p.StaticExtra
	quoted := func(hook common.Address) bool {
		_, ok := GetHook(hook, nil)
		p.StaticExtra = strings.Replace(staticExtra, poolHook, hexutil.Encode(hook[:]), 1)
		_, err := NewPoolSimulator(p, valueobject.ChainIDEthereum)
		require.True(t, err == nil || errors.Is(err, shared.ErrUnsupportedHook), err)
		require.Equal(t, ok, err == nil)
		return ok
	}

	for _, hook := range []string{
		"0xbebc5239e446bc78b46654ad943b0fc19a31e088",
		"0xb39aa0fea3d8940c1106059460934ccfb0212088",
	} {
		require.True(t, HasSwapPermissions(common.HexToAddress(hook)), hook)
		require.False(t, quoted(common.HexToAddress(hook)), hook)
	}
	require.True(t, quoted(common.HexToAddress("0x0000000000000000000000000000000000002088")))
}
