package ekubo

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
	if os.Getenv("CI") != "" {
		t.Skip("Skipping testing in CI environment")
	}

	rpcClient := ethrpc.
		New("https://eth.llamarpc.com").
		SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))

	e := NewEventParser(&Config{
		Core:  common.HexToAddress("0xe0e0e08A6A4b9Dc7bD67BCB7aadE5cF48157d444"),
		Twamm: common.HexToAddress("0xd4279c050da1f5c5b2830558c7a08e57e12b54ec"),
	})

	t.Parallel()

	tests := []struct {
		name        string
		txHash      string
		poolAddress []string
	}{
		{
			name:   "Position updated",
			txHash: "0x10d8e276994d28e82bd67a915fea15a0d7e5da43333f00fbc1b1d09cf8bd6322",
			poolAddress: []string{
				"0xdbcf2dab4ba020756b3f44836b5dfc85d95b67d0d35849d3e8d0f00e93d4c763",
			},
		},
		{
			name:   "Anonymous event",
			txHash: "0x26d7555c237c64968f06f662c91f111c9824efc54decdb68e5ad1c35b384dc17",
			poolAddress: []string{
				"0x0e647f6d174aa84c22fddeef0af92262b878ba6f86094e54dbec558c0a53ab79",
				"0x7d7ee01726b349da3cf2c5af88a965579f3f241693e4c63b19dcbb02ed3c6ff3",
			},
		},
		{
			name:   "Anonymous event",
			txHash: "0xe13b37e5729409f2df6309ff041e203a0a2df538482c01363d61ddcc4a7d9ff2",
			poolAddress: []string{
				"0x91ed49e8b9bf72bda26928351a3bbf93b7bb964ee2b22ca35dce6460ce33e9ee",
			},
		},
		{
			name:   "Order updated event",
			txHash: "0xbd9e24145c6e3c936c7617d2a7756a0a7d1b3cf491e145d21f201a06899b1f01",
			poolAddress: []string{
				"0x91ed49e8b9bf72bda26928351a3bbf93b7bb964ee2b22ca35dce6460ce33e9ee",
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
