package elfomofi

import (
	"fmt"
	"math/big"
	"sync/atomic"
	"testing"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	orderbook "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/order-book"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type testChainIDRPC struct {
	chainID uint64
	calls   atomic.Int32
}

func (s *testChainIDRPC) ChainId() string {
	s.calls.Add(1)
	return fmt.Sprintf("0x%x", s.chainID)
}

func TestResolveHelperAddress(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		chainID uint64
		address string
	}{
		{56, "0x78015E3544d989f0712a0b9986cB05838c3fE06D"},
		{8453, "0xc1b13606FC7227f2554067aFb3fb12De75C02d81"},
		{4663, "0x20D24Ee45c9b338Ef7ACf2eEde75183B9E2C9E3c"},
		{196, ""},
		{5042, ""},
	} {
		t.Run(fmt.Sprint(tc.chainID), func(t *testing.T) {
			t.Parallel()
			service := &testChainIDRPC{chainID: tc.chainID}
			server := rpc.NewServer()
			require.NoError(t, server.RegisterName("eth", service))
			client := rpc.DialInProc(server)
			defer client.Close()
			tracker := NewPoolTracker(&Config{}, ethrpc.NewWithClient(ethclient.NewClient(client)))
			address, err := tracker.resolveHelperAddress(t.Context())
			if tc.address == "" {
				require.ErrorContains(t, err, "no helper address")
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.address, address)
				address, err = tracker.resolveHelperAddress(t.Context())
				require.NoError(t, err)
				require.Equal(t, tc.address, address)
			}
			require.Equal(t, int32(1), service.calls.Load(), "known chain should resolve once per tracker")
		})
	}

	address, err := NewPoolTracker(&Config{HelperAddress: "0x78015E3544d989f0712a0b9986cB05838c3fE06D"}, nil).resolveHelperAddress(t.Context())
	require.NoError(t, err)
	require.Equal(t, helperAddresses[56], address)
	_, err = NewPoolTracker(&Config{HelperAddress: "bad address"}, nil).resolveHelperAddress(t.Context())
	require.ErrorContains(t, err, "invalid helperAddress")
}

func TestHelperOrderbookABIDecode(t *testing.T) {
	t.Parallel()
	base := common.HexToAddress("0x4200000000000000000000000000000000000006")
	quote := common.HexToAddress("0x833589fcd6edb6e08f4c7c32d4f71b54bda02913")
	book := HelperOrderbook{
		Base:                base,
		Quote:               quote,
		AskCumulativeLevels: []CumulativeLevel{{big.NewInt(100), big.NewInt(200)}},
		BidCumulativeLevels: []CumulativeLevel{{big.NewInt(300), big.NewInt(400)}},
		BalanceBase:         big.NewInt(500),
		BalanceQuote:        big.NewInt(600),
	}
	encoded, err := helperABI.Methods["getOrderbook"].Outputs.Pack(book, big.NewInt(123), big.NewInt(456))
	require.NoError(t, err)
	var decoded getOrderbookResult
	require.NoError(t, helperABI.UnpackIntoInterface(&decoded, "getOrderbook", encoded))
	require.Equal(t, book, decoded.Book)
	require.Equal(t, uint64(123), decoded.BlockNumber.Uint64())
	require.Equal(t, int64(456), decoded.BlockTimestamp.Int64())
}

// Base block 51,988,102: the helper's bid probes and swap.getAmountOut
// agree at 5 WETH. The former decade-size probes gave 10,374.575373 USDC
// because they spanned the 1-10 WETH depth break in a single level.
func TestHelperLevelsMatchOnchainQuote(t *testing.T) {
	t.Parallel()
	probes := []CumulativeLevel{
		{mustBigInt("600000000000000000"), big.NewInt(1616347179)},
		{mustBigInt("3600000000000000000"), big.NewInt(9693153117)},
		{mustBigInt("4836006608468564358"), big.NewInt(13020415384)},
		{mustBigInt("38299425196365215770"), big.NewInt(58095068741)},
	}
	bids, err := cumulativeLevelsToOrderbook(probes, 18, 6, 10_000)
	require.NoError(t, err)
	require.Len(t, bids, 5)
	base := "0x4200000000000000000000000000000000000006"
	quote := "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913"
	simulator := newFixtureSimulator(t, base, quote, 18, 6, bids)
	res, err := simulator.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: base, Amount: mustBigInt("5000000000000000000")},
		TokenOut:      quote,
	})
	require.NoError(t, err)
	diff := new(big.Int).Sub(res.TokenAmountOut.Amount, big.NewInt(13241311681))
	require.LessOrEqual(t, new(big.Int).Abs(diff).Int64(), int64(2), "helper quote should match getAmountOut to USDC wei")
}

// Base block 51,989,050. Kyber's ElfomoFi API returned 522.956931 USDC
// for 2 AAPLc and 1,104.637735 for 5 AAPLc. The direct contract quotes
// were 657.925161 and 1,251.317860; helper book levels reproduce them.
func TestHelperLevelsFixAAPLUnderquote(t *testing.T) {
	t.Parallel()
	probes := []CumulativeLevel{
		{big.NewInt(117000000), big.NewInt(384999675)},
		{big.NewInt(260712482), big.NewInt(857563518)},
		{big.NewInt(20600311981), big.NewInt(34326946722)},
	}
	bids, err := cumulativeLevelsToOrderbook(probes, 8, 6, 10_000)
	require.NoError(t, err)
	base := "0xb200000000000000000000c2e324d24d7eecd1fb"
	quote := "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913"
	simulator := newFixtureSimulator(t, base, quote, 8, 6, bids)
	for _, tc := range []struct {
		input, direct, oldAPI int64
	}{
		{200000000, 657925161, 522956931},
		{500000000, 1251317860, 1104637735},
	} {
		result, err := simulator.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: base, Amount: big.NewInt(tc.input)},
			TokenOut:      quote,
		})
		require.NoError(t, err)
		actual := result.TokenAmountOut.Amount.Int64()
		require.InDelta(t, tc.direct, actual, 2)
		require.Greater(t, actual, tc.oldAPI)
	}
}

func TestCumulativeLevelsToOrderbookBuffer(t *testing.T) {
	t.Parallel()
	probes := []CumulativeLevel{
		{big.NewInt(0), big.NewInt(0)},
		{big.NewInt(10), big.NewInt(100)},
		{big.NewInt(20), big.NewInt(200)},
	}
	levels, err := cumulativeLevelsToOrderbook(probes, 0, 0, 9_900)
	require.NoError(t, err)
	require.Equal(t, []orderbook.Level{{0, 0}, {10, 9.9}, {10, 9.9}}, levels)
	require.Equal(t, int64(100), probes[1].AmountOut.Int64(), "conversion must not modify helper data")
	_, err = cumulativeLevelsToOrderbook([]CumulativeLevel{{AmountIn: big.NewInt(1)}}, 0, 0, 10_000)
	require.Error(t, err)
}

func mustBigInt(value string) *big.Int {
	n, ok := new(big.Int).SetString(value, 10)
	if !ok {
		panic(value)
	}
	return n
}

func newFixtureSimulator(t *testing.T, base, quote string, baseDecimals, quoteDecimals uint8, bids []orderbook.Level) *PoolSimulator {
	t.Helper()
	extra, err := json.Marshal(orderbook.Extra{LevelsFrom: [2][]orderbook.Level{bids, nil}})
	require.NoError(t, err)
	simulator, err := NewPoolSimulator(pool.FactoryParams{EntityPool: entity.Pool{
		Address:     "elfomofi_" + base + "_" + quote,
		Exchange:    DexType,
		Type:        DexType,
		Timestamp:   time.Now().Unix(),
		Reserves:    []string{"999999999999999999999999", "999999999999999999999999"},
		Tokens:      []*entity.PoolToken{{Address: base, Decimals: baseDecimals, Swappable: true}, {Address: quote, Decimals: quoteDecimals, Swappable: true}},
		Extra:       string(extra),
		StaticExtra: `{"factoryAddress":"0xf0f0F0F0FB0d738452EfD03A28e8be14C76d5f73"}`,
	}})
	require.NoError(t, err)
	return simulator
}
