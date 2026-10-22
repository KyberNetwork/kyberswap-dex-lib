package ekubov3

import (
	"math/big"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/v3/abis"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/v3/math"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/v3/pools"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/v3/quoting"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var testMevCaptureV21 = common.HexToAddress("0x51f831c23fa4051862881Ef8EfC0c3C37905Ce49")

func TestConfigSupportsMevCaptureV21(t *testing.T) {
	t.Parallel()

	require.Equal(t, ExtensionTypeUnknown, (&Config{}).ExtensionType(testMevCaptureV21))
	require.Equal(t, ExtensionTypeMevCaptureV21, (&Config{MevCaptureV21: testMevCaptureV21}).ExtensionType(testMevCaptureV21))
	require.Equal(t, ExtensionTypeMevCapture, (&Config{
		MevCapture:    common.HexToAddress("0x5555fF9Ff2757500BF4EE020DcfD0210CFfa41Be"),
		MevCaptureV21: testMevCaptureV21,
	}).ExtensionType(common.HexToAddress("0x5555fF9Ff2757500BF4EE020DcfD0210CFfa41Be")))
}

func TestDecodeMevCaptureV21ConfigAndAnchor(t *testing.T) {
	t.Parallel()

	getConfig := abis.MevCaptureV21ABI.Methods[mevCaptureV21GetConfigMethod]
	encoded, err := getConfig.Outputs.Pack(struct {
		HalfLife               uint32
		SlopeK                 uint8
		SegmentExp             uint8
		SegmentWidthMultiplier uint32
		JLin                   uint8
		MaxFee                 uint64
		ClampTicks             uint32
		MGate                  uint8
	}{120, 4, 0, 1, 16, 1 << 63, 2500, 3})
	require.NoError(t, err)
	var config struct{ mevCaptureV21ConfigData }
	require.NoError(t, abis.MevCaptureV21ABI.UnpackIntoInterface(&config, mevCaptureV21GetConfigMethod, encoded))
	require.Equal(t, mevCaptureV21ConfigData{120, 4, 0, 1, 16, 1 << 63, 2500, 3}, config.mevCaptureV21ConfigData)

	getAnchor := abis.MevCaptureV21ABI.Methods[mevCaptureV21GetAnchorMethod]
	encoded, err = getAnchor.Outputs.Pack(uint32(1), uint32(2), uint32(3), uint8(4), uint8(5), uint32(6), int64(-7))
	require.NoError(t, err)
	var anchor mevCaptureV21AnchorData
	require.NoError(t, abis.MevCaptureV21ABI.UnpackIntoInterface(&anchor, mevCaptureV21GetAnchorMethod, encoded))
	require.Equal(t, mevCaptureV21AnchorData{1, 2, 3, 4, 5, 6, -7}, anchor)
}

func TestEventParserDecodesAnchorGateFailed(t *testing.T) {
	t.Parallel()

	parser := &EventParser{config: &Config{MevCaptureV21: testMevCaptureV21}}
	poolID := common.HexToHash("0x65299fa649a7199f782036efb4cacb7a402eef6ea898fabe1c05b23f8e59954f")
	addresses, err := parser.DecodePoolAddressesFromFactoryLog(t.Context(), types.Log{
		Address: testMevCaptureV21,
		Topics:  []common.Hash{abis.AnchorGateFailedEvent.ID, poolID},
		Data:    make([]byte, 64),
	})
	require.NoError(t, err)
	require.Equal(t, []string{hexutil.Encode(poolID[:])}, addresses)
}

func testMevCaptureV21Entity(t *testing.T) entity.Pool {
	t.Helper()

	usdt := common.HexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7")
	poolKey := pools.AnyPoolKey{PoolKey: pools.NewPoolKey(common.Address{}, usdt,
		pools.NewPoolConfig[pools.PoolTypeConfig](testMevCaptureV21, 3689348814741910, pools.NewConcentratedPoolTypeConfig(100)))}
	liquidity := big256.New("100000000000000000")
	const tick, width = -19853200, 40_000
	state := pools.NewMevCaptureV21PoolState(
		&pools.MevCaptureV21PoolSwapState{
			SqrtRatio: math.ToSqrtRatio(tick),
			Liquidity: liquidity,
			Tick:      tick,
			Anchor: pools.MevCaptureV21Anchor{
				LastUpdateTime: 1_000,
				LRefTime:       1_000,
				LRefRaiseTime:  880,
				AnchorX16:      int64(tick) << 16,
			},
		},
		pools.MevCaptureV21Config{HalfLife: 120, SlopeK: 4, JLin: 16, MaxFee: 1 << 63, ClampTicks: 2500, MGate: 3},
		[]pools.Tick{
			{Number: tick - width, LiquidityDelta: big256.SInt256(liquidity)},
			{Number: tick + width, LiquidityDelta: big256.SNeg(liquidity)},
		},
		[2]int32{tick - 2*256*100, tick + 2*256*100},
	)
	extra, err := json.Marshal(state)
	require.NoError(t, err)
	staticExtra, err := json.Marshal(StaticExtra{
		Core:          MainnetConfig.Core,
		ExtensionType: ExtensionTypeMevCaptureV21,
		PoolKey:       poolKey,
	})
	require.NoError(t, err)
	address, err := poolKey.ToPoolAddress()
	require.NoError(t, err)
	return entity.Pool{
		Address:  address,
		Exchange: string(DexType),
		Type:     DexType,
		Reserves: []string{"0", "0"},
		Tokens: []*entity.PoolToken{
			{Address: valueobject.ZeroToWrappedLower(valueobject.ZeroAddress, valueobject.ChainIDEthereum), Swappable: true},
			{Address: hexutil.Encode(usdt[:]), Swappable: true},
		},
		Extra:       string(extra),
		StaticExtra: string(staticExtra),
	}
}

func TestMevCaptureV21PoolSimulator(t *testing.T) {
	t.Parallel()

	entityPool := testMevCaptureV21Entity(t)
	sim, err := NewPoolSimulator(entityPool)
	require.NoError(t, err)
	_, ok := sim.EkuboPool.(*pools.MevCaptureV21Pool)
	require.True(t, ok)

	// the persisted state round-trips
	extra, err := json.Marshal(sim.GetState())
	require.NoError(t, err)
	require.JSONEq(t, entityPool.Extra, string(extra))

	weth, usdt := sim.GetTokens()[0], sim.GetTokens()[1]
	params := pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: weth, Amount: big.NewInt(1e18)},
		TokenOut:      usdt,
	}
	quote, err := sim.CalcAmountOut(params)
	require.NoError(t, err)
	require.Positive(t, quote.TokenAmountOut.Amount.Sign())
	require.Zero(t, quote.RemainingTokenAmountIn.Amount.Sign())
	require.Greater(t, quote.Gas, int64(quoting.BaseGasCost+quoting.BaseGasCostOfOneMevCaptureV21Swap))
	info := quote.SwapInfo.(quoting.SwapInfo)
	require.Equal(t, testMevCaptureV21, *info.Forward)
	require.Zero(t, info.SkipAhead)
	require.False(t, info.IsToken1)
	// decreasing: the limit is the lower edge of the known tick range
	require.Equal(t, math.FixedSqrtRatioToFloat(math.ToSqrtRatio(-19853200-2*256*100), false), info.PriceLimit)
	stateAfter := info.SwapStateAfter.(*pools.MevCaptureV21PoolSwapState)
	require.Greater(t, stateAfter.Anchor.LastUpdateTime, uint32(1_000))

	// a clone with the swap applied runs the next swap at the same timestamp without another anchor update
	cloned := sim.CloneState()
	cloned.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: quote.SwapInfo})
	next, err := cloned.CalcAmountOut(params)
	require.NoError(t, err)
	require.Less(t, next.TokenAmountOut.Amount.Cmp(quote.TokenAmountOut.Amount), 0)
	nextState := next.SwapInfo.(quoting.SwapInfo).SwapStateAfter.(*pools.MevCaptureV21PoolSwapState)
	require.Equal(t, stateAfter.Anchor.LastUpdateTime, nextState.Anchor.LastUpdateTime)
	require.Less(t, next.Gas, quote.Gas)

	// the original is unchanged
	again, err := sim.CalcAmountOut(params)
	require.NoError(t, err)
	require.Equal(t, quote.TokenAmountOut.Amount, again.TokenAmountOut.Amount)

	// exact output
	exactOut, err := sim.CalcAmountIn(pool.CalcAmountInParams{
		TokenAmountOut: pool.TokenAmount{Token: weth, Amount: big.NewInt(1e17)},
		TokenIn:        usdt,
	})
	require.NoError(t, err)
	require.Positive(t, exactOut.TokenAmountIn.Amount.Sign())
	require.Zero(t, exactOut.RemainingTokenAmountOut.Amount.Sign())
	require.True(t, exactOut.SwapInfo.(quoting.SwapInfo).IsToken1 == false)

	// a swap larger than the known tick range stops at its edge
	huge, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: weth, Amount: new(big.Int).Lsh(big.NewInt(1), 100)},
		TokenOut:      usdt,
	})
	require.NoError(t, err)
	require.Positive(t, huge.RemainingTokenAmountIn.Amount.Sign())

	balances, err := sim.CalcBalances()
	require.NoError(t, err)
	require.Len(t, balances, 2)
	require.False(t, balances[0].IsZero() || balances[1].IsZero())
}

func TestMevCaptureV21PoolTrackerRefetchesOnEvents(t *testing.T) {
	t.Parallel()

	entityPool := testMevCaptureV21Entity(t)
	config := *MainnetConfig
	config.supportedExtensions = nil
	config.MevCaptureV21 = testMevCaptureV21
	tracker := NewPoolTracker(&config, ethrpc.New(testFailingRPCURL))
	// any event of a MEVCapture v2.1 pool triggers a refetch: with a failing RPC it fails instead of applying the event
	_, err := tracker.GetNewPoolState(t.Context(), entityPool, pool.GetNewPoolStateParams{
		Logs: []types.Log{{Address: MainnetConfig.Core, BlockNumber: 1}},
	})
	require.ErrorContains(t, err, "fetching pool state")

	staticExtra := StaticExtra{}
	require.NoError(t, json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra))
	ekuboPool, err := unmarshalPool([]byte(entityPool.Extra), &staticExtra)
	require.NoError(t, err)
	require.ErrorIs(t, ekuboPool.ApplyEvent(pools.EventSwapped, nil, 0), pools.ErrMevCaptureV21RefreshRequired)
}

func TestMevCaptureV21RejectsInvalidState(t *testing.T) {
	t.Parallel()

	entityPool := testMevCaptureV21Entity(t)
	var state map[string]any
	require.NoError(t, json.Unmarshal([]byte(entityPool.Extra), &state))
	state["config"].(map[string]any)["halfLife"] = 0
	extra, err := json.Marshal(state)
	require.NoError(t, err)
	entityPool.Extra = string(extra)
	_, err = NewPoolSimulator(entityPool)
	require.ErrorIs(t, err, pools.ErrMevCaptureV21InvalidConfig)
}
