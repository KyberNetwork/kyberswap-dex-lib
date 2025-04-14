package ekubo

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

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

type PoolSimulatorTestSuite struct {
	suite.Suite

	pools map[string]string
	sims  map[string]*PoolSimulator
}

// https://github.com/EkuboProtocol/evm-rust-sdk/commits/d6a6e7df76030a8f6c18c2e2cf75086d8a58d16b
func (ts *PoolSimulatorTestSuite) SetupSuite() {
	ts.pools = map[string]string{
		"lvlUSD-USDC-base":        `{"address":"0x7c1156e515aa1a2e851674120074968c905aaf37/0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48_0_200_0x0000000000000000000000000000000000000000","exchange":"ekubo","type":"ekubo","timestamp":1744552055,"reserves":["22230236553469695333225","32442057326"],"tokens":[{"address":"0x7c1156e515aa1a2e851674120074968c905aaf37","symbol":"lvlUSD","decimals":18,"swappable":true},{"address":"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48","symbol":"USDC","decimals":6,"swappable":true}],"extra":"{\"liquidity\":190444832097070393212,\"sqrtRatio\":340297432795514877548017330683904,\"activeTick\":-27630947,\"ticks\":[{\"number\":-27733347,\"liquidityDelta\":0},{\"number\":-27634400,\"liquidityDelta\":1357532262696882268},{\"number\":-27631400,\"liquidityDelta\":61232925196865067418},{\"number\":-27631200,\"liquidityDelta\":127854374637508443526},{\"number\":-27630800,\"liquidityDelta\":-127854374637508443526},{\"number\":-27630600,\"liquidityDelta\":-61232925196865067418},{\"number\":-27627600,\"liquidityDelta\":-1357532262696882268},{\"number\":-27528547,\"liquidityDelta\":0}],\"tickBounds\":[-27733347,-27528547]}","staticExtra":"{\"extensionType\":0,\"poolKey\":{\"token0\":\"0x7c1156e515aa1a2e851674120074968c905aaf37\",\"token1\":\"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48\",\"config\":{\"fee\":0,\"tickSpacing\":200,\"extension\":\"0x0000000000000000000000000000000000000000\"}}}"}`,
		"ETH-USDC-oracle-42527c":  `{"address":"0x0000000000000000000000000000000000000000/0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48_0_0_0x51d02a5948496a67827242eabc5725531342527c","exchange":"ekubo","type":"ekubo","timestamp":1744554592,"reserves":["16211767033603422046","25582559997"],"tokens":[{"address":"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2","symbol":"WETH","decimals":18,"swappable":true},{"address":"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48","symbol":"USDC","decimals":6,"swappable":true}],"extra":"{\"liquidity\":644001943172367,\"sqrtRatio\":13517496585667842734787457760362496,\"activeTick\":-20267103,\"ticks\":[{\"number\":-88722835,\"liquidityDelta\":644001943172367},{\"number\":88722835,\"liquidityDelta\":-644001943172367}],\"tickBounds\":[-88722835,88722835]}","staticExtra":"{\"extensionType\":1,\"poolKey\":{\"token0\":\"0x0000000000000000000000000000000000000000\",\"token1\":\"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48\",\"config\":{\"fee\":0,\"tickSpacing\":0,\"extension\":\"0x51d02a5948496a67827242eabc5725531342527c\"}}}"}`,
		"ETH-EKUBO-base-1":        `{"address":"0x0000000000000000000000000000000000000000/0x04c46e830bb56ce22735d5d8fc9cb90309317d0f_184467440737095516_19802_0x0000000000000000000000000000000000000000","exchange":"ekubo","type":"ekubo","timestamp":1744554808,"reserves":["412153040861140123","18352063799468475546949"],"tokens":[{"address":"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2","symbol":"WETH","decimals":18,"swappable":true},{"address":"0x04c46e830bb56ce22735d5d8fc9cb90309317d0f","symbol":"EKUBO","decimals":18,"swappable":true}],"extra":"{\"liquidity\":69269646872393240672,\"sqrtRatio\":6843420854794309313390943859390472519680,\"activeTick\":6002537,\"ticks\":[{\"number\":-4136087,\"liquidityDelta\":0},{\"number\":2950498,\"liquidityDelta\":568821477503452479021},{\"number\":5207926,\"liquidityDelta\":3345854232514988052480},{\"number\":5544560,\"liquidityDelta\":69269646872393240672},{\"number\":5564362,\"liquidityDelta\":24355412055252046472},{\"number\":5623768,\"liquidityDelta\":-568741057254962452977},{\"number\":5643570,\"liquidityDelta\":-3345934652763478078524},{\"number\":5940600,\"liquidityDelta\":-24355412055252046472},{\"number\":6257432,\"liquidityDelta\":-69269646872393240672},{\"number\":16141161,\"liquidityDelta\":0}],\"tickBounds\":[-4136087,16141161]}","staticExtra":"{\"extensionType\":0,\"poolKey\":{\"token0\":\"0x0000000000000000000000000000000000000000\",\"token1\":\"0x04c46e830bb56ce22735d5d8fc9cb90309317d0f\",\"config\":{\"fee\":184467440737095516,\"tickSpacing\":19802,\"extension\":\"0x0000000000000000000000000000000000000000\"}}}"}`,
		"ETH-EKUBO-base-2":        `{"address":"0x0000000000000000000000000000000000000000/0x04c46e830bb56ce22735d5d8fc9cb90309317d0f_184467440737095516_0_0x0000000000000000000000000000000000000000","exchange":"ekubo","type":"ekubo","timestamp":1744554808,"reserves":["2969312898133367","1187588195557490576"],"tokens":[{"address":"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2","symbol":"WETH","decimals":18,"swappable":true},{"address":"0x04c46e830bb56ce22735d5d8fc9cb90309317d0f","symbol":"EKUBO","decimals":18,"swappable":true}],"extra":"{\"liquidity\":59382833771552102,\"sqrtRatio\":6805254927144693263794887740749196034048,\"activeTick\":5991352,\"ticks\":[{\"number\":-88722835,\"liquidityDelta\":59382833771552102},{\"number\":88722835,\"liquidityDelta\":-59382833771552102}],\"tickBounds\":[-88722835,88722835]}","staticExtra":"{\"extensionType\":0,\"poolKey\":{\"token0\":\"0x0000000000000000000000000000000000000000\",\"token1\":\"0x04c46e830bb56ce22735d5d8fc9cb90309317d0f\",\"config\":{\"fee\":184467440737095516,\"tickSpacing\":0,\"extension\":\"0x0000000000000000000000000000000000000000\"}}}"}`,
		"ETH-EKUBO-oracle-42527c": `{"address":"0x0000000000000000000000000000000000000000/0x04c46e830bb56ce22735d5d8fc9cb90309317d0f_0_0_0x51d02a5948496a67827242eabc5725531342527c","exchange":"ekubo","type":"ekubo","timestamp":1744554808,"reserves":["12362031325829643375","4974045697871814700863"],"tokens":[{"address":"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2","symbol":"WETH","decimals":18,"swappable":true},{"address":"0x04c46e830bb56ce22735d5d8fc9cb90309317d0f","symbol":"EKUBO","decimals":18,"swappable":true}],"extra":"{\"liquidity\":247970378741493120494,\"sqrtRatio\":6825734798789139554821795794043866710016,\"activeTick\":5997362,\"ticks\":[{\"number\":-88722835,\"liquidityDelta\":247970378741493120494},{\"number\":88722835,\"liquidityDelta\":-247970378741493120494}],\"tickBounds\":[-88722835,88722835]}","staticExtra":"{\"extensionType\":1,\"poolKey\":{\"token0\":\"0x0000000000000000000000000000000000000000\",\"token1\":\"0x04c46e830bb56ce22735d5d8fc9cb90309317d0f\",\"config\":{\"fee\":0,\"tickSpacing\":0,\"extension\":\"0x51d02a5948496a67827242eabc5725531342527c\"}}}"}`,
	}

	ts.sims = map[string]*PoolSimulator{}
	for k, p := range ts.pools {
		var ep entity.Pool
		err := json.Unmarshal([]byte(p), &ep)
		ts.Require().Nil(err)

		sim, err := NewPoolSimulator(ep)
		ts.Require().Nil(err)
		ts.Require().NotNil(sim)

		ts.sims[k] = sim
	}
}

func (ts *PoolSimulatorTestSuite) TestCalcAmountOut() {
	ts.T().Parallel()

	testCases := []struct {
		pool     string
		tokenIn  string
		tokenOut string
		amountIn string

		expectedAmountOut           string
		expectedTickSpacingsCrossed uint32
		expectedErr                 error
	}{
		{
			pool:        "lvlUSD-USDC-base",
			tokenIn:     "0x7c1156e515aa1a2e851674120074968c905aaf37",
			tokenOut:    "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			amountIn:    "1000000",
			expectedErr: ekubopool.ErrZeroAmount,
		},
		{
			pool:                        "lvlUSD-USDC-base",
			tokenIn:                     "0x7c1156e515aa1a2e851674120074968c905aaf37",
			tokenOut:                    "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			amountIn:                    "10000000000000000",
			expectedAmountOut:           "10000",
			expectedTickSpacingsCrossed: 0,
		},
		{
			pool:                        "lvlUSD-USDC-base",
			tokenIn:                     "0x7c1156e515aa1a2e851674120074968c905aaf37",
			tokenOut:                    "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			amountIn:                    "50000000000000000000",
			expectedAmountOut:           "50004414",
			expectedTickSpacingsCrossed: 0,
		},
		{
			pool:                        "lvlUSD-USDC-base",
			tokenIn:                     "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			tokenOut:                    "0x7c1156e515aa1a2e851674120074968c905aaf37",
			amountIn:                    "10000000000000",
			expectedAmountOut:           "22230236553469695333225",
			expectedTickSpacingsCrossed: 581768,
		},
		{
			pool:                        "lvlUSD-USDC-base",
			tokenIn:                     "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			tokenOut:                    "0x7c1156e515aa1a2e851674120074968c905aaf37",
			amountIn:                    "1000000000000000000",
			expectedAmountOut:           "22230236553469695333225",
			expectedTickSpacingsCrossed: 581768,
		},

		{
			pool:        "ETH-USDC-oracle-42527c",
			tokenIn:     "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
			tokenOut:    "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			amountIn:    "1000000",
			expectedErr: ekubopool.ErrZeroAmount,
		},
		{
			pool:                        "ETH-USDC-oracle-42527c",
			tokenIn:                     "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
			tokenOut:                    "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			amountIn:                    "100000000000000000",
			expectedAmountOut:           "156835001",
			expectedTickSpacingsCrossed: 0,
		},
		{
			pool:                        "ETH-USDC-oracle-42527c",
			tokenIn:                     "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			tokenOut:                    "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
			amountIn:                    "100000000",
			expectedAmountOut:           "63123641237103297",
			expectedTickSpacingsCrossed: 0,
		},
		{
			pool:                        "ETH-USDC-oracle-42527c",
			tokenIn:                     "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			tokenOut:                    "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
			amountIn:                    "100000000000000",
			expectedAmountOut:           "16207620709311223961",
			expectedTickSpacingsCrossed: 0,
		},
		{
			pool:                        "ETH-USDC-oracle-42527c",
			tokenIn:                     "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
			tokenOut:                    "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			amountIn:                    "100000000000000000",
			expectedAmountOut:           "156835001",
			expectedTickSpacingsCrossed: 0,
		},
		{
			pool:                        "ETH-USDC-oracle-42527c",
			tokenIn:                     "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
			tokenOut:                    "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			amountIn:                    "100000000000000000000",
			expectedAmountOut:           "22013743230",
			expectedTickSpacingsCrossed: 0,
		},
		{
			pool:                        "ETH-EKUBO-oracle-42527c",
			tokenIn:                     "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
			tokenOut:                    "0x04c46e830bb56ce22735d5d8fc9cb90309317d0f",
			amountIn:                    "900000000000000",
			expectedAmountOut:           "362101916616786920",
			expectedTickSpacingsCrossed: 0,
		},
		{
			pool:                        "ETH-EKUBO-oracle-42527c",
			tokenIn:                     "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
			tokenOut:                    "0x04c46e830bb56ce22735d5d8fc9cb90309317d0f",
			amountIn:                    "100000000000000000000",
			expectedAmountOut:           "4426802932609840856309",
			expectedTickSpacingsCrossed: 0,
		},
		{
			pool:                        "ETH-EKUBO-oracle-42527c",
			tokenIn:                     "0x04c46e830bb56ce22735d5d8fc9cb90309317d0f",
			tokenOut:                    "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
			amountIn:                    "10000000000000",
			expectedAmountOut:           "24853071426",
			expectedTickSpacingsCrossed: 0,
		},
		{
			pool:                        "ETH-EKUBO-oracle-42527c",
			tokenIn:                     "0x04c46e830bb56ce22735d5d8fc9cb90309317d0f",
			tokenOut:                    "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
			amountIn:                    "100000000000000000",
			expectedAmountOut:           "248525718318197",
			expectedTickSpacingsCrossed: 0,
		},

		{
			pool:                        "ETH-EKUBO-base-1",
			tokenIn:                     "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
			tokenOut:                    "0x04c46e830bb56ce22735d5d8fc9cb90309317d0f",
			amountIn:                    "1000000",
			expectedAmountOut:           "400407818",
			expectedTickSpacingsCrossed: 0,
		},
		{
			pool:                        "ETH-EKUBO-base-1",
			tokenIn:                     "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
			tokenOut:                    "0x04c46e830bb56ce22735d5d8fc9cb90309317d0f",
			amountIn:                    "1000000000000000000",
			expectedAmountOut:           "326276313187628668418",
			expectedTickSpacingsCrossed: 18,
		},
		{
			pool:                        "ETH-EKUBO-base-1",
			tokenIn:                     "0x04c46e830bb56ce22735d5d8fc9cb90309317d0f",
			tokenOut:                    "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
			amountIn:                    "1000000000000000000",
			expectedAmountOut:           "2446014701861857",
			expectedTickSpacingsCrossed: 0,
		},
		{
			pool:                        "ETH-EKUBO-base-1",
			tokenIn:                     "0x04c46e830bb56ce22735d5d8fc9cb90309317d0f",
			tokenOut:                    "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
			amountIn:                    "1000000000000000000000000",
			expectedAmountOut:           "412153040861140123",
			expectedTickSpacingsCrossed: 4177,
		},

		{
			pool:                        "ETH-EKUBO-base-2",
			tokenIn:                     "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
			tokenOut:                    "0x04c46e830bb56ce22735d5d8fc9cb90309317d0f",
			amountIn:                    "1000000",
			expectedAmountOut:           "395954099",
			expectedTickSpacingsCrossed: 0,
		},
		{
			pool:                        "ETH-EKUBO-base-2",
			tokenIn:                     "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
			tokenOut:                    "0x04c46e830bb56ce22735d5d8fc9cb90309317d0f",
			amountIn:                    "1000000000000000",
			expectedAmountOut:           "296948572606173404",
			expectedTickSpacingsCrossed: 0,
		},
		{
			pool:                        "ETH-EKUBO-base-2",
			tokenIn:                     "0x04c46e830bb56ce22735d5d8fc9cb90309317d0f",
			tokenOut:                    "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
			amountIn:                    "1000000000000000000",
			expectedAmountOut:           "1349942920865004",
			expectedTickSpacingsCrossed: 0,
		},
		{
			pool:                        "ETH-EKUBO-base-2",
			tokenIn:                     "0x04c46e830bb56ce22735d5d8fc9cb90309317d0f",
			tokenOut:                    "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
			amountIn:                    "10000000000000000000000",
			expectedAmountOut:           "2968956746821686",
			expectedTickSpacingsCrossed: 0,
		},
	}

	for _, tc := range testCases {
		ts.T().Run(tc.pool, func(t *testing.T) {
			sim := ts.sims[tc.pool]

			res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{
					Token:  tc.tokenIn,
					Amount: bignum.NewBig(tc.amountIn),
				},
				TokenOut: tc.tokenOut,
			})

			if tc.expectedErr == nil {
				require.NotNil(t, res)
				require.Equal(t, tc.expectedAmountOut, res.TokenAmountOut.Amount.String())

				swapInfo := res.SwapInfo.(quoting.SwapInfo)
				require.Equal(t, tc.expectedTickSpacingsCrossed, swapInfo.TickSpacingsCrossed)
			} else {
				require.ErrorContains(t, err, tc.expectedErr.Error())
			}
		})
	}
}

func (ts *PoolSimulatorTestSuite) TestCalcAmountIn() {
	ts.T().Parallel()

	for p, sim := range ts.sims {
		ts.T().Run(p, func(t *testing.T) {
			testutil.TestCalcAmountIn(t, sim)
		})
	}
}

func TestPoolSimulatorTestSuite(t *testing.T) {
	suite.Run(t, new(PoolSimulatorTestSuite))
}
