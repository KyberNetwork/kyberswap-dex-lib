package volatile

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/goccy/go-json"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type Extra struct {
	Tracked        bool                `json:"t,omitempty"`
	FeeZeroForOne  uniswapv4.FeeAmount `json:"f0,omitempty"`
	FeeOneForZero  uniswapv4.FeeAmount `json:"f1,omitempty"`
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
	if param.Cfg == nil || param.Cfg.ChainID != valueobject.ChainIDEthereum || param.HookAddress != HookAddress {
		return nil, ErrUnsupportedDeployment
	}
	if err := validateImplementation(ctx, param); err != nil {
		return nil, err
	}

	staticExtra := new(uniswapv4.StaticExtra)
	if err := json.Unmarshal([]byte(param.Pool.StaticExtra), staticExtra); err != nil {
		return nil, err
	}

	var feeZeroForOne, feeOneForZero uint32
	poolKey := uniswapv4.PoolKey{
		Currency0:   currency(staticExtra.IsNative[0], param.Pool.Tokens[0].Address),
		Currency1:   currency(staticExtra.IsNative[1], param.Pool.Tokens[1].Address),
		Fee:         big.NewInt(int64(staticExtra.Fee)),
		TickSpacing: big.NewInt(int64(staticExtra.TickSpacing)),
		Hooks:       param.HookAddress,
	}
	if _, err := param.RpcClient.NewRequest().SetContext(ctx).SetBlockNumber(param.BlockNumber).
		SetOverrides(param.Overrides).
		AddCall(&ethrpc.Call{
			ABI:    volatilePairHookABI,
			Target: hexutil.Encode(param.HookAddress[:]),
			Method: "getFee",
			Params: []any{poolKey},
		}, []any{&feeZeroForOne, &feeOneForZero}).Aggregate(); err != nil {
		return nil, err
	}

	if feeZeroForOne >= uint32(uniswapv4.FeeMax) || feeOneForZero >= uint32(uniswapv4.FeeMax) {
		return nil, ErrInvalidSwapFee
	}

	h.Extra = Extra{
		Tracked:       true,
		FeeZeroForOne: uniswapv4.FeeAmount(feeZeroForOne),
		FeeOneForZero: uniswapv4.FeeAmount(feeOneForZero),
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
	}, nil
}

func (h *Hook) CloneState() uniswapv4.Hook {
	cloned := *h
	return &cloned
}

func validateImplementation(ctx context.Context, param *uniswapv4.HookParam) error {
	account := param.Overrides[HookAddress]
	if account.Code != nil && crypto.Keccak256Hash(account.Code) != proxyCodeHash {
		return ErrUnsupportedDeployment
	}

	var slot []byte
	if account.State != nil {
		slot = account.State[implementationSlot].Bytes()
	} else if value, ok := account.StateDiff[implementationSlot]; ok {
		slot = value.Bytes()
	} else {
		var err error
		slot, err = param.RpcClient.GetETHClient().StorageAt(ctx, HookAddress, implementationSlot, param.BlockNumber)
		if err != nil {
			return err
		}
	}
	if common.BytesToAddress(slot) != ImplementationAddress {
		return ErrUnsupportedDeployment
	}
	return nil
}

func currency(isNative bool, token string) common.Address {
	if isNative {
		return common.Address{}
	}
	return common.HexToAddress(token)
}
