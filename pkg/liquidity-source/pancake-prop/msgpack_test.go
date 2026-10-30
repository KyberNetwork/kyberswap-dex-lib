package pancakeprop_test

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	pancakeprop "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake-prop"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/msgpack"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// The router ships simulators between services via msgpack, so a decoded simulator must quote
// the same as the original, including the input already consumed in the epoch.
func TestMsgpackRoundTrip(t *testing.T) {
	const usdt, wbnb = "0x55d398326f99059ff775485246999027b3197955", "0xbb4cdb9cbd36b01bd1cbaebf2de08d9173bc095c"
	extra, err := json.Marshal(&pancakeprop.Extra{
		Packed:      *uint256.MustFromHex("0x2ee0bb89c4271020138800008ca0000000003e528f51f885b690001eada49401"),
		MaxInput:    [2]uint256.Int{*uint256.MustFromDecimal("5000000000000000000000"), *uint256.MustFromDecimal("7500000000000000000")},
		ExpiredAtMs: 1 << 62,
	})
	require.NoError(t, err)
	sim, err := pancakeprop.NewPoolSimulator(pool.FactoryParams{EntityPool: entity.Pool{
		Address: "0xabc", Exchange: pancakeprop.DexType, Type: pancakeprop.DexType, Extra: string(extra),
		StaticExtra: `{"entry":"0x12b76204f5a1b390002a73e3d869196f0e6e8b1c","vault":"0x00fc010f0d2e5010635a0c8702f8b9779fa6befe","pairId":1}`,
		Tokens:      []*entity.PoolToken{{Address: usdt}, {Address: wbnb}},
		Reserves:    entity.PoolReserves{"24244540955976161967236", "4130696923886726262"},
	}})
	require.NoError(t, err)

	amt := func(n int64) *big.Int { return new(big.Int).Mul(big.NewInt(n), big.NewInt(1e18)) }
	params := func(n int64) pool.CalcAmountOutParams {
		return pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: usdt, Amount: amt(n)}, TokenOut: wbnb}
	}
	res, err := sim.CalcAmountOut(params(1800))
	require.NoError(t, err)
	sim.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: res.SwapInfo})

	enc, err := msgpack.EncodePoolSimulatorsMap(map[string]pool.IPoolSimulator{"p": sim})
	require.NoError(t, err)
	dec, err := msgpack.DecodePoolSimulatorsMap(enc)
	require.NoError(t, err)

	want, err := sim.CalcAmountOut(params(1000))
	require.NoError(t, err)
	got, err := dec["p"].CalcAmountOut(params(1000))
	require.NoError(t, err)
	assert.Equal(t, want.TokenAmountOut.Amount.String(), got.TokenAmountOut.Amount.String())
	assert.Equal(t, 1, want.TokenAmountOut.Amount.Cmp(big.NewInt(0)))
}
