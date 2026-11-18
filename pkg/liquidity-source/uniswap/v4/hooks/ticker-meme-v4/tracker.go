package tickermemev4

import (
	"context"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/goccy/go-json"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// Track validates the selected deployment before using the shared Pons tracker.
// The legacy address remains a proxy; the new deployment executes its code directly.
func (h *Hook) Track(ctx context.Context, param *uniswapv4.HookParam) (json.RawMessage, error) {
	if param.Cfg == nil || param.Cfg.ChainID != valueobject.ChainIDMonad || (param.HookAddress != HookAddress && param.HookAddress != ImmutableHookAddress) {
		return nil, ErrUnsupportedDeployment
	}
	if param.HookAddress == ImmutableHookAddress {
		if err := checkImmutableCode(ctx, param); err != nil {
			return nil, err
		}
		if err := h.Extra.Track(ctx, param); err != nil {
			return nil, err
		}
		return json.Marshal(h.Extra)
	}
	var slot []byte
	account := param.Overrides[HookAddress]
	if account.Code != nil && crypto.Keccak256Hash(account.Code) != proxyCodeHash {
		return nil, ErrUnsupportedDeployment
	}
	if account.State != nil {
		slot = account.State[implementationSlot].Bytes()
	} else if value, ok := account.StateDiff[implementationSlot]; ok {
		slot = value.Bytes()
	} else {
		var err error
		slot, err = param.RpcClient.GetETHClient().StorageAt(ctx, HookAddress, implementationSlot, param.BlockNumber)
		if err != nil {
			return nil, err
		}
	}
	if common.BytesToAddress(slot) != ImplementationAddress {
		return nil, ErrUnsupportedDeployment
	}
	// Check supplied code without fetching bytecode from the chain.
	if code := param.Overrides[ImplementationAddress].Code; code != nil && crypto.Keccak256Hash(code) != ImplementationCodeHash {
		return nil, ErrUnsupportedDeployment
	}
	if err := h.Extra.Track(ctx, param); err != nil {
		return nil, err
	}
	return json.Marshal(h.Extra)
}

// Cache only verified chain code. Simulation replacement code must always be checked.
var immutableCodeVerified sync.Map

func checkImmutableCode(ctx context.Context, param *uniswapv4.HookParam) error {
	code := param.Overrides[ImmutableHookAddress].Code
	overridden := code != nil
	if !overridden {
		if _, ok := immutableCodeVerified.Load(ImmutableHookAddress); ok {
			return nil
		}
		var err error
		code, err = param.RpcClient.GetETHClient().CodeAt(ctx, ImmutableHookAddress, param.BlockNumber)
		if err != nil {
			return err
		}
	}
	if crypto.Keccak256Hash(code) != immutableHookCodeHash {
		return ErrUnsupportedDeployment
	}
	if !overridden {
		immutableCodeVerified.Store(ImmutableHookAddress, struct{}{})
	}
	return nil
}
