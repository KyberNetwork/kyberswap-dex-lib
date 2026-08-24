package hooks

import (
	"context"
	"encoding/json"
	"math"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

const (
	// FeeUseStaticFee mirrors EulerSwap's QuoteLib.getFee/getFeeReadOnly: when
	// the swap hook answers getFee with type(uint64).max, the pool falls back
	// to its static fee0/fee1.
	FeeUseStaticFee uint64 = math.MaxUint64
)

// WhitelistHookAddresses lists the deployed access-control EulerSwap swap
// hooks. These contracts register pools by address, gate every swap behind an
// owner-managed swapper whitelist inside beforeSwap (reverting with
// Unauthorized() otherwise), verify output amounts against EVault cash, and
// always answer getFee with type(uint64).max so pools keep their static fees.
var WhitelistHookAddresses = []common.Address{
	common.HexToAddress("0x4B18F757c90856718DE70Be35aF6683b44bf72CA"),
}

// WhitelistExtra is the tracked hook state persisted into Extra.HookExtra.
type WhitelistExtra struct {
	Enabled bool `json:"e"` // pool registered and not paused/disabled on the hook
}

type WhitelistHook struct {
	Extra WhitelistExtra
}

var _ Hook = (*WhitelistHook)(nil)

func init() {
	RegisterHooksFactory(NewWhitelistHook, WhitelistHookAddresses...)
}

func NewWhitelistHook(param *HookParam) Hook {
	hook := &WhitelistHook{Extra: WhitelistExtra{Enabled: true}}
	if param != nil && len(param.HookExtra) > 0 {
		if err := json.Unmarshal([]byte(param.HookExtra), &hook.Extra); err != nil {
			hook.Extra = WhitelistExtra{Enabled: true}
		}
	}
	return hook
}

func (h *WhitelistHook) GetFee(_ *GetFeeParams) (uint64, error) {
	return FeeUseStaticFee, nil
}

func (h *WhitelistHook) BeforeSwap(_ *BeforeSwapParams) error {
	if !h.Extra.Enabled {
		return ErrPoolDisabled
	}

	// The hook only allows whitelisted swappers past beforeSwap. dex-lib has no
	// notion of the taker address and public routers are not whitelisted, so
	// any swap we could simulate would revert with Unauthorized() on-chain.
	return ErrSwapperNotAuthorized
}

func (h *WhitelistHook) AfterSwap(_ *AfterSwapParams) error {
	return nil
}

func (h *WhitelistHook) Track(ctx context.Context, param *HookParam) (string, error) {
	if param == nil || param.RpcClient == nil || param.Pool == nil {
		return "", ErrHookCallFailed
	}

	var enabled bool
	req := param.RpcClient.NewRequest().SetContext(ctx).SetBlockNumber(param.BlockNumber)
	req.AddCall(&ethrpc.Call{
		ABI:    HookABI,
		Target: hexutil.Encode(param.HookAddress[:]),
		Method: "poolEnabled",
		Params: []any{common.HexToAddress(param.Pool.Address)},
	}, []any{&enabled})

	if _, err := req.TryAggregate(); err != nil {
		return "", err
	}

	bz, err := json.Marshal(WhitelistExtra{Enabled: enabled})
	if err != nil {
		return "", err
	}
	return string(bz), nil
}

func (h *WhitelistHook) CloneState() Hook {
	cloned := *h
	return &cloned
}
