package tickerdualpool

import (
	"math/big"
	"testing"

	wiremsgpack "github.com/KyberNetwork/msgpack/v5"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

func poolSnapshotFixture() *Hook {
	h := &Hook{State: fixtureState(), BlockNumber: 123}
	h.Currencies = [2]common.Address{common.HexToAddress("0x1"), common.HexToAddress("0x2")}
	h.Vaults[0] = common.HexToAddress("0x3")
	for side := range 2 {
		h.Raw[side] = new(uint256.Int)
		h.Claims[side] = new(uint256.Int)
		h.ManagerCash[side] = uint256.MustFromDecimal("100000000000000000000000000")
	}
	h.Raw[1] = h.Total1.Clone()
	shares := uint256.MustFromDecimal("1000000000000000000000000000000")
	h.Aave[0] = &AaveInventory{Raw: new(uint256.Int), Claims: new(uint256.Int), PoolShares: shares.Clone(), Supply: shares.Clone(), Idle: new(uint256.Int), StrategyShares: h.Total0.Clone(), Rate: ray.Clone(), PendingYield: new(uint256.Int)}
	h.Reserves[0] = &AaveReserve{Cash: new(uint256.Int).Add(h.Total0, uint256.MustFromDecimal("1000000000000000000")), ScaledSupply: uint256.MustFromDecimal("100000000000000000000000000"), SupplyCap: new(uint256.Int), PendingTreasury: new(uint256.Int), Active: true}
	return h
}

func snapshotSell() *uniswapv4.BeforeSwapParams {
	return &uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: false, AmountSpecified: new(big.Int).Exp(big.NewInt(10), big.NewInt(21), nil)}
}

func TestPoolSnapshotQuoteAndUpdateAreIndependent(t *testing.T) {
	a, b := poolSnapshotFixture(), poolSnapshotFixture()
	b.BlockNumber++
	aBefore, err := json.Marshal(a.Snapshot)
	require.NoError(t, err)
	bBefore, err := json.Marshal(b.Snapshot)
	require.NoError(t, err)

	quote, err := a.BeforeSwap(snapshotSell())
	require.NoError(t, err)
	aAfterQuote, err := json.Marshal(a.Snapshot)
	require.NoError(t, err)
	require.Equal(t, aBefore, aAfterQuote, "quoting must not mutate the pool snapshot")

	otherQuote, err := b.BeforeSwap(snapshotSell())
	require.NoError(t, err, "a different snapshot block must not prevent quoting")
	a.UpdateBalance(quote.SwapInfo)
	require.Less(t, a.Reserves[0].Cash.Cmp(b.Reserves[0].Cash), 0, "the traded pool updates its own Aave cash")
	bAfter, err := json.Marshal(b.Snapshot)
	require.NoError(t, err)
	require.Equal(t, bBefore, bAfter, "another pool retains its own snapshot even when vault addresses match")

	// The model does not propagate a trade's reserve changes to another pool.
	again, err := b.BeforeSwap(snapshotSell())
	require.NoError(t, err)
	require.Equal(t, otherQuote.SwapInfo.(Quote).NetOutput.Dec(), again.SwapInfo.(Quote).NetOutput.Dec())
}

func TestPoolSnapshotShortageSurvivesSerialization(t *testing.T) {
	h := poolSnapshotFixture()
	h.Reserves[0].Cash.SubUint64(h.Total0, 1)
	before, err := json.Marshal(h.Snapshot)
	require.NoError(t, err)
	for _, calcOut := range []bool{true, false} {
		params := snapshotSell()
		params.CalcOut = calcOut
		if !calcOut {
			params.AmountSpecified = big.NewInt(1000)
		}
		_, err := h.BeforeSwap(params)
		require.ErrorIs(t, err, ErrVaultIlliquid)
	}
	after, err := json.Marshal(h.Snapshot)
	require.NoError(t, err)
	require.Equal(t, before, after, "rejected quotes must not mutate accounting state")

	data, err := wiremsgpack.Marshal(h)
	require.NoError(t, err)
	var restored Hook
	require.NoError(t, wiremsgpack.Unmarshal(data, &restored))
	_, err = restored.BeforeSwap(snapshotSell())
	require.ErrorIs(t, err, ErrVaultIlliquid, "saving and loading must preserve the pool's cash shortage")
}

func TestParentSimulatorUsesEachPoolSnapshot(t *testing.T) {
	ah, bh := poolSnapshotFixture(), poolSnapshotFixture()
	bh.BlockNumber++
	a, b := localSimulator(t, ah.Snapshot), localSimulator(t, bh.Snapshot)
	input := pool.TokenAmount{Token: a.GetTokens()[1], Amount: snapshotSell().AmountSpecified}
	params := pool.CalcAmountOutParams{TokenAmountIn: input, TokenOut: a.GetTokens()[0]}
	aQuote, err := a.CalcAmountOut(params)
	require.NoError(t, err)
	bQuote, err := b.CalcAmountOut(params)
	require.NoError(t, err)
	bReserves := [2]string{b.GetReserves()[0].String(), b.GetReserves()[1].String()}

	info := aQuote.SwapInfo.(uniswapv4.SwapInfo).HookSwapInfo.(Quote)
	a.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: input, TokenAmountOut: *aQuote.TokenAmountOut, SwapInfo: aQuote.SwapInfo})
	require.Equal(t, info.Swap.Balance0.Dec(), a.GetReserves()[0].String())
	require.Equal(t, info.Swap.Balance1.Dec(), a.GetReserves()[1].String())
	require.Zero(t, a.V3Pool.SqrtRatioX96.Cmp(info.Swap.SqrtPriceX96))
	require.Equal(t, int(info.Swap.Tick), a.V3Pool.TickCurrent)
	require.Equal(t, bReserves, [2]string{b.GetReserves()[0].String(), b.GetReserves()[1].String()})

	again, err := b.CalcAmountOut(params)
	require.NoError(t, err)
	require.Equal(t, bQuote.TokenAmountOut.Amount.String(), again.TokenAmountOut.Amount.String())
	_, err = b.CalcAmountIn(pool.CalcAmountInParams{TokenAmountOut: pool.TokenAmount{Token: b.GetTokens()[0], Amount: big.NewInt(1000)}, TokenIn: b.GetTokens()[1]})
	require.NoError(t, err)
	require.Equal(t, ah.BlockNumber, a.GetMetaInfo(input.Token, params.TokenOut).(uniswapv4.PoolMetaInfo).BlockNumber)
	require.Equal(t, bh.BlockNumber, b.GetMetaInfo(input.Token, params.TokenOut).(uniswapv4.PoolMetaInfo).BlockNumber)
}
