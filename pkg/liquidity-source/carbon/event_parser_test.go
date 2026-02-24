package carbon

import (
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/test"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

func TestEventParserDecode(t *testing.T) {
	t.Parallel()
	test.SkipCI(t)

	controller := common.HexToAddress("0xc537e898cd774e2dcba3b14ea6f34c93d5ea45e1")

	rpcClient := ethrpc.New("https://eth.drpc.org").
		SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))

	e := NewEventParser(&Config{Controller: controller}, rpcClient)

	tests := []struct {
		name        string
		txHash      string
		poolAddress []string
	}{
		{
			name:   "TokensTraded",
			txHash: "0xc53b428ba28fc133124601b0d79356afe383d6c9f519ebfd25fb216208e1409d",
			poolAddress: []string{
				generatePoolAddress(
					controller,
					valueobject.AddrNative,
					common.HexToAddress("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"),
				),
			},
		},
		{
			name:   "StrategyCreated",
			txHash: "0xf74aa8c3e5dd23f23a32c840ed8b1aadb6fc2ca44f2f8c64b933b1702e2d7f86",
			poolAddress: []string{
				generatePoolAddress(
					controller,
					common.HexToAddress("0xfc60fc0145D7330e5abcFc52AF7B043a1cE18e7d"),
					common.HexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7"),
				),
			},
		},
		{
			name:   "StrategyUpdated",
			txHash: "0x47807947929a7baf9cd4ae291c368e3ef0a90dc12b2329d728a6094b4a993c96",
			poolAddress: []string{
				generatePoolAddress(
					controller,
					valueobject.AddrNative,
					common.HexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7"),
				),
			},
		},
		{
			name:   "StrategyDeleted",
			txHash: "0x3629e473d9edce6d9f3b8e3b2eca9db60832f71ff59a3cb7d67b5839abaefc94",
			poolAddress: []string{
				generatePoolAddress(
					controller,
					common.HexToAddress("0xfc60fc0145D7330e5abcFc52AF7B043a1cE18e7d"),
					common.HexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7"),
				),
			},
		},
		{
			name:   "PairTradingFeePPMUpdated",
			txHash: "0x59cd4e996649cbef1b39e6c06f440e93340e1ed83303c55380526a60edb0dc8a",
			poolAddress: []string{
				generatePoolAddress(
					controller,
					common.HexToAddress("0x5f98805A4E8be255a32880FDeC7F6728C6568bA0"),
					common.HexToAddress("0x6B175474E89094C44Da98b954EedeAC495271d0F"),
				),
			},
		},
		{
			name:   "PairTradingFeePPMUpdated",
			txHash: "0x59cd4e996649cbef1b39e6c06f440e93340e1ed83303c55380526a60edb0dc8a",
			poolAddress: []string{
				generatePoolAddress(
					controller,
					common.HexToAddress("0x5f98805A4E8be255a32880FDeC7F6728C6568bA0"),
					common.HexToAddress("0x6B175474E89094C44Da98b954EedeAC495271d0F"),
				),
			},
		},
		{
			name:        "TradingFeePPMUpdated",
			txHash:      "0xa8c33dd6939277c3dae2307d5ccf98fa35d2f727cb4f0d95111089615775b77e",
			poolAddress: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			txReceipt, err := rpcClient.GetETHClient().TransactionReceipt(t.Context(), common.HexToHash(tt.txHash))
			if err != nil {
				t.Fatalf("failed to get tx receipt: %v", err)
			}

			logs := lo.Map(txReceipt.Logs, func(log *types.Log, _ int) types.Log {
				return *log
			})

			logByAddress, err := e.Decode(t.Context(), logs)
			require.NoError(t, err)

			if t.Name() != "TradingFeePPMUpdated" {
				for _, poolAddress := range tt.poolAddress {
					require.Contains(t, logByAddress, poolAddress)
					require.Equal(t, len(tt.poolAddress), len(logByAddress))
				}
			} else {
				allPairs, err := getPairs(t.Context(), rpcClient, controller)
				require.NoError(t, err)
				require.Equal(t, len(allPairs), len(logByAddress))
			}
		})
	}
}
