package tickerdualpool

import (
	"math/big"
	"testing"

	wiremsgpack "github.com/KyberNetwork/msgpack/v5"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/swaplimit"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// The router builds shared inventories only for exchanges in pool.UseSwapLimit.
func TestExchangeUsesSwapLimit(t *testing.T) {
	require.Contains(t, pool.UseSwapLimit, valueobject.ExchangeUniswapV4TickerDualPool)
}

func sharedFixture() *Hook {
	h := &Hook{Snapshot: Snapshot{State: fixtureState(), BlockNumber: 123}}
	h.Currencies = [2]common.Address{common.HexToAddress("0x1"), common.HexToAddress("0x2")}
	h.Vaults[0] = common.HexToAddress("0x3")
	for side := 0; side < 2; side++ {
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
func sharedSell(limit *swaplimit.Inventory) *uniswapv4.BeforeSwapParams {
	p := &uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: false, AmountSpecified: new(big.Int).Exp(big.NewInt(10), big.NewInt(21), nil)}
	if limit != nil {
		p.Limit = limit
	}
	return p
}
func TestAnotherPoolsSwapReducesSharedAaveCash(t *testing.T) {
	a := sharedFixture()
	b := sharedFixture()
	b.Vaults[0] = common.HexToAddress("0x4")
	limits := a.CalculateLimit()
	for key, value := range b.CalculateLimit() {
		limits[key] = value
	}
	inventory := swaplimit.NewInventory(a.GetExchange(), limits)
	before := inventory.Clone()
	quote, err := a.BeforeSwap(sharedSell(inventory))
	require.NoError(t, err)
	require.Equal(t, before.GetLimit("ticker:143:aave:0x0000000000000000000000000000000000000001:cash"), inventory.GetLimit("ticker:143:aave:0x0000000000000000000000000000000000000001:cash"), "quoting cannot mutate shared cash")
	a.UpdateBalanceWithLimit(quote.SwapInfo, inventory)
	require.NoError(t, a.decodeErr)
	// B's own wrapper still owns the same shares; the common Aave reserve has less cash.
	_, err = b.BeforeSwap(sharedSell(nil))
	require.NoError(t, err, "an isolated stale snapshot would still quote")
	_, err = b.BeforeSwap(sharedSell(inventory))
	require.ErrorIs(t, err, ErrVaultIlliquid)
	untouched := swaplimit.NewInventory(a.GetExchange(), b.CalculateLimit())
	_, err = b.BeforeSwap(sharedSell(untouched))
	require.NoError(t, err, "independent route state remains unchanged")
}
func TestSharedInventoryRejectsStaleQuotesAndMixedBlocks(t *testing.T) {
	h := sharedFixture()
	inventory := swaplimit.NewInventory(h.GetExchange(), h.CalculateLimit())
	quote, err := h.BeforeSwap(sharedSell(inventory))
	require.NoError(t, err)
	h.UpdateBalanceWithLimit(quote.SwapInfo, inventory)
	require.NoError(t, h.decodeErr)
	h.UpdateBalanceWithLimit(quote.SwapInfo, inventory)
	require.ErrorIs(t, h.decodeErr, ErrInvalidState)
	data, err := wiremsgpack.Marshal(h)
	require.NoError(t, err)
	var restored Hook
	require.NoError(t, wiremsgpack.Unmarshal(data, &restored))
	_, err = restored.BeforeSwap(sharedSell(nil))
	require.ErrorIs(t, err, ErrInvalidState, "stale quote rejection must survive persistence")
	fresh := sharedFixture()
	fresh.BlockNumber++
	_, err = fresh.BeforeSwap(sharedSell(inventory))
	require.ErrorIs(t, err, ErrInvalidState)
}

func TestParentSimulatorPassesSharedInventory(t *testing.T) {
	ah, bh := sharedFixture(), sharedFixture()
	bh.Vaults[0] = common.HexToAddress("0x4")
	a, b := localSimulator(t, ah.Snapshot), localSimulator(t, bh.Snapshot)
	limits := a.CalculateLimit()
	for key, value := range b.CalculateLimit() {
		limits[key] = value
	}
	inventory := swaplimit.NewInventory(a.GetExchange(), limits)
	original := inventory.Clone()
	input := pool.TokenAmount{Token: a.GetTokens()[1], Amount: sharedSell(nil).AmountSpecified}
	quote, err := a.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: input, TokenOut: a.GetTokens()[0], Limit: inventory})
	require.NoError(t, err)
	info := quote.SwapInfo.(uniswapv4.SwapInfo).HookSwapInfo.(Quote)
	a.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: input, TokenAmountOut: *quote.TokenAmountOut, SwapInfo: quote.SwapInfo, SwapLimit: inventory})
	require.Equal(t, info.Swap.Balance0.Dec(), a.GetReserves()[0].String())
	_, err = b.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: input, TokenOut: b.GetTokens()[0], Limit: inventory})
	require.ErrorIs(t, err, ErrVaultIlliquid)
	_, err = b.CalcAmountIn(pool.CalcAmountInParams{TokenAmountOut: pool.TokenAmount{Token: b.GetTokens()[0], Amount: big.NewInt(1000)}, TokenIn: b.GetTokens()[1], Limit: inventory})
	require.ErrorIs(t, err, ErrVaultIlliquid)
	_, err = b.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: input, TokenOut: b.GetTokens()[0], Limit: original})
	require.NoError(t, err, "cloned route inventory stays independent")
}
