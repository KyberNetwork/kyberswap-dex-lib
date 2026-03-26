package canonic

import (
	"context"
	"math/big"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

func TestPoolSimulatorUnit(t *testing.T) {
	extra := Extra{
		MidPrice:   uint256.MustFromDecimal("2000000000000000000"),
		MidPrec:    uint256.MustFromDecimal("1000000000000000000"),
		TakerFee:   uint256.NewInt(300),
		BaseScale:  uint256.MustFromDecimal("1000000000000000000"),
		QuoteScale: uint256.MustFromDecimal("1000000"),
		AskBps:     []uint16{10, 20, 30},
		AskVols: []*uint256.Int{
			uint256.MustFromDecimal("500000000000000000"),
			uint256.MustFromDecimal("500000000000000000"),
			uint256.MustFromDecimal("500000000000000000"),
		},
		BidBps: []uint16{10, 20, 30},
		BidVols: []*uint256.Int{
			uint256.MustFromDecimal("4000000"),
			uint256.MustFromDecimal("4000000"),
			uint256.MustFromDecimal("4000000"),
		},
		Active: true,
	}
	extraBytes, _ := json.Marshal(extra)

	staticExtra := StaticExtra{
		BaseToken:  "0xweth",
		QuoteToken: "0xusdm",
	}
	staticExtraBytes, _ := json.Marshal(staticExtra)

	ep := entity.Pool{
		Address:     "0xpool",
		Exchange:    "canonic",
		Type:        DexType,
		Reserves:    entity.PoolReserves{"1500000000000000000", "12000000"},
		StaticExtra: string(staticExtraBytes),
		Extra:       string(extraBytes),
		Tokens: []*entity.PoolToken{
			{Address: "0xweth", Swappable: true},
			{Address: "0xusdm", Swappable: true},
		},
	}

	sim, err := NewPoolSimulator(ep)
	require.NoError(t, err)

	t.Run("sell base (WETH->USDC)", func(t *testing.T) {
		result, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{
				Token:  "0xweth",
				Amount: big.NewInt(1e17),
			},
			TokenOut: "0xusdm",
		})
		require.NoError(t, err)
		require.True(t, result.TokenAmountOut.Amount.Sign() > 0)
		t.Logf("sell 0.1 ETH -> %s USDC (fee: %s)", result.TokenAmountOut.Amount, result.Fee.Amount)
	})

	t.Run("buy base (USDC->WETH)", func(t *testing.T) {
		result, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{
				Token:  "0xusdm",
				Amount: big.NewInt(200_000),
			},
			TokenOut: "0xweth",
		})
		require.NoError(t, err)
		require.True(t, result.TokenAmountOut.Amount.Sign() > 0)
		t.Logf("buy with 0.2 USDC -> %s WETH (fee: %s)", result.TokenAmountOut.Amount, result.Fee.Amount)
	})

	t.Run("CalcAmountIn sell base", func(t *testing.T) {
		result, err := sim.CalcAmountIn(pool.CalcAmountInParams{
			TokenAmountOut: pool.TokenAmount{
				Token:  "0xusdm",
				Amount: big.NewInt(100_000),
			},
			TokenIn: "0xweth",
		})
		require.NoError(t, err)
		require.True(t, result.TokenAmountIn.Amount.Sign() > 0)
		t.Logf("get 0.1 USDC needs %s WETH", result.TokenAmountIn.Amount)
	})

	t.Run("CloneState", func(t *testing.T) {
		cloned := sim.CloneState().(*PoolSimulator)
		require.Equal(t, sim.askVols[0].String(), cloned.askVols[0].String())

		cloned.askVols[0] = uint256.NewInt(0)
		require.NotEqual(t, sim.askVols[0].String(), cloned.askVols[0].String())
	})

	t.Run("inactive market", func(t *testing.T) {
		sim2 := *sim
		sim2.active = false
		_, err := sim2.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: "0xweth", Amount: big.NewInt(1e17)},
			TokenOut:      "0xusdm",
		})
		require.ErrorIs(t, err, ErrMarketNotActive)
	})
}

func TestPoolTrackerIntegration(t *testing.T) {
	rpcClient := ethrpc.New("https://megaeth.drpc.org")
	rpcClient.SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))

	tracker := NewPoolTracker(&Config{DexId: "canonic"}, rpcClient)

	wethUsdm := entity.Pool{
		Address: "0x23469683e25b780DFDC11410a8e83c923caDF125",
		Tokens: []*entity.PoolToken{
			{Address: "0x4200000000000000000000000000000000000006", Swappable: true},
			{Address: "0xbF5feaFeABE8B926E2453960B30e4574dbeA9fe7", Swappable: true},
		},
		Reserves: entity.PoolReserves{"0", "0"},
		Extra:    "{}",
	}

	updated, err := tracker.GetNewPoolState(context.Background(), wethUsdm, pool.GetNewPoolStateParams{})
	require.NoError(t, err)

	var extra Extra
	require.NoError(t, json.Unmarshal([]byte(updated.Extra), &extra))

	t.Logf("midPrice: %s", extra.MidPrice)
	t.Logf("midPrec: %s", extra.MidPrec)
	t.Logf("takerFee: %s", extra.TakerFee)
	t.Logf("baseScale: %s", extra.BaseScale)
	t.Logf("quoteScale: %s", extra.QuoteScale)
	t.Logf("askRungs: %d, bidRungs: %d", len(extra.AskBps), len(extra.BidBps))
	t.Logf("active: %v", extra.Active)
	t.Logf("reserves: %v", updated.Reserves)
	t.Logf("blockNumber: %d", updated.BlockNumber)

	require.True(t, extra.MidPrice.Gt(uint256.NewInt(0)))
	require.True(t, len(extra.AskBps) > 0 || len(extra.BidBps) > 0)

	sim, err := NewPoolSimulator(updated)
	require.NoError(t, err)

	if extra.Active {
		result, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{
				Token:  updated.Tokens[0].Address,
				Amount: big.NewInt(1e15),
			},
			TokenOut: updated.Tokens[1].Address,
		})
		if err == nil {
			t.Logf("sell 0.001 WETH -> %s USDm (fee: %s)", result.TokenAmountOut.Amount, result.Fee.Amount)
		} else {
			t.Logf("sell 0.001 WETH err: %v", err)
		}
	}
}

func TestPreviewerComparison(t *testing.T) {
	rpcClient := ethrpc.New("https://megaeth.drpc.org")
	rpcClient.SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))

	tracker := NewPoolTracker(&Config{DexId: "canonic"}, rpcClient)

	wethUsdm := entity.Pool{
		Address: "0x23469683e25b780DFDC11410a8e83c923caDF125",
		Tokens: []*entity.PoolToken{
			{Address: "0x4200000000000000000000000000000000000006", Swappable: true},
			{Address: "0xbF5feaFeABE8B926E2453960B30e4574dbeA9fe7", Swappable: true},
		},
		Reserves: entity.PoolReserves{"0", "0"},
		Extra:    "{}",
	}

	updated, err := tracker.GetNewPoolState(context.Background(), wethUsdm, pool.GetNewPoolStateParams{})
	require.NoError(t, err)

	sim, err := NewPoolSimulator(updated)
	require.NoError(t, err)

	previewerAddr := "0xEaeD40cC4bA1e7A2A7CA3f1A22C815B628B074Ea"
	maobAddr := common.HexToAddress(updated.Address)

	sellAmount := big.NewInt(1e15)
	var previewResult struct {
		QuoteOut     *big.Int
		QuoteFeePaid *big.Int
		BaseUsed     *big.Int
	}

	req := rpcClient.R().SetContext(context.Background())
	req.AddCall(&ethrpc.Call{
		ABI:    previewerABI,
		Target: previewerAddr,
		Method: "previewSellBaseTargetIn",
		Params: []any{maobAddr, sellAmount, big.NewInt(0), uint16(64), big.NewInt(0)},
	}, []any{&previewResult})

	_, err = req.Call()
	require.NoError(t, err)

	t.Logf("previewer: quoteOut=%s fee=%s baseUsed=%s", previewResult.QuoteOut, previewResult.QuoteFeePaid, previewResult.BaseUsed)

	simResult, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  updated.Tokens[0].Address,
			Amount: sellAmount,
		},
		TokenOut: updated.Tokens[1].Address,
	})
	require.NoError(t, err)

	t.Logf("simulator: quoteOut=%s fee=%s", simResult.TokenAmountOut.Amount, simResult.Fee.Amount)

	diff := new(big.Int).Sub(previewResult.QuoteOut, simResult.TokenAmountOut.Amount)
	if diff.Sign() < 0 {
		diff.Neg(diff)
	}
	t.Logf("diff: %s", diff)
}
