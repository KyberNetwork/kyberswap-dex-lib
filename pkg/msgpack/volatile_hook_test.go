package msgpack

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	v4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/volatile"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// A ChangingFeeHook pool keeps its 20,000 pip fee ceiling through the production msgpack cache (the ceiling is a
// field of the hook, set by the factory from the hook address, and is not in hX): the restored simulator quotes the
// same amount, and that amount is the 2% quote, not the one Uniswap's hook address would give for the same state.
func TestVolatileChangingFeeCeilingSurvivesRoundTrip(t *testing.T) {
	// The tracked state: the pool price fell about 17% (sqrt) since the stored block-open price, so the buyer's side
	// is the backrun side and the top-of-block advance prices a backrun fee far above 2%.
	const hX = `{"c":{"rs":800,"bc":500000,"mr":4000000,"mm":200,"sr":1000000,"ss":300,"ms":800,"bt":500,"rf":1000000,"rt":500},` +
		`"s":{"p":"95073795017117205112252740403","a":400}}`
	build := func(hook common.Address) *v4.PoolSimulator {
		static, err := json.Marshal(map[string]any{"0x0": [2]bool{false, false}, "fee": 8388608, "tS": 1, "hooks": hook})
		require.NoError(t, err)
		extra := `{"liquidity":1000000000000,"sqrtPriceX96":79228162514264337593543950336,"tickSpacing":1,"tick":0,` +
			`"ticks":[{"index":-887272,"liquidityGross":1000000000000,"liquidityNet":1000000000000},` +
			`{"index":887272,"liquidityGross":1000000000000,"liquidityNet":-1000000000000}],"hX":` + hX + `}`
		sim, err := v4.NewPoolSimulator(entity.Pool{
			Address: "0x2222222222222222222222222222222222222222222222222222222222222222", Type: v4.DexType,
			SwapFee: 0, Reserves: []string{"1000000000000", "1000000000000"},
			Tokens: []*entity.PoolToken{
				{Address: "0xe000000000000000000000000000000000000000", Decimals: 6, Swappable: true},
				{Address: "0xf000000000000000000000000000000000000000", Decimals: 6, Swappable: true},
			},
			Extra: extra, StaticExtra: string(static),
		}, valueobject.ChainIDPolygon)
		require.NoError(t, err)
		return sim
	}
	quote := func(sim *v4.PoolSimulator) *big.Int {
		tokens := sim.Info.Tokens
		// token1 in, token0 out: the buy, against the fall
		res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: tokens[1], Amount: big.NewInt(1_000_000_000)}, TokenOut: tokens[0]})
		require.NoError(t, err)
		return res.TokenAmountOut.Amount
	}

	ours := build(volatile.ChangingFeeHookAddresses[0])
	uniswaps := build(volatile.HookAddress)
	want := quote(ours)

	encoded, err := EncodePoolSimulatorsMap(map[string]pool.IPoolSimulator{ours.GetAddress(): ours})
	require.NoError(t, err)
	decoded, err := DecodePoolSimulatorsMap(encoded)
	require.NoError(t, err)
	restored := decoded[ours.GetAddress()].(*v4.PoolSimulator)

	require.Equal(t, want.String(), quote(restored).String())
	// the same state under Uniswap's address pays a fee several times larger than 2%, so it gets back much less
	uncapped := quote(uniswaps)
	require.Greater(t, new(big.Int).Mul(want, big.NewInt(100)).Cmp(new(big.Int).Mul(uncapped, big.NewInt(105))), 0,
		"the 2%% ceiling must leave at least 5%% more output than the uncapped fee")
	t.Logf("1,000,000,000 in: out %s with the 20,000 pip ceiling (before and after msgpack), %s under Uniswap's address", want, uncapped)
}
