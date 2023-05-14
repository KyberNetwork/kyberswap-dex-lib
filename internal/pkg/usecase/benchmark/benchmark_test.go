package benchmark

//
//import (
//	"context"
//	_ "embed"
//	"encoding/csv"
//	"encoding/json"
//	"errors"
//	"fmt"
//	"math"
//	"math/big"
//	"os"
//	"runtime"
//	"strconv"
//	"strings"
//	"sync"
//	"testing"
//	"time"
//
//	"github.com/pkg/profile"
//	"github.com/stretchr/testify/assert"
//	"github.com/stretchr/testify/require"
//
//	"github.com/KyberNetwork/router-service/internal/pkg/constant"
//	"github.com/KyberNetwork/router-service/internal/pkg/core"
//	poolPkg "github.com/KyberNetwork/router-service/internal/pkg/core/pool"
//	"github.com/KyberNetwork/router-service/internal/pkg/entity"
//	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute"
//	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute/bruteforce"
//	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute/spfa"
//	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute/spfav2"
//	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute/uniswap"
//	"github.com/KyberNetwork/router-service/internal/pkg/utils/eth"
//	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
//)
//
//const (
//	configFilePath = "../../config/files/dev/polygon.yaml"
//	saveGas        = false
//	gasInclude     = false
//)
//
//type finderAlgo int
//
//const (
//	finderAlgoSPFA finderAlgo = iota
//	finderAlgoUniswap
//	finderAlgoBruteforce
//	finderAlgoSpfav2
//)
//
//var algoName = []string{"spfa", "uniswap", "bruteforce", "spfav2"}
//
//var (
//	algoToBenchmark = []finderAlgo{finderAlgoSPFA, finderAlgoSpfav2}
//
//	// we will compare rate of other algo to baseAlgo
//	// index in algoToBenchmark
//	baseAlgoIndex = 0
//)
//
//type multiplyOfAvgPriceRange struct {
//	minMultiply, maxMultiply int
//}
//
//var multiplyRange = []multiplyOfAvgPriceRange{
//	{1, 1},
//	{10, 10},
//	{50, 51},
//	{100, 100},
//	{500, 501},
//	{1000, 1000},
//	{5000, 5001},
//	{10000, 10000},
//}
//
////go:embed benchmark_tokens.json
//var benchmarkTokensJSON string
//
//type tokenData struct {
//	Address  string `json:"address"`
//	Symbol   string `json:"symbol"`
//	Decimals uint8  `json:"decimals"`
//}
//
//var benchmarkTokens []tokenData
//
//type topTradingPairs struct {
//	Pair      string
//	TokenIn   string
//	TokenOut  string
//	Volume    float64
//	AvgAmount float64
//}
//
//type testcase struct {
//	tokenInAddress, tokenOutAddress string
//	tokenInSymbol, tokenOutSymbol   string
//	amountIn                        string
//	testName                        string
//}
//
//type finderContext struct {
//	input             findroute.Input
//	pools             []entity.Pool
//	finderDataFactory func() findroute.FinderData
//}
//
//func init() {
//	if err := json.Unmarshal([]byte(benchmarkTokensJSON), &benchmarkTokens); err != nil {
//		panic(err)
//	}
//}
//
//func newDefaultIFinderFromID(algoID finderAlgo, uc *benchmarkUseCase, finderCtx *finderContext) findroute.IFinder {
//	switch algoID {
//	case finderAlgoSPFA:
//		return spfa.NewDefaultSPFAFinder()
//	case finderAlgoUniswap:
//		return uniswap.NewDefaultUniswapFinder()
//	case finderAlgoBruteforce:
//		return bruteforce.NewDefaultBruteforceFinder(finderCtx.pools, uc.poolFactory)
//	case finderAlgoSpfav2:
//		return spfav2.NewDefaultSPFAv2Finder()
//	default:
//		panic("invalid algo")
//	}
//}
//
//func makeFinderContext(uc *benchmarkUseCase, tc *testcase) (*finderContext, error) {
//	tokenInAddress, err := eth.ConvertEtherToWETH(tc.tokenInAddress, uc.config.ChainID)
//	if err != nil {
//		return nil, err
//	}
//
//	tokenOutAddress, err := eth.ConvertEtherToWETH(tc.tokenOutAddress, uc.config.ChainID)
//	if err != nil {
//		return nil, err
//	}
//
//	gasTokenAddress := strings.ToLower(uc.config.GasTokenAddress)
//
//	// allow all sources
//	sources := uc.getSources(nil, nil)
//
//	pools, err := uc.listPools(context.TODO(), tokenInAddress, tokenOutAddress, sources)
//	if err != nil {
//		return nil, err
//	}
//
//	tokenAddresses := getTokenAddresses(pools, gasTokenAddress, tokenInAddress, tokenOutAddress)
//
//	tokenByAddress, err := uc.getTokenByAddress(context.TODO(), tokenAddresses)
//	if err != nil {
//		return nil, err
//	}
//
//	priceByAddress, err := uc.getPriceByAddress(context.TODO(), tokenAddresses)
//	if err != nil {
//		return nil, err
//	}
//
//	gasPrice, err := uc.getGasPrice(context.TODO(), nil)
//	if err != nil {
//		return nil, err
//	}
//
//	preferredPriceUSDByAddress := make(map[string]float64, len(priceByAddress))
//	for address, price := range priceByAddress {
//		preferredPrice, _ := price.GetPreferredPrice()
//		preferredPriceUSDByAddress[address] = preferredPrice
//	}
//
//	gasTokenPriceUSD := preferredPriceUSDByAddress[gasTokenAddress]
//
//	amountInBi, ok := new(big.Int).SetString(tc.amountIn, 10)
//	// fmt.Println(tokenInAddress, amountInBi)
//	// fmt.Println(preferredPriceUSDByAddress[tokenInAddress])
//	if !ok {
//		return nil, errors.New("invalid amountIn")
//	}
//
//	input := findroute.Input{
//		TokenInAddress:   tokenInAddress,
//		TokenOutAddress:  tokenOutAddress,
//		AmountIn:         amountInBi,
//		GasPrice:         gasPrice,
//		GasTokenPriceUSD: gasTokenPriceUSD,
//		SaveGas:          saveGas,
//		GasInclude:       gasInclude,
//	}
//	finderDataFactory := func() findroute.FinderData {
//		return findroute.FinderData{
//			PoolByAddress:     uc.poolFactory.NewPoolByAddress(pools),
//			TokenByAddress:    tokenByAddress,
//			PriceUSDByAddress: preferredPriceUSDByAddress,
//		}
//	}
//
//	return &finderContext{
//		input:             input,
//		pools:             pools,
//		finderDataFactory: finderDataFactory,
//	}, nil
//}
//
//// adapted from GetRouteUseCase.Handle
//func testRun(uc *benchmarkUseCase, w *csv.Writer, tc testcase) error {
//	finderCtx, err := makeFinderContext(uc, &tc)
//	if err != nil {
//		return err
//	}
//
//	allAlgoBestRoute := make([]*core.Route, len(algoToBenchmark))
//	allAlgoSummamize := make([]valueobject.Route, len(algoToBenchmark))
//	allAlgoRunningTime := make([]string, len(algoToBenchmark))
//	var allAlgoDiffVsBaseAlgo []string
//	for i, algoID := range algoToBenchmark {
//
//		finder := newDefaultIFinderFromID(algoID, uc, finderCtx)
//
//		algoStartTime := time.Now()
//		algoBestRoutes, err := finder.Find(context.TODO(), finderCtx.input, finderCtx.finderDataFactory())
//		if err != nil {
//			fmt.Printf("[%v] failed to find route, err: %v\n", algoName[algoID], err)
//			return nil
//		}
//		allAlgoRunningTime[i] = strconv.FormatInt(time.Since(algoStartTime).Milliseconds(), 10)
//
//		allAlgoBestRoute[i] = extractBestRoute(algoBestRoutes)
//
//		algoSummarize, err := allAlgoBestRoute[i].Summarize(uc.poolFactory.NewPools(finderCtx.pools))
//		if err != nil {
//			fmt.Printf("[%v] failed to summarize, err: %v\n", algoName[algoID], err)
//			return nil
//		}
//		allAlgoSummamize[i] = algoSummarize
//
//	}
//
//	for i := range algoToBenchmark {
//		if i != baseAlgoIndex {
//			allAlgoDiffVsBaseAlgo = append(allAlgoDiffVsBaseAlgo,
//				new(big.Int).Sub(allAlgoSummamize[i].OutputAmount, allAlgoSummamize[baseAlgoIndex].OutputAmount).String())
//		}
//	}
//
//	for i, summarizedRoute := range allAlgoSummamize {
//		fmt.Printf("[%v] Summarize Result: %v \n", algoName[algoToBenchmark[i]], summarizedRoute.OutputAmount.String())
//		for _, path := range summarizedRoute.Route {
//			fmt.Println(path[0].SwapAmount, "(", new(big.Int).Div(new(big.Int).Mul(path[0].SwapAmount, big.NewInt(100)), summarizedRoute.InputAmount).String(), ")")
//			for j, swap := range path {
//				fmt.Print("pool: ", swap.Pool, " ", swap.Exchange, " => ")
//				if j == len(path)-1 {
//					fmt.Println(swap.AmountOut)
//				}
//			}
//		}
//		fmt.Println()
//	}
//
//	var testResult = []string{tc.tokenInSymbol, tc.tokenOutSymbol, fmt.Sprintf("%f", allAlgoBestRoute[baseAlgoIndex].Input.AmountUsd)}
//
//	for _, summarizedRoute := range allAlgoSummamize {
//		testResult = append(testResult, summarizedRoute.OutputAmount.String())
//	}
//
//	testResult = append(testResult, allAlgoDiffVsBaseAlgo...)
//
//	testResult = append(testResult, allAlgoRunningTime...)
//
//	for _, summarizedRoute := range allAlgoSummamize {
//		testResult = append(testResult, strconv.Itoa(len(summarizedRoute.Route)))
//	}
//
//	w.Write(testResult)
//	w.Flush()
//	return nil
//}
//
//func TestSwap(t *testing.T) {
//	t.Skip("The benchmark is skipped on CI, if you want to run it manually, please comment this line")
//
//	uc, err := newMockBenchmarkUseCase(configFilePath)
//	assert.Nil(t, err)
//
//	// WETH
//	tokenInAddress := "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"
//
//	// KNCL
//	tokenOutAddress := "0xdd974d5c2e2928dea5f71b9825b8b646686bd200"
//
//	// gasTokenAddress := strings.ToLower(uc.config.GasTokenAddress)
//
//	// allow all sources
//	sources := uc.getSources(nil, nil)
//
//	pools, err := uc.listPools(context.TODO(), tokenInAddress, tokenOutAddress, sources)
//	assert.Nil(t, err)
//
//	for _, pool := range pools {
//		// fmt.Println(pool.Address)
//		if pool.Address == "0x76838fd2f22bdc1d3e96069971e65653173edb2a" {
//			iswap := uc.poolFactory.NewPools([]entity.Pool{pool})
//			out, err := iswap[0].CalcAmountOut(poolPkg.TokenAmount{
//				Token:     tokenInAddress,
//				Amount:    big.NewInt(1750000000000000000),
//				AmountUsd: 0,
//			}, tokenOutAddress)
//			assert.Nil(t, err)
//			expect, _ := new(big.Int).SetString("3121467533032199786769", 10)
//			assert.Equal(t, out.TokenAmountOut.Amount.Cmp(expect), 0)
//
//			iswap = uc.poolFactory.NewPools([]entity.Pool{pool})
//			out, err = iswap[0].CalcAmountOut(poolPkg.TokenAmount{
//				Token:     tokenInAddress,
//				Amount:    big.NewInt(2000000000000000000),
//				AmountUsd: 0,
//			}, tokenOutAddress)
//			assert.Nil(t, err)
//
//			expect, _ = new(big.Int).SetString("3516176456323544536541", 10)
//			assert.Equal(t, out.TokenAmountOut.Amount.Cmp(expect), 0)
//
//		}
//	}
//}
//
//func makeTestCaseFromTokens(uc *benchmarkUseCase) ([]testcase, error) {
//	var (
//		tests []testcase
//		pairs []topTradingPairs
//	)
//
//	// open file
//	f, err := os.Open("./toptradingpairs.csv")
//	if err != nil {
//		return nil, err
//	}
//	// remember to close the file at the end of the program
//	defer f.Close()
//
//	// read csv values using csv.Reader
//	csvReader := csv.NewReader(f)
//	data, err := csvReader.ReadAll()
//	if err != nil {
//		return nil, err
//	}
//	for i, topPairSlice := range data {
//		if i > 0 {
//			volume, err := strconv.ParseFloat(topPairSlice[3], 64)
//			if err != nil {
//				return nil, err
//			}
//			avgAmount, err := strconv.ParseFloat(topPairSlice[4], 64)
//			if err != nil {
//				return nil, err
//			}
//			pairs = append(pairs, topTradingPairs{
//				Pair:      topPairSlice[0],
//				TokenIn:   topPairSlice[1],
//				TokenOut:  topPairSlice[2],
//				Volume:    volume,
//				AvgAmount: avgAmount,
//			})
//		}
//	}
//
//	for _, tokenIn := range benchmarkTokens {
//		for _, tokenOut := range benchmarkTokens {
//			if tokenIn != tokenOut {
//				for _, topPair := range pairs {
//					// only test top trade pairs
//					if tokenIn.Address == topPair.TokenIn && tokenOut.Address == topPair.TokenOut {
//						price, err := uc.priceRepository.FindByAddresses(context.Background(), []string{tokenIn.Address})
//						if err != nil {
//							return nil, err
//						}
//						if len(price) == 0 {
//							return nil, errors.New("can't get price from db")
//						}
//						tokenInPreferredPrice, _ := price[0].GetPreferredPrice()
//						minAmount := math.Max(topPair.AvgAmount/tokenInPreferredPrice, 1)
//						for _, r := range multiplyRange {
//							for i := r.minMultiply; i <= r.maxMultiply; i++ {
//								amountIn := new(big.Int).Mul(constant.TenPowInt(tokenIn.Decimals), new(big.Int).Mul(big.NewInt(int64(i)), big.NewInt(int64(minAmount)))).String()
//								tests = append(tests, testcase{
//									tokenInAddress:  strings.ToLower(tokenIn.Address),
//									tokenOutAddress: strings.ToLower(tokenOut.Address),
//									tokenInSymbol:   tokenIn.Symbol,
//									tokenOutSymbol:  tokenOut.Symbol,
//									amountIn:        amountIn,
//									testName:        fmt.Sprintf("%v %v %v", tokenIn.Symbol, tokenOut.Symbol, amountIn),
//								})
//							}
//						}
//					}
//				}
//			}
//		}
//	}
//
//	return tests, nil
//}
//
//func TestBenchmarkAlgorithm(t *testing.T) {
//	t.Skip("The benchmark is skipped on CI, if you want to run it manually, please comment this line")
//
//	uc, err := newMockBenchmarkUseCase(configFilePath)
//	assert.Nil(t, err)
//
//	tests, err := makeTestCaseFromTokens(uc)
//	assert.Nil(t, err)
//
//	f, err := os.Create("test_results.csv")
//	assert.Nil(t, err)
//
//	defer f.Close()
//	w := csv.NewWriter(f)
//	defer w.Flush()
//
//	attributeNames := []string{"tokenIn", "tokenOut", "amountInUsd"}
//	for _, algo := range algoToBenchmark {
//		attributeNames = append(attributeNames, fmt.Sprintf("%vAmountOut", algoName[algo]))
//	}
//	for i, algo := range algoToBenchmark {
//		if i == baseAlgoIndex {
//			continue
//		}
//		attributeNames = append(attributeNames, fmt.Sprintf("%vDiff", algoName[algo]))
//	}
//	for _, algo := range algoToBenchmark {
//		attributeNames = append(attributeNames, fmt.Sprintf("%vExecTime", algoName[algo]))
//	}
//	for _, algo := range algoToBenchmark {
//		attributeNames = append(attributeNames, fmt.Sprintf("%vSplit", algoName[algo]))
//	}
//
//	w.Write(attributeNames)
//
//	for _, test := range tests {
//		t.Run(test.testName, func(t *testing.T) {
//			assert.Nil(t, testRun(uc, w, test))
//		})
//	}
//}
//
//func findRouteByAlgorithmAndTestCase(algo finderAlgo, uc *benchmarkUseCase, tc testcase, result chan *big.Int) error {
//	finderCtx, err := makeFinderContext(uc, &tc)
//	if err != nil {
//		return err
//	}
//
//	var finder = newDefaultIFinderFromID(algo, uc, finderCtx)
//
//	routes, err := finder.Find(context.TODO(), finderCtx.input, finderCtx.finderDataFactory())
//	if err != nil {
//		return err
//	}
//	bestRoute := extractBestRoute(routes)
//	summary, err := bestRoute.Summarize(uc.poolFactory.NewPools(finderCtx.pools))
//	if err != nil {
//		return err
//	}
//	fmt.Printf("amountOut = %s\n", summary.OutputAmount)
//	_ = summary
//	result <- summary.OutputAmount
//
//	return nil
//}
//
//func TestProfileSingleAlgorithmConcurrently(t *testing.T) {
//	t.Skip()
//
//	uc, err := newMockBenchmarkUseCase(configFilePath)
//	require.NoError(t, err)
//
//	var (
//		algo = finderAlgoSPFA
//		// rng  = rand.New(rand.NewSource(time.Now().UnixNano()))
//	)
//
//	tests, err := makeTestCaseFromTokens(uc)
//	assert.Nil(t, err)
//
//	defer profile.Start(profile.CPUProfile).Stop()
//
//	taskCh := make(chan uint64, 1024)
//	wg := new(sync.WaitGroup)
//
//	numWorkers := runtime.NumCPU()
//	var numTasks = uint64(len(tests))
//	amountOutResultChan := make(chan *big.Int, numTasks)
//
//	worker := func() {
//		defer wg.Done()
//		for i := range taskCh {
//			tc := tests[i]
//			fmt.Printf("running testcase %s\n", tc.testName)
//			if err := findRouteByAlgorithmAndTestCase(algo, uc, tc, amountOutResultChan); err != nil {
//				panic(err)
//			}
//		}
//	}
//
//	for i := 0; i < numWorkers; i++ {
//		wg.Add(1)
//		go worker()
//	}
//
//	wg.Add(1)
//	go func() {
//		defer wg.Done()
//		var i uint64
//		for ; i < numTasks; i++ {
//			taskCh <- i
//		}
//		close(taskCh)
//	}()
//
//	go func() {
//		defer close(amountOutResultChan)
//		wg.Wait()
//	}()
//
//	sumAllResult := big.NewInt(0)
//	for x := range amountOutResultChan {
//		sumAllResult = new(big.Int).Add(sumAllResult, x)
//	}
//	fmt.Println("Sum amountOut of all testcases: ", sumAllResult)
//
//}
