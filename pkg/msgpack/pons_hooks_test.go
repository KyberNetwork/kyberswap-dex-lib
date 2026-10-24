package msgpack

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	v4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/coocoo"
	ponsv2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/pons-v2"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/reafund"
	tickermeme "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/ticker-meme-v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// pons-v2 and coocoo embed the v4 Hook interface (holding a *BaseHook), ticker-meme embeds BaseHook by value.
// Each must survive the production msgpack cache with its Extra and CalcAmountIn/Out unchanged.
func TestPonsFamilyHooksRoundTrip(t *testing.T) {
	for name, addr := range map[string]common.Address{
		"pons-v2":     ponsv2.HookAddresses[0],
		"coocoo":      coocoo.HookAddresses[0],
		"ticker-meme": tickermeme.HookAddress,
		"reafund":     reafund.HookAddresses[0],
	} {
		t.Run(name, func(t *testing.T) {
			static, err := json.Marshal(map[string]any{"0x0": [2]bool{false, false}, "fee": 2000, "tS": 60, "hooks": addr})
			require.NoError(t, err)
			extra := `{"liquidity":1000000000000,"sqrtPriceX96":79228162514264337593543950336,"tickSpacing":60,"tick":0,` +
				`"ticks":[{"index":-887220,"liquidityGross":1000000000000,"liquidityNet":1000000000000},` +
				`{"index":887220,"liquidityGross":1000000000000,"liquidityNet":-1000000000000}],"hX":{"r":true,"f":233}}`
			sim, err := v4.NewPoolSimulator(entity.Pool{
				Address: "0x1111111111111111111111111111111111111111111111111111111111111111", Type: v4.DexType, SwapFee: 2000,
				Reserves: []string{"1000000000000", "1000000000000"},
				Tokens: []*entity.PoolToken{
					{Address: "0xe000000000000000000000000000000000000000", Decimals: 18, Swappable: true},
					{Address: "0xf000000000000000000000000000000000000000", Decimals: 18, Swappable: true},
				},
				Extra: extra, StaticExtra: string(static),
			}, valueobject.ChainIDEthereum)
			require.NoError(t, err)

			encoded, err := EncodePoolSimulatorsMap(map[string]pool.IPoolSimulator{sim.GetAddress(): sim})
			require.NoError(t, err)
			decoded, err := DecodePoolSimulatorsMap(encoded)
			require.NoError(t, err)
			restored := decoded[sim.GetAddress()].(*v4.PoolSimulator)

			tokens := sim.Info.Tokens
			amt := big.NewInt(1_000_000)
			out := pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: tokens[0], Amount: amt}, TokenOut: tokens[1]}
			want, err := sim.CalcAmountOut(out)
			require.NoError(t, err)
			got, err := restored.CalcAmountOut(out)
			require.NoError(t, err)
			require.Equal(t, want.TokenAmountOut, got.TokenAmountOut)

			in := pool.CalcAmountInParams{TokenAmountOut: pool.TokenAmount{Token: tokens[1], Amount: amt}, TokenIn: tokens[0]}
			wantIn, err := sim.CalcAmountIn(in)
			require.NoError(t, err)
			gotIn, err := restored.CalcAmountIn(in)
			require.NoError(t, err)
			require.Equal(t, wantIn.TokenAmountIn, gotIn.TokenAmountIn)
			require.Equal(t, sim.GetExchange(), restored.GetExchange())
		})
	}
}
