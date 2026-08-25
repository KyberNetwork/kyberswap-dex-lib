package fables

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// Hook prices Fables dynamic-fee pools on Robinhood Chain.
//
// Every Fables pool uses the v4 dynamic-fee flag and resolves its LP fee inside beforeSwap
// via the fee-override return — no swap deltas, no hook fee, no custom calldata. The hook
// composes a per-pool base fee and a TTL-bounded operator poke, clamped to the pool's on-chain
// cap and floor, and exposes the fully resolved result through the view
// `currentFee(poolId, zeroForOne)`. We read that value per direction in Track and replay it in
// BeforeSwap, so the simulator never re-derives the fee logic. slot0.lpFee is not authoritative
// for these pools. The currently deployed hooks return the same fee for both directions; the
// per-direction read is used so the handler stays correct if that changes.
//
// Fables deploys one immutable hook per pool; the fee is still tracked per poolId so the
// handler remains correct if a hook ever serves more than one pool.
type Hook struct {
	uniswapv4.Hook `json:"-"`
	// Fee0For1 is the resolved fee (pips) for a zeroForOne swap; Fee1For0 for oneForZero.
	Fee0For1 uniswapv4.FeeAmount `json:"f01"`
	Fee1For0 uniswapv4.FeeAmount `json:"f10"`
}

type Extra struct {
	Fee0For1 int64 `json:"f01"`
	Fee1For0 int64 `json:"f10"`
}

var _ = uniswapv4.RegisterHooksFactory(func(param *uniswapv4.HookParam) uniswapv4.Hook {
	hook := &Hook{
		Hook: &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4Fables},
	}
	var extra Extra
	if err := param.HookExtra.Unmarshal(&extra); err == nil {
		hook.Fee0For1 = uniswapv4.FeeAmount(extra.Fee0For1)
		hook.Fee1For0 = uniswapv4.FeeAmount(extra.Fee1For0)
	}
	return hook
}, HookAddresses...)

// Track reads the on-chain resolved fee for both swap directions. `currentFee` collapses the
// full Fables fee stack (autonomous curve + directional premium + live poke, clamped to cap)
// into a single uint24 per direction, so one call per side is all the simulator needs.
func (h *Hook) Track(ctx context.Context, param *uniswapv4.HookParam) (json.RawMessage, error) {
	hookTarget := hexutil.Encode(param.HookAddress[:])
	poolId := common.HexToHash(param.Pool.Address)

	var fee0For1, fee1For0 *big.Int
	if _, err := param.RpcClient.NewRequest().SetContext(ctx).SetBlockNumber(param.BlockNumber).
		AddCall(&ethrpc.Call{
			ABI:    fablesHookABI,
			Target: hookTarget,
			Method: "currentFee",
			Params: []any{poolId, true},
		}, []any{&fee0For1}).
		AddCall(&ethrpc.Call{
			ABI:    fablesHookABI,
			Target: hookTarget,
			Method: "currentFee",
			Params: []any{poolId, false},
		}, []any{&fee1For0}).
		Aggregate(); err != nil {
		return nil, err
	}
	return json.Marshal(Extra{
		Fee0For1: fee0For1.Int64(),
		Fee1For0: fee1For0.Int64(),
	})
}

// BeforeSwap returns the direction's resolved fee. No delta, no hook fee.
func (h *Hook) BeforeSwap(params *uniswapv4.BeforeSwapParams) (*uniswapv4.BeforeSwapResult, error) {
	fee := h.Fee1For0
	if params.ZeroForOne {
		fee = h.Fee0For1
	}
	return &uniswapv4.BeforeSwapResult{
		SwapFee:          fee,
		DeltaSpecified:   bignumber.ZeroBI,
		DeltaUnspecified: bignumber.ZeroBI,
		Gas:              gasBeforeSwap,
	}, nil
}

func (h *Hook) CloneState() uniswapv4.Hook {
	cloned := *h
	return &cloned
}
