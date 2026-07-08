package ekubov3

import (
	"context"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/test"
)

func TestEventParserDecode(t *testing.T) {
	t.Parallel()
	test.SkipCI(t)

	rpcClient := ethrpc.
		New("https://ethereum.drpc.org").
		SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))

	e := NewPoolFactory(MainnetConfig, rpcClient)

	tests := []struct {
		name            string
		txHash          string
		poolEventCounts map[string]int
	}{
		{
			name:   "Swapped",
			txHash: "0xee56e1f3bad803bd857fb118e55d7eabb5368a94ae8f11e83724278f474294ca",
			poolEventCounts: map[string]int{
				"0x21ae00a8bbb307ce790c612a71c5ce300918ddca939255bd5e26a8fdcf04b0de": 1,
			},
		},
		{
			name:   "PositionUpdated",
			txHash: "0x2757427086944621c7fb8eca63a01809be4c76bb5b7b32596ced53d7fd17a691",
			poolEventCounts: map[string]int{
				"0x21ae00a8bbb307ce790c612a71c5ce300918ddca939255bd5e26a8fdcf04b0de": 1,
			},
		},
		{
			name:   "VirtualOrdersExecutedAndOrderUpdatedV1",
			txHash: "0xde6812e959a49e245f15714d1b50571f43ca7711c91d2df1087178a38bc554b7",
			poolEventCounts: map[string]int{
				"0x8d04fa3b0df99076064daf0511006a8a06b0f988922db81c1e596ddfd1f3da12": 2,
			},
		},
		{
			name:   "VirtualOrdersExecutedAndOrderUpdatedV2",
			txHash: "0x32de015a5cd9a2a3f7fab3e19ad6bed01af3f91aeeb49d936831d97919504ed9",
			poolEventCounts: map[string]int{
				"0xedeaae143f233a3a5d4fabd3166afa0e2108fe7741489237274b939ca17fcff8": 3, // Third event is a TWAMM-initiated swap and comes from a virtual order execution
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			txReceipt, err := rpcClient.
				GetETHClient().
				TransactionReceipt(context.Background(), common.HexToHash(tt.txHash))
			if err != nil {
				t.Fatalf("failed to get tx receipt: %v", err)
			}

			logs := lo.Map(txReceipt.Logs, func(log *types.Log, _ int) types.Log {
				return *log
			})

			logByAddress, err := e.Decode(context.Background(), logs)

			assert.NoError(t, err)
			for expectedPool, expectedEventCnt := range tt.poolEventCounts {
				assert.Equal(t, expectedEventCnt, len(logByAddress[expectedPool]))
			}
		})
	}
}
