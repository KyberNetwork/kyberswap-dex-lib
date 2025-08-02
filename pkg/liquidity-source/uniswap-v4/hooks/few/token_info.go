package few

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"

var fewTokens = []TokenInfo{
	{
		FewTokenAddress:    "0xa250cc729bb3323e7933022a67b52200fe354767",
		UnwrapTokenAddress: "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
		HookAddress:        "0x12b504160222d66c38d916d9fba11b613c51e888",
		PoolAddress:        "0xd233d590a34569a9256d167d3990c1164e357ad6cb76eef4e043358f4f6bf343",
		TickSpacing:        60,
		Fee:                0,
		ChainID:            valueobject.ChainIDEthereum,
		IsNative:           true,
	},
	{
		FewTokenAddress:    "0xe8e1f50392bd61d0f8f48e8e7af51d3b8a52090a",
		UnwrapTokenAddress: "0x1f9840a85d5af5bf1d1762f925bdaddc4201f984",
		HookAddress:        "0xcf1e7189264a84d7454077dc713c3d11400de888",
		PoolAddress:        "0x7a81621f11b9023e67f5cfedb7feaa8946a225cba91cc53929ea84edf2cd194a",
		TickSpacing:        60,
		Fee:                0,
		ChainID:            valueobject.ChainIDEthereum,
	},
	{
		FewTokenAddress:    "0x2078f336fdd260f708bec4a20c82b063274e1b23",
		UnwrapTokenAddress: "0x2260fac5e5542a773aa44fbcfedf7c193bc2c599",
		HookAddress:        "0x948922b055187c7366e71b876ab1242ebbaea888",
		PoolAddress:        "0x884c00abc9b0fa843ea2dfdd025e1df5611db552f396e2f17c88fb2ceef199e1",
		TickSpacing:        60,
		Fee:                0,
		ChainID:            valueobject.ChainIDEthereum,
	},
}

type TokenInfo struct {
	FewTokenAddress    string
	UnwrapTokenAddress string
	HookAddress        string
	PoolAddress        string
	TickSpacing        int32
	Fee                uint32
	ChainID            valueobject.ChainID
	IsNative           bool
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
