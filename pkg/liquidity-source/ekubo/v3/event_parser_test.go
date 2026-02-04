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

	e := NewEventParser(MainnetConfig)

	tests := []struct {
		name        string
		txHash      string
		poolAddress []string
	}{
		{
			name:   "Swapped",
			txHash: "0xee56e1f3bad803bd857fb118e55d7eabb5368a94ae8f11e83724278f474294ca",
			poolAddress: []string{
				"0x21ae00a8bbb307ce790c612a71c5ce300918ddca939255bd5e26a8fdcf04b0de",
			},
		},
		{
			name:   "PositionUpdated",
			txHash: "0x2757427086944621c7fb8eca63a01809be4c76bb5b7b32596ced53d7fd17a691",
			poolAddress: []string{
				"0x21ae00a8bbb307ce790c612a71c5ce300918ddca939255bd5e26a8fdcf04b0de",
			},
		},
		{
			name:   "VirtualOrdersExecuted",
			txHash: "0xde6812e959a49e245f15714d1b50571f43ca7711c91d2df1087178a38bc554b7",
			poolAddress: []string{
				"0x8d04fa3b0df99076064daf0511006a8a06b0f988922db81c1e596ddfd1f3da12",
			},
		},
		{
			name:   "OrderUpdated",
			txHash: "0x67bb5ba44397d8b9d9ffe753e9c7f1b478eadfac22464a39521bdd3541f6a68f",
			poolAddress: []string{
				"0x8d04fa3b0df99076064daf0511006a8a06b0f988922db81c1e596ddfd1f3da12",
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
			assert.Equal(t, len(tt.poolAddress), len(logByAddress))
			for _, poolAddress := range tt.poolAddress {
				assert.Contains(t, logByAddress, poolAddress)
			}
		})
	}
}
