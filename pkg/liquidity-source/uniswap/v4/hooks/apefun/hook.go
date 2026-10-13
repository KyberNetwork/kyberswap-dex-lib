// Package apefun implements LaunchFeeProtectionHook, the one Uniswap v4 hook of every coin launched on ape.fun (Arc
// and Robinhood, see Deployments). Pools have fee 0 and tickSpacing 1; the hook takes every fee in the quote asset:
//
//   - buy (quote in): 1% + creatorTaxBps + an anti-snipe rate in the first 3 s, each floored on the gross input and
//     taken in beforeSwap as a specified delta, so the pool swaps what is left.
//   - sell (coin in): 1% + creatorTaxBps, floored on the pool's quote output and taken in afterSwap.
//   - only exact-input swaps are accepted, and only once the creation transaction has ended (Launch.stage 3).
//
// Snipe exemptions are keyed by the ape.fun router's recipient; any other sender is never exempt, so routed quotes
// are exact, read one second early in the window (see snipeRate). Per-pool state is written once at launch.
package apefun

import (
	"context"
	"errors"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var (
	ErrNotTracked     = errors.New("apefun: pool not registered, not launched or not tracked yet")
	ErrAmountTooLarge = errors.New("apefun: amount above int128 max, the hook reverts UnsupportedSwap")
)

// NowFn is the clock the anti-snipe window reads; a variable so tests can pin it.
var NowFn = func() int64 { return time.Now().Unix() }

var maxInt128 = new(big.Int).Sub(new(big.Int).Lsh(bignumber.One, 127), bignumber.One)

// Extra is the hook state kept per pool, all written once at launch.
type Extra struct {
	Live             bool  `json:"v,omitempty"`  // registered with the hook and Launch.stage == 3
	QuoteIsCurrency0 bool  `json:"q0,omitempty"` // which side is the quote asset (buy = quote in)
	TaxBps           int64 `json:"t,omitempty"`  // policies(id).tax, creatorTaxBps
	Launched         int64 `json:"l,omitempty"`  // policies(id).launched, the creation block's timestamp
	BuybackVault     bool  `json:"bv,omitempty"` // buybackVaults(id) != 0: afterSwap records an observation (gas)
}

// Hook embeds BaseHook by value, not the Hook interface, so a simulator survives the msgpack round trip.
type Hook struct {
	uniswapv4.BaseHook `json:"-"`
	Extra
}

var _ = uniswapv4.RegisterHooksFactory(func(param *uniswapv4.HookParam) uniswapv4.Hook {
	h := &Hook{BaseHook: uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4ApeFun}}
	if onDeploymentChain(param) {
		_ = param.HookExtra.Unmarshal(&h.Extra)
	}
	return h
}, HookAddresses...)

// onDeploymentChain: the hook address is only ape.fun's on the chain it was deployed on.
func onDeploymentChain(param *uniswapv4.HookParam) bool {
	d, ok := Deployments[param.HookAddress]
	return ok && param.Cfg != nil && param.Cfg.ChainID == d.ChainID
}

// Track reads policies(id) and buybackVaults(id) from the hook and launchForToken(coin) from the factory in one
// multicall, pinned to param.BlockNumber when set (on Robinhood pool-service's ArbMulticall2 reports the L2 block).
// All are immutable once the creation transaction ends, so a live pool is never re-read.
func (h *Hook) Track(ctx context.Context, param *uniswapv4.HookParam) (json.RawMessage, error) {
	if h.Live {
		return json.Marshal(h.Extra)
	}
	h.Extra = Extra{}
	if !onDeploymentChain(param) || len(param.Pool.Tokens) != 2 {
		return json.Marshal(h.Extra)
	}
	d := Deployments[param.HookAddress]
	token0, token1 := common.HexToAddress(param.Pool.Tokens[0].Address), common.HexToAddress(param.Pool.Tokens[1].Address)
	quote0, coin := token0 == d.QuoteAsset, token0
	if quote0 {
		coin = token1
	} else if token1 != d.QuoteAsset {
		return json.Marshal(h.Extra)
	}

	poolID := common.HexToHash(param.Pool.Address)
	hook := hexutil.Encode(param.HookAddress[:])
	var (
		policy policyRaw
		vault  common.Address
		launch launchResult
	)
	req := param.RpcClient.NewRequest().SetContext(ctx).SetOverrides(param.Overrides)
	if param.BlockNumber != nil {
		req.SetBlockNumber(param.BlockNumber)
	}
	if _, err := req.
		AddCall(&ethrpc.Call{ABI: hookABI, Target: hook, Method: "policies", Params: []any{poolID}}, []any{&policy}).
		AddCall(&ethrpc.Call{ABI: hookABI, Target: hook, Method: "buybackVaults", Params: []any{poolID}}, []any{&vault}).
		AddCall(&ethrpc.Call{ABI: factoryABI, Target: hexutil.Encode(d.Factory[:]), Method: "launchForToken",
			Params: []any{coin}}, []any{&launch}).
		Aggregate(); err != nil {
		return nil, err
	}

	h.Extra = extraFrom(&policy, &launch.Launch, poolID, quote0, vault)
	return json.Marshal(h.Extra)
}

// extraFrom keeps a pool only if the hook registered it (destination != 0, beforeSwap reverts otherwise), the
// factory's launch for the coin is this pool and has left its creation transaction (stage 3, beforeSwap reverts
// InvalidState for any outside sender before), and the tax is one register accepts.
func extraFrom(policy *policyRaw, launch *launchRaw, poolID common.Hash, quote0 bool, vault common.Address) Extra {
	if policy.Destination == (common.Address{}) || launch.Stage != stageTrades || launch.PoolId != poolID ||
		policy.Tax > maxTaxBps {
		return Extra{}
	}
	return Extra{
		Live:             true,
		QuoteIsCurrency0: quote0,
		TaxBps:           int64(policy.Tax),
		Launched:         int64(policy.Launched),
		BuybackVault:     vault != (common.Address{}),
	}
}

// snipeRate mirrors the rate in LaunchFeeProtectionHook._fees for a buy by a non-exempt recipient (every sender
// other than the ape.fun router passes recipient 0, never exempt): by elapsed second, capped at 9800 - tax. The
// quoting clock can run ahead of block timestamps, so elapsed is read clockMarginSeconds earlier: a quote never
// charges less than the chain, and is exact again from 3 + clockMarginSeconds seconds after launch.
func (e *Extra) snipeRate(now int64) int64 {
	elapsed := max(now-e.Launched-clockMarginSeconds, 0)
	if elapsed >= int64(len(snipeBps)) {
		return 0
	}
	return min(snipeBps[elapsed], maxTakeBps-e.TaxBps)
}

// fees mirrors LaunchFeeProtectionHook._fees: snipe, base fee and tax each floored separately on gross (the input
// of a buy, the pool's quote output of a sell). It returns fee + tax, what the hook keeps.
func (e *Extra) fees(gross *big.Int, buy bool, now int64) *big.Int {
	var total, part big.Int
	addFloor := func(rateBps int64) {
		if rateBps > 0 {
			total.Add(&total, part.Quo(part.Mul(gross, big.NewInt(rateBps)), bignumber.BasisPoint))
		}
	}
	if buy {
		addFloor(e.snipeRate(now))
	}
	addFloor(baseFeeBps)
	addFloor(e.TaxBps)
	return &total
}

// grossFor returns the smallest gross with gross - fees(gross) >= net. Each floor rounds a fee down, so the
// unfloored gross-up bounds it from above.
func (e *Extra) grossFor(net *big.Int, buy bool, now int64) (*big.Int, error) {
	var takeBps int64 = baseFeeBps + e.TaxBps
	if buy {
		takeBps += e.snipeRate(now)
	}
	guess := uniswapv4.GrossBeforeFee(net, big.NewInt(takeBps), bignumber.BasisPoint)
	return uniswapv4.MinAmountIn(net, guess, func(gross *big.Int) (*big.Int, error) {
		return new(big.Int).Sub(gross, e.fees(gross, buy, now)), nil
	})
}

// CloneState returns a copy; the state is never written after Track (UpdateBalance is the base no-op).
func (h *Hook) CloneState() uniswapv4.Hook {
	c := *h
	return &c
}

func (e *Extra) isBuy(zeroForOne bool) bool {
	return zeroForOne == e.QuoteIsCurrency0
}

// BeforeSwap mirrors LaunchFeeProtectionHook.beforeSwap (LP fee override 0). CalcOut: a buy's fee and tax come
// off the gross input. CalcIn reverses exact-in: a sell grosses the pool's quote output up so the afterSwap fee
// leaves AmountSpecified; a buy is grossed up in AfterSwap, once the pool's input is known.
func (h *Hook) BeforeSwap(params *uniswapv4.BeforeSwapParams) (*uniswapv4.BeforeSwapResult, error) {
	if !h.Live {
		return nil, ErrNotTracked
	}
	buy := h.isBuy(params.ZeroForOne)
	res := &uniswapv4.BeforeSwapResult{
		DeltaSpecified:   bignumber.ZeroBI,
		DeltaUnspecified: bignumber.ZeroBI,
		Gas:              gasBeforeSwapSell,
	}
	if buy {
		res.Gas = gasBeforeSwapBuy
	}
	if params.CalcOut {
		// beforeSwap: amountSpecified < 0 (exact input) and -amountSpecified <= type(int128).max
		if params.AmountSpecified.Cmp(maxInt128) > 0 {
			return nil, ErrAmountTooLarge
		}
		if buy {
			res.DeltaSpecified = h.fees(params.AmountSpecified, true, NowFn())
		}
	} else if !buy {
		gross, err := h.grossFor(params.AmountSpecified, false, NowFn())
		if err != nil {
			return nil, err
		}
		res.DeltaSpecified = gross.Sub(gross, params.AmountSpecified)
	}
	return res, nil
}

// AfterSwap mirrors LaunchFeeProtectionHook.afterSwap: a sell returns fee + tax of the pool's quote output as the
// unspecified delta. CalcIn: a buy's input is grossed up so the beforeSwap fee leaves the pool's AmountIn.
func (h *Hook) AfterSwap(params *uniswapv4.AfterSwapParams) (*uniswapv4.AfterSwapResult, error) {
	if !h.Live {
		return nil, ErrNotTracked
	}
	buy := h.isBuy(params.ZeroForOne)
	res := &uniswapv4.AfterSwapResult{HookFee: bignumber.ZeroBI, Gas: gasAfterSwapSell}
	if buy {
		res.Gas = gasAfterSwapBuy
	}
	if h.BuybackVault {
		res.Gas += gasObservation
	}
	if params.CalcOut && !buy {
		res.HookFee = h.fees(params.AmountOut, false, NowFn())
	} else if !params.CalcOut && buy {
		gross, err := h.grossFor(params.AmountIn, true, NowFn())
		if err != nil {
			return nil, err
		}
		res.HookFee = gross.Sub(gross, params.AmountIn)
	}
	return res, nil
}
