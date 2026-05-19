package stable

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/ethrpc"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	stableng "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/stable-ng"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake/infinity/cl"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type Hook struct {
	cl.Hook
	exchange string

	hookAddress common.Address

	inner *stableng.PoolSimulator
}

func (h *Hook) GetExchange() string {
	if h == nil || h.exchange == "" {
		return valueobject.ExchangePancakeInfinityCLStable
	}
	return h.exchange
}

func (h *Hook) AllowEmptyTicks() bool { return true }

func (h *Hook) GetReserves(_ context.Context, param *cl.HookParam) (entity.PoolReserves, error) {
	var hx HookExtra
	if err := json.Unmarshal(param.HookExtra, &hx); err != nil {
		return nil, err
	}

	return hx.Balances, nil
}

func (h *Hook) BeforeSwap(p *cl.BeforeSwapParams) (*cl.BeforeSwapResult, error) {
	if h.inner == nil {
		return nil, fmt.Errorf("stable hook %s: inner simulator unavailable", h.hookAddress.Hex())
	}

	tokens := h.inner.GetTokens()

	if p.CalcOut {
		var i, j int
		if p.ZeroForOne {
			i, j = 0, 1
		} else {
			i, j = 1, 0
		}
		res, err := h.inner.CalcAmountOut(poolpkg.CalcAmountOutParams{
			TokenAmountIn: poolpkg.TokenAmount{
				Token:  tokens[i],
				Amount: new(big.Int).Set(p.AmountSpecified),
			},
			TokenOut: tokens[j],
		})
		if err != nil {
			return nil, err
		}

		return &cl.BeforeSwapResult{
			DeltaSpecified:   new(big.Int).Set(p.AmountSpecified),
			DeltaUnspecified: new(big.Int).Neg(res.TokenAmountOut.Amount),
			Gas:              defaultGas,
			SwapInfo: updateBalanceInfo{
				in:  poolpkg.TokenAmount{Token: tokens[i], Amount: new(big.Int).Set(p.AmountSpecified)},
				out: poolpkg.TokenAmount{Token: tokens[j], Amount: new(big.Int).Set(res.TokenAmountOut.Amount)},
				fee: poolpkg.TokenAmount{Token: res.Fee.Token, Amount: new(big.Int).Set(res.Fee.Amount)},
			},
		}, nil
	}

	var i, j int
	if p.ZeroForOne {
		i, j = 0, 1
	} else {
		i, j = 1, 0
	}
	res, err := h.inner.CalcAmountIn(poolpkg.CalcAmountInParams{
		TokenAmountOut: poolpkg.TokenAmount{
			Token:  tokens[j],
			Amount: new(big.Int).Set(p.AmountSpecified),
		},
		TokenIn: tokens[i],
	})
	if err != nil {
		return nil, err
	}

	return &cl.BeforeSwapResult{
		DeltaSpecified:   new(big.Int).Neg(p.AmountSpecified),
		DeltaUnspecified: new(big.Int).Set(res.TokenAmountIn.Amount),
		Gas:              defaultGas,
		SwapInfo: updateBalanceInfo{
			in:  poolpkg.TokenAmount{Token: tokens[i], Amount: new(big.Int).Set(res.TokenAmountIn.Amount)},
			out: poolpkg.TokenAmount{Token: tokens[j], Amount: new(big.Int).Set(p.AmountSpecified)},
			fee: poolpkg.TokenAmount{Token: res.Fee.Token, Amount: new(big.Int).Set(res.Fee.Amount)},
		},
	}, nil
}

func (h *Hook) AfterSwap(_ *cl.AfterSwapParams) (*cl.AfterSwapResult, error) {
	return &cl.AfterSwapResult{HookFee: bignumber.ZeroBI}, nil
}

func (h *Hook) CloneState() cl.Hook {
	cloned := *h
	if h.inner != nil {
		cloned.inner = h.inner.CloneState().(*stableng.PoolSimulator)
	}

	return &cloned
}

func (h *Hook) UpdateBalance(swapInfo any) {
	if h.inner == nil {
		return
	}
	info, ok := swapInfo.(updateBalanceInfo)
	if !ok {
		return
	}

	h.inner.UpdateBalance(poolpkg.UpdateBalanceParams{
		TokenAmountIn:  info.in,
		TokenAmountOut: info.out,
		Fee:            info.fee,
	})
}

type updateBalanceInfo struct {
	in, out, fee poolpkg.TokenAmount
}

func (h *Hook) Track(ctx context.Context, param *cl.HookParam) ([]byte, error) {
	if param.RpcClient == nil {
		return param.HookExtra, nil
	}
	hookAddr := param.HookAddress.Hex()

	out := struct {
		balances            []*big.Int
		storedRates         []*big.Int
		totalSupply         *big.Int
		initialA            *big.Int
		futureA             *big.Int
		initialATime        *big.Int
		futureATime         *big.Int
		fee                 *big.Int
		adminFee            *big.Int
		offpegFeeMultiplier *big.Int
		nCoins              *big.Int
	}{
		initialATime: new(big.Int),
		futureATime:  new(big.Int),
	}

	req := param.RpcClient.NewRequest().SetContext(ctx)
	if param.BlockNumber != nil {
		req = req.SetBlockNumber(param.BlockNumber)
	}

	addHookCall(req, hookAddr, "get_balances", &out.balances)
	addHookCall(req, hookAddr, "stored_rates", &out.storedRates)
	addHookCall(req, hookAddr, "totalSupply", &out.totalSupply)
	addHookCall(req, hookAddr, "initial_A", &out.initialA)
	addHookCall(req, hookAddr, "future_A", &out.futureA)
	addHookCall(req, hookAddr, "initial_A_time", &out.initialATime)
	addHookCall(req, hookAddr, "future_A_time", &out.futureATime)
	addHookCall(req, hookAddr, "fee", &out.fee)
	addHookCall(req, hookAddr, "admin_fee", &out.adminFee)
	addHookCall(req, hookAddr, "offpeg_fee_multiplier", &out.offpegFeeMultiplier)
	addHookCall(req, hookAddr, "N_COINS", &out.nCoins)

	if _, err := req.Aggregate(); err != nil {
		return nil, fmt.Errorf("stable hook %s track: %w", hookAddr, err)
	}

	return json.Marshal(HookExtra{
		Balances:            lo.Map(out.balances, func(b *big.Int, _ int) string { return b.String() }),
		Rates:               lo.Map(out.storedRates, func(r *big.Int, _ int) string { return r.String() }),
		LpSupply:            out.totalSupply.String(),
		InitialA:            out.initialA.String(),
		FutureA:             out.futureA.String(),
		InitialATime:        out.initialATime.Int64(),
		FutureATime:         out.futureATime.Int64(),
		SwapFee:             out.fee.String(),
		AdminFee:            out.adminFee.String(),
		OffpegFeeMultiplier: out.offpegFeeMultiplier.String(),
	})
}

func addHookCall(req *ethrpc.Request, hookAddr, method string, dest any) {
	req.AddCall(&ethrpc.Call{
		ABI:    hookABI,
		Target: hookAddr,
		Method: method,
		Params: []any{},
	}, []any{dest})
}

// buildInner constructs the curve-stable-ng simulator from HookExtra.
func buildInner(p *entity.Pool, hookExtraBytes []byte) (*stableng.PoolSimulator, error) {
	var hx HookExtra
	if err := json.Unmarshal(hookExtraBytes, &hx); err != nil {
		return nil, err
	}

	nCoins := len(p.Tokens)

	rates := make([]uint256.Int, nCoins)
	for i := 0; i < nCoins; i++ {
		if err := rates[i].SetFromDecimal(hx.Rates[i]); err != nil {
			return nil, fmt.Errorf("parse rates[%d]=%q: %w", i, hx.Rates[i], err)
		}
	}

	seBytes, err := json.Marshal(stableng.StaticExtra{
		APrecision:          uint256.NewInt(APrecision),
		OffpegFeeMultiplier: uint256.MustFromDecimal(hx.OffpegFeeMultiplier),
		IsNativeCoins:       make([]bool, len(p.Tokens)),
	})
	if err != nil {
		return nil, err
	}

	exBytes, err := json.Marshal(stableng.Extra{
		InitialA:        uint256.MustFromDecimal(hx.InitialA),
		FutureA:         uint256.MustFromDecimal(hx.FutureA),
		InitialATime:    hx.InitialATime,
		FutureATime:     hx.FutureATime,
		SwapFee:         uint256.MustFromDecimal(hx.SwapFee),
		AdminFee:        uint256.MustFromDecimal(hx.AdminFee),
		RateMultipliers: rates,
	})
	if err != nil {
		return nil, err
	}

	reserves := make(entity.PoolReserves, nCoins+1)
	for i := 0; i < nCoins; i++ {
		reserves[i] = hx.Balances[i]
	}
	reserves[nCoins] = hx.LpSupply

	inner, err := stableng.NewPoolSimulator(entity.Pool{
		Address:     p.Address,
		Exchange:    p.Exchange,
		Type:        p.Type,
		Tokens:      p.Tokens,
		Reserves:    reserves,
		StaticExtra: string(seBytes),
		Extra:       string(exBytes),
	})
	if err != nil {
		return nil, err
	}

	return inner, nil
}
