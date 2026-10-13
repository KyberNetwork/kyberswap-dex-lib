package tickermemev4

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// Track checks Ticker's upgradeable proxy before using the shared Pons tracker.
func (h *Hook) Track(ctx context.Context, param *uniswapv4.HookParam) (json.RawMessage, error) {
	if param.Cfg == nil || param.Cfg.ChainID != valueobject.ChainIDMonad || param.HookAddress != HookAddress {
		return nil, ErrUnsupportedDeployment
	}
	var slot []byte
	account := param.Overrides[HookAddress]
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
	if err := h.Extra.Track(ctx, param); err != nil {
		return nil, err
	}
	return json.Marshal(h.Extra)
}
