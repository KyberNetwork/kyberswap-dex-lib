package curve

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	testHook  = "0x695cff9c7f11fa87ca05c7b0a0fa64c3554b3ece"
	testUsdc  = "0x3600000000000000000000000000000000000000"
	testToken = "0x0fe35c3d33fddc8c8dcaf271efcea03251d81fc4"
)

func u(s string) *uint256.Int { return uint256.MustFromDecimal(s) }

func newSim(t *testing.T, extra Extra) *PoolSimulator {
	t.Helper()
	extraBytes, err := json.Marshal(extra)
	require.NoError(t, err)
	sim, err := NewPoolSimulator(entity.Pool{
		Address:     PoolAddress("0x" + "11"),
		Exchange:    DexType,
		Type:        DexType,
		Tokens:      []*entity.PoolToken{{Address: testUsdc, Swappable: true}, {Address: testToken, Swappable: true}},
		Reserves:    entity.PoolReserves{"0", "0"},
		StaticExtra: `{"hook":"` + testHook + `"}`,
		Extra:       string(extraBytes),
	})
	require.NoError(t, err)
	return sim
}

func curving(sold, real string) Extra {
	return Extra{Tracked: true, Mode: modePump, Status: statusCurving, TokensSold: u(sold), RealUsdcReserve: u(real)}
}

// Vectors: ArcadeHook.buy / sell executed on the deployed hook bytecode against a
// real v4-core PoolManager in Foundry (contracts/v4test ArcadeHookSwap harness), each
// from a state snapshot, reading the caller's balance deltas and the stored curve
// state before and after.
func TestParity_Buy(t *testing.T) {
	NowFn = func() int64 { return 1 }
	cases := []struct {
		name, sold, real, amountIn, tokensOut, pulled, newSold, newReal string
		graduates                                                       bool
	}{
		{"fresh buy 1 USDC", "0", "0", "1000000", "196920554300225959327322", "1000000", "196920554300225959327322", "990000", false},
		{"fresh buy 123.456789 USDC", "0", "0", "123456789", "23786956479430314485352977", "123456789", "23786956479430314485352977", "122222222", false},
		{"fresh buy 5000 USDC", "0", "0", "5000000000", "518305263157894736842105264", "5000000000", "518305263157894736842105264", "4950000000", false},
		{"mid buy 2 USDC", "638283333333333333333333334", "7700000000", "2000000", "68377243413487976803479", "2000000", "638351710576746821310136813", "7701980000", false},
		{"mid buy 3333 USDC", "638283333333333333333333334", "7700000000", "3333000000", "91176038520770415408308166", "3333000000", "729459371854103748741641500", "10999670000", false},
		// Crosses CURVE_SUPPLY: clipped to the remaining tokens, 14,169.118676 USDC
		// never leaves the buyer, and the launch graduates in the same transaction.
		{"graduating buy 20000 USDC", "638283333333333333333333334", "7700000000", "20000000000", "138716666666666666666666666", "5830881324", "777000000000000000000000000", "13472572511", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			sim := newSim(t, curving(c.sold, c.real))
			amountIn := u(c.amountIn).ToBig()
			res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{Token: testUsdc, Amount: amountIn},
				TokenOut:      testToken,
			})
			require.NoError(t, err)
			assert.Equal(t, c.tokensOut, res.TokenAmountOut.Amount.String())

			info := res.SwapInfo.(*SwapInfo)
			assert.Equal(t, c.pulled, info.AmountIn.String())
			refund := new(big.Int).Sub(amountIn, u(c.pulled).ToBig())
			if refund.Sign() > 0 {
				require.NotNil(t, res.RemainingTokenAmountIn)
				assert.Equal(t, refund, res.RemainingTokenAmountIn.Amount)
			} else {
				assert.Nil(t, res.RemainingTokenAmountIn)
			}
			assert.Equal(t, c.newSold, info.NewTokensSold.Dec())
			assert.Equal(t, c.newReal, info.NewRealUsdcReserve.Dec())
			assert.Equal(t, c.graduates, info.Graduates)

			sim.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: info})
			_, err = sim.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{Token: testUsdc, Amount: big.NewInt(1_000_000)},
				TokenOut:      testToken,
			})
			if c.graduates {
				assert.ErrorIs(t, err, ErrNotCurving, "a graduated launch no longer trades on the curve")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestParity_Sell(t *testing.T) {
	cases := []struct{ name, tokensIn, usdcOut, newSold, newReal string }{
		{"sell 1 token", "1000000000000000000", "29", "638283332333333333333333334", "7699999971"},
		{"sell 12.5M tokens", "12500000000000000000000000", "348727986", "625783333333333333333333334", "7347749510"},
		{"sell 300M tokens", "300000000000000000000000000", "5186285968", "338283333333333333333333334", "2461327306"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			sim := newSim(t, curving("638283333333333333333333334", "7700000000"))
			res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{Token: testToken, Amount: u(c.tokensIn).ToBig()},
				TokenOut:      testUsdc,
			})
			require.NoError(t, err)
			assert.Equal(t, c.usdcOut, res.TokenAmountOut.Amount.String())
			info := res.SwapInfo.(*SwapInfo)
			assert.Equal(t, c.newSold, info.NewTokensSold.Dec())
			assert.Equal(t, c.newReal, info.NewRealUsdcReserve.Dec())
			assert.Nil(t, res.RemainingTokenAmountIn)
		})
	}
}

// A launch created with startBps 3000 over 600s (launchedAt 1): the buyer receives the
// curve output minus the decaying token haircut, while the curve state moves by the
// full, pre-tax output.
func TestParity_AntiSniperTax(t *testing.T) {
	extra := curving("0", "0")
	extra.SnipeStartBps, extra.SnipeDecaySeconds, extra.SnipeLaunchedAt = 3_000, 600, 1
	cases := []struct {
		now       int64
		tokensOut string
	}{
		{1, "32983062200956937799043063"},
		{151, "36516961722488038277511963"},
		{600, "47095100956937799043062202"},
		{601, "47118660287081339712918661"}, // window over: no tax
	}
	for _, c := range cases {
		NowFn = func() int64 { return c.now }
		sim := newSim(t, extra)
		res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: testUsdc, Amount: big.NewInt(250_000_000)},
			TokenOut:      testToken,
		})
		require.NoError(t, err)
		assert.Equal(t, c.tokensOut, res.TokenAmountOut.Amount.String(), "t=%d", c.now)
		info := res.SwapInfo.(*SwapInfo)
		assert.Equal(t, "47118660287081339712918661", info.NewTokensSold.Dec(), "state moves by the pre-tax output")
		assert.Equal(t, "247500000", info.NewRealUsdcReserve.Dec())
	}
	NowFn = func() int64 { return 1 }
}

func TestRefusals(t *testing.T) {
	buy := pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: testUsdc, Amount: big.NewInt(1_000_000)}, TokenOut: testToken}

	_, err := newSim(t, Extra{}).CalcAmountOut(buy)
	assert.ErrorIs(t, err, ErrNotTracked)

	for _, e := range []Extra{
		{Tracked: true, Mode: modePump, Status: 1, TokensSold: u("0"), RealUsdcReserve: u("0")},               // graduating
		{Tracked: true, Mode: modePump, Status: statusGraduated, TokensSold: u("0"), RealUsdcReserve: u("0")}, // graduated
		{Tracked: true, Mode: 1, Status: statusGraduated, TokensSold: u("0"), RealUsdcReserve: u("0")},        // CLANKER
	} {
		_, err = newSim(t, e).CalcAmountOut(buy)
		assert.ErrorIs(t, err, ErrNotCurving)
	}

	paused := curving("0", "0")
	paused.Paused = true
	_, err = newSim(t, paused).CalcAmountOut(buy)
	assert.ErrorIs(t, err, ErrPaused)

	_, err = newSim(t, curving("1000", "0")).CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testToken, Amount: big.NewInt(1001)}, TokenOut: testUsdc,
	})
	assert.ErrorIs(t, err, ErrSellExceedsSold)

	_, err = newSim(t, curving("0", "0")).CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testUsdc, Amount: big.NewInt(0)}, TokenOut: testToken,
	})
	assert.ErrorIs(t, err, ErrZeroAmount)
}

func TestCloneStateIsIndependent(t *testing.T) {
	sim := newSim(t, curving("0", "0"))
	clone := sim.CloneState().(*PoolSimulator)
	res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testUsdc, Amount: big.NewInt(5_000_000_000)}, TokenOut: testToken,
	})
	require.NoError(t, err)
	sim.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: res.SwapInfo})
	assert.Equal(t, "0", clone.extra.TokensSold.Dec())
	assert.NotEqual(t, "0", sim.extra.TokensSold.Dec())
}

func TestPoolFactoryDecoder(t *testing.T) {
	d := NewPoolFactoryDecoder(&Config{DexID: DexType, ChainID: valueobject.ChainIDArc, Hook: testHook})
	poolID := common.HexToHash("0xe9ab261f1c77caeefd37aec54859484c93c8856b95739e779e4827616cab54fe")
	launch := func(mode uint8) types.Log {
		data, err := arcadeHookABI.Events["LaunchCreated"].Inputs.NonIndexed().Pack(common.HexToAddress("0xC0FFEE"), mode)
		require.NoError(t, err)
		return types.Log{
			Address: common.HexToAddress(testHook),
			Topics:  []common.Hash{launchCreatedEventHash, poolID, common.BytesToHash(common.HexToAddress(testToken).Bytes())},
			Data:    data,
		}
	}

	p, err := d.DecodePoolCreated(launch(modePump))
	require.NoError(t, err)
	require.NotNil(t, p)
	assert.Equal(t, PoolAddress(poolID.Hex()), p.Address)
	assert.Equal(t, testUsdc, p.Tokens[0].Address)
	assert.Equal(t, testToken, p.Tokens[1].Address)
	assert.JSONEq(t, `{"hook":"`+testHook+`"}`, p.StaticExtra)

	p, err = d.DecodePoolCreated(launch(1)) // CLANKER: no curve
	require.NoError(t, err)
	assert.Nil(t, p)

	foreign := launch(modePump)
	foreign.Address = common.HexToAddress("0xdead")
	p, err = d.DecodePoolCreated(foreign)
	require.NoError(t, err)
	assert.Nil(t, p)

	for _, ev := range []common.Hash{curveBuyEventHash, curveSellEventHash, graduatedEventHash} {
		addrs, err := d.DecodePoolAddressesFromFactoryLog(context.Background(), types.Log{Address: common.HexToAddress(testHook), Topics: []common.Hash{ev, poolID}})
		require.NoError(t, err)
		assert.Equal(t, []string{PoolAddress(poolID.Hex())}, addrs)
	}
	assert.Equal(t, poolID.Hex(), PoolIDFromAddress(PoolAddress(poolID.Hex())))
}
