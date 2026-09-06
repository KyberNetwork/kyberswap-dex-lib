// Package ponsv2 implements PonsV2MemeHook, the singleton Uniswap v4 hook shared by
// every graduated pons-v2 pool on Robinhood chain (see the pons-v2 package for the
// pre-graduation bonding curve). Verified via Sourcify at
// 0xE5e702641Ea86F4ae6cC3cDaeD2B886f976Be044: only afterSwap (+ afterSwapReturnDelta)
// is enabled -- the hook takes hookFeeBps + creatorTaxBps of the swap's unspecified
// currency straight out of the pool manager's flash-accounting ledger via a single
// `take`, so the fee always lands on the realized output (exact-in) or is added on
// top of the realized input (exact-out). Both bps are frozen per pool at
// registerPool time, read here via the `launches` public mapping getter.
package ponsv2

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"
	"github.com/samber/lo"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type Extra struct {
	Registered bool  `json:"r,omitempty"`
	FeeBps     int64 `json:"f,omitempty"`
}

type Hook struct {
	uniswapv4.Hook `json:"-"`
	Extra
}

var _ = uniswapv4.RegisterHooksFactory(func(param *uniswapv4.HookParam) uniswapv4.Hook {
	h := &Hook{Hook: &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4PonsV2}}
	_ = param.HookExtra.Unmarshal(&h.Extra)
	return h
}, HookAddresses...)

// Track reads the pool's frozen hookFeeBps + creatorTaxBps from the hook's `launches`
// getter. Both are snapshotted immutably at registerPool time (later owner-level fee
// updates only affect pools registered afterward), so an already-registered pool is
// never re-fetched.
func (h *Hook) Track(ctx context.Context, param *uniswapv4.HookParam) (json.RawMessage, error) {
	if h.Registered {
		return json.Marshal(h)
	}

	var raw launchRaw
	if _, err := param.RpcClient.NewRequest().SetContext(ctx).SetBlockNumber(param.BlockNumber).
		SetOverrides(param.Overrides).AddCall(&ethrpc.Call{
		ABI:    hookABI,
		Target: hexutil.Encode(param.HookAddress[:]),
		Method: "launches",
		Params: []any{common.HexToHash(param.Pool.Address)},
	}, []any{&raw}).Call(); err != nil {
		return nil, err
	}

	h.Extra = Extra{
		Registered: raw.Registered,
		FeeBps:     int64(raw.HookFeeBps) + int64(raw.CreatorTaxBps),
	}
	return json.Marshal(h)
}

// AfterSwap mirrors PonsV2MemeHook._afterSwap: the fee is a flat feeBps of the
// unspecified currency's magnitude -- the realized output on exact-in, the realized
// input on exact-out -- taken directly from the pool manager, never recomputed here.
func (h *Hook) AfterSwap(params *uniswapv4.AfterSwapParams) (*uniswapv4.AfterSwapResult, error) {
	if !h.Registered || h.FeeBps == 0 {
		return &uniswapv4.AfterSwapResult{HookFee: bignumber.ZeroBI}, nil
	}
	unspecified := lo.Ternary(params.CalcOut, params.AmountOut, params.AmountIn)
	return &uniswapv4.AfterSwapResult{
		HookFee: bignumber.MulDivDown(new(big.Int), unspecified, big.NewInt(h.FeeBps), big.NewInt(basisPoints)),
	}, nil
}
