package angstrom

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/samber/lo"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var ErrOutdatedAttestations = errors.New("outdated attestations")

type Hook struct {
	uniswapv4.Hook

	hook   common.Address
	asset0 common.Address
	asset1 common.Address

	extra HookExtra

	attestationController *AttestationController
}

var _ = uniswapv4.RegisterHooksFactory(NewHook, HookAddresses...)

func NewHook(param *uniswapv4.HookParam) uniswapv4.Hook {
	hook := &Hook{
		Hook: &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4Angstrom},
		hook: param.HookAddress,
	}

	if param.Pool != nil {
		hook.asset0 = common.HexToAddress(param.Pool.Tokens[0].Address)
		hook.asset1 = common.HexToAddress(param.Pool.Tokens[1].Address)
	}

	if param.HookExtra != "" {
		var extra HookExtra
		if err := json.Unmarshal([]byte(param.HookExtra), &extra); err != nil {
			return nil
		}

		hook.extra = extra
	}

	hookCfgProperties, exist := param.Cfg.HookConfigs[param.HookAddress]
	if exist {
		var hookCfg HookConfig
		data, err := json.Marshal(hookCfgProperties)
		if err != nil {
			return hook
		}

		if err := json.Unmarshal(data, &hookCfg); err != nil {
			return hook
		}

		hook.attestationController = GetAttestationController(hookCfg)
	}

	return hook
}

func (h *Hook) Track(ctx context.Context, param *uniswapv4.HookParam) (string, error) {
	var extra HookExtra
	if param.HookExtra != "" {
		if err := json.Unmarshal([]byte(param.HookExtra), &extra); err != nil {
			return "", err
		}
	}

	req := param.RpcClient.NewRequest().SetContext(ctx)
	if param.BlockNumber != nil {
		req.SetBlockNumber(param.BlockNumber)
	}

	key := keyFromAssetsUnchecked(h.asset0, h.asset1)
	slot := calculateUnlockedFeeSlot(key, StorageSlotUnlockedFeesVariable)

	var extsloadRes *big.Int

	req.AddCall(&ethrpc.Call{
		ABI:    hookABI,
		Target: h.hook.Hex(),
		Method: "extsload",
		Params: []any{slot.Big()},
	}, []any{&extsloadRes})

	if _, err := req.Aggregate(); err != nil {
		return "", err
	}

	unlockedFee, protocolUnlockedFee := extractUnlockedFee(extsloadRes)
	extra.UnlockedFee = unlockedFee
	extra.ProtocolUnlockedFee = protocolUnlockedFee

	if h.attestationController != nil {
		attestations, attestationTime := h.attestationController.GetLatestAttestations()
		if len(attestations) > 0 {
			extra.LatestAttestations = attestations
			extra.LatestAttestationsTime = attestationTime.Unix()
		}
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return "", err
	}

	return string(extraBytes), nil
}

func (h *Hook) BeforeSwap(swapHookParams *uniswapv4.BeforeSwapParams) (*uniswapv4.BeforeSwapResult, error) {
	if time.Since(time.Unix(h.extra.LatestAttestationsTime, 0)) > time.Minute {
		return nil, ErrOutdatedAttestations
	}

	return &uniswapv4.BeforeSwapResult{
		DeltaSpecified:   bignumber.ZeroBI,
		DeltaUnspecified: bignumber.ZeroBI,
		SwapFee:          uniswapv4.FeeAmount(h.extra.UnlockedFee.Uint64()),

		SwapInfo: SwapInfo{
			Adapter:      Adapter,
			Attestations: h.extra.LatestAttestations,
		},
	}, nil
}

func (h *Hook) AfterSwap(swapHookParams *uniswapv4.AfterSwapParams) (*uniswapv4.AfterSwapResult, error) {
	exactIn := swapHookParams.ExactIn
	targetAmount := swapHookParams.AmountOut

	var tmp big.Int

	fee := lo.Ternary(
		exactIn,

		new(big.Int).Div(
			tmp.Mul(targetAmount, h.extra.ProtocolUnlockedFee),
			ONE_E6,
		),

		new(big.Int).Sub(
			tmp.Div(
				tmp.Mul(targetAmount, ONE_E6),
				tmp.Sub(ONE_E6, h.extra.ProtocolUnlockedFee),
			),
			targetAmount,
		),
	)

	return &uniswapv4.AfterSwapResult{
		HookFee: fee,
	}, nil
}
