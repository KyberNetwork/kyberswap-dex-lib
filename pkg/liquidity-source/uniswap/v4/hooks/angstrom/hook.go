package angstrom

import (
	"context"
	"errors"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var ErrOutdatedAttestations = errors.New("outdated attestations")

var uOneE6 = uint256.NewInt(1_000_000)

type Hook struct {
	uniswapv4.Hook

	hook   common.Address
	asset0 common.Address
	asset1 common.Address

	extra HookExtra

	attestationController *AttestationController
	staleCheck            bool
}

var _ = uniswapv4.RegisterHooksFactory(NewHook, HookAddresses...)

func NewHook(param *uniswapv4.HookParam) uniswapv4.Hook {
	hook := &Hook{
		Hook: &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4Angstrom},
		hook: param.HookAddress, staleCheck: param.StaleCheck,
	}

	if param.Pool != nil {
		hook.asset0 = common.HexToAddress(param.Pool.Tokens[0].Address)
		hook.asset1 = common.HexToAddress(param.Pool.Tokens[1].Address)
	}

	_ = param.HookExtra.Unmarshal(&hook.extra)

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

func (h *Hook) Track(ctx context.Context, param *uniswapv4.HookParam) (json.RawMessage, error) {
	var extra HookExtra
	_ = param.HookExtra.Unmarshal(&extra)

	req := param.RpcClient.NewRequest().SetContext(ctx).SetOverrides(param.Overrides)
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
		return nil, err
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

	return json.Marshal(extra)
}

func (h *Hook) BeforeSwap(_ *uniswapv4.BeforeSwapParams) (*uniswapv4.BeforeSwapResult, error) {
	if h.staleCheck && time.Since(time.Unix(h.extra.LatestAttestationsTime, 0)) > time.Minute {
		return nil, ErrOutdatedAttestations
	}

	return &uniswapv4.BeforeSwapResult{
		DeltaSpecified:   new(uint256.Int),
		DeltaUnspecified: new(uint256.Int),
		SwapFee:          uniswapv4.FeeAmount(h.extra.UnlockedFee.Uint64()),

		SwapInfo: SwapInfo{
			Adapter:      Adapter,
			Attestations: h.extra.LatestAttestations,
		},
	}, nil
}

func (h *Hook) AfterSwap(params *uniswapv4.AfterSwapParams) (*uniswapv4.AfterSwapResult, error) {
	exactIn := params.CalcOut
	targetAmount := params.AmountOut
	protocolFee := u256.FromBig(h.extra.ProtocolUnlockedFee)

	var fee uint256.Int
	if exactIn {
		fee.Mul(targetAmount, protocolFee).Div(&fee, uOneE6)
	} else {
		// floor(t * 1e6 / (1e6 - p)) - t, two sequential floors.
		var denom uint256.Int
		denom.Sub(uOneE6, protocolFee)
		fee.Mul(targetAmount, uOneE6).Div(&fee, &denom).Sub(&fee, targetAmount)
	}

	return &uniswapv4.AfterSwapResult{
		HookFee: &fee,
	}, nil
}
