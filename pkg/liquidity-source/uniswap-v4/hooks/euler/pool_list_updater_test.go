package euler

import (
	"context"
	"encoding/json"
	"log"
	"math/big"
	"os"
	"strings"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

var eulerSwapABI = `[{"inputs":[],"name":"AmountInMoreThanMax","type":"error"},{"inputs":[],"name":"AmountOutLessThanMin","type":"error"},{"inputs":[],"name":"DeadlineExpired","type":"error"},{"inputs":[{"internalType":"address","name":"token","type":"address"}],"name":"SafeERC20FailedOperation","type":"error"},{"inputs":[{"internalType":"address","name":"eulerSwap","type":"address"},{"internalType":"address","name":"tokenIn","type":"address"},{"internalType":"address","name":"tokenOut","type":"address"}],"name":"getLimits","outputs":[{"internalType":"uint256","name":"","type":"uint256"},{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},{"inputs":[{"internalType":"address","name":"eulerSwap","type":"address"},{"internalType":"address","name":"tokenIn","type":"address"},{"internalType":"address","name":"tokenOut","type":"address"},{"internalType":"uint256","name":"amountIn","type":"uint256"}],"name":"quoteExactInput","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},{"inputs":[{"internalType":"address","name":"eulerSwap","type":"address"},{"internalType":"address","name":"tokenIn","type":"address"},{"internalType":"address","name":"tokenOut","type":"address"},{"internalType":"uint256","name":"amountOut","type":"uint256"}],"name":"quoteExactOutput","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},{"inputs":[{"internalType":"address","name":"eulerSwap","type":"address"},{"internalType":"address","name":"tokenIn","type":"address"},{"internalType":"address","name":"tokenOut","type":"address"},{"internalType":"uint256","name":"amountIn","type":"uint256"},{"internalType":"address","name":"receiver","type":"address"},{"internalType":"uint256","name":"amountOutMin","type":"uint256"},{"internalType":"uint256","name":"deadline","type":"uint256"}],"name":"swapExactIn","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[{"internalType":"address","name":"eulerSwap","type":"address"},{"internalType":"address","name":"tokenIn","type":"address"},{"internalType":"address","name":"tokenOut","type":"address"},{"internalType":"uint256","name":"amountOut","type":"uint256"},{"internalType":"address","name":"receiver","type":"address"},{"internalType":"uint256","name":"amountInMax","type":"uint256"},{"internalType":"uint256","name":"deadline","type":"uint256"}],"name":"swapExactOut","outputs":[],"stateMutability":"nonpayable","type":"function"}]`

func TestPoolListUpdater(t *testing.T) {
	t.Parallel()
	if os.Getenv("CI") != "" {
		t.Skip("Skipping testing in CI environment")
	}

	plUpdater := NewPoolsListUpdater(&Config{
		DexID:          DexType,
		FactoryAddress: "0xFb9FE66472917F0F8966506A3bf831Ac0c10caD4",
	}, ethrpc.New("https://ethereum.kyberengineering.io").
		SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11")))

	newPools, _, err := plUpdater.GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	require.Greater(t, len(newPools), 0)

	log.Fatalf("%+v", newPools[1])
}

func TestQuoteExactInputRPC(t *testing.T) {
	// Test data from pool_simulator_test.go
	poolStr := `{"address":"0x69058613588536167ba0aa94f0cc1fe420ef28a8","exchange":"uniswap-v4-euler","type":"uniswap-v4-euler","timestamp":1749726634,"reserves":["845320505406","266499434512256655425"],"tokens":[{"address":"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48","symbol":"USDC","decimals":6,"swappable":true},{"address":"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2","symbol":"WETH","decimals":18,"swappable":true}],"extra":"{\"p\":1,\"v\":[{\"Cash\":\"3744304484283\",\"Debt\":\"0\",\"MaxDeposit\":\"46752543110116\",\"MaxWithdraw\":\"67500000000000\",\"TotalBorrows\":\"24503152405600\",\"EulerAccountAssets\":\"345894015181\"},{\"Cash\":\"4640741482437344429395\",\"Debt\":\"35006069375450104270\",\"MaxDeposit\":\"58923720760039413629508\",\"MaxWithdraw\":\"90000000000000000000000\",\"TotalBorrows\":\"36435537757523241941096\",\"EulerAccountAssets\":\"0\"}]}","staticExtra":"{\"v0\":\"0x797DD80692c3b2dAdabCe8e30C07fDE5307D48a9\",\"v1\":\"0xD8b27CF359b7D15710a5BE299AF6e7Bf904984C2\",\"ea\":\"0x0afBf798467F9b3b97F90d05bf7DF592D89A6CF1\",\"f\":\"500000000000000\",\"pf\":\"0\",\"er0\":\"751024805196\",\"er1\":\"301566016943501539193\",\"px\":\"379218809252938\",\"py\":\"1000000\",\"cx\":\"850000000000000000\",\"cy\":\"850000000000000000\"}","blockNumber":22688100}`

	var pool entity.Pool
	err := json.Unmarshal([]byte(poolStr), &pool)
	require.Nil(t, err)

	// Create contract instance
	client := ethrpc.New("https://ethereum.kyberengineering.io").
		SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))

	// Contract address for EulerSwap
	contractAddress := common.HexToAddress("0x208fF5Eb543814789321DaA1B5Eb551881D16b06")

	// Test quoteExactInput
	amountIn, _ := new(big.Int).SetString("1000000", 10)                          // 1 USDC
	tokenIn := common.HexToAddress("0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48")  // USDC
	tokenOut := common.HexToAddress("0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2") // WETH

	abi, err := abi.JSON(strings.NewReader(eulerSwapABI))
	require.Nil(t, err)

	var out *big.Int
	request := client.NewRequest()
	request.AddCall(&ethrpc.Call{
		ABI:    abi,
		Target: contractAddress.String(),
		Method: "quoteExactInput",
		Params: []any{common.HexToAddress("0x69058613588536167ba0aa94f0cc1fe420ef28a8"), tokenIn, tokenOut, amountIn},
	}, []any{&out})

	_, err = request.Call()
	require.Nil(t, err)

	t.Fatalf("Amount out: %s", out.String())
}
