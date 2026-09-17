package few

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"

// TokenInfo describes a Ring Protocol FewToken wrapper pool (an underlying
// asset paired 1:1 against its FewToken representation via a dedicated hook).
// Shared by every generation package (few/v1, few/v2, ...) so their token
// lists carry identical shape and wrapping semantics.
type TokenInfo struct {
	ChainID            valueobject.ChainID
	PoolAddress        string
	HookAddress        string
	IsNative           bool
	UnwrapTokenAddress string
	FewTokenAddress    string
	Fee                uint32
	TickSpacing        int32
}

func (t TokenInfo) GetWrapToken() string {
	return t.FewTokenAddress
}

func (t TokenInfo) GetUnwrapToken() string {
	return t.UnwrapTokenAddress
}

func (t TokenInfo) GetHook() string {
	return t.HookAddress
}

func (t TokenInfo) GetPool() string {
	return t.PoolAddress
}

func (t TokenInfo) GetTickSpacing() int32 {
	return t.TickSpacing
}

func (t TokenInfo) GetFee() uint32 {
	return t.Fee
}

func (t TokenInfo) GetHookData() []byte {
	return nil
}

func (t TokenInfo) IsUnwrapNative() bool {
	return t.IsNative
}
