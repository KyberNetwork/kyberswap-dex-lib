package volatile

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type Extra struct {
	Tracked       bool                `json:"t,omitempty"`
	FeeZeroForOne uniswapv4.FeeAmount `json:"f0,omitempty"`
	FeeOneForZero uniswapv4.FeeAmount `json:"f1,omitempty"`
}

type Hook struct {
	uniswapv4.Hook `json:"-"`
	Extra
}

var _ = uniswapv4.RegisterHooksFactory(func(param *uniswapv4.HookParam) uniswapv4.Hook {
	h := &Hook{Hook: &uniswapv4.BaseHook{Exchange: DexType}}
	_ = param.HookExtra.Unmarshal(&h.Extra)
	return h
}, HookAddress)

func (h *Hook) Track(ctx context.Context, param *uniswapv4.HookParam) (json.RawMessage, error) {
	staticExtra := new(uniswapv4.StaticExtra)
	if err := json.Unmarshal([]byte(param.Pool.StaticExtra), staticExtra); err != nil {
		return nil, err
	}

	poolKey := newPoolKey(param.Pool, staticExtra, param.HookAddress)
	// ethrpc needs one struct target (abi tags) for a multi-output call.
	var fees struct {
		FeeZeroForOne *big.Int `abi:"feeE6ZeroForOne"` // uint24 decodes to *big.Int
		FeeOneForZero *big.Int `abi:"feeE6OneForZero"`
	}
	if _, err := param.RpcClient.NewRequest().SetContext(ctx).SetBlockNumber(param.BlockNumber).
		SetOverrides(param.Overrides).
		AddCall(&ethrpc.Call{
			ABI:    volatilePairHookABI,
			Target: hexutil.Encode(param.HookAddress[:]),
			Method: "getFee",
			Params: []any{poolKey},
		}, []any{&fees}).Aggregate(); err != nil {
		return nil, err
	}
	if fees.FeeZeroForOne == nil || fees.FeeOneForZero == nil ||
		fees.FeeZeroForOne.Uint64() >= uint64(uniswapv4.FeeMax) || fees.FeeOneForZero.Uint64() >= uint64(uniswapv4.FeeMax) {
		return nil, ErrInvalidSwapFee
	}

	h.Extra = Extra{
		Tracked:       true,
		FeeZeroForOne: uniswapv4.FeeAmount(fees.FeeZeroForOne.Uint64()),
		FeeOneForZero: uniswapv4.FeeAmount(fees.FeeOneForZero.Uint64()),
	}
	return json.Marshal(h)
}

func (h *Hook) BeforeSwap(params *uniswapv4.BeforeSwapParams) (*uniswapv4.BeforeSwapResult, error) {
	if !h.Tracked {
		return nil, ErrPoolIsNotTracked
	}

	fee := h.FeeOneForZero
	if params.ZeroForOne {
		fee = h.FeeZeroForOne
	}

	return &uniswapv4.BeforeSwapResult{
		DeltaSpecified:   bignumber.ZeroBI,
		DeltaUnspecified: bignumber.ZeroBI,
		SwapFee:          fee,
		Gas:              HookGas,
	}, nil
}

func (h *Hook) CloneState() uniswapv4.Hook {
	cloned := *h
	return &cloned
}

func newPoolKey(pool *entity.Pool, staticExtra *uniswapv4.StaticExtra, hook common.Address) uniswapv4.PoolKey {
	return uniswapv4.PoolKey{
		Currency0:   currency(staticExtra.IsNative[0], pool.Tokens[0].Address),
		Currency1:   currency(staticExtra.IsNative[1], pool.Tokens[1].Address),
		Fee:         big.NewInt(int64(staticExtra.Fee)),
		TickSpacing: big.NewInt(int64(staticExtra.TickSpacing)),
		Hooks:       hook,
	}
}

func currency(isNative bool, token string) common.Address {
	if isNative {
		return common.Address{}
	}
	return common.HexToAddress(token)
}
