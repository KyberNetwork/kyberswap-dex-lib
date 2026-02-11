package test

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	cloberob "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/clober-ob"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/pooltypes"
)

func TestDecodePoolAddress(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		dexType  string
		events   []types.Log
		pools    []string
		decodeFn func(event types.Log) ([]string, error)
	}{
		{
			dexType: pooltypes.PoolTypes.CloberOB,
			events: []types.Log{
				{
					// Open
					Address: common.HexToAddress("0x6657d192273731c3cac646cc82d5f28d0cbe8ccc"),
					Topics: []common.Hash{
						common.HexToHash("0x803427d75ce3214f82dc7aa4910635170a6655e2c1663dc03429dd04100cba5a"),
						common.HexToHash("0x0000000000000000e4a3d4d6a29767bc7d085d3326b161a4d4aac7cb642fa150"),
						common.HexToHash("0xecac9c5f704e954931349da37f60e39f515c11c1"),
						common.HexToHash("0x754704bc059f8c67012fed69bc8a327a5aafb603"),
					},
					Data: common.FromHex("0x0000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000087a120000000000000000000000000000000000000000000000000000000000087a1840000000000000000000000000000000000000000000000000000000000000000"),
				},
				{
					// Make
					Address: common.HexToAddress("0x6657d192273731c3cac646cc82d5f28d0cbe8ccc"),
					Topics: []common.Hash{
						common.HexToHash("0x251db4df45fa692f68b4e3f072919384c5b71995c71bf22888385168930fd22a"),
						common.HexToHash("0x00000000000000001bbc7dcc9cf384619b36b8a9a8eddc9354def557a3ff0ac7"),
						common.HexToHash("0x000000000000000000000000b09684f5486d1af80699bbc27f14dd5a905da873"),
					},
					Data: common.FromHex("0xfffffffffffffffffffffffffffffffffffffffffffffffffffffffffffcfff8000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000010f902f0000000000000000000000000000000000000000000000000000000000000000"),
				},
				{
					// Take
					Address: common.HexToAddress("0x6657d192273731c3cac646cc82d5f28d0cbe8ccc"),
					Topics: []common.Hash{
						common.HexToHash("0xc4c20b9c4a5ada3b01b7a391a08dd81a1be01dd8ef63170dd9da44ecee3db11b"),
						common.HexToHash("0x0000000000000000f2dbe84fb6e603efc401eb30ab4a34fd881c4d3a14f024a2"),
						common.HexToHash("0x00000000000000000000000019b68a2b909d96c05b623050c276fbd457de8e83"),
					},
					Data: common.FromHex("0xfffffffffffffffffffffffffffffffffffffffffffffffffffffffffffb4f46000000000000000000000000000000000000000000000000000000000697b7a7"),
				},
				{
					// Claim
					Address: common.HexToAddress("0x6657d192273731c3cac646cc82d5f28d0cbe8ccc"),
					Topics: []common.Hash{
						common.HexToHash("0xfc7df80a30ee916cc040221cf6fcfb3c6dc994b3fa4c4ab23e8a0f134de5c0c0"),
						common.HexToHash("0xf2dbe84fb6e603efc401eb30ab4a34fd881c4d3a14f024a2fb4f460000000000"),
					},
					Data: common.FromHex("0x000000000000000000000000000000000000000000000000000000000697b7a7"),
				},
				{
					// Cancel
					Address: common.HexToAddress("0x6657d192273731c3cac646cc82d5f28d0cbe8ccc"),
					Topics: []common.Hash{
						common.HexToHash("0x0c6ba7ef5064094c17cce013aa4c617a23e2582f867774d07a5931de43b85d72"),
						common.HexToHash("0x9e107a8ef13ee448ec85ff7f89c6d797437205b08331906004b0ac0000000000"),
					},
					Data: common.FromHex("0x0000000000000000000000000000000000000000000000000000000251fe180a"),
				},
				{
					// Cancel
					Address: common.HexToAddress("0x6657d192273731c3cac646cc82d5f28d0cbe8ccc"),
					Topics: []common.Hash{
						common.HexToHash("0x0c6ba7ef5064094c17cce013aa4c617a23e2582f867774d07a5931de43b85d72"),
						common.HexToHash("0xf2dbe84fb6e603efc401eb30ab4a34fd881c4d3a14f024a2fb4b640000000000"),
					},
					Data: common.FromHex("0x00000000000000000000000000000000000000000000000000000000017b37b9"),
				},
			},
			pools: []string{
				"5606235663707778363715349868981881053004659724322268488016",
				"680091963353999958661303284433884846705699901928885914311",
				"5954885684956363054050231031211743946744177791604395877538",
				"5954885684956363054050231031211743946744177791604395877538",
				"3875727077379471850923186002296331935053867847116966170720",
				"5954885684956363054050231031211743946744177791604395877538",
			},
			decodeFn: cloberob.NewPoolFactory(nil).DecodePoolAddress,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.dexType, func(t *testing.T) {
			for i, event := range tc.events {
				pools, err := tc.decodeFn(event)
				require.NoError(t, err)
				require.Equal(t, tc.pools[i], pools[0])
			}
		})
	}
}
