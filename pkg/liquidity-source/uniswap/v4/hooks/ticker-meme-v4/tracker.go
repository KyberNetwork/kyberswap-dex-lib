package tickermemev4

import (
	"context"
	"sync"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/goccy/go-json"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// Track checks the immutable deployment before using the shared Pons tracker.
func (h *Hook) Track(ctx context.Context, param *uniswapv4.HookParam) (json.RawMessage, error) {
	if param == nil || param.Cfg == nil || param.Cfg.ChainID != valueobject.ChainIDMonad || param.HookAddress != HookAddress {
		return nil, ErrUnsupportedDeployment
	}
	if err := checkHookCode(ctx, param); err != nil {
		return nil, err
	}
	if err := h.Extra.Track(ctx, param); err != nil {
		return nil, err
	}
	return json.Marshal(h.Extra)
}

// Cache only verified chain code. Simulation replacement code must always be checked.
var verifiedCode sync.Map

func checkHookCode(ctx context.Context, param *uniswapv4.HookParam) error {
	code := param.Overrides[HookAddress].Code
	overridden := code != nil
	if !overridden {
		if _, ok := verifiedCode.Load(HookAddress); ok {
			return nil
		}
		var err error
		code, err = param.RpcClient.GetETHClient().CodeAt(ctx, HookAddress, param.BlockNumber)
		if err != nil {
			return err
		}
	}
	if crypto.Keccak256Hash(code) != hookCodeHash {
		return ErrUnsupportedDeployment
	}
	if !overridden {
		verifiedCode.Store(HookAddress, struct{}{})
	}
	return nil
}
