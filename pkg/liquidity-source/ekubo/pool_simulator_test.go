package ekubo

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/math"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/quoting"
	ekubopool "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/quoting/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	bignum "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

var (
	token0 = common.HexToAddress("0x0000000000000000000000000000000000000001")
	token1 = common.HexToAddress("0x0000000000000000000000000000000000000002")

	oracleAddress = "0x0000000000000000000000000000000000000003"
)

func poolKey(fee uint64, tickSpacing uint32, extension common.Address) *quoting.PoolKey {
	return quoting.NewPoolKey(
		token0, token1,
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

	pk := staticExtra.PoolKey

	return &entity.Pool{
		Tokens: []*entity.PoolToken{
			{Address: FromEkuboAddress(pk.Token0.String(), MainnetConfig.ChainId)},
			{Address: FromEkuboAddress(pk.Token1.String(), MainnetConfig.ChainId)},
		},
		Extra:       string(extraJson),
		StaticExtra: string(staticExtraJson),
	}
}

func TestBasePool(t *testing.T) {
	entityPool := marshalPool(t,
		&Extra{PoolState: quoting.NewPoolState(
			big.NewInt(99999),
			bignum.NewBig("13967539110995781342936001321080700"),
			-20201601,
			[]quoting.Tick{
				{Number: -88722000, LiquidityDelta: bignum.NewBig("99999")},
				{Number: -24124600, LiquidityDelta: bignum.NewBig("103926982998885")},
				{Number: -24124500, LiquidityDelta: bignum.NewBig("-103926982998885")},
				{Number: -20236100, LiquidityDelta: bignum.NewBig("20192651866847")},
				{Number: -20235900, LiquidityDelta: bignum.NewBig("676843433645")},
				{Number: -20235400, LiquidityDelta: bignum.NewBig("620315686813")},
				{Number: -20235000, LiquidityDelta: bignum.NewBig("3899271022058")},
				{Number: -20234900, LiquidityDelta: bignum.NewBig("1985516133391")},
				{Number: -20233000, LiquidityDelta: bignum.NewBig("2459469409600")},
				{Number: -20232100, LiquidityDelta: bignum.NewBig("-20192651866847")},
				{Number: -20231900, LiquidityDelta: bignum.NewBig("-663892969024")},
				{Number: -20231400, LiquidityDelta: bignum.NewBig("-620315686813")},
				{Number: -20231000, LiquidityDelta: bignum.NewBig("-3516445235227")},
				{Number: -20230900, LiquidityDelta: bignum.NewBig("-1985516133391")},
				{Number: -20229000, LiquidityDelta: bignum.NewBig("-2459469409600")},
				{Number: -20227900, LiquidityDelta: bignum.NewBig("-12950464621")},
				{Number: -20227000, LiquidityDelta: bignum.NewBig("-382825786831")},
				{Number: -2000, LiquidityDelta: bignum.NewBig("140308196")},
				{Number: 2000, LiquidityDelta: bignum.NewBig("-140308196")},
				{Number: 88722000, LiquidityDelta: bignum.NewBig("-99999")},
			},
			[2]int32{-88722000, 88722000}),
		},
		&StaticExtra{
			PoolKey:       poolKey(922337203685477, 100, common.Address{}),
			ExtensionType: ekubopool.Base,
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
	entityPool := marshalPool(t, &Extra{PoolState: quoting.NewPoolState(
		big.NewInt(10_000_000),
		math.TwoPow128,
		0,
		[]quoting.Tick{
			{Number: math.MinTick, LiquidityDelta: big.NewInt(10_000_000)},
			{Number: math.MaxTick, LiquidityDelta: big.NewInt(-10_000_000)},
		},
		[2]int32{math.MinTick, math.MaxTick},
	)},
		&StaticExtra{
			PoolKey:       poolKey(0, 0, common.HexToAddress(oracleAddress)),
			ExtensionType: ekubopool.Oracle,
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
