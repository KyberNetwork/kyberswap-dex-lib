package stonkbrokersfunv2

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	abiutil "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
)

// topic0 values pinned against the deployed StonkSafeLaunchpadV2 ABI. Hard-coded
// rather than derived so a silent ABI swap under the package fails here instead
// of quietly matching nothing on chain.
func TestPadEventIDs_MatchDeployedSignatures(t *testing.T) {
	t.Parallel()

	for name, want := range map[string]string{
		"SafeBuy":       "0xba22b06917da96d20a8f4f80d45cbdaaf3294856de78268558edcce22e4298df",
		"SafeSell":      "0x2de6d6d1573ee69658d3daae2e752379e6eb0676622a5ade2812088d7cb56581",
		"LaunchArmed":   "0x617dff9af409b81e92edd8d62fb192f25509657b96d203585e3e6b605d4dea26",
		"CurveClosed":   "0x35438c0a652764ec4e53a7c6ea24f69d38de411205cb46565a621d1ecec1b4f0",
		"LaunchBonded":  "0x93a0dc53b22eaff6d32706900ad4faf8945af529c8178e4556a9cba1b688cc87",
		"LaunchAborted": "0x580dfdb1b506406bf92732277b3b526bdf395c999ba181ddd0c576be4b95dc8d",
	} {
		id := abiutil.MustABIEvent(PadABI, name).ID
		require.Equal(t, want, id.Hex(), name)
		require.Contains(t, padEventIDs, id, name)
	}
}

func TestDecodePoolAddressesFromFactoryLog(t *testing.T) {
	t.Parallel()

	const padHex = "0xfcd61b25bbf3abd6cf0070d6328e351cc30eec9f"
	pad := common.HexToAddress(padHex)
	parser := NewEventParser(&EventParserConfig{DexID: string(DexType)})

	idTopic := func(n int64) common.Hash { return common.BigToHash(big.NewInt(n)) }
	padLog := func(event string, rest ...common.Hash) types.Log {
		return types.Log{
			Address: pad,
			Topics:  append([]common.Hash{abiutil.MustABIEvent(PadABI, event).ID}, rest...),
		}
	}

	tests := []struct {
		name string
		log  types.Log
		want []string
	}{
		{"buy", padLog("SafeBuy", idTopic(176), common.Hash{}), []string{padHex + "_176"}},
		{"sell", padLog("SafeSell", idTopic(176), common.Hash{}), []string{padHex + "_176"}},
		{"arm", padLog("LaunchArmed", idTopic(1)), []string{padHex + "_1"}},
		{"close", padLog("CurveClosed", idTopic(291)), []string{padHex + "_291"}},
		{"bond", padLog("LaunchBonded", idTopic(291)), []string{padHex + "_291"}},
		{"abort", padLog("LaunchAborted", idTopic(42)), []string{padHex + "_42"}},

		// Each log names exactly one launch, which is the whole point: routing a
		// pad's trades by dependency instead would wake every launch on the lane.
		{"a sibling's trade routes to the sibling only",
			padLog("SafeBuy", idTopic(177), common.Hash{}), []string{padHex + "_177"}},

		// OperatorSet indexes `account` first, so treating topics[1] as a launch
		// id would point at a wholly unrelated pool.
		{"operator grant", padLog("OperatorSet", idTopic(176), idTopic(176), common.Hash{}), nil},
		{"launch creation", padLog("LaunchCreated", idTopic(176), common.Hash{}, common.Hash{}), nil},
		{"unindexed pad event", padLog("LaunchFeeSet"), nil},
		{"log with no topics", types.Log{Address: pad}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parser.DecodePoolAddressesFromFactoryLog(context.Background(), tt.log)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

// The key has to match what discovery stored, byte for byte, or the decoded
// address resolves to nothing.
func TestDecodePoolAddressesFromFactoryLog_MatchesDiscoveryKey(t *testing.T) {
	t.Parallel()

	const padHex = "0xfcd61b25bbf3abd6cf0070d6328e351cc30eec9f"
	parser := NewEventParser(&EventParserConfig{})

	got, err := parser.DecodePoolAddressesFromFactoryLog(context.Background(), types.Log{
		// Checksummed on the wire; the key is lowercase.
		Address: common.HexToAddress("0xFCD61B25BBF3ABD6CF0070D6328E351CC30EEC9F"),
		Topics: []common.Hash{
			abiutil.MustABIEvent(PadABI, "SafeBuy").ID,
			common.BigToHash(big.NewInt(176)),
			{},
		},
	})
	require.NoError(t, err)
	require.Equal(t, []string{PoolAddress(padHex, 176)}, got)
}
