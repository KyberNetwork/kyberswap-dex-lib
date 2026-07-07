package stable

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	stableng "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/stable-ng"
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

func TestHook_BeforeSwap_DustPool(t *testing.T) {
	const poolJSON = `{"address":"0x80a2f355f4694286aa79c22f4c644c42141b07e68546ce6cdb23d7183f8db1d0","exchange":"pancake-infinity-cl-stable","type":"pancake-infinity-cl","timestamp":1778040091,"reserves":["285462141838064","1975110172833"],"tokens":[{"address":"0x38c207dbe12e84a4b3eafa62f492391df95ae36e","symbol":"MIT","decimals":18,"swappable":true},{"address":"0xaa60ca2e1bada3641bb94785d7ae3bbd5a6cb520","symbol":"USDT.z","decimals":18,"swappable":true}],"extra":"{\"liquidity\":0,\"sqrtPriceX96\":79228162514264337593543950336,\"tickSpacing\":1,\"tick\":0,\"ticks\":[],\"hX\":{\"balances\":[\"285462141838064\",\"1975110172833\"],\"rates\":[\"1000000000000000000\",\"1000000000000000000\"],\"lpSupply\":\"284924283676128\",\"initialA\":\"200000\",\"futureA\":\"200000\",\"initialATime\":0,\"futureATime\":0,\"swapFee\":\"1000000\",\"adminFee\":\"5000000000\",\"offpegFeeMultiplier\":\"50000000000\"}}","staticExtra":"{\"hsp\":true,\"0x0\":[false,false],\"fee\":0,\"params\":\"0x0000000000000000000000000000000000000000000000000000000000010455\",\"tS\":1,\"pm\":\"0xa0ffb9c1ce1fe56963b0321b32e7a0302114058b\",\"hooks\":\"0xd7829fa3188734420659601ac17ccc16980dabf0\",\"p2\":\"0x31c2f6fcff4f8759b3bd5bf0e1084a055615c768\",\"vault\":\"0x238a358808379702088667322f80ac48bad5e6c4\",\"m3\":\"0xca11bde05977b3631167028862be2a173976ca11\"}","blockNumber":89704036}`

	var pool entity.Pool
	require.NoError(t, json.Unmarshal([]byte(poolJSON), &pool))

	tokenUSDT := "0xaa60ca2e1bada3641bb94785d7ae3bbd5a6cb520"
	tokenMIT := "0x38c207dbe12e84a4b3eafa62f492391df95ae36e"
	e18 := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)

	t.Run("1e18 routes", func(t *testing.T) {
		sim, err := cl.NewPoolSimulator(pool, valueobject.ChainID(56))
		require.NoError(t, err)
		res, err := sim.CalcAmountOut(poolpkg.CalcAmountOutParams{
			TokenAmountIn: poolpkg.TokenAmount{Token: tokenUSDT, Amount: new(big.Int).Set(e18)},
			TokenOut:      tokenMIT,
		})
		require.NoError(t, err)
		require.Equal(t, 1, res.TokenAmountOut.Amount.Sign())
	})

	t.Run("100e18 rejected by stable-ng drain guard", func(t *testing.T) {
		sim, err := cl.NewPoolSimulator(pool, valueobject.ChainID(56))
		require.NoError(t, err)
		huge := new(big.Int).Mul(big.NewInt(100), e18)
		_, err = sim.CalcAmountOut(poolpkg.CalcAmountOutParams{
			TokenAmountIn: poolpkg.TokenAmount{Token: tokenUSDT, Amount: huge},
			TokenOut:      tokenMIT,
		})
		require.ErrorIs(t, err, stableng.ErrPoolDrained)
	})
}

func TestHook_UpdateBalance(t *testing.T) {
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

	pre0 := new(big.Int).Set(h.inner.Info.Reserves[0])
	pre1 := new(big.Int).Set(h.inner.Info.Reserves[1])

	dx := bignum.NewBig("1731028333861421")
	res, err := h.BeforeSwap(&cl.BeforeSwapParams{
		CalcOut:         true,
		ZeroForOne:      true,
		AmountSpecified: dx,
	})
	require.NoError(t, err)
	require.NotNil(t, res.SwapInfo)

	h.UpdateBalance(res.SwapInfo)

	require.NotEqual(t, pre0.String(), h.inner.Info.Reserves[0].String(), "reserve[0] must have moved")
	require.NotEqual(t, pre1.String(), h.inner.Info.Reserves[1].String(), "reserve[1] must have moved")
	require.Equal(t, 1, h.inner.Info.Reserves[0].Cmp(pre0), "reserve[0] (in) increased")
	require.Equal(t, -1, h.inner.Info.Reserves[1].Cmp(pre1), "reserve[1] (out) decreased")
}
