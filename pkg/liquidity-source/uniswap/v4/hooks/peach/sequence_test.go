package peach

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	v4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// References execute real source Hooks against the real v4 PoolManager. Only
// fixture-local Hook addresses are admitted by this test's temporary factory;
// production chain/address registration is tested separately, without msgpack.
func TestSoliditySequentialSwapParity(t *testing.T) {
	data, err := os.ReadFile("testdata/sequence.json")
	require.NoError(t, err)
	var fixture struct {
		Cases []struct {
			Kind, Variant, Direction                                    int
			Hook, PoolID, Currency0, Currency1, Liquidity, SqrtPriceX96 string
			Timestamp                                                   uint64
			ProtocolFee                                                 uint32
			TickSpacing, Tick                                           int
			Base0, Base1, Auto0, Auto1, Floor, Cap                      uint32
			Mid                                                         MidBlockConfig
			Steps                                                       []struct {
				ZeroForOne                                 bool
				AmountIn, AmountOut, SqrtBefore, SqrtAfter string
				Fee                                        uint32
			}
		}
	}
	require.NoError(t, json.Unmarshal(data, &fixture))
	require.Len(t, fixture.Cases, 30)
	surcharged, capped, crossed, compared := 0, 0, 0, 0
	for _, c := range fixture.Cases {
		t.Run(fmt.Sprintf("kind%d/variant%d/direction%d", c.Kind, c.Variant, c.Direction), func(t *testing.T) {
			require.Zero(t, c.ProtocolFee)
			addr := common.HexToAddress(c.Hook)
			old, existed := v4.HookFactories[addr]
			var active *Hook
			v4.HookFactories[addr] = func(param *v4.HookParam) v4.Hook {
				h := &Hook{BaseHook: v4.BaseHook{Exchange: DexType}, ValidDeployment: true, QuoteAt: int64(c.Timestamp), RWA: c.Kind == 2}
				require.NoError(t, param.HookExtra.Unmarshal(&h.Extra))
				active = h
				return h
			}
			defer func() {
				if existed {
					v4.HookFactories[addr] = old
				} else {
					delete(v4.HookFactories, addr)
				}
			}()
			state := Extra{Tracked: true, Timestamp: c.Timestamp, Autonomous0For1: c.Auto0, Autonomous1For0: c.Auto1, Fee0For1: c.Base0, Fee1For0: c.Base1, Floor: c.Floor, Cap: c.Cap, AbsoluteCap: 20000, Mid: c.Mid}
			hx, err := json.Marshal(state)
			require.NoError(t, err)
			liq, ok := new(big.Int).SetString(c.Liquidity, 10)
			require.True(t, ok)
			initial, ok := new(big.Int).SetString(c.SqrtPriceX96, 10)
			require.True(t, ok)
			ticks := []v4.Tick{{Index: -60000, LiquidityGross: liq, LiquidityNet: liq}, {Index: 60000, LiquidityGross: liq, LiquidityNet: new(big.Int).Neg(liq)}}
			epExtra := v4.FetchRPCResult{Liquidity: liq, Slot0: v4.Slot0Data{SqrtPriceX96: initial, Tick: big.NewInt(int64(c.Tick))}, TickSpacing: int32(c.TickSpacing)}
			ex := epExtra.ToExtra(ticks)
			ex.HookExtra = hx
			raw, err := json.Marshal(ex)
			require.NoError(t, err)
			native := c.Currency0 == "0x0000000000000000000000000000000000000000"
			token0 := c.Currency0
			if native {
				token0 = "0xbb4cdb9cbd36b01bd1cbaebf2de08d9173bc095c"
			}
			stat, err := json.Marshal(v4.StaticExtra{IsNative: [2]bool{native, false}, Fee: 8388608, TickSpacing: int32(c.TickSpacing), HooksAddress: addr})
			require.NoError(t, err)
			r0, r1 := v4.EstimateReservesFromTicks(initial, ticks)
			ep := entity.Pool{Address: c.PoolID, Type: v4.DexType, Exchange: string(DexType), Timestamp: int64(c.Timestamp), Reserves: []string{r0.String(), r1.String()}, Tokens: []*entity.PoolToken{{Address: token0, Decimals: 18, Swappable: true}, {Address: c.Currency1, Decimals: 18, Swappable: true}}, Extra: string(raw), StaticExtra: string(stat)}
			sim, err := v4.NewPoolSimulator(ep, 56)
			require.NoError(t, err)
			require.Len(t, c.Steps, 4)
			for i, step := range c.Steps {
				require.Equal(t, step.SqrtBefore, sim.V3Pool.SqrtRatioX96.Dec(), "leg %d pre-price", i)
				before, err := active.BeforeSwap(&v4.BeforeSwapParams{CalcOut: true, ZeroForOne: step.ZeroForOne, SqrtPriceX96: &sim.V3Pool.SqrtRatioX96})
				require.NoError(t, err)
				require.EqualValues(t, step.Fee, before.SwapFee, "leg %d actual Swap event fee", i)
				in, out := 0, 1
				if !step.ZeroForOne {
					in, out = 1, 0
				}
				amount, ok := new(big.Int).SetString(step.AmountIn, 10)
				require.True(t, ok)
				params := pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: ep.Tokens[in].Address, Amount: amount}, TokenOut: ep.Tokens[out].Address}
				result, err := sim.CalcAmountOut(params)
				require.NoError(t, err)
				require.Equal(t, step.AmountOut, result.TokenAmountOut.Amount.String(), "leg %d real EVM output", i)
				info := result.SwapInfo.(v4.SwapInfo)
				require.Equal(t, step.SqrtAfter, info.NextStateSqrtRatioX96.Dec(), "leg %d post-price", i)
				again, err := sim.CalcAmountOut(params)
				require.NoError(t, err)
				require.Equal(t, result.TokenAmountOut, again.TokenAmountOut)
				base := c.Base1
				if step.ZeroForOne {
					base = c.Base0
				}
				if step.Fee > base {
					surcharged++
				}
				if step.Fee == c.Cap {
					capped++
				}
				prior := uint256.MustFromDecimal(step.SqrtBefore)
				after := uint256.MustFromDecimal(step.SqrtAfter)
				opening := uint256.MustFromDecimal(c.SqrtPriceX96)
				if prior.Cmp(opening)*after.Cmp(opening) < 0 {
					crossed++
				}
				sim.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: params.TokenAmountIn, TokenAmountOut: *result.TokenAmountOut, Fee: *result.Fee, SwapInfo: result.SwapInfo})
				compared++
			}
		})
	}
	require.Equal(t, 120, compared)
	require.Positive(t, surcharged)
	require.Positive(t, capped)
	require.Positive(t, crossed)
}
