package clanker

import (
	"context"
	"encoding/json"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/eth"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type StaticFeeHook struct {
	uniswapv4.Hook

	pool             string
	token0           common.Address
	hook             string
	protocolFee      *big.Int
	clankerFee       uniswapv4.FeeAmount
	pairedFee        uniswapv4.FeeAmount
	clankerIsToken0  bool
	clankerTracked   bool
	clankerCaller    *ClankerCaller
	crankerCallerErr error
	rpcClient        *ethrpc.Client
}

type StaticFeeExtra struct {
	ProtocolFee     *big.Int
	ClankerFee      *big.Int
	PairedFee       *big.Int
	ClankerIsToken0 bool
	ClankerTracked  bool
}

var _ = uniswapv4.RegisterHooksFactory(func(param *uniswapv4.HookParam) uniswapv4.Hook {
	var extra StaticFeeExtra
	if param.HookExtra != "" {
		if err := json.Unmarshal([]byte(param.HookExtra), &extra); err != nil {
			return nil
		}
	}

	hook := &StaticFeeHook{
		Hook:            &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4Clanker},
		pool:            param.Pool.Address,
		token0:          common.HexToAddress(param.Pool.Tokens[0].Address),
		hook:            param.HookAddress.Hex(),
		protocolFee:     extra.ProtocolFee,
		pairedFee:       uniswapv4.FeeAmount(extra.PairedFee.Uint64()),
		clankerFee:      uniswapv4.FeeAmount(extra.ClankerFee.Uint64()),
		clankerIsToken0: extra.ClankerIsToken0,
		clankerTracked:  extra.ClankerTracked,
		rpcClient:       param.RpcClient,
	}

	return hook
}, StaticFeeHookAddresses...)

func (h *StaticFeeHook) Track(ctx context.Context, _ *uniswapv4.HookParam) (string, error) {
	var pairedFee, clankerFee *big.Int
	poolBytes := eth.StringToBytes32(h.pool)

	req := h.rpcClient.NewRequest().SetContext(ctx)
	req.AddCall(&ethrpc.Call{
		ABI:    dynamicFeeHookABI,
		Target: h.hook,
		Method: "protocolFee",
	}, []any{&h.protocolFee})

	req.AddCall(&ethrpc.Call{
		ABI:    staticFeeHookABI,
		Target: h.hook,
		Method: "clankerFee",
		Params: []any{poolBytes},
	}, []any{&clankerFee})
	req.AddCall(&ethrpc.Call{
		ABI:    staticFeeHookABI,
		Target: h.hook,
		Method: "pairedFee",
		Params: []any{poolBytes},
	}, []any{&pairedFee})

	if _, err := req.Aggregate(); err != nil {
		return "", err
	}

	extra := StaticFeeExtra{
		ProtocolFee: h.protocolFee,
		ClankerFee:  clankerFee,
		PairedFee:   pairedFee,
	}

	if !h.clankerTracked {
		if err := h.crankerCallerErr; err != nil {
			return "", err
		}
		info, err := h.clankerCaller.TokenDeploymentInfo(&bind.CallOpts{Context: ctx}, h.token0)
		if err != nil {
			return "", err
		}
		extra.ClankerTracked = true
		extra.ClankerIsToken0 = info.Token.Cmp(h.token0) == 0
	}

	extraBytes, err := json.Marshal(&extra)
	if err != nil {
		return "", err
	}

	return string(extraBytes), nil
}

func (h *StaticFeeHook) BeforeSwap(params *uniswapv4.SwapParam) (hookFeeAmt *big.Int, swapFee uniswapv4.FeeAmount) {
	swappingForClanker := params.ZeroForOne != h.clankerIsToken0

	if !swappingForClanker {
		return big.NewInt(0), h.clankerFee
	}

	var scaledProtocolFee, fee big.Int

	scaledProtocolFee.Mul(h.protocolFee, bignumber.BONE)
	fee.Add(MILLION, h.protocolFee)
	scaledProtocolFee.Div(&scaledProtocolFee, &fee)
	fee.Mul(params.AmountSpecified, &scaledProtocolFee)
	fee.Div(&fee, bignumber.BONE)

	return &fee, h.pairedFee
}

func (h *StaticFeeHook) AfterSwap(params *uniswapv4.SwapParam) (hookFeeAmt *big.Int) {
	swappingForClanker := params.ZeroForOne != h.clankerIsToken0

	if swappingForClanker {
		return big.NewInt(0)
	}

	var delta big.Int
	delta.Mul(params.AmountOut, h.protocolFee)
	delta.Div(&delta, FEE_DENOMINATOR)

	return &delta
}

func (h *StaticFeeHook) GetReserves(ctx context.Context, param *uniswapv4.HookParam) (entity.PoolReserves, error) {
	return nil, nil
}
