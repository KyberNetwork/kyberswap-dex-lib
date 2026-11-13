package ozlimitorder

import (
	"testing"

	"github.com/goccy/go-json"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// A WMON/USDC 0.01% pool (tick spacing 1) with this hook on Monad, holding two resting limit orders to sell WMON
// one and two ticks above the price. Expected outputs are the WMON the PoolManager paid for each USDC input on a
// Monad mainnet fork (block 119361375), with the hook deployed at its CREATE2 address.
var (
	entityPool entity.Pool
	_          = json.Unmarshal([]byte(`{"address":"0x3957bb3c02c319231713c06853de9bfc8e7d1aaa276873cd65fc028efc0b1314","swapFee":100,"exchange":"uniswap-v4","type":"uniswap-v4","timestamp":1793820000,"reserves":["2500000000000000000000000","0"],"tokens":[{"address":"0x3bd359c1119da7da1d913d1c4d2b7c461115433a","symbol":"WMON","decimals":18,"swappable":true},{"address":"0x754704bc059f8c67012fed69bc8a327a5aafb603","symbol":"USDC","decimals":6,"swappable":true}],"extra":"{\"liquidity\":0,\"sqrtPriceX96\":10861044632387367993344,\"tickSpacing\":1,\"tick\":-316069,\"ticks\":[{\"index\":-316068,\"liquidityGross\":2742055839388159881746,\"liquidityNet\":2742055839388159881746},{\"index\":-316067,\"liquidityGross\":6855345247517256198162,\"liquidityNet\":1371233568740936434670},{\"index\":-316066,\"liquidityGross\":4113289408129096316416,\"liquidityNet\":-4113289408129096316416}]}","staticExtra":"{\"0x0\":[false,false],\"fee\":100,\"tS\":1,\"hooks\":\"0x94df2bffa788bbd918179f4ccb5a22dbad7e1040\",\"uR\":\"0x0d97dc33264bfc1c226207428a79b26757fb9dc3\",\"pm2\":\"0x000000000022d473030f116ddee9f6b43ac78ba3\",\"mc3\":\"0xca11bde05977b3631167028862be2a173976ca11\"}","blockNumber":119361375}`), &entityPool)
	poolSim    = lo.Must(uniswapv4.NewPoolSimulator(entityPool, valueobject.ChainIDMonad))
)

func TestHookIsRegistered(t *testing.T) {
	t.Parallel()
	for _, addr := range HookAddresses {
		hook, ok := uniswapv4.GetHook(addr, nil)
		require.True(t, ok)
		require.IsType(t, &Hook{}, hook)
		require.True(t, hook.CanAfterSwap(addr))
		require.False(t, hook.CanBeforeSwap(addr))
		before, err := hook.BeforeSwap(&uniswapv4.BeforeSwapParams{})
		require.NoError(t, err)
		require.Zero(t, before.DeltaSpecified.Sign())
		require.Zero(t, before.DeltaUnspecified.Sign())
		after, err := hook.AfterSwap(&uniswapv4.AfterSwapParams{})
		require.NoError(t, err)
		require.Zero(t, after.HookFee.Sign())
	}
}

// USDC in -> WMON out matches the chain exactly: the hook never changes a swap's amounts.
func TestCalcAmountOutMatchesChain(t *testing.T) {
	t.Parallel()
	testutil.TestCalcAmountOut(t, poolSim, map[int]map[int]map[string]string{
		1: {
			0: {
				"1000000000":  "53201999728098375278079",
				"20000000000": "1063986290870219584973742",
			},
		},
	})
}

func TestCloneStateKeepsHook(t *testing.T) {
	t.Parallel()
	cloned := poolSim.CloneState()
	in := pool.TokenAmount{Token: "0x754704bc059f8c67012fed69bc8a327a5aafb603", Amount: bignumber.NewBig10("1000000000")}
	res, err := cloned.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: in, TokenOut: "0x3bd359c1119da7da1d913d1c4d2b7c461115433a"})
	require.NoError(t, err)
	require.Equal(t, "53201999728098375278079", res.TokenAmountOut.Amount.String())
}
