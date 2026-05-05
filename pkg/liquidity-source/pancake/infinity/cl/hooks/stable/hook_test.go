package stable

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake/infinity/cl"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	bignum "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

func poolHookExtra() HookExtra {
	return HookExtra{
		Balances:            []string{"1962090690918989043", "1731028333861421373"},
		Rates:               []string{"1000000000000000000", "1000000000000000000"},
		LpSupply:            "3693102188763670554",
		InitialA:            "100000",
		FutureA:             "100000",
		InitialATime:        0,
		FutureATime:         0,
		SwapFee:             "100",
		AdminFee:            "5000000000",
		OffpegFeeMultiplier: "100000000000",
	}
}

var fixtures = []struct {
	dx       *big.Int
	zeroFor1 bool
	wantDy   *big.Int
	desc     string
}{
	{bignum.NewBig("17310283338614"), true, bignum.NewBig("17308102232003"), "0.001%, 0->1"},
	{bignum.NewBig("17310283338614"), false, bignum.NewBig("17312464041894"), "0.001%, 1->0"},
	{bignum.NewBig("1731028333861421"), true, bignum.NewBig("1730808580239110"), "0.1%, 0->1"},
	{bignum.NewBig("1731028333861421"), false, bignum.NewBig("1731244761381204"), "0.1%, 1->0"},
	{bignum.NewBig("17310283338614213"), true, bignum.NewBig("17307936080849785"), "1%, 0->1"},
	{bignum.NewBig("17310283338614213"), false, bignum.NewBig("17312298602254781"), "1%, 1->0"},
	{bignum.NewBig("173102833386142137"), true, bignum.NewBig("173063889843910692"), "10%, 0->1"},
	{bignum.NewBig("173102833386142137"), false, bignum.NewBig("173108285972012985"), "10%, 1->0"},
	{bignum.NewBig("432757083465355343"), true, bignum.NewBig("432586062517552385"), "25%, 0->1"},
	{bignum.NewBig("432757083465355343"), false, bignum.NewBig("432708242422232854"), "25%, 1->0"},
}

func stableFixtureTokens() []*entity.PoolToken {
	return []*entity.PoolToken{
		{
			Address:   "0x55d398326f99059ff775485246999027b3197955",
			Decimals:  18,
			Swappable: true,
		},
		{
			Address:   "0x8ac76a51cc950d9822d68b83fe1ad97b32cd580d",
			Decimals:  18,
			Swappable: true,
		},
	}
}

func makeFixturePool(t *testing.T) entity.Pool {
	t.Helper()

	hxBytes, err := json.Marshal(poolHookExtra())
	require.NoError(t, err)

	hookAddr := common.HexToAddress("0x78b58a50e66956d9a4828c229c31541de7c79bbf")

	staticExtra := map[string]any{
		"hsp":    true, // HasSwapPermissions=true → cl errors on unknown hook (good guard)
		"0x0":    [2]bool{false, false},
		"fee":    uint32(0),
		"params": "0x0000000000000000000000000000000000000000000000000000000000000001",
		"tS":     uint64(1),
		"pm":     common.HexToAddress("0xa0FfB9c1CE1Fe56963B0321B32E7A0302114058b"),
		"hooks":  hookAddr,
		"p2":     common.HexToAddress("0x31c2F6fcFf4F8759b3Bd5Bf0e1084A055615c768"),
		"vault":  common.HexToAddress("0x238a358808379702088667322f80aC48bAd5e6c4"),
		"m3":     common.HexToAddress("0x0000000000000000000000000000000000000000"),
	}
	seBytes, err := json.Marshal(staticExtra)
	require.NoError(t, err)

	exBytes := []byte(`{
		"liquidity": 0,
		"sqrtPriceX96": 79228162514264337593543950336,
		"tickSpacing": 1,
		"tick": 0,
		"ticks": [],
		"hX": ` + string(hxBytes) + `
	}`)

	hx := poolHookExtra()
	tokens := stableFixtureTokens()
	reserves := make(entity.PoolReserves, len(tokens))
	for i := range tokens {
		reserves[i] = hx.Balances[i]
	}

	return entity.Pool{
		Address:     hookAddr.Hex(),
		SwapFee:     0,
		Exchange:    valueobject.ExchangePancakeInfinityCLStable,
		Type:        cl.DexType,
		Tokens:      tokens,
		Reserves:    reserves,
		StaticExtra: string(seBytes),
		Extra:       string(exBytes),
	}
}

func TestHook_BeforeSwap_DirectMath(t *testing.T) {
	hxBytes, err := json.Marshal(poolHookExtra())
	require.NoError(t, err)

	pool := &entity.Pool{
		Address:  "0x78b58a50e66956d9a4828c229c31541de7c79bbf",
		Exchange: valueobject.ExchangePancakeInfinityCLStable,
		Type:     cl.DexType,
		Tokens:   stableFixtureTokens(),
		Reserves: entity.PoolReserves(poolHookExtra().Balances),
	}

	h := Factory(&cl.HookParam{
		Pool:        pool,
		HookExtra:   hxBytes,
		HookAddress: common.HexToAddress(pool.Address),
	}).(*Hook)

	require.NotNil(t, h.inner, "inner curve simulator must be built from HookExtra")

	for _, fx := range fixtures {
		res, err := h.BeforeSwap(&cl.BeforeSwapParams{
			CalcOut:         true,
			ZeroForOne:      fx.zeroFor1,
			AmountSpecified: fx.dx,
		})
		require.NoError(t, err, fx.desc)

		// DeltaSpecified must equal the input (full math override).
		require.Equal(t, fx.dx.String(), res.DeltaSpecified.String())

		// DeltaUnspecified is -wantDy (negated output).
		gotOut := new(big.Int).Neg(res.DeltaUnspecified)
		require.Equal(t, fx.wantDy.String(), gotOut.String())
	}
}

func TestHook_GetReserves_FromHookExtra(t *testing.T) {
	hxBytes, err := json.Marshal(poolHookExtra())
	require.NoError(t, err)

	pool := &entity.Pool{
		Address:  "0x78b58a50e66956d9a4828c229c31541de7c79bbf",
		Exchange: valueobject.ExchangePancakeInfinityCLStable,
	}

	h := Factory(&cl.HookParam{
		Pool:        pool,
		HookExtra:   hxBytes,
		HookAddress: common.HexToAddress(pool.Address),
	}).(*Hook)

	reserves, err := h.GetReserves(context.Background(), &cl.HookParam{
		Pool:        pool,
		HookExtra:   hxBytes,
		HookAddress: common.HexToAddress(pool.Address),
	})
	require.NoError(t, err)
	require.Equal(t, entity.PoolReserves(poolHookExtra().Balances), reserves)
}

func Test_ClSimulator_StableHook(t *testing.T) {
	pool := makeFixturePool(t)

	sim, err := cl.NewPoolSimulator(pool, valueobject.ChainID(56))
	require.NoError(t, err, "cl.NewPoolSimulator must build the stable pool")

	for _, fx := range fixtures {
		var tokenIn, tokenOut string
		if fx.zeroFor1 {
			tokenIn, tokenOut = pool.Tokens[0].Address, pool.Tokens[1].Address
		} else {
			tokenIn, tokenOut = pool.Tokens[1].Address, pool.Tokens[0].Address
		}

		cloned := sim.CloneState()
		res, err := cloned.CalcAmountOut(poolpkg.CalcAmountOutParams{
			TokenAmountIn: poolpkg.TokenAmount{
				Token:  tokenIn,
				Amount: new(big.Int).Set(fx.dx),
			},
			TokenOut: tokenOut,
		})

		require.NoError(t, err, fx.desc)
		require.NotNil(t, res)
		require.NotNil(t, res.TokenAmountOut)
		require.Equal(t, fx.wantDy.String(), res.TokenAmountOut.Amount.String())
	}
}
