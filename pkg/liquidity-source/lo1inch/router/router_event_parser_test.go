package router

import (
	context "context"
	"encoding/json"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestDecode(t *testing.T) {
	t.Run("1. Decode pool address from logs", func(t *testing.T) {
		// eth_getLogs(0xd6edcfa282b86c5b194a569656de1f16d7cb38c49f47480df0a923544a5129f1)
		jsonStr := `[{"address": "0x111111125421ca6dc452d289314280a0f8842a65","topics": ["0xfec331350fce78ba658e082a71da20ac9f8d798a99b3c79681c8440cbfe77e07"],"data": "0x346fd1aa60dd2558f7f25f905b185da6e9f42e75841275d2dce4b11fd421a11b0000000000000000000000000000000000000000000004ae863226441d02352e","blockNumber": "0x153c0a1","transactionHash": "0xd6edcfa282b86c5b194a569656de1f16d7cb38c49f47480df0a923544a5129f1","transactionIndex": "0xeb","blockHash": "0xcf26c32f5fbe29875e74bb3035def726c8e099e38cdc905a0cd3ab691b5ef08a","logIndex": "0x205","removed": false}]`
		var logs []types.Log
		_ = json.Unmarshal([]byte(jsonStr), &logs)
		poolDecoder := NewEventParser(&Config{
			Router:     "0x111111125421ca6dc452d289314280a0f8842a65",
			Chain:      "1",
			HTTPClient: &HTTPClientConfig{},
		})

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockClient := NewMockI1inchClient(ctrl)
		mockClient.EXPECT().GetOrder(gomock.Any(), gomock.Any()).Return(&OrderResp{
			makerAsset: "0x6b175474e89094c44da98b954eedeac495271d0f",
			takerAsset: "0xf939e0a03fb07f59a73314e73794be0e57ac1b4e",
		}, nil)
		poolDecoder.SetClient(mockClient)
		got, err := poolDecoder.Decode(context.Background(), logs)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(got))
		assert.Equal(t, 1, len(got["lo1inch_0x6b175474e89094c44da98b954eedeac495271d0f_0xf939e0a03fb07f59a73314e73794be0e57ac1b4e"]))
		assert.Equal(t, 517, int(got["lo1inch_0x6b175474e89094c44da98b954eedeac495271d0f_0xf939e0a03fb07f59a73314e73794be0e57ac1b4e"][0].Index))
	})
}

// func getLogs() {
// 	client, err := ethclient.Dial("https://ethereum.kyberengineering.io")
// 	if err != nil {
// 		log.Fatalf("Failed to connect to the Ethereum client: %v", err)
// 	}

// 	// Transaction hash from the input
// 	txHash := common.HexToHash("0xd6edcfa282b86c5b194a569656de1f16d7cb38c49f47480df0a923544a5129f1")

// 	// Get transaction receipt which contains logs
// 	receipt, err := client.TransactionReceipt(context.Background(), txHash)
// 	if err != nil {
// 		log.Fatalf("Failed to get transaction receipt: %v", err)
// 	}

// 	// receipt.Logs is already of type []*types.Log, so no unmarshaling needed
// 	var logs []types.Log
// 	for _, log := range receipt.Logs {
// 		logs = append(logs, *log) // Convert from []*types.Log to []types.Log if needed
// 	}

// 	// Pretty print the logs for debugging
// 	logsJSON, err := json.MarshalIndent(logs, "", "  ")
// 	if err != nil {
// 		log.Fatalf("Failed to marshal logs to JSON: %v", err)
// 	}

// 	fmt.Printf("Transaction logs: %s\n", logsJSON)
// }
