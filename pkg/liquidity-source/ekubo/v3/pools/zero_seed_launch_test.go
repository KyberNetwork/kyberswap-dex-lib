package pools

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/KyberNetwork/int256"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	ekubomath "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/v3/math"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

type launchFixture struct {
	Create []struct {
		Order     string `json:"order"`
		Decimals  string `json:"dec"`
		Spacing   uint32 `json:"spacing,string"`
		Liquidity string `json:"liquidity"`
		Lower     int32  `json:"tickLower,string"`
		Upper     int32  `json:"tickUpper,string"`
		Initial   int32  `json:"initialTick,string"`
	} `json:"create"`
	Trade []map[string]string `json:"trade"`
}

func frozenLaunchPools(t *testing.T) ([]*ZeroSeedLaunchPool, launchFixture) {
	t.Helper()
	data, err := os.ReadFile("../testdata/zero-seed-launch.json")
	require.NoError(t, err)
	var fixture launchFixture
	require.NoError(t, json.Unmarshal(data, &fixture))
	var result []*ZeroSeedLaunchPool
	for _, v := range fixture.Create {
		key := NewPoolKey(common.Address{}, common.HexToAddress("0x100"), NewPoolConfig(common.HexToAddress("0x517621Ea1bf51F31826819b118fa4dfEb4D198D2"), 0, NewConcentratedPoolTypeConfig(v.Spacing)))
		liquidity := big256.New(v.Liquidity)
		delta := (*int256.Int)(liquidity.Clone())
		ticks := []Tick{{Number: v.Lower, LiquidityDelta: delta}, {Number: v.Upper, LiquidityDelta: new(int256.Int).Neg(delta)}}
		index := 0
		if v.Initial == v.Upper {
			liquidity = new(uint256.Int)
			index = 1
		}
		state := NewConcentratedPoolState(NewConcentratedPoolSwapState(ekubomath.ToSqrtRatio(v.Initial), liquidity, index), ticks, [2]int32{v.Lower, v.Upper}, v.Initial)
		p, err := NewZeroSeedLaunchPool(key, &ZeroSeedLaunchPoolState{state, Launch{Controller: common.HexToAddress("0x123"), TradingStart: 1000, DecayRateX32: 68913100933, InitialFee: 922337203685477580, FinalFee: 55340232221128654, TickLower: v.Lower, TickUpper: v.Upper}, 1000})
		require.NoError(t, err)
		result = append(result, p)
	}
	return result, fixture
}

func TestZeroSeedLaunchFrozenTrades(t *testing.T) {
	ps, fixture := frozenLaunchPools(t)
	for i, p := range ps {
		v := fixture.Create[i]
		t.Run(v.Order+"/"+v.Decimals, func(t *testing.T) {
			tokenIs1 := v.Order == "t1"
			q, err := p.Quote(uint256.NewInt(1), tokenIs1)
			require.NoError(t, err)
			require.True(t, q.ConsumedAmount.IsZero(), "there is no bid before a buy")
			for _, trade := range fixture.Trade {
				if trade["order"] != v.Order || trade["dec"] != v.Decimals {
					continue
				}
				var amount, output string
				var isToken1, exactOut bool
				switch trade["step"] {
				case "buyExactIn":
					amount, output, isToken1 = trade["quoteIn"], trade["net"], !tokenIs1
				case "buyExactOut":
					amount, output, isToken1, exactOut = trade["tokenOut"], trade["paid"], tokenIs1, true
				case "sellExactIn":
					amount, output, isToken1 = trade["tokenIn"], trade["quoteOut"], tokenIs1
				default:
					continue
				}
				a := big256.New(amount)
				if exactOut {
					a.Neg(a)
				}
				before, err := json.Marshal(p.GetState())
				require.NoError(t, err)
				q, err := p.Quote(a, isToken1)
				require.NoError(t, err, trade["step"])
				after, err := json.Marshal(p.GetState())
				require.NoError(t, err)
				require.Equal(t, before, after, "quoting must be pure")
				require.Equal(t, output, q.CalculatedAmount.String(), trade["step"])
				require.Equal(t, a, q.ConsumedAmount)
				if fee := trade["fee"]; fee != "" {
					require.Equal(t, fee, q.FeesPaid.String())
				}
				require.Equal(t, p.GetKey().Extension(), *q.SwapInfo.Forward)
				require.Equal(t, uint32(1), q.SwapInfo.SkipAhead)
				clone := p.CloneSwapStateOnly()
				clone.SetSwapState(q.SwapInfo.SwapStateAfter)
				unchanged, err := json.Marshal(p.GetState())
				require.NoError(t, err)
				require.Equal(t, before, unchanged)
				p.SetSwapState(q.SwapInfo.SwapStateAfter)
			}
		})
	}
}

func TestZeroSeedLaunchLimits(t *testing.T) {
	ps, _ := frozenLaunchPools(t)
	for i, p := range ps {
		isQuoteToken1 := i < 2
		p.blockTimestamp = 999
		_, err := p.Quote(uint256.NewInt(1), isQuoteToken1)
		require.ErrorContains(t, err, "not started")
		p.blockTimestamp = 1000
		p.launch.InitialFee = ^uint64(0)
		p.launch.FinalFee = 0
		q, err := p.Quote(uint256.NewInt(1), isQuoteToken1)
		require.NoError(t, err)
		require.True(t, q.CalculatedAmount.IsZero(), "maximum fee may leave zero net output")
		large := new(uint256.Int).Lsh(uint256.NewInt(1), 126)
		q, err = p.Quote(large, isQuoteToken1)
		require.NoError(t, err)
		require.True(t, q.ConsumedAmount.Lt(large), "exhaustion must be a partial fill")
		require.Equal(t, ekubomath.ToSqrtRatio(map[bool]int32{true: p.launch.TickUpper, false: p.launch.TickLower}[isQuoteToken1]), q.SwapInfo.SwapStateAfter.(*ConcentratedPoolSwapState).SqrtRatio)
		tooLarge := new(uint256.Int).Lsh(uint256.NewInt(1), 127)
		_, err = p.Quote(tooLarge, isQuoteToken1)
		require.Error(t, err)
		// Maximum fee grosses up one raw input by 2**64; a sufficiently
		// large exact-output request exceeds the signed input delta.
		large.Neg(large)
		q, err = p.Quote(large, !isQuoteToken1)
		if i%2 == 0 {
			require.ErrorIs(t, err, ekubomath.ErrOverflow)
		} else {
			require.NoError(t, err, "six-decimal input gross-up still fits int128")
			require.NotEqual(t, large, q.ConsumedAmount)
		}
	}
}
