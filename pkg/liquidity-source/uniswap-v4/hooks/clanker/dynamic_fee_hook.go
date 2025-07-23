package clanker

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/uniswapv3"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type Hook struct {
	uniswapv4.Hook

	v3PoolSim        *uniswapv3.PoolSimulator
	token0, token1   string
	hook             string
	protocolFee      *big.Int
	clankerIsToken0  bool
	clankerTracked   bool
	clankerCaller    *ClankerCaller
	crankerCallerErr error
	rpcClient        *ethrpc.Client
}

type DynamicFeeExtra struct {
	ProtocolFee     *big.Int
	ClankerFee      *big.Int `json:"clankerFee,omitempty"`
	PairedFee       *big.Int `json:"pairedFee,omitempty"`
	ClankerIsToken0 bool
	ClankerTracked  bool
}

var _ = uniswapv4.RegisterHooksFactory(func(param *uniswapv4.HookParam) uniswapv4.Hook {
	var extra DynamicFeeExtra
	if param.HookExtra != "" {
		if err := json.Unmarshal([]byte(param.HookExtra), &extra); err != nil {
			return nil
		}
	}

	chainID := valueobject.ChainID(param.Cfg.ChainID)

	hook := &Hook{
		Hook:            &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4Clanker},
		token0:          param.Pool.Tokens[0].Address,
		token1:          param.Pool.Tokens[1].Address,
		hook:            param.HookAddress.Hex(),
		protocolFee:     extra.ProtocolFee,
		clankerIsToken0: extra.ClankerIsToken0,
		clankerTracked:  extra.ClankerTracked,
		rpcClient:       param.RpcClient,
	}

	param.Pool.SwapFee = 0
	v3PoolSimulator, err := uniswapv3.NewPoolSimulator(param.Pool, chainID)
	if err != nil {
		return nil
	}

	hook.v3PoolSim = v3PoolSimulator

	if param.RpcClient == nil {
		hook.crankerCallerErr = errors.New("rpc client is nil")
	} else {
		hook.clankerCaller, hook.crankerCallerErr = NewClankerCaller(ClankerAddressByChain[chainID],
			param.RpcClient.GetETHClient())
	}

	return hook
}, DynamicFeeHookAddresses...)

func (h *Hook) Track(ctx context.Context, _ *uniswapv4.HookParam) (string, error) {
	req := h.rpcClient.NewRequest().SetContext(ctx)
	req.AddCall(&ethrpc.Call{
		ABI:    dynamicFeeHookABI,
		Target: h.hook,
		Method: "protocolFee",
	}, []any{&h.protocolFee})

	if _, err := req.Aggregate(); err != nil {
		return "", err
	}

	extra := DynamicFeeExtra{
		ProtocolFee: h.protocolFee,
	}

	if !h.clankerTracked {
		if err := h.crankerCallerErr; err != nil {
			return "", err
		}
		token0 := common.HexToAddress(h.token0)
		info, err := h.clankerCaller.TokenDeploymentInfo(&bind.CallOpts{Context: ctx}, token0)
		if err != nil {
			return "", err
		}
		extra.ClankerTracked = true
		extra.ClankerIsToken0 = info.Token.Cmp(token0) == 0
	}

	extraBytes, err := json.Marshal(&extra)
	if err != nil {
		return "", err
	}

	return string(extraBytes), nil
}

func (h *Hook) BeforeSwap(params *uniswapv4.SwapParam) (hookFeeAmt *big.Int, swapFee uniswapv4.FeeAmount) {
	swappingForClanker := params.ZeroForOne != h.clankerIsToken0

	_, err := h.getTicks(params.AmountSpecified, params.ZeroForOne)
	if err != nil {
		return nil, 0
	}

	if !swappingForClanker {
		return big.NewInt(0), swapFee
	}

	var scaledProtocolFee, fee big.Int

	scaledProtocolFee.Mul(h.protocolFee, bignumber.BONE)
	fee.Add(MILLION, h.protocolFee)
	scaledProtocolFee.Div(&scaledProtocolFee, &fee)
	fee.Mul(params.AmountSpecified, &scaledProtocolFee)
	fee.Div(&fee, bignumber.BONE)

	return &fee, swapFee
}

func (h *Hook) AfterSwap(params *uniswapv4.SwapParam) (hookFeeAmt *big.Int) {
	swappingForClanker := params.ZeroForOne != h.clankerIsToken0

	if swappingForClanker {
		return big.NewInt(0)
	}

	var delta big.Int
	delta.Mul(params.AmountOut, h.protocolFee)
	delta.Div(&delta, FEE_DENOMINATOR)

	return &delta
}

func (h *Hook) GetReserves(ctx context.Context, param *uniswapv4.HookParam) (entity.PoolReserves, error) {
	return nil, nil
}

func (h *Hook) getTicks(amountIn *big.Int, zeroForOne bool) (int, error) {
	tokenIn, tokenOut := h.token0, h.token1
	if !zeroForOne {
		tokenIn, tokenOut = tokenOut, tokenIn
	}

	result, err := h.v3PoolSim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  tokenIn,
			Amount: amountIn,
		},
		TokenOut: tokenOut,
	})

	if err != nil {
		return 0, err
	}

	swapInfo := result.SwapInfo.(uniswapv3.SwapInfo)

	return swapInfo.NextStateTickCurrent, nil
}
