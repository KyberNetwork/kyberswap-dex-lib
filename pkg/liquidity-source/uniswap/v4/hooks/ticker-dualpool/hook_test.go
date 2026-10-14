package tickerdualpool

import (
	"math/big"
	"testing"

	wiremsgpack "github.com/KyberNetwork/msgpack/v5"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

func localSimulator(t *testing.T, snapshots ...Snapshot) *uniswapv4.PoolSimulator {
	t.Helper()
	address := common.HexToAddress("0x0000000000000000000000000000000000012ac4")
	previous, exists := uniswapv4.HookFactories[address]
	uniswapv4.RegisterHooksFactory(NewHook, address)
	t.Cleanup(func() {
		if exists {
			uniswapv4.HookFactories[address] = previous
		} else {
			delete(uniswapv4.HookFactories, address)
		}
	})
	seed := Snapshot{State: fixtureState(), BlockNumber: 1}
	if len(snapshots) > 0 {
		seed = snapshots[0]
	}
	state := seed.State
	snapshot, err := json.Marshal(seed)
	require.NoError(t, err)
	fetch := uniswapv4.FetchRPCResult{Liquidity: big.NewInt(0), TickSpacing: 10,
		Slot0: uniswapv4.Slot0Data{SqrtPriceX96: state.Pool.SqrtPriceX96.ToBig(), Tick: big.NewInt(0), LpFee: 2000}, HookExtra: snapshot}
	extra, err := json.Marshal(fetch.ToExtra([]uniswapv4.Tick{}))
	require.NoError(t, err)
	static, err := json.Marshal(uniswapv4.StaticExtra{Fee: 2000, TickSpacing: 10, HooksAddress: address})
	require.NoError(t, err)
	p := entity.Pool{Address: "0x0000000000000000000000000000000000000000000000000000000000000001", Type: uniswapv4.DexType,
		Tokens:   []*entity.PoolToken{{Address: "0x0000000000000000000000000000000000000001", Decimals: 18, Swappable: true}, {Address: "0x0000000000000000000000000000000000000002", Decimals: 18, Swappable: true}},
		Reserves: entity.PoolReserves{state.Pool.Balance0.Dec(), state.Pool.Balance1.Dec()}, Extra: string(extra), StaticExtra: string(static), BlockNumber: seed.BlockNumber}
	sim, err := uniswapv4.NewPoolSimulator(p, valueobject.ChainIDMonad)
	require.NoError(t, err)
	return sim
}

func TestKyberSimulatorAppliesTickerFees(t *testing.T) {
	sim := localSimulator(t)
	input := pool.TokenAmount{Token: sim.GetTokens()[0], Amount: big.NewInt(1001)}
	result, err := sim.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: input, TokenOut: sim.GetTokens()[1]})
	require.NoError(t, err)
	require.Equal(t, "984", result.TokenAmountOut.Amount.String())
	clone := sim.CloneState()
	sim.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: input, TokenAmountOut: *result.TokenAmountOut, SwapInfo: result.SwapInfo})
	// An independent clone must still give the original quote.
	again, err := clone.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: input, TokenOut: sim.GetTokens()[1]})
	require.NoError(t, err)
	require.Equal(t, result.TokenAmountOut.Amount.String(), again.TokenAmountOut.Amount.String())
	reverse, err := sim.CalcAmountIn(pool.CalcAmountInParams{TokenAmountOut: pool.TokenAmount{Token: sim.GetTokens()[1], Amount: big.NewInt(198)}, TokenIn: sim.GetTokens()[0]})
	require.NoError(t, err)
	check, err := sim.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: *reverse.TokenAmountIn, TokenOut: sim.GetTokens()[1]})
	require.NoError(t, err)
	require.GreaterOrEqual(t, check.TokenAmountOut.Amount.Int64(), int64(198))
}

func TestHookCloneAndVaultRefresh(t *testing.T) {
	h := &Hook{Snapshot: Snapshot{State: fixtureState()}}
	h.Vaults[1] = common.HexToAddress("0x1234")
	q, err := QuoteExactInput(h.State, true, uint256.NewInt(1001))
	require.NoError(t, err)
	clone := h.CloneState().(*Hook)
	clone.Pool.Buckets[0].WeightBps = 1
	require.Equal(t, uint16(10000), h.Pool.Buckets[0].WeightBps)
	h.UpdateBalance(q)
	_, err = h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: true, AmountSpecified: big.NewInt(1001)})
	require.ErrorIs(t, err, ErrRefreshRequired)
	require.False(t, clone.RefreshRequired)
	q.Swap.Balance0.Clear()
	require.False(t, h.Pool.Balance0.IsZero(), "UpdateBalance must not retain mutable quote pointers")
}

func TestCloneAccountingDoesNotSharePointers(t *testing.T) {
	h := &Hook{Snapshot: Snapshot{State: fixtureState()}}
	for i := 0; i < 2; i++ {
		h.Raw[i] = uint256.NewInt(1)
		h.Claims[i] = uint256.NewInt(2)
		h.ManagerCash[i] = uint256.NewInt(3)
		h.Limits[i] = &AaveLimits{uint256.NewInt(4), uint256.NewInt(5)}
	}
	clone := h.CloneState().(*Hook)
	for i := 0; i < 2; i++ {
		clone.Raw[i].Clear()
		clone.Claims[i].Clear()
		clone.ManagerCash[i].Clear()
		clone.Limits[i].MaxDeposit.Clear()
		clone.Limits[i].MaxWithdraw.Clear()
		require.Equal(t, uint64(1), h.Raw[i].Uint64())
		require.Equal(t, uint64(2), h.Claims[i].Uint64())
		require.Equal(t, uint64(3), h.ManagerCash[i].Uint64())
		require.Equal(t, uint64(4), h.Limits[i].MaxDeposit.Uint64())
		require.Equal(t, uint64(5), h.Limits[i].MaxWithdraw.Uint64())
	}
}

func TestMalformedSnapshotRemainsRejectedAfterMsgpack(t *testing.T) {
	for _, malformed := range []string{`,"SwapGas":"bad"}`, `,"HookFeeBps":65536}`} {
		raw, err := json.Marshal(Snapshot{State: fixtureState(), BlockNumber: 1})
		require.NoError(t, err)
		raw = append(raw[:len(raw)-1], []byte(malformed)...)
		h := NewHook(&uniswapv4.HookParam{HookExtra: raw}).(*Hook)
		params := &uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: true, AmountSpecified: big.NewInt(1001)}
		_, err = h.BeforeSwap(params)
		require.Error(t, err)
		require.Nil(t, h.Total0, "failed decoding must retain no partial snapshot")
		data, err := wiremsgpack.Marshal(h)
		require.NoError(t, err)
		var restored Hook
		require.NoError(t, wiremsgpack.Unmarshal(data, &restored))
		_, err = restored.BeforeSwap(params)
		require.ErrorIs(t, err, ErrInvalidState)
	}
}
func TestRefreshRequiredRemainsRejectedAfterMsgpack(t *testing.T) {
	h := &Hook{Snapshot: Snapshot{State: fixtureState()}}
	h.Vaults[1] = common.HexToAddress("0x1234")
	q, err := QuoteExactInput(h.State, true, uint256.NewInt(1001))
	require.NoError(t, err)
	h.UpdateBalance(q)
	params := &uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: true, AmountSpecified: big.NewInt(1001)}
	_, err = h.BeforeSwap(params)
	require.ErrorIs(t, err, ErrRefreshRequired)
	data, err := wiremsgpack.Marshal(h)
	require.NoError(t, err)
	var restored Hook
	require.NoError(t, wiremsgpack.Unmarshal(data, &restored))
	_, err = restored.BeforeSwap(params)
	require.ErrorIs(t, err, ErrRefreshRequired)
}
