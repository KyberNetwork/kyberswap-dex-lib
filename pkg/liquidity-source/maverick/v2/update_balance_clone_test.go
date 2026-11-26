package maverickv2

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// stateDump is the part of a simulator's state a quote reads, as strings.
func stateDump(s *MaverickPoolState) string {
	b, err := json.Marshal(struct {
		Active int32
		Bins   map[uint32]Bin
		Ticks  map[int32]Tick
	}{s.ActiveTick, s.Bins, s.Ticks})
	if err != nil {
		panic(err)
	}
	return string(b)
}

// newMovingBinPool has a movement bin (kind 1) at tick -1 that the TWA moves to tick 0 once a swap lifts the active
// tick, so UpdateBalance's moveBins writes bins and ticks.
func newMovingBinPool(t *testing.T) *PoolSimulator {
	t.Helper()
	const s = `"1000000000000000000000"`
	bin := func(tick int) string {
		return fmt.Sprintf(`{"tick":%d,"totalSupply":%s,"tickBalance":%s}`, tick, s, s)
	}
	tick := func(a, b string, ids string) string {
		return fmt.Sprintf(`{"ReserveA":%s,"ReserveB":%s,"TotalSupply":%s,"BinIdsByTick":%s}`, a, b, s, ids)
	}
	extra := `{"feeAIn":100000000000000,"feeBIn":100000000000000,"protocolFeeRatio":0,"activeTick":0,"lastTwaD8":50000000,` +
		`"bins":{"1":` + bin(0) + `,"2":` + bin(-1) + `,"3":` + bin(1) + `},` +
		`"ticks":{"0":` + tick(s, s, `{"0":1}`) + `,"-1":` + tick(s, `"0"`, `{"1":2}`) + `,"1":` + tick(`"0"`, s, `{"0":3}`) + `}}`
	p, err := NewPoolSimulator(entity.Pool{Address: "0xmav", Exchange: "maverick-v2", Type: "maverick-v2",
		Reserves: entity.PoolReserves{"3000000000000000000000", "2000000000000000000000"}, Tokens: cloneTestTokens(),
		Extra: extra, StaticExtra: `{"tickSpacing":10}`})
	require.NoError(t, err)
	return p
}

// A swap that lifts the active tick moves the movement bin. Applied to a clone with the quote's SwapInfo, the move
// must not write into the bins and ticks the original holds.
func TestUpdateBalanceMovingBinOnACloneKeepsTheOriginal(t *testing.T) {
	orig := newMovingBinPool(t)
	before := stateDump(orig.state)

	amountIn := new(big.Int).Mul(big.NewInt(1500), bignumber.TenPowInt(18))
	res, err := orig.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: cloneTokenA, Amount: amountIn}, TokenOut: cloneTokenB})
	require.NoError(t, err)
	require.Greater(t, res.SwapInfo.(maverickSwapInfo).activeTick, int32(0), "the swap must lift the active tick")

	clone := orig.CloneState().(*PoolSimulator)
	clone.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: pool.TokenAmount{Token: cloneTokenA, Amount: amountIn},
		TokenAmountOut: *res.TokenAmountOut, Fee: *res.Fee, SwapInfo: res.SwapInfo})

	require.NotEqual(t, before, stateDump(clone.state), "the movement bin did not move: the test shows nothing")
	require.Equal(t, before, stateDump(orig.state), "a swap applied to a clone changed the original's bins or ticks")
	require.Equal(t, "0", big256.U0.String(), "a shared zero was written")
}
