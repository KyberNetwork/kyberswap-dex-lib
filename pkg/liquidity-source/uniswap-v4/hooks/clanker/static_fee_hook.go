package clanker

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/eth"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type StaticFeeHook struct {
	uniswapv4.Hook

	hook            string
	protocolFee     *big.Int
	clankerFee      uniswapv4.FeeAmount
	pairedFee       uniswapv4.FeeAmount
	clankerIsToken0 bool
}

type StaticFeeExtra struct {
	ProtocolFee     *big.Int
	ClankerFee      *big.Int
	PairedFee       *big.Int
	ClankerIsToken0 bool
	ClankerTracked  bool
}

var _ = uniswapv4.RegisterHooksFactory(NewStaticFeeHook, StaticFeeHookAddresses...)

func NewStaticFeeHook(param *uniswapv4.HookParam) uniswapv4.Hook {
	hook := &StaticFeeHook{
		Hook: &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4Clanker},
		hook: param.HookAddress.Hex(),
	}

	if param.HookExtra != "" {
		var extra StaticFeeExtra
		if err := json.Unmarshal([]byte(param.HookExtra), &extra); err != nil {
			return nil
		}

		hook.clankerIsToken0 = extra.ClankerIsToken0
		hook.protocolFee = extra.ProtocolFee

		if extra.PairedFee != nil {
			hook.pairedFee = uniswapv4.FeeAmount(extra.PairedFee.Uint64())
		}
		if extra.ClankerFee != nil {
			hook.clankerFee = uniswapv4.FeeAmount(extra.ClankerFee.Uint64())
		}
	}

	return hook
}

func (h *StaticFeeHook) Track(ctx context.Context, param *uniswapv4.HookParam) (string, error) {
	var extra StaticFeeExtra
	if param.HookExtra != "" {
		if err := json.Unmarshal([]byte(param.HookExtra), &extra); err != nil {
			return "", err
		}
	}

	poolBytes := eth.StringToBytes32(param.Pool.Address)
	token0 := common.HexToAddress(param.Pool.Tokens[0].Address)
	var info ClankerDeploymentInfo

	req := param.RpcClient.NewRequest().SetContext(ctx)
	req.AddCall(&ethrpc.Call{
		ABI:    dynamicFeeHookABI,
		Target: h.hook,
		Method: "protocolFee",
	}, []any{&extra.ProtocolFee})

	req.AddCall(&ethrpc.Call{
		ABI:    staticFeeHookABI,
		Target: h.hook,
		Method: "clankerFee",
		Params: []any{poolBytes},
	}, []any{&extra.ClankerFee})
	req.AddCall(&ethrpc.Call{
		ABI:    staticFeeHookABI,
		Target: h.hook,
		Method: "pairedFee",
		Params: []any{poolBytes},
	}, []any{&extra.PairedFee})

	if !extra.ClankerTracked {
		req.AddCall(&ethrpc.Call{
			ABI:    clankerABI,
			Target: ClankerAddressByChain[valueobject.ChainID(param.Cfg.ChainID)],
			Method: "tokenDeploymentInfo",
			Params: []any{token0},
		}, []any{&info})

		extra.ClankerTracked = true
		extra.ClankerIsToken0 = info.Data.Token.Cmp(token0) == 0
	}

	if _, err := req.Aggregate(); err != nil {
		return "", err
	}

	if !extra.ClankerTracked {
		extra.ClankerTracked = true
		extra.ClankerIsToken0 = info.Data.Token.Cmp(token0) == 0
	}

	extraBytes, err := json.Marshal(&extra)
	if err != nil {
		return "", err
	}

	return string(extraBytes), nil
}

func (h *StaticFeeHook) BeforeSwap(params *uniswapv4.BeforeSwapParams) (*uniswapv4.BeforeSwapResult, error) {
	swappingForClanker := params.ZeroForOne != h.clankerIsToken0

	if params.ExactIn && !swappingForClanker || !params.ExactIn && swappingForClanker {
		return &uniswapv4.BeforeSwapResult{
			DeltaSpecific:   bignumber.ZeroBI,
			DeltaUnSpecific: bignumber.ZeroBI,
			SwapFee:         h.clankerFee,
		}, nil
	}

	var scaledProtocolFee, fee big.Int

	scaledProtocolFee.Mul(h.protocolFee, bignumber.BONE)
	if params.ExactIn && swappingForClanker {
		fee.Add(MILLION, h.protocolFee)
	} else { // !params.ExactIn && !swappingForClanker
		fee.Sub(MILLION, h.protocolFee)
	}
	scaledProtocolFee.Div(&scaledProtocolFee, &fee)
	fee.Mul(params.AmountSpecified, &scaledProtocolFee)
	fee.Div(&fee, bignumber.BONE)

	return &uniswapv4.BeforeSwapResult{
		DeltaSpecific:   &fee,
		DeltaUnSpecific: bignumber.ZeroBI,
		SwapFee:         h.pairedFee,
	}, nil
}

func (h *StaticFeeHook) AfterSwap(params *uniswapv4.AfterSwapParams) (*uniswapv4.AfterSwapResult, error) {
	swappingForClanker := params.ZeroForOne != h.clankerIsToken0

	if params.ExactIn && swappingForClanker || !params.ExactIn && !swappingForClanker {
		return &uniswapv4.AfterSwapResult{
			HookFee: bignumber.ZeroBI,
		}, nil
	}

	var delta big.Int
	if params.ExactIn && !swappingForClanker {
		delta.Mul(params.AmountOut, h.protocolFee)
	} else { // !params.ExactIn && swappingForClanker
		delta.Mul(params.AmountIn, h.protocolFee)
	}
	delta.Div(&delta, FEE_DENOMINATOR)

	return &uniswapv4.AfterSwapResult{
		HookFee: &delta,
		Gas:     0,
	}, nil
}
