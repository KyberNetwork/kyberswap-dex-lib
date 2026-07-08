package clanker

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

type StaticFeeHook struct {
	uniswapv4.Hook  `json:"-"`
	*Fork           `json:"-"`
	ProtocolFee     *big.Int            `json:"p,omitempty"`
	ClankerFee      uniswapv4.FeeAmount `json:"c,omitempty"`
	PairedFee       uniswapv4.FeeAmount `json:"f,omitempty"`
	ClankerIsToken0 bool                `json:"0,omitempty"`
	ClankerTracked  bool                `json:"t,omitempty"`
}

var _ = uniswapv4.RegisterHooksFactory(NewStaticFeeHook(Clanker), StaticFeeHookAddresses...)
var _ = uniswapv4.RegisterHooksFactory(NewStaticFeeHook(Liquid), LiquidStaticFeeHookAddresses...)

func NewStaticFeeHook(fork *Fork) func(param *uniswapv4.HookParam) uniswapv4.Hook {
	return func(param *uniswapv4.HookParam) uniswapv4.Hook {
		hook := &StaticFeeHook{
			Hook: &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4Clanker},
			Fork: fork,
		}
		_ = param.HookExtra.Unmarshal(&hook)
		return hook
	}
}

func (h *StaticFeeHook) Track(ctx context.Context, param *uniswapv4.HookParam) (json.RawMessage, error) {
	hook := hexutil.Encode(param.HookAddress[:])
	poolBytes := common.HexToHash(param.Pool.Address)
	token0 := common.HexToAddress(param.Pool.Tokens[0].Address)

	var info TokenDeploymentInfo
	req := param.RpcClient.NewRequest().SetContext(ctx).SetBlockNumber(param.BlockNumber).AddCall(&ethrpc.Call{
		ABI:    dynamicFeeHookABI,
		Target: hook,
		Method: "protocolFee",
	}, []any{&h.ProtocolFee}).AddCall(&ethrpc.Call{
		ABI:    staticFeeHookABI,
		Target: hook,
		Method: h.Name + "Fee",
		Params: []any{poolBytes},
	}, []any{(*uint64)(&h.ClankerFee)}).AddCall(&ethrpc.Call{
		ABI:    staticFeeHookABI,
		Target: hook,
		Method: "pairedFee",
		Params: []any{poolBytes},
	}, []any{(*uint64)(&h.PairedFee)})
	if !h.ClankerTracked {
		req.AddCall(&ethrpc.Call{
			ABI:    clankerABI,
			Target: h.AddressByChain[param.Cfg.ChainID],
			Method: "tokenDeploymentInfo",
			Params: []any{token0},
		}, []any{&info})
	}

	if _, err := req.Aggregate(); err != nil {
		return nil, err
	}

	if !h.ClankerTracked {
		h.ClankerTracked = true
		h.ClankerIsToken0 = info.Data.Token == token0
	}

	return json.Marshal(h)
}

func (h *StaticFeeHook) BeforeSwap(params *uniswapv4.BeforeSwapParams) (*uniswapv4.BeforeSwapResult, error) {
	if h.ProtocolFee == nil {
		return nil, ErrPoolIsNotTracked
	}

	if params.ZeroForOne == h.ClankerIsToken0 {
		return &uniswapv4.BeforeSwapResult{
			DeltaSpecified:   bignumber.ZeroBI,
			DeltaUnspecified: bignumber.ZeroBI,
			SwapFee:          h.ClankerFee,
		}, nil
	}

	var scaledProtocolFee, fee big.Int
	scaledProtocolFee.Mul(h.ProtocolFee, bignumber.BONE)
	fee.Add(Million, h.ProtocolFee)
	scaledProtocolFee.Div(&scaledProtocolFee, &fee)
	fee.Mul(params.AmountSpecified, &scaledProtocolFee)
	fee.Div(&fee, bignumber.BONE)

	return &uniswapv4.BeforeSwapResult{
		DeltaSpecified:   &fee,
		DeltaUnspecified: bignumber.ZeroBI,
		SwapFee:          h.PairedFee,
	}, nil
}

func (h *StaticFeeHook) AfterSwap(params *uniswapv4.AfterSwapParams) (*uniswapv4.AfterSwapResult, error) {
	if params.ZeroForOne != h.ClankerIsToken0 {
		return &uniswapv4.AfterSwapResult{
			HookFee: bignumber.ZeroBI,
		}, nil
	}

	var delta big.Int
	delta.Mul(params.AmountOut, h.ProtocolFee)
	delta.Div(&delta, FeeDenominator)

	return &uniswapv4.AfterSwapResult{
		HookFee: &delta,
	}, nil
}
