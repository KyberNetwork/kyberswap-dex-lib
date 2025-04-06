package ekubo

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/ekubo/math"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/ekubo/quoting"
	ekubo_pool "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/ekubo/quoting/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

var (
	token0 = common.HexToAddress("0x0000000000000000000000000000000000000001")
	token1 = common.HexToAddress("0x0000000000000000000000000000000000000002")

	oracleAddress = common.HexToAddress("0x0000000000000000000000000000000000000003")
)

func poolKey(fee uint64, tickSpacing uint32, extension common.Address) quoting.PoolKey {
	return quoting.NewPoolKey(
		token0,
		token1,
		quoting.Config{
			Fee:         fee,
			TickSpacing: tickSpacing,
			Extension:   extension,
		},
	)
}

func marshalPool(t *testing.T, extra *Extra, staticExtra *StaticExtra) *entity.Pool {
	extraJson, err := json.Marshal(extra)
	require.NoError(t, err)

	staticExtraJson, err := json.Marshal(staticExtra)
	require.NoError(t, err)

	return &entity.Pool{
		Extra:       string(extraJson),
		StaticExtra: string(staticExtraJson),
	}
}

func TestBasePool(t *testing.T) {
	entityPool := marshalPool(
		t,
		&Extra{
			State: quoting.NewPoolState(
				big.NewInt(99999),
				math.IntFromString("13967539110995781342936001321080700"),
				-20201601,
				[]quoting.Tick{
					{
						Number:         -88722000,
						LiquidityDelta: math.IntFromString("99999"),
					},
					{
						Number:         -24124600,
						LiquidityDelta: math.IntFromString("103926982998885"),
					},
					{
						Number:         -24124500,
						LiquidityDelta: math.IntFromString("-103926982998885"),
					},
					{
						Number:         -20236100,
						LiquidityDelta: math.IntFromString("20192651866847"),
					},
					{
						Number:         -20235900,
						LiquidityDelta: math.IntFromString("676843433645"),
					},
					{
						Number:         -20235400,
						LiquidityDelta: math.IntFromString("620315686813"),
					},
					{
						Number:         -20235000,
						LiquidityDelta: math.IntFromString("3899271022058"),
					},
					{
						Number:         -20234900,
						LiquidityDelta: math.IntFromString("1985516133391"),
					},
					{
						Number:         -20233000,
						LiquidityDelta: math.IntFromString("2459469409600"),
					},
					{
						Number:         -20232100,
						LiquidityDelta: math.IntFromString("-20192651866847"),
					},
					{
						Number:         -20231900,
						LiquidityDelta: math.IntFromString("-663892969024"),
					},
					{
						Number:         -20231400,
						LiquidityDelta: math.IntFromString("-620315686813"),
					},
					{
						Number:         -20231000,
						LiquidityDelta: math.IntFromString("-3516445235227"),
					},
					{
						Number:         -20230900,
						LiquidityDelta: math.IntFromString("-1985516133391"),
					},
					{
						Number:         -20229000,
						LiquidityDelta: math.IntFromString("-2459469409600"),
					},
					{
						Number:         -20227900,
						LiquidityDelta: math.IntFromString("-12950464621"),
					},
					{
						Number:         -20227000,
						LiquidityDelta: math.IntFromString("-382825786831"),
					},
					{
						Number:         -2000,
						LiquidityDelta: math.IntFromString("140308196"),
					},
					{
						Number:         2000,
						LiquidityDelta: math.IntFromString("-140308196"),
					},
					{
						Number:         88722000,
						LiquidityDelta: math.IntFromString("-99999"),
					},
				},
				[2]int32{-88722000, 88722000},
			),
		},
		&StaticExtra{
			PoolKey: poolKey(
				922337203685477,
				100,
				common.Address{},
			),
			Extension: ekubo_pool.Base,
		},
	)

	poolSim, err := NewPoolSimulator(*entityPool)
	require.NoError(t, err)

	expectedToken0Amount := big.NewInt(2436479431)

	resExactOut, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountOutResult, error) {
		return poolSim.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{
				Token:  token1.Hex(),
				Amount: big.NewInt(1000000),
			},
			TokenOut: token0.Hex(),
		})
	})
	require.NoError(t, err)

	require.True(t, resExactOut.TokenAmountOut.Amount.Cmp(expectedToken0Amount) == 0)

	resExactIn, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountInResult, error) {
		return poolSim.CalcAmountIn(pool.CalcAmountInParams{
			TokenAmountOut: pool.TokenAmount{
				Token:  token1.Hex(),
				Amount: big.NewInt(-1000000),
			},
			TokenIn: token0.Hex(),
		})
	})
	require.NoError(t, err)

	require.True(t, resExactIn.TokenAmountIn.Amount.Cmp(expectedToken0Amount) == 0)
}

func TestOraclePool(t *testing.T) {
	entityPool := marshalPool(
		t,
		&Extra{
			State: quoting.NewPoolState(
				big.NewInt(10_000_000),
				math.TwoPow128,
				0,
				[]quoting.Tick{
					{
						Number:         math.MinTick,
						LiquidityDelta: big.NewInt(10_000_000),
					},
					{
						Number:         math.MaxTick,
						LiquidityDelta: big.NewInt(-10_000_000),
					},
				},
				[2]int32{math.MinTick, math.MaxTick},
			),
		},
		&StaticExtra{
			PoolKey: poolKey(
				0,
				0,
				oracleAddress,
			),
			Extension: ekubo_pool.Oracle,
		},
	)

	poolSim, err := NewPoolSimulator(*entityPool)
	require.NoError(t, err)

	expectedToken0Amount := big.NewInt(999)

	resExactOut, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountOutResult, error) {
		return poolSim.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{
				Token:  token1.Hex(),
				Amount: big.NewInt(1000),
			},
			TokenOut: token0.Hex(),
		})
	})
	require.NoError(t, err)

	require.True(t, resExactOut.TokenAmountOut.Amount.Cmp(expectedToken0Amount) == 0)

	resExactIn, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountInResult, error) {
		return poolSim.CalcAmountIn(pool.CalcAmountInParams{
			TokenAmountOut: pool.TokenAmount{
				Token:  token1.Hex(),
				Amount: big.NewInt(-1000),
			},
			TokenIn: token0.Hex(),
		})
	})
	require.NoError(t, err)

	require.True(t, resExactIn.TokenAmountIn.Amount.Cmp(expectedToken0Amount) == 0)
}
