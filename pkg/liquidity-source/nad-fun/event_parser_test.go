package nadfun

import (
	"context"
	"os"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
)

func TestEventParserDecode(t *testing.T) {
	t.Parallel()
	if os.Getenv("CI") != "" {
		t.Skip("Skipping testing in CI environment")
	}

	rpcClient := ethrpc.
		New("https://rpc-mainnet.monadinfra.com/rpc/ICLJSp4IKDWLSpZ4laJATUQfL0ucwxiK").
		SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))

	e := NewEventParser(&EventParserConfig{
		BondingCurve: "0xa7283d07812a02afb7c09b60f8896bcea3f90ace",
	})

	tests := []struct {
		name        string
		txHash      string
		poolAddress []string
	}{
		{
			name:   "Sell",
			txHash: "0x991f7f9d2298e06148cec1450ffe4caf808b671d072b303ee12b560ae77060e6",
			poolAddress: []string{
				"bc-0x47c6ac22d1fbb5747b711b7d6602090c8dd37777",
			},
		},
		{
			name:   "Buy",
			txHash: "0xf8334a96f9fa0ff38078058325120d1fe28898048906db732b97028923ad94a7",
			poolAddress: []string{
				"bc-0x47c6ac22d1fbb5747b711b7d6602090c8dd37777",
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

			logs := make([]types.Log, len(txReceipt.Logs))
			for _, log := range txReceipt.Logs {
				logs = append(logs, *log)
			}

			logByAddress, err := e.Decode(context.Background(), logs)

			assert.NoError(t, err)
			assert.Equal(t, len(tt.poolAddress), len(logByAddress))
			for _, poolAddress := range tt.poolAddress {
				assert.Contains(t, logByAddress, poolAddress)
			}
		})
	}
}
