package tickermemev4

import (
	"context"
	"fmt"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/goccy/go-json"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	ponsv2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/pons-v2"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// Track checks the proxy's implementation address and reads fees at the pinned block.
func (h *Hook) Track(ctx context.Context, param *uniswapv4.HookParam) (json.RawMessage, error) {
	if param.Cfg == nil || param.Cfg.ChainID != valueobject.ChainIDMonad || param.HookAddress != HookAddress {
		return nil, ErrUnsupportedDeployment
	}
	if param.BlockNumber == nil || param.BlockNumber.Sign() <= 0 || param.RpcClient == nil || param.Pool == nil {
		return nil, fmt.Errorf("%s: missing pinned tracking context", DexType)
	}
	poolID, decodeErr := hexutil.Decode(param.Pool.Address)
	if decodeErr != nil || len(poolID) != common.HashLength {
		return nil, fmt.Errorf("%s: invalid pool id", DexType)
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
	implementation := common.BytesToAddress(slot)
	if implementation != ImplementationAddress {
		return nil, ErrUnsupportedDeployment
	}
	// Check supplied code overrides without downloading on-chain bytecode.
	if code := param.Overrides[implementation].Code; code != nil && crypto.Keccak256Hash(code) != ImplementationCodeHash {
		return nil, ErrUnsupportedDeployment
	}
	next := *h
	next.decodeErr = nil
	var fees launchInfo
	_, err := param.RpcClient.NewRequest().SetContext(ctx).SetBlockNumber(param.BlockNumber).
		SetOverrides(param.Overrides).AddCall(&ethrpc.Call{
		ABI: ponsv2.HookABI(), Target: hexutil.Encode(HookAddress[:]), Method: "launches",
		Params: []any{common.BytesToHash(poolID)},
	}, []any{&fees}).Call()
	if err != nil {
		return nil, err
	}
	next.Extra = Extra{
		// Tag the supported implementation; its address was checked above.
		ImplementationHash: ImplementationCodeHash,
		Registered:         fees.Registered,
		HookFeeBps:         fees.HookFeeBps,
		CreatorTaxBps:      fees.CreatorTaxBps,
	}
	if err := next.validate(); err != nil {
		return nil, err
	}
	result, err := json.Marshal(next.Extra)
	if err != nil {
		return nil, err
	}
	*h = next
	param.Pool.BlockNumber = param.BlockNumber.Uint64()
	return result, nil
}
