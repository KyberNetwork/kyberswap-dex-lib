// Package o1 implements the Uniswap v4 hook shared by every o1 launchpad pool. It
// is the same LaunchHook.sol lineage as hooks/b20 (same decay/fee logic, reused
// directly via b20.Extra), deployed separately on Base at
// 0x1f91c998e7c2f4b690d75bdbf6502bdcd6e02acc with a reworked poolConfig layout
// that doesn't change the swap-visible fee amount -- only Track's ABI decode
// differs from b20.
package o1

import (
	"context"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/b20"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var _ = uniswapv4.RegisterHooksFactory(func(param *uniswapv4.HookParam) uniswapv4.Hook {
	h := &Hook{Hook: &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4O1}}
	_ = param.HookExtra.Unmarshal(&h.Extra)
	return h
}, HookAddresses...)

type Hook struct {
	uniswapv4.Hook `json:"-"`
	b20.Extra
}

// Track reads poolConfig, whose field set/order differs from b20's (see
// hooks/b20/abis.go) but whose pricing-relevant fields decode straight into the
// shared b20.Extra. Reuses an already-populated Extra as-is (LaunchTime != 0)
// since baseFeeBps/antiSnipe* never change after registration.
func (h *Hook) Track(ctx context.Context, param *uniswapv4.HookParam) (json.RawMessage, error) {
	if h.LaunchTime != 0 {
		return json.Marshal(h)
	}

	var raw poolConfigRaw
	req := param.RpcClient.NewRequest().SetContext(ctx).SetBlockNumber(param.BlockNumber).
		SetOverrides(param.Overrides).AddCall(&ethrpc.Call{
		ABI:    LaunchHookABI,
		Target: hexutil.Encode(param.HookAddress[:]),
		Method: "poolConfig",
		Params: []any{common.HexToHash(param.Pool.Address)},
	}, []any{&raw})
	if _, err := req.Aggregate(); err != nil {
		return nil, err
	}
	if !raw.Initialized {
		return nil, b20.ErrNotRegistered
	}

	h.Extra = b20.Extra{
		TokenIsCurrency0:       raw.TokenIsCurrency0,
		BaseFeeBps:             int64(raw.BaseFeeBps),
		AntiSnipeStartTotalBps: int64(raw.AntiSnipeStartTotalBps),
		AntiSnipeWindowSeconds: int64(raw.AntiSnipeWindowSeconds),
		LaunchTime:             raw.LaunchTime.Int64(),
	}
	return json.Marshal(h)
}

// Delegate rather than rely on promotion: Hook embeds both the uniswapv4.Hook
// interface and b20.Extra at the same depth, both with BeforeSwap/AfterSwap, so an
// unqualified call would otherwise be an ambiguous selector.
func (h *Hook) BeforeSwap(params *uniswapv4.BeforeSwapParams) (*uniswapv4.BeforeSwapResult, error) {
	return h.Extra.BeforeSwap(params)
}

func (h *Hook) AfterSwap(params *uniswapv4.AfterSwapParams) (*uniswapv4.AfterSwapResult, error) {
	return h.Extra.AfterSwap(params)
}
