package tickerdualpool

import (
	"context"
	_ "embed"
	"errors"
	"math/big"
	"strings"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/dualpool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

//go:embed abis/Ticker.json
var tickerABIJSON string
var tickerABI = func() abi.ABI {
	a, err := abi.JSON(strings.NewReader(tickerABIJSON))
	if err != nil {
		panic(err)
	}
	return a
}()
var ErrRefreshRequired = errors.New("Ticker vault state must be refreshed after simulated JIT teardown")

// Snapshot holds one pool's tracked state and local vault/reserve accounting.
type Snapshot struct {
	Currencies               [2]common.Address
	Reserves                 [2]*AaveReserve
	Limits                   [2]*AaveLimits
	Raw, Claims, ManagerCash [2]*uint256.Int
	Aave                     [2]*AaveInventory
	State
	Vaults      [2]common.Address
	BlockNumber uint64
	// Supplied from measured execution, not borrowed from Twofold's gas estimate.
	SwapGas int64
}

type Hook struct {
	uniswapv4.BaseHook
	Snapshot
	// Persist rejection flags so saving/loading cannot revive invalid quotes.
	Invalid         bool
	RefreshRequired bool
	address         common.Address
	decodeErr       error
}

var _ uniswapv4.Hook = (*Hook)(nil)

var productionHookAddress = common.HexToAddress("0x047ea2f55148b7601fdde0d28588b55c09192ac4")

var _ = uniswapv4.RegisterHooksFactory(NewHook, productionHookAddress)

// NewHook loads the tracked snapshot; Track validates the proxy implementation.
func NewHook(p *uniswapv4.HookParam) uniswapv4.Hook {
	h := &Hook{address: p.HookAddress}
	if p.HookAddress == productionHookAddress && (p.Cfg == nil || p.Cfg.ChainID != valueobject.ChainIDMonad) {
		h.invalidate(ErrInvalidState)
		return h
	}
	if len(p.HookExtra) > 0 {
		var snapshot Snapshot
		if err := p.HookExtra.Unmarshal(&snapshot); err != nil {
			h.invalidate(err)
		} else {
			h.Snapshot = snapshot
		}
	}
	return h
}
func (h *Hook) invalidate(err error) {
	h.decodeErr = err
	h.Invalid = true
}

func (h *Hook) GetExchange() string               { return valueobject.ExchangeUniswapV4TickerDualPool }
func (h *Hook) AllowEmptyTicks() bool             { return true }
func (h *Hook) CanBeforeSwap(common.Address) bool { return true }
func (h *Hook) CanAfterSwap(common.Address) bool  { return false }

func (h *Hook) BeforeSwap(p *uniswapv4.BeforeSwapParams) (*uniswapv4.BeforeSwapResult, error) {
	if h.decodeErr != nil {
		return nil, h.decodeErr
	}
	if h.Invalid {
		return nil, ErrInvalidState
	}
	if h.RefreshRequired {
		return nil, ErrRefreshRequired
	}
	if p.AmountSpecified == nil || p.AmountSpecified.Sign() <= 0 {
		return nil, ErrInvalidState
	}
	amount, overflow := uint256.FromBig(p.AmountSpecified)
	if overflow {
		return nil, ErrInvalidState
	}
	state := h.State
	// Recompute available vault cash after each simulated reserve change.
	// Keep total share value separate so the 100% check sees any shortage.
	for side, inventory := range h.Aave {
		if inventory == nil || h.Reserves[side] == nil {
			continue
		}
		available, err := inventory.realizableAssets(*h.Reserves[side])
		if err != nil {
			return nil, err
		}
		if side == 0 {
			state.Pool.Balance0 = available
		} else {
			state.Pool.Balance1 = available
		}
	}
	var q Quote
	var err error
	if p.CalcOut {
		q, err = QuoteExactInput(state, p.ZeroForOne, amount)
	} else {
		q, err = QuoteRequiredInput(state, p.ZeroForOne, amount)
	}
	if err != nil {
		return nil, err
	}
	for side, inventory := range h.Aave {
		if inventory == nil {
			continue
		}
		minted, returned := q.Swap.Mint0, q.Swap.Burn0
		if side == 1 {
			minted, returned = q.Swap.Mint1, q.Swap.Burn1
		}
		var next AaveInventory
		var err error
		if reserve := h.Reserves[side]; reserve != nil {
			var nextReserve AaveReserve
			next, nextReserve, err = inventory.AfterJITWithReserve(minted, returned, *reserve)
			q.ReservesAfter[side] = &nextReserve
		} else {
			next, err = inventory.AfterJIT(minted, returned)
		}
		if err != nil {
			return nil, err
		}
		assets, err := next.poolAssets()
		if err != nil {
			return nil, err
		}
		q.VaultsAfter[side] = &next
		if side == 0 {
			q.Swap.Balance0 = assets
		} else {
			q.Swap.Balance1 = assets
		}
	}
	for side := 0; side < 2; side++ {
		if h.Raw[side] == nil || h.Claims[side] == nil || h.ManagerCash[side] == nil {
			continue
		}
		minted, returned := q.Swap.Mint0, q.Swap.Burn0
		if side == 1 {
			minted, returned = q.Swap.Mint1, q.Swap.Burn1
		}
		if next := q.VaultsAfter[side]; next != nil {
			q.RawAfter[side], q.ClaimsAfter[side] = next.Raw.Clone(), next.Claims.Clone()
		} else {
			q.RawAfter[side], q.ClaimsAfter[side], err = inventoryAfterJIT(h.Raw[side], h.Claims[side], minted, returned)
			if err != nil {
				return nil, err
			}
		}
		cash, err := managerCashAfterJIT(h.ManagerCash[side], h.Claims[side], minted, returned)
		if err != nil {
			return nil, err
		}
		isInput := (side == 0) == p.ZeroForOne
		// Exact-input hook fees are taken as tokens before the router settles.
		// CalcAmountIn also executes exact-input; only its search is reversed.
		if !isInput {
			fees, err := addChecked(q.HookFee, q.CreatorTax)
			if err != nil {
				return nil, err
			}
			if cash.Cmp(fees) < 0 {
				return nil, ErrVaultIlliquid
			}
			cash.Sub(cash, fees)
		}
		if isInput {
			cash, err = addChecked(cash, q.Swap.AmountIn)
			if err != nil {
				return nil, err
			}
		} else {
			if cash.Cmp(q.NetOutput) < 0 {
				return nil, ErrVaultIlliquid
			}
			cash.Sub(cash, q.NetOutput)
		}
		q.ManagerCashAfter[side] = cash
	}
	result := &uniswapv4.BeforeSwapResult{SwapFee: uniswapv4.FeeAmount(h.State.Pool.LpFee), Gas: h.quoteGas(q), SwapInfo: q}
	if p.CalcOut {
		result.DeltaSpecified = q.Swap.AmountIn.ToBig()
		result.DeltaUnspecified = new(big.Int).Neg(q.NetOutput.ToBig())
	} else {
		// CalcAmountIn reverses exact-in: add the output needed before fees.
		// DualPool still supplies the input price for its temporary liquidity.
		result.DeltaSpecified = new(big.Int).Sub(q.Swap.AmountOut.ToBig(), p.AmountSpecified)
		result.DeltaUnspecified = q.Swap.AmountIn.ToBig()
	}
	return result, nil
}

// UpdateBalance models a settled leg, including its input payment and output payout.
func (h *Hook) UpdateBalance(info any) {
	q, ok := info.(Quote)
	if !ok || q.Swap.Balance0 == nil || q.Swap.Balance1 == nil {
		return
	}
	h.State.Pool.Balance0 = q.Swap.Balance0.Clone()
	h.State.Pool.Balance1 = q.Swap.Balance1.Clone()
	h.Total0 = q.Swap.Balance0.Clone()
	h.Total1 = q.Swap.Balance1.Clone()
	h.State.Pool.SqrtPriceX96 = q.Swap.SqrtPriceX96.Clone()
	h.State.Pool.Tick = q.Swap.Tick
	// JIT arithmetic alone does not model an ERC-4626 withdrawal/redeposit.
	// Keep this unfinished path explicit rather than publish inaccurate quotes.
	h.RefreshRequired = false
	for side := range h.Aave {
		if q.RawAfter[side] != nil && q.ClaimsAfter[side] != nil && q.ManagerCashAfter[side] != nil {
			h.Raw[side], h.Claims[side] = q.RawAfter[side].Clone(), q.ClaimsAfter[side].Clone()
			cash := q.ManagerCashAfter[side]
			h.ManagerCash[side] = cash.Clone()
			unbacked := unbackedClaims(h.Claims[side], cash)
			if side == 0 {
				h.Unbacked0 = unbacked
			} else {
				h.Unbacked1 = unbacked
			}
		}
		if q.VaultsAfter[side] != nil {
			next := q.VaultsAfter[side].Clone()
			h.Aave[side] = &next
			if q.ReservesAfter[side] != nil {
				reserve := q.ReservesAfter[side].Clone()
				h.Reserves[side] = &reserve
			}
		} else if h.Vaults[side] != (common.Address{}) {
			h.RefreshRequired = true
		}
	}
}

func (h *Hook) CloneState() uniswapv4.Hook {
	clone := *h
	for side, inventory := range h.Aave {
		if inventory != nil {
			copied := inventory.Clone()
			clone.Aave[side] = &copied
		}
	}
	copyInt := func(v *uint256.Int) *uint256.Int {
		if v == nil {
			return nil
		}
		return v.Clone()
	}
	for side := range h.Raw {
		if h.Reserves[side] != nil {
			reserve := h.Reserves[side].Clone()
			clone.Reserves[side] = &reserve
		}
		clone.Raw[side] = copyInt(h.Raw[side])
		clone.Claims[side] = copyInt(h.Claims[side])
		clone.ManagerCash[side] = copyInt(h.ManagerCash[side])
		if limits := h.Limits[side]; limits != nil {
			clone.Limits[side] = &AaveLimits{MaxDeposit: copyInt(limits.MaxDeposit), MaxWithdraw: copyInt(limits.MaxWithdraw)}
		}
	}
	clone.Total0 = copyInt(h.Total0)
	clone.Total1 = copyInt(h.Total1)
	clone.Unbacked0 = copyInt(h.Unbacked0)
	clone.Unbacked1 = copyInt(h.Unbacked1)
	clone.State.Pool.Balance0 = copyInt(h.State.Pool.Balance0)
	clone.State.Pool.Balance1 = copyInt(h.State.Pool.Balance1)
	clone.State.Pool.SqrtPriceX96 = copyInt(h.State.Pool.SqrtPriceX96)
	clone.State.Pool.Buckets = append([]dualpool.Bucket(nil), h.State.Pool.Buckets...)
	return &clone
}
func (h *Hook) GetReserves(context.Context, *uniswapv4.HookParam) (entity.PoolReserves, error) {
	if h.State.Pool.Balance0 == nil || h.State.Pool.Balance1 == nil {
		return nil, ErrInvalidState
	}
	return entity.PoolReserves{h.State.Pool.Balance0.Dec(), h.State.Pool.Balance1.Dec()}, nil
}

type launchResponse struct {
	Registered, MemecoinIsCurrency0                                                           bool
	Memecoin, QuoteToken, Creator, BuybackCreatorRecipient, ProtocolFeeRecipient              common.Address
	CreatorTaxBps, ProtocolFeeShareBps, BuybackBurnBps, HookFeeBps, MaxInternalPriceImpactBps uint16
	BuybackEnabled                                                                            bool
}

func (h *Hook) trackBase(ctx context.Context, p *uniswapv4.HookParam) (json.RawMessage, error) {
	if p.BlockNumber == nil || p.BlockNumber.Sign() <= 0 || p.Cfg == nil || p.Cfg.StateViewAddress == "" || p.Pool == nil || len(p.Pool.Tokens) != 2 {
		return nil, ErrInvalidState
	}
	var config uniswapv4.StaticExtra
	if err := json.Unmarshal([]byte(p.Pool.StaticExtra), &config); err != nil {
		return nil, err
	}
	if config.HooksAddress != h.address {
		return nil, ErrInvalidState
	}
	key := uniswapv4.PoolKey{Fee: big.NewInt(int64(config.Fee)), TickSpacing: big.NewInt(int64(config.TickSpacing)), Hooks: h.address}
	currencies := [2]common.Address{common.HexToAddress(p.Pool.Tokens[0].Address), common.HexToAddress(p.Pool.Tokens[1].Address)}
	for i, native := range config.IsNative {
		if native {
			currencies[i] = common.Address{}
		}
	}
	key.Currency0 = currencies[0]
	key.Currency1 = currencies[1]
	id := common.HexToHash(p.Pool.Address)
	// Both requests use the block chosen by the parent v4 tracker, and the same overrides.
	raw, err := dualpool.NewStateReader(h.address).Track(ctx, p)
	if err != nil {
		return nil, err
	}
	next := Snapshot{BlockNumber: p.BlockNumber.Uint64(), SwapGas: h.SwapGas}
	if err = json.Unmarshal(raw, &next.State.Pool); err != nil {
		return nil, err
	}
	var total struct{ Amount0, Amount1 *big.Int }
	var launch launchResponse
	req := p.RpcClient.NewRequest().SetContext(ctx).SetBlockNumber(p.BlockNumber).SetOverrides(p.Overrides)
	target := hexutil.Encode(h.address[:])
	req.AddCall(&ethrpc.Call{ABI: tickerABI, Target: target, Method: "totalAssets", Params: []any{key}}, []any{&total})
	req.AddCall(&ethrpc.Call{ABI: tickerABI, Target: target, Method: "launches", Params: []any{id}}, []any{&launch})
	for i := range currencies {
		req.AddCall(&ethrpc.Call{ABI: tickerABI, Target: target, Method: "vaults", Params: []any{id, currencies[i]}}, []any{&next.Vaults[i]})
	}
	if _, err = req.Aggregate(); err != nil {
		return nil, err
	}
	if total.Amount0 == nil || total.Amount1 == nil {
		return nil, ErrInvalidState
	}
	next.Total0 = uint256.MustFromBig(total.Amount0)
	next.Total1 = uint256.MustFromBig(total.Amount1)
	next.TickSpacing = int(config.TickSpacing)
	next.Registered = launch.Registered
	next.HookFeeBps = launch.HookFeeBps
	next.CreatorTaxBps = launch.CreatorTaxBps
	h.Snapshot = next
	h.decodeErr = nil
	h.Invalid = false
	h.RefreshRequired = false
	return json.Marshal(next)
}

// JIT positions are removed after every swap. Publish the hook's exact backing
// and price to the parent simulator instead of its generic reserve deltas.
func (h *Hook) PoolState() uniswapv4.HookPoolState {
	return uniswapv4.HookPoolState{Reserves: [2]*big.Int{h.Total0.ToBig(), h.Total1.ToBig()}, SqrtPriceX96: *h.State.Pool.SqrtPriceX96, Tick: int(h.State.Pool.Tick)}
}
