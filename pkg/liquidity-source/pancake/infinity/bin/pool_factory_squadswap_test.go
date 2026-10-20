package bin

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake/infinity/shared"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// Real SquadSwap Bin Initialize logs (BSC manager 0x7f8e5a02): a Thanos-family hook pool (bitmap 0x442)
// and a bitmap-2 (afterInitialize only) hook pool.
func squadswapInitLog(t *testing.T, id, currency0, hooks, params, activeID string) ethtypes.Log {
	t.Helper()
	topic := func(h string) common.Hash { return common.HexToHash(h) }
	return ethtypes.Log{
		Address: common.HexToAddress("0x7f8e5a02f9007b19408ef731982adbba7ae08eb1"),
		Topics: []common.Hash{
			shared.BinPoolManagerABI.Events["Initialize"].ID, topic(id), topic(currency0),
			topic("0xbb4cdb9cbd36b01bd1cbaebf2de08d9173bc095c"),
		},
		Data: common.FromHex("0x" + common.Bytes2Hex(common.LeftPadBytes(common.FromHex(hooks), 32)) +
			common.Bytes2Hex(common.LeftPadBytes([]byte{0x80, 0, 0}, 32)) +
			common.Bytes2Hex(common.LeftPadBytes(common.FromHex(params), 32)) +
			common.Bytes2Hex(common.LeftPadBytes(common.FromHex(activeID), 32))),
		BlockNumber: 126780625,
	}
}

func TestSquadSwapExchangeAndHookPermissions(t *testing.T) {
	thanosLog := squadswapInitLog(t, "0x999e8f48cbf0c7a6a20cf54567acd1cd2aea4c6ae6c09e31b2c35d9b1dcc0716",
		"0x1d2f0da169ceb9fc7b3144628db156f3f6c60dbe", "0xe7c583425bb682ac321701aa954e99a352c5aa53",
		"0x001e0442", "0x7ff7ca")
	feeMgrLog := squadswapInitLog(t, "0x59b91e2a3f1d6515ac187bd821821a1c37a9a827c6ed31b47539da398d250396",
		"0x570a5d26f7765ecb712c0924e4de545b89fd43df", "0xe1a70d8873c0f8480681ca758b98113d7644bfe0",
		"0x000a0002", "0x7ff8ba")

	for _, tc := range []struct {
		name, dexID, exchange string
		log                   ethtypes.Log
		hasSwapPerms          bool
	}{
		{"thanos hook", "squadswap-bin", valueobject.ExchangeSquadSwapBin, thanosLog, true},
		{"afterInitialize-only hook", "squadswap-bin", valueobject.ExchangeSquadSwapBin, feeMgrLog, false},
		{"pancake unchanged", "pancake-infinity-bin", valueobject.ExchangePancakeInfinityBin, thanosLog, true},
		{"empty dex id", "", valueobject.ExchangePancakeInfinityBin, thanosLog, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p, err := NewPoolFactory(&Config{DexID: tc.dexID, ChainID: 56}).DecodePoolCreated(tc.log)
			require.NoError(t, err)
			require.Equal(t, tc.exchange, p.Exchange)
			var se StaticExtra
			require.NoError(t, json.Unmarshal([]byte(p.StaticExtra), &se))
			require.Equal(t, tc.hasSwapPerms, se.HasSwapPermissions)
			require.Equal(t, hexutil.Encode(tc.log.Topics[1][:]), p.Address)
			// the hook is modelled iff it has swap callbacks; the simulator skips any other hooked pool
			_, modelled := GetHook(se.HooksAddress)
			require.Equal(t, tc.hasSwapPerms, modelled)
		})
	}
}
