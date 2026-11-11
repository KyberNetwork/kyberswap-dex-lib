package oneinchaqua

import (
	"encoding/json"
	"math/big"
	"strings"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/swaplimit"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type poolOpts struct {
	program    []byte
	app        common.Address
	maker      common.Address
	raw        [2]*uint256.Int
	wallet     [2]*uint256.Int
	allowance  [2]*uint256.Int
	rawBlock   uint64
	walletBlk  uint64
	docked     bool
	inactive   bool
	tokens     [2]string
	strategyId string
}

var unlimited = new(uint256.Int).Not(new(uint256.Int))

func defaultProgram() []byte {
	return prog(iConcentrate(sqrtPrice(e18(1)), sqrtPrice(e18(4))), iFlat(10_000_000), iSwap(), iSalt(1))
}

func newEntity(t testing.TB, o poolOpts) entity.Pool {
	t.Helper()
	if o.program == nil {
		o.program = defaultProgram()
	}
	if o.app == (common.Address{}) {
		o.app = testRouter
	}
	if o.maker == (common.Address{}) {
		o.maker = testMaker
	}
	if o.tokens == [2]string{} {
		o.tokens = [2]string{testToken0, testToken1}
	}
	for i := range 2 {
		if o.raw[i] == nil {
			o.raw[i] = e18(1000)
		}
		if o.wallet[i] == nil {
			o.wallet[i] = e18(1_000_000)
		}
		if o.allowance[i] == nil {
			o.allowance[i] = unlimited
		}
	}
	if o.rawBlock == 0 && o.walletBlk == 0 {
		o.rawBlock, o.walletBlk = 100, 100
	}
	order := orderOf(o.maker, o.program)
	extra := Extra{
		RawBalance0: o.raw[0], RawBalance1: o.raw[1], RawBlock: o.rawBlock,
		WalletBalance0: o.wallet[0], WalletBalance1: o.wallet[1],
		Allowance0: o.allowance[0], Allowance1: o.allowance[1], WalletBlock: o.walletBlk,
		Docked: o.docked, Inactive: o.inactive,
	}
	traits := order.Traits
	static := StaticExtra{
		Maker: addrHex(o.maker), App: addrHex(o.app), StrategyHash: order.Hash().Hex(), Launch: o.app == testLaunch,
		Traits: traits.Hex(), Data: hexutil.Encode(order.Data),
	}
	eb, err := json.Marshal(extra)
	require.NoError(t, err)
	sb, err := json.Marshal(static)
	require.NoError(t, err)
	reserves := []string{minDec(o.raw[0], o.wallet[0], o.allowance[0]), minDec(o.raw[1], o.wallet[1], o.allowance[1])}
	return entity.Pool{
		Address:     "0xpool" + o.strategyId,
		Exchange:    valueobject.ExchangeOneInchAqua,
		Type:        DexType,
		Reserves:    reserves,
		Tokens:      []*entity.PoolToken{{Address: o.tokens[0], Swappable: true}, {Address: o.tokens[1], Swappable: true}},
		Extra:       string(eb),
		StaticExtra: string(sb),
		BlockNumber: o.rawBlock,
	}
}

func minDec(vals ...*uint256.Int) string {
	m := vals[0]
	for _, v := range vals[1:] {
		if v.Lt(m) {
			m = v
		}
	}
	return m.Dec()
}

func newSim(t testing.TB, o poolOpts) *PoolSimulator {
	t.Helper()
	s, err := NewPoolSimulator(newEntity(t, o))
	require.NoError(t, err)
	return s
}

func quoteOut(s *PoolSimulator, in int, amount *uint256.Int, limit pool.SwapLimit) (*pool.CalcAmountOutResult, error) {
	return s.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: s.Info.Tokens[in], Amount: amount.ToBig()},
		TokenOut:      s.Info.Tokens[1-in],
		Limit:         limit,
	})
}

func swapOnce(s *PoolSimulator, in int, amount *uint256.Int, limit pool.SwapLimit) (*pool.CalcAmountOutResult, error) {
	res, err := quoteOut(s, in, amount, limit)
	if err != nil {
		return nil, err
	}
	s.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: s.Info.Tokens[in], Amount: amount.ToBig()},
		TokenAmountOut: *res.TokenAmountOut,
		Fee:            *res.Fee,
		SwapInfo:       res.SwapInfo,
		SwapLimit:      limit,
	})
	return res, nil
}

func TestNewPoolSimulatorErrors(t *testing.T) {
	t.Parallel()
	good := newEntity(t, poolOpts{})
	mutate := func(f func(e *entity.Pool)) entity.Pool {
		e := good
		e.Tokens = append([]*entity.PoolToken(nil), good.Tokens...)
		f(&e)
		return e
	}
	staticWith := func(f func(s *StaticExtra)) entity.Pool {
		var s StaticExtra
		require.NoError(t, json.Unmarshal([]byte(good.StaticExtra), &s))
		f(&s)
		b, _ := json.Marshal(s)
		return mutate(func(e *entity.Pool) { e.StaticExtra = string(b) })
	}
	guarded := orderOf(testMaker, prog(guard(), iSwap()))
	for name, e := range map[string]entity.Pool{
		"one token":   mutate(func(e *entity.Pool) { e.Tokens = e.Tokens[:1] }),
		"one reserve": mutate(func(e *entity.Pool) { e.Reserves = e.Reserves[:1] }),
		"same token": mutate(func(e *entity.Pool) {
			e.Tokens[1] = &entity.PoolToken{Address: strings.ToUpper(testToken0[:2]) + testToken0[2:]}
		}),
		"bad extra":      mutate(func(e *entity.Pool) { e.Extra = "{" }),
		"bad static":     mutate(func(e *entity.Pool) { e.StaticExtra = "{" }),
		"bad traits":     staticWith(func(s *StaticExtra) { s.Traits = "0xzz" }),
		"bad data":       staticWith(func(s *StaticExtra) { s.Data = "nothex" }),
		"gated program":  staticWith(func(s *StaticExtra) { s.Data = hexutil.Encode(guarded.Data) }),
		"signature mode": staticWith(func(s *StaticExtra) { s.Traits = "0x0" }),
	} {
		_, err := NewPoolSimulator(e)
		require.Error(t, err, name)
	}
}

// The golden quotes through the simulator: same amounts as the router, in both token orders.
func TestCalcAmountOutGolden(t *testing.T) {
	t.Parallel()
	for _, v := range loadGolden(t) {
		o, err := DecodeOrder(hexutil.MustDecode(v.Strategy))
		require.NoError(t, err)
		raw := [2]*uint256.Int{u(v.Raw[0]), u(v.Raw[1])}
		s := newSim(t, poolOpts{program: o.Data, maker: o.Maker, app: common.HexToAddress(v.App), raw: raw,
			tokens: [2]string{strings.ToLower(v.Tokens[0]), strings.ToLower(v.Tokens[1])}})
		res, err := quoteOut(s, v.In, u(v.AmountIn), nil)
		// The router quotes without the wallet caps; skip the vectors those caps would refuse.
		if err != nil {
			require.ErrorIs(t, err, pool.ErrNotEnoughInventory)
			continue
		}
		require.Equal(t, v.AmountOut, res.TokenAmountOut.Amount.String())
		require.Equal(t, s.Info.Tokens[1-v.In], res.TokenAmountOut.Token)
	}
}

func TestCalcAmountOutBasics(t *testing.T) {
	t.Parallel()
	s := newSim(t, poolOpts{})
	for in := range 2 {
		res, err := quoteOut(s, in, e18(10), nil)
		require.NoError(t, err)
		require.True(t, res.IsValid())
		require.Equal(t, s.Info.Tokens[1-in], res.TokenAmountOut.Token)
		require.Equal(t, int64(defaultGas), res.Gas)
		require.Equal(t, "0", res.Fee.Amount.String())
		require.Equal(t, s.Info.Tokens[in], res.Fee.Token)
		info, ok := res.SwapInfo.(*SwapInfo)
		require.True(t, ok)
		require.Equal(t, e18(10), &info.AmountIn)
		require.Equal(t, res.TokenAmountOut.Amount.String(), info.AmountOut.Dec())
	}
	// The same quote twice: the call is pure.
	a, _ := quoteOut(s, 0, e18(10), nil)
	b, _ := quoteOut(s, 0, e18(10), nil)
	require.Equal(t, a.TokenAmountOut.Amount, b.TokenAmountOut.Amount)

	// Token order does not change the price: swapping the token addresses mirrors the concentrate bound.
	flipped := newSim(t, poolOpts{tokens: [2]string{testToken1, testToken0}})
	r0, _ := quoteOut(s, 0, e18(10), nil)
	r1, _ := quoteOut(flipped, 1, e18(10), nil)
	require.NotEmpty(t, r0)
	require.NotEmpty(t, r1)
	_, err := quoteOut(flipped, 0, e18(10), nil)
	require.NoError(t, err)
}

func TestCalcAmountOutInvalidInput(t *testing.T) {
	t.Parallel()
	s := newSim(t, poolOpts{})
	tok0, tok1 := s.Info.Tokens[0], s.Info.Tokens[1]
	call := func(in, out string, amount *big.Int) error {
		_, err := s.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: in, Amount: amount}, TokenOut: out})
		return err
	}
	require.ErrorIs(t, call("0xunknown", tok1, big.NewInt(1)), ErrInvalidToken)
	require.ErrorIs(t, call(tok0, "0xunknown", big.NewInt(1)), ErrInvalidToken)
	require.ErrorIs(t, call(tok0, tok0, big.NewInt(1)), ErrInvalidToken)
	require.ErrorIs(t, call(tok0, tok1, nil), ErrInvalidAmountIn)
	require.ErrorIs(t, call(tok0, tok1, big.NewInt(0)), ErrInvalidAmountIn)
	require.ErrorIs(t, call(tok0, tok1, big.NewInt(-5)), ErrInvalidAmountIn)
	require.ErrorIs(t, call(tok0, tok1, new(big.Int).Lsh(big.NewInt(1), 256)), ErrInvalidAmountIn)
	_, err := quoteOut(s, 0, uint256.NewInt(1), nil)
	require.ErrorIs(t, err, ErrZeroAmountOut)
	// A plain xyc strategy with an empty side cannot quote.
	empty := newSim(t, poolOpts{program: prog(iSwap()), raw: [2]*uint256.Int{new(uint256.Int), e18(5)}})
	_, err = quoteOut(empty, 0, e18(1), nil)
	require.ErrorIs(t, err, ErrZeroBalance)
}

// A strategy only quotes with one wallet snapshot taken at the
// block of its raw balances.
func TestCalcAmountOutStateGuards(t *testing.T) {
	t.Parallel()
	_, err := quoteOut(newSim(t, poolOpts{rawBlock: 50, walletBlk: 51}), 0, e18(1), nil)
	require.ErrorIs(t, err, ErrStateMixedBlock)
	_, err = quoteOut(newSim(t, poolOpts{rawBlock: 51, walletBlk: 50}), 0, e18(1), nil)
	require.ErrorIs(t, err, ErrStateMixedBlock)

	// No wallet snapshot: both blocks zero, a lister-created pool.
	fresh := newEntity(t, poolOpts{rawBlock: 1, walletBlk: 1})
	fresh.Extra = "{}"
	s, err := NewPoolSimulator(fresh)
	require.NoError(t, err)
	_, err = quoteOut(s, 0, e18(1), nil)
	require.ErrorIs(t, err, ErrNoWalletSnapshot)
	// A raw read without a wallet read never quotes either.
	_, err = quoteOut(newSim(t, poolOpts{rawBlock: 7, walletBlk: 0}), 0, e18(1), nil)
	require.ErrorIs(t, err, ErrNoWalletSnapshot)
	require.Nil(t, s.CalculateLimit())

	for name, o := range map[string]poolOpts{"docked": {docked: true}, "inactive": {inactive: true}} {
		s := newSim(t, o)
		_, err := quoteOut(s, 0, e18(1), nil)
		require.ErrorIs(t, err, ErrStrategyTerminal, name)
		require.Nil(t, s.CalculateLimit(), name)
	}
}

// A strategy that is wallet-limited (raw > 0, wallet empty) gives no quote; once the
// maker tops the wallet up and the strategy is seen docked it is terminal and never quotes.
func TestDockedWhileDormant(t *testing.T) {
	t.Parallel()
	dormant := newSim(t, poolOpts{wallet: [2]*uint256.Int{new(uint256.Int), new(uint256.Int)}})
	for in := range 2 {
		_, err := quoteOut(dormant, in, e18(1), nil)
		require.ErrorIs(t, err, ErrInsufficientWallet)
	}
	require.Equal(t, "0", dormant.Info.Reserves[0].String())
	require.Equal(t, "0", dormant.Info.Reserves[1].String())

	toppedUpAndDocked := newSim(t, poolOpts{docked: true, wallet: [2]*uint256.Int{e18(500), e18(500)}})
	for in := range 2 {
		_, err := quoteOut(toppedUpAndDocked, in, e18(1), nil)
		require.ErrorIs(t, err, ErrStrategyTerminal)
	}
	require.Nil(t, toppedUpAndDocked.CalculateLimit())
}

func TestCalcAmountOutCaps(t *testing.T) {
	t.Parallel()
	// One-sided strategy: 1000e18 of token1 to sell, almost nothing of token0.
	s := newSim(t, poolOpts{raw: [2]*uint256.Int{uint256.NewInt(1), e18(1000)}})
	res, err := quoteOut(s, 0, e18(1), nil)
	require.NoError(t, err)
	out := new(uint256.Int)
	out.SetFromBig(res.TokenAmountOut.Amount)

	// Raw cap: a quote above the strategy balance is refused even with a huge wallet.
	small := newSim(t, poolOpts{raw: [2]*uint256.Int{uint256.NewInt(1), uint256.NewInt(10)}})
	_, err = quoteOut(small, 0, e18(1000), nil)
	require.ErrorIs(t, err, ErrInsufficientRaw)
	require.ErrorIs(t, err, pool.ErrNotEnoughInventory)

	// Wallet cap from the pool's own snapshot: min(balance, allowance).
	byBalance := newSim(t, poolOpts{raw: [2]*uint256.Int{uint256.NewInt(1), e18(1000)}, wallet: [2]*uint256.Int{nil, new(uint256.Int).SubUint64(out, 1)}})
	_, err = quoteOut(byBalance, 0, e18(1), nil)
	require.ErrorIs(t, err, ErrInsufficientWallet)
	byAllowance := newSim(t, poolOpts{raw: [2]*uint256.Int{uint256.NewInt(1), e18(1000)}, allowance: [2]*uint256.Int{nil, new(uint256.Int).SubUint64(out, 1)}})
	_, err = quoteOut(byAllowance, 0, e18(1), nil)
	require.ErrorIs(t, err, ErrInsufficientWallet)
	exact := newSim(t, poolOpts{raw: [2]*uint256.Int{uint256.NewInt(1), e18(1000)}, wallet: [2]*uint256.Int{nil, out}})
	_, err = quoteOut(exact, 0, e18(1), nil)
	require.NoError(t, err, "a limit equal to the output is enough")

	// A shared limit replaces the own snapshot: a missing key means no balance.
	key := s.limitKeys[1]
	inv := swaplimit.NewInventory(DexType, map[string]*big.Int{key: new(big.Int).Set(out.ToBig())})
	_, err = quoteOut(s, 0, e18(1), inv)
	require.NoError(t, err)
	_, err = quoteOut(s, 0, e18(1), swaplimit.NewInventory(DexType, map[string]*big.Int{key: new(big.Int).Sub(out.ToBig(), big.NewInt(1))}))
	require.ErrorIs(t, err, ErrInsufficientWallet)
	_, err = quoteOut(s, 0, e18(1), swaplimit.NewInventory(DexType, map[string]*big.Int{}))
	require.ErrorIs(t, err, ErrInsufficientWallet)
	_, err = quoteOut(s, 0, e18(1), swaplimit.NewInventory(DexType, map[string]*big.Int{key: big.NewInt(-1)}))
	require.ErrorIs(t, err, ErrInsufficientWallet)
	huge := new(big.Int).Lsh(big.NewInt(1), 300)
	_, err = quoteOut(s, 0, e18(1), swaplimit.NewInventory(DexType, map[string]*big.Int{key: huge}))
	require.NoError(t, err, "a limit above 2^256 is not a cap")
}

// Protocol fee collection: collected iff the strategy and the maker wallet cover it; the current router
// skips what it cannot pull, the launch router reverts so the quote is refused.
func TestProtocolFee(t *testing.T) {
	t.Parallel()
	feeProgram := prog(iProtocol(10_000_000, testRecipient), iSwap(), iSalt(1)) // 1%
	amount := e18(1)
	fee := new(uint256.Int).Div(new(uint256.Int).Mul(amount, uint256.NewInt(10_000_000)), uBps)

	collected := newSim(t, poolOpts{program: feeProgram})
	res, err := quoteOut(collected, 0, amount, nil)
	require.NoError(t, err)
	require.Equal(t, fee.Dec(), res.Fee.Amount.String())
	require.Equal(t, int64(defaultGasFeeCollected), res.Gas)
	info := res.SwapInfo.(*SwapInfo)
	require.Equal(t, fee.Dec(), info.FeeIn.Dec())
	require.Equal(t, fee.Dec(), info.FeeCollected.Dec())

	// Boundary: the strategy holds exactly the fee, so it is still collected (THIN_BALANCE_IN case of
	// the SwapVM fee-skip tests: 0.01e18 in, 1% fee, 1e18 in is the largest payable amount).
	thin := poolOpts{program: feeProgram, raw: [2]*uint256.Int{u("10000000000000000"), e18(20_000_000)},
		wallet: [2]*uint256.Int{nil, e18(30_000_000)}}
	s := newSim(t, thin)
	res, err = quoteOut(s, 0, e18(1), nil)
	require.NoError(t, err)
	require.Equal(t, "10000000000000000", res.Fee.Amount.String(), "the fee drains the whole in side")
	require.Equal(t, int64(defaultGasFeeCollected), res.Gas)
	// One step above the ceiling the current router skips the pull and the swap still succeeds.
	above := new(uint256.Int).Add(e18(1), u("100000000000000000"))
	res, err = quoteOut(s, 0, above, nil)
	require.NoError(t, err)
	require.Equal(t, "0", res.Fee.Amount.String())
	require.Equal(t, int64(defaultGas), res.Gas)
	info = res.SwapInfo.(*SwapInfo)
	wantFeeIn := new(uint256.Int).Div(new(uint256.Int).Mul(above, uint256.NewInt(10_000_000)), uBps)
	require.Equal(t, wantFeeIn.Dec(), info.FeeIn.Dec())
	require.True(t, info.FeeCollected.IsZero())
	// The launch router reverts there.
	launch := newSim(t, poolOpts{program: feeProgram, app: testLaunch, raw: thin.raw, wallet: thin.wallet})
	_, err = quoteOut(launch, 0, above, nil)
	require.ErrorIs(t, err, ErrFeeShortfall)
	_, err = quoteOut(launch, 0, e18(1), nil)
	require.NoError(t, err)

	// The maker wallet limit decides too, with or without a shared limit.
	walletShort := poolOpts{program: feeProgram, wallet: [2]*uint256.Int{new(uint256.Int).SubUint64(fee, 1), nil}}
	res, err = quoteOut(newSim(t, walletShort), 0, amount, nil)
	require.NoError(t, err)
	require.Equal(t, "0", res.Fee.Amount.String())
	_, err = quoteOut(newSim(t, poolOpts{program: feeProgram, wallet: walletShort.wallet, app: testLaunch}), 0, amount, nil)
	require.ErrorIs(t, err, ErrFeeShortfall)
	cur := newSim(t, poolOpts{program: feeProgram})
	res, err = quoteOut(cur, 0, amount, swaplimit.NewInventory(DexType, map[string]*big.Int{
		cur.limitKeys[0]: new(big.Int).Sub(fee.ToBig(), big.NewInt(1)), cur.limitKeys[1]: e18(1000).ToBig()}))
	require.NoError(t, err)
	require.Equal(t, "0", res.Fee.Amount.String())

	// A zero fee is not a skipped fee, and no protocol fee instruction leaves FeeIns empty.
	zero := newSim(t, poolOpts{program: prog(iProtocol(0, common.Address{}), iSwap())})
	res, err = quoteOut(zero, 0, amount, nil)
	require.NoError(t, err)
	require.Equal(t, "0", res.Fee.Amount.String())
	require.Equal(t, int64(defaultGas), res.Gas)
	launchZero := newSim(t, poolOpts{program: prog(iProtocol(1, testRecipient), iSwap()), app: testLaunch})
	_, err = quoteOut(launchZero, 0, uint256.NewInt(10), nil) // fee rounds to zero
	require.NoError(t, err)
}

// Two protocol fees on one strategy: each is checked against what the earlier pulls left.
func TestProtocolFeeSequential(t *testing.T) {
	t.Parallel()
	program := prog(iSalt(1), iProtocol(100_000_000, testRecipient), iProtocol(200_000_000, testRecipient), iFlat(50_000_000), iSwap())
	amount := uint256.NewInt(1_000_000) // inner fee 180000 (20%), outer fee 100000 (10%)
	for _, tc := range []struct {
		rawIn     uint64
		collected uint64
	}{
		{1_000_000, 280_000},
		{280_000, 280_000},
		{279_999, 180_000}, // the outer fee no longer fits after the inner pull
		{180_000, 180_000},
		{179_999, 100_000}, // the inner fee is skipped, the outer one still fits
		{100_000, 100_000},
		{99_999, 0},
	} {
		s := newSim(t, poolOpts{program: program, raw: [2]*uint256.Int{uint256.NewInt(tc.rawIn), e18(10)}})
		res, err := quoteOut(s, 0, amount, nil)
		require.NoError(t, err, tc.rawIn)
		require.Equal(t, uint256.NewInt(tc.collected).Dec(), res.Fee.Amount.String(), "rawIn %d", tc.rawIn)
		require.Equal(t, "280000", decOf(res.SwapInfo.(*SwapInfo).FeeIn))
		launch := newSim(t, poolOpts{program: program, app: testLaunch, raw: [2]*uint256.Int{uint256.NewInt(tc.rawIn), e18(10)}})
		_, err = quoteOut(launch, 0, amount, nil)
		if tc.collected == 280_000 {
			require.NoError(t, err)
		} else {
			require.ErrorIs(t, err, ErrFeeShortfall, tc.rawIn)
		}
	}
	// The wallet limit is shared by both pulls.
	s := newSim(t, poolOpts{program: program, wallet: [2]*uint256.Int{uint256.NewInt(250_000), nil}})
	res, err := quoteOut(s, 0, amount, nil)
	require.NoError(t, err)
	require.Equal(t, "180000", res.Fee.Amount.String())
}

// Aqua balances are uint248: a push that would overflow them reverts. The product amountIn*balanceOut
// must fit in 256 bits too, so the out side is kept small enough to leave a non-zero output.
func TestUint248PushBound(t *testing.T) {
	t.Parallel()
	rawOut := new(uint256.Int).Lsh(uint256.NewInt(1), 180)
	amount := u("1000000000000000000000") // 1e21
	rawIn := func(headroom *uint256.Int) [2]*uint256.Int {
		return [2]*uint256.Int{new(uint256.Int).Sub(max248, headroom), rawOut}
	}
	exact := newSim(t, poolOpts{program: prog(iSwap()), raw: rawIn(amount)})
	_, err := quoteOut(exact, 0, amount, nil)
	require.NoError(t, err, "a push of exactly the headroom fits")
	over := newSim(t, poolOpts{program: prog(iSwap()), raw: rawIn(new(uint256.Int).SubUint64(amount, 1))})
	_, err = quoteOut(over, 0, amount, nil)
	require.ErrorIs(t, err, ErrOverflow, "one wei above the headroom")
	// With a fee collected the push is measured after the pull.
	withFee := newSim(t, poolOpts{program: prog(iProtocol(10_000_000, testRecipient), iSwap()), raw: rawIn(new(uint256.Int).SubUint64(amount, 1))})
	_, err = quoteOut(withFee, 0, amount, nil)
	require.NoError(t, err, "the fee pull frees room for the push")
}

func TestUpdateBalance(t *testing.T) {
	t.Parallel()
	program := prog(iProtocol(10_000_000, testRecipient), iConcentrate(sqrtPrice(e18(1)), sqrtPrice(e18(4))), iFlat(10_000_000), iSwap(), iSalt(1))
	s := newSim(t, poolOpts{program: program, wallet: [2]*uint256.Int{e18(500), e18(900)},
		allowance: [2]*uint256.Int{e18(400), e18(2000)}})
	res, err := swapOnce(s, 0, e18(10), nil)
	require.NoError(t, err)
	info := res.SwapInfo.(*SwapInfo)
	require.False(t, info.FeeCollected.IsZero())

	// Strategy: rawIn += amountIn - fee, rawOut -= amountOut.
	wantIn := new(uint256.Int).Add(e18(1000), new(uint256.Int).Sub(&info.AmountIn, &info.FeeCollected))
	wantOut := new(uint256.Int).Sub(e18(1000), &info.AmountOut)
	require.Equal(t, wantIn.Dec(), s.State.Raw[0].Dec())
	require.Equal(t, wantOut.Dec(), s.State.Raw[1].Dec())
	// Own wallet snapshot: out leaves the wallet and lowers the finite allowance; in arrives net of the fee.
	require.Equal(t, new(uint256.Int).Sub(e18(900), &info.AmountOut).Dec(), s.State.Wallet[1].Dec())
	require.Equal(t, new(uint256.Int).Sub(e18(2000), &info.AmountOut).Dec(), s.State.Allowance[1].Dec())
	require.Equal(t, new(uint256.Int).Add(e18(500), wantNet(info)).Dec(), s.State.Wallet[0].Dec())
	require.Equal(t, new(uint256.Int).Sub(e18(400), &info.FeeCollected).Dec(), s.State.Allowance[0].Dec())
	// Blocks never move on a swap.
	require.Equal(t, uint64(100), s.State.RawBlock)
	require.Equal(t, uint64(100), s.State.WalletBlock)
	// The published reserves are the fillable amounts.
	require.Equal(t, minDec(&s.State.Raw[0], &s.State.Wallet[0], &s.State.Allowance[0]), s.Info.Reserves[0].String())
	require.Equal(t, minDec(&s.State.Raw[1], &s.State.Wallet[1], &s.State.Allowance[1]), s.Info.Reserves[1].String())

	// A second swap sees the first: worse price.
	res2, err := quoteOut(s, 0, e18(10), nil)
	require.NoError(t, err)
	require.True(t, res2.TokenAmountOut.Amount.Cmp(res.TokenAmountOut.Amount) < 0)

	// Unknown swap info or tokens change nothing.
	before := s.State
	s.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: "x"})
	s.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: info, TokenAmountIn: pool.TokenAmount{Token: "0xno"}, TokenAmountOut: pool.TokenAmount{Token: s.Info.Tokens[1]}})
	s.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: info, TokenAmountIn: pool.TokenAmount{Token: s.Info.Tokens[0]}, TokenAmountOut: pool.TokenAmount{Token: s.Info.Tokens[0]}})
	require.Equal(t, before, s.State)
}

func wantNet(info *SwapInfo) *uint256.Int {
	return new(uint256.Int).Sub(&info.AmountIn, &info.FeeCollected)
}

// The wallet snapshot of the pool floors at zero and an unlimited allowance does not move.
func TestUpdateBalanceSnapshotEdges(t *testing.T) {
	t.Parallel()
	s := newSim(t, poolOpts{wallet: [2]*uint256.Int{nil, nil}, allowance: [2]*uint256.Int{unlimited, unlimited}})
	_, err := swapOnce(s, 0, e18(10), nil)
	require.NoError(t, err)
	require.True(t, s.State.Allowance[0].Eq(unlimited))
	require.True(t, s.State.Allowance[1].Eq(unlimited))

	floor := newSim(t, poolOpts{})
	floor.State.Wallet[1] = *uint256.NewInt(5)
	floor.State.Allowance[1] = *uint256.NewInt(5)
	info := SwapInfo{AmountIn: *uint256.NewInt(10), AmountOut: *uint256.NewInt(9)}
	floor.updateWallet(0, 1, &info)
	require.True(t, floor.State.Wallet[1].IsZero())
	require.True(t, floor.State.Allowance[1].IsZero())
}

func TestCalculateLimit(t *testing.T) {
	t.Parallel()
	s := newSim(t, poolOpts{wallet: [2]*uint256.Int{e18(5), e18(7)}, allowance: [2]*uint256.Int{e18(9), e18(3)}})
	limit := s.CalculateLimit()
	require.Len(t, limit, 2)
	require.Equal(t, e18(5).Dec(), limit[s.limitKeys[0]].String(), "min(balance, allowance)")
	require.Equal(t, e18(3).Dec(), limit[s.limitKeys[1]].String())
	require.Equal(t, addrHex(testMaker)+":"+testToken0, s.limitKeys[0])
	require.Equal(t, pool.SwapLimitMergeMin, s.GetLimitMergePolicy(s.limitKeys[0]))
	_, usesLimit := pool.UseSwapLimit[valueobject.ExchangeOneInchAqua]
	require.True(t, usesLimit)
}

// Two strategies of one maker publish snapshots of the same wallet at different times: the lower wins.
func TestMergeMinOfSnapshots(t *testing.T) {
	t.Parallel()
	a := newSim(t, poolOpts{strategyId: "a", wallet: [2]*uint256.Int{e18(100), e18(100)}})
	b := newSim(t, poolOpts{strategyId: "b", wallet: [2]*uint256.Int{e18(40), e18(250)}})
	limits := map[string]*big.Int{}
	pool.MergeSwapLimits(limits, a)
	pool.MergeSwapLimits(limits, b)
	require.Equal(t, e18(40).Dec(), limits[a.limitKeys[0]].String())
	require.Equal(t, e18(100).Dec(), limits[a.limitKeys[1]].String())
	// Order does not matter.
	limits = map[string]*big.Int{}
	pool.MergeSwapLimits(limits, b)
	pool.MergeSwapLimits(limits, a)
	require.Equal(t, e18(40).Dec(), limits[a.limitKeys[0]].String())
}

// Shared maker wallet inside one route: the first hop lowers what the second strategy of the same maker
// can pay out, and a finite allowance is not credited by the taker's push.
func TestSharedLimitTwoStrategies(t *testing.T) {
	t.Parallel()
	one := newSim(t, poolOpts{strategyId: "1", program: prog(iFlat(10_000_000), iSwap())})
	two := newSim(t, poolOpts{strategyId: "2", program: prog(iFlat(10_000_000), iSwap())})
	first, err := quoteOut(one, 0, e18(100), nil)
	require.NoError(t, err)
	out1 := first.TokenAmountOut.Amount

	// The wallet holds exactly that much token1: after hop one nothing is left for the second strategy.
	inv := swaplimit.NewInventory(DexType, map[string]*big.Int{
		one.limitKeys[0]: e18(10).ToBig(),
		one.limitKeys[1]: new(big.Int).Set(out1),
	})
	_, err = swapOnce(one, 0, e18(100), inv)
	require.NoError(t, err)
	require.Equal(t, "0", inv.GetLimit(one.limitKeys[1]).String(), "tokenOut decreased by amountOut")
	_, err = quoteOut(two, 0, e18(100), inv)
	require.ErrorIs(t, err, ErrInsufficientWallet)
	// tokenIn credit: unlimited allowance credits amountIn into the wallet limit.
	require.Equal(t, new(big.Int).Add(e18(10).ToBig(), e18(100).ToBig()).String(), inv.GetLimit(one.limitKeys[0]).String())

	// With a finite allowance on tokenIn nothing is credited.
	finite := newSim(t, poolOpts{strategyId: "3", program: prog(iFlat(10_000_000), iSwap()), allowance: [2]*uint256.Int{e18(50), nil}})
	inv2 := swaplimit.NewInventory(DexType, map[string]*big.Int{finite.limitKeys[0]: e18(10).ToBig(), finite.limitKeys[1]: e18(900).ToBig()})
	_, err = swapOnce(finite, 0, e18(100), inv2)
	require.NoError(t, err)
	require.Equal(t, e18(10).String(), inv2.GetLimit(finite.limitKeys[0]).String())
	// An allowance exactly at 2^128 counts as unlimited, one below does not.
	edge := new(uint256.Int).Lsh(uint256.NewInt(1), 128)
	for allowance, credited := range map[string]bool{edge.Dec(): true, new(uint256.Int).SubUint64(edge, 1).Dec(): false} {
		sim := newSim(t, poolOpts{strategyId: "4", program: prog(iSwap()), allowance: [2]*uint256.Int{u(allowance), nil}})
		l := swaplimit.NewInventory(DexType, map[string]*big.Int{sim.limitKeys[0]: big.NewInt(0), sim.limitKeys[1]: e18(900).ToBig()})
		_, err = swapOnce(sim, 0, e18(100), l)
		require.NoError(t, err)
		require.Equal(t, credited, l.GetLimit(sim.limitKeys[0]).Sign() > 0, allowance)
	}
}

// The protocol fee debit on the shared limit is clamped to what the key holds, so
// UpdateLimit never sees a decrease above the balance, and a missing key skips it.
func TestFeeDebitClamp(t *testing.T) {
	t.Parallel()
	program := prog(iProtocol(10_000_000, testRecipient), iSwap())
	s := newSim(t, poolOpts{program: program, allowance: [2]*uint256.Int{e18(1), nil}})           // finite: no credit
	fee := new(uint256.Int).Div(new(uint256.Int).Mul(e18(100), uint256.NewInt(10_000_000)), uBps) // 1e18

	// The in key is below the fee: the pull is skipped by the quote, the debit clamps to the key.
	inv := swaplimit.NewInventory(DexType, map[string]*big.Int{s.limitKeys[0]: big.NewInt(1000), s.limitKeys[1]: e18(900).ToBig()})
	res, err := swapOnce(s, 0, e18(100), inv)
	require.NoError(t, err)
	collected := res.SwapInfo.(*SwapInfo).FeeCollected
	require.True(t, collected.IsZero())
	require.Equal(t, fee.Dec(), decOf(res.SwapInfo.(*SwapInfo).FeeIn))
	require.Equal(t, "0", inv.GetLimit(s.limitKeys[0]).String(), "debit clamped to the balance of the key")

	// Collected fee with a roomy key: the key ends at credit minus fee.
	s2 := newSim(t, poolOpts{program: program})
	inv2 := swaplimit.NewInventory(DexType, map[string]*big.Int{s2.limitKeys[0]: e18(5).ToBig(), s2.limitKeys[1]: e18(900).ToBig()})
	_, err = swapOnce(s2, 0, e18(100), inv2)
	require.NoError(t, err)
	want := new(big.Int).Sub(new(big.Int).Add(e18(5).ToBig(), e18(100).ToBig()), fee.ToBig())
	require.Equal(t, want.String(), inv2.GetLimit(s2.limitKeys[0]).String())

	// A missing in key: the credit creates it, so the debit finds it; with no in key and no credit the
	// debit is skipped without error.
	s3 := newSim(t, poolOpts{program: program, allowance: [2]*uint256.Int{e18(1), nil}})
	swap100 := func(limit pool.SwapLimit) {
		s3.UpdateBalance(pool.UpdateBalanceParams{
			TokenAmountIn:  pool.TokenAmount{Token: s3.Info.Tokens[0], Amount: e18(100).ToBig()},
			TokenAmountOut: pool.TokenAmount{Token: s3.Info.Tokens[1], Amount: big.NewInt(1)},
			SwapInfo:       &SwapInfo{AmountIn: *e18(100), AmountOut: *uint256.NewInt(1), FeeIn: *fee},
			SwapLimit:      limit,
		})
	}
	inv3 := swaplimit.NewInventory(DexType, map[string]*big.Int{s3.limitKeys[1]: e18(900).ToBig()})
	swap100(inv3)
	require.NotNil(t, inv3.GetLimit(s3.limitKeys[0]))
	require.Equal(t, "0", inv3.GetLimit(s3.limitKeys[0]).String())
	// The out key missing makes UpdateLimit refuse and nothing else is touched.
	inv4 := swaplimit.NewInventory(DexType, map[string]*big.Int{})
	swap100(inv4)
	require.Nil(t, inv4.GetLimit(s3.limitKeys[0]))
	// A key that the credit left at zero and a zero clamp skip the debit.
	inv5 := swaplimit.NewInventory(DexType, map[string]*big.Int{s3.limitKeys[0]: big.NewInt(0), s3.limitKeys[1]: e18(900).ToBig()})
	swap100(inv5)
	require.Equal(t, "0", inv5.GetLimit(s3.limitKeys[0]).String())
}

func TestMetaAndApproval(t *testing.T) {
	t.Parallel()
	s := newSim(t, poolOpts{app: testLaunch})
	meta, ok := s.GetMetaInfo(s.Info.Tokens[0], s.Info.Tokens[1]).(PoolMeta)
	require.True(t, ok)
	require.Equal(t, addrHex(testLaunch), meta.ApprovalAddress)
	require.Equal(t, addrHex(testMaker), meta.Maker)
	require.Equal(t, uint64(100), meta.BlockNumber)
	require.Equal(t, hexutil.Encode(defaultProgram()), meta.Data)
	require.Equal(t, addrHex(testLaunch), s.GetApprovalAddress("", ""))
	// What the encoder receives rebuilds the order whose hash is the strategy hash.
	order, err := (&StaticExtra{Maker: meta.Maker, Traits: meta.Traits, Data: meta.Data}).order()
	require.NoError(t, err)
	require.Equal(t, s.StaticExtra.StrategyHash, order.Hash().Hex())
	require.Equal(t, []string{s.Info.Tokens[1]}, s.CanSwapTo(s.Info.Tokens[0]))
	require.Equal(t, []string{s.Info.Tokens[0]}, s.CanSwapFrom(s.Info.Tokens[1]))
}

func decOf(z uint256.Int) string { return z.Dec() }

// limitWithoutKeys is a SwapLimit that accepts every update and holds no key.
type limitWithoutKeys struct{ updates int }

func (l *limitWithoutKeys) Clone() pool.SwapLimit           { return l }
func (l *limitWithoutKeys) GetExchange() string             { return DexType }
func (l *limitWithoutKeys) GetLimit(string) *big.Int        { return nil }
func (l *limitWithoutKeys) GetSwapped() map[string]*big.Int { return nil }
func (l *limitWithoutKeys) GetAllowedSenders() string       { return "" }
func (l *limitWithoutKeys) UpdateLimit(_, _ string, _, _ *big.Int) (*big.Int, *big.Int, error) {
	l.updates++
	return nil, nil, nil
}

func TestUpdateBalanceLimitWithoutKey(t *testing.T) {
	t.Parallel()
	s := newSim(t, poolOpts{program: prog(iProtocol(10_000_000, testRecipient), iSwap())})
	limit := &limitWithoutKeys{}
	s.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: s.Info.Tokens[0], Amount: big.NewInt(100)},
		TokenAmountOut: pool.TokenAmount{Token: s.Info.Tokens[1], Amount: big.NewInt(1)},
		SwapInfo:       &SwapInfo{AmountIn: *uint256.NewInt(100), AmountOut: *uint256.NewInt(1), FeeIn: *uint256.NewInt(1)},
		SwapLimit:      limit,
	})
	require.Equal(t, 1, limit.updates, "no debit when the limit holds no key for tokenIn")
}

type snap struct {
	st       state
	reserves [2]string
}

func snapOf(s *PoolSimulator) snap {
	return snap{st: s.State, reserves: [2]string{s.Info.Reserves[0].String(), s.Info.Reserves[1].String()}}
}

func raceSims(t *testing.T) []*PoolSimulator {
	return []*PoolSimulator{
		newSim(t, poolOpts{}),
		newSim(t, poolOpts{program: prog(iProtocol(10_000_000, testRecipient), iFlat(10_000_000), iSwap())}),
		newSim(t, poolOpts{program: prog(iProtocol(10_000_000, testRecipient), iSwap()), allowance: [2]*uint256.Int{e18(1), e18(50)}}),
		newSim(t, poolOpts{app: testLaunch, program: prog(iProtocol(10_000_000, testRecipient), iSwap()), wallet: [2]*uint256.Int{e18(1), nil}}),
	}
}

var raceAmounts = []*uint256.Int{uint256.NewInt(1), e18(1), e18(10), e18(500), e18(5000)}

type q struct {
	out string
	err error
}

func quoteAll(s *PoolSimulator, limit func() pool.SwapLimit) []q {
	var r []q
	for in := range 2 {
		for _, a := range raceAmounts {
			res, err := quoteOut(s, in, a, limit())
			if err != nil {
				r = append(r, q{err: err})
				continue
			}
			r = append(r, q{out: res.TokenAmountOut.Amount.String() + "/" + res.Fee.Amount.String()})
		}
	}
	return r
}

func newInv(s *PoolSimulator) pool.SwapLimit {
	return swaplimit.NewInventory(DexType, map[string]*big.Int{s.limitKeys[0]: e18(3).ToBig(), s.limitKeys[1]: e18(400).ToBig()})
}

// Concurrent quotes on one shared simulator (the router's pattern: one pool set, many goroutines), while
// other goroutines clone it and swap the clones. The original and every quote must be unaffected.
func TestRaceCalcAmountOutCloneState(t *testing.T) {
	for _, s := range raceSims(t) {
		base := snapOf(s)
		noLimit := func() pool.SwapLimit { return nil }
		wantNil := quoteAll(s, noLimit)
		wantInv := quoteAll(s, func() pool.SwapLimit { return newInv(s) })
		var wg sync.WaitGroup
		for g := range 16 {
			wg.Go(func() {
				for range 50 {
					switch g % 4 {
					case 0:
						assert.Equal(t, wantNil, quoteAll(s, noLimit))
					case 1:
						assert.Equal(t, wantInv, quoteAll(s, func() pool.SwapLimit { return newInv(s) }))
					case 2: // clone, swap the clone (and a clone of the clone) both ways
						c := s.CloneState().(*PoolSimulator)
						inv := newInv(s)
						for in := range 2 {
							_, _ = swapOnce(c, in, e18(7), inv)
							cc := c.CloneState().(*PoolSimulator)
							_, _ = swapOnce(cc, 1-in, e18(3), nil)
						}
					case 3: // mutate everything a quote hands out
						res, err := quoteOut(s, 0, e18(10), nil)
						if err == nil {
							res.TokenAmountOut.Amount.SetInt64(0)
							res.Fee.Amount.SetInt64(-1)
							info := res.SwapInfo.(*SwapInfo)
							info.AmountOut.Clear()
							info.FeeCollected.SetAllOne()
						}
						_ = s.CalculateLimit()
						_ = s.GetMetaInfo("", "")
					}
				}
			})
		}
		wg.Wait()
		require.Equal(t, base, snapOf(s), "original state unchanged")
		require.Equal(t, wantNil, quoteAll(s, noLimit))
	}
}

type qIn struct {
	in  string
	out string
	err error
}

// quoteAllIn is quoteAll for CalcAmountIn: every direction and target.
func quoteAllIn(s *PoolSimulator, limit func() pool.SwapLimit) []qIn {
	var r []qIn
	for in := range 2 {
		for _, a := range raceAmounts {
			res, err := s.CalcAmountIn(inParams(s, in, a, limit()))
			if err != nil {
				r = append(r, qIn{err: err})
				continue
			}
			r = append(r, qIn{in: res.TokenAmountIn.Amount.String(), out: res.Fee.Amount.String() + "/" + res.SwapInfo.(*SwapInfo).AmountOut.Dec()})
		}
	}
	return r
}

// CalcAmountIn and CalcAmountOut run concurrently on one shared simulator, next to clones that swap, and
// every result is the same as when run alone; results can be mutated freely.
func TestRaceCalcAmountIn(t *testing.T) {
	for _, s := range raceSims(t) {
		base := snapOf(s)
		noLimit := func() pool.SwapLimit { return nil }
		wantNil := quoteAllIn(s, noLimit)
		wantInv := quoteAllIn(s, func() pool.SwapLimit { return newInv(s) })
		wantOut := quoteAll(s, noLimit)
		var wg sync.WaitGroup
		for g := range 16 {
			wg.Go(func() {
				for range 50 {
					switch g % 4 {
					case 0:
						assert.Equal(t, wantNil, quoteAllIn(s, noLimit))
					case 1:
						assert.Equal(t, wantInv, quoteAllIn(s, func() pool.SwapLimit { return newInv(s) }))
					case 2:
						assert.Equal(t, wantOut, quoteAll(s, noLimit))
						c := s.CloneState().(*PoolSimulator)
						if res, err := c.CalcAmountIn(inParams(c, 0, e18(7), nil)); err == nil {
							c.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: *res.TokenAmountIn,
								TokenAmountOut: pool.TokenAmount{Token: c.Info.Tokens[1], Amount: e18(7).ToBig()}, Fee: *res.Fee, SwapInfo: res.SwapInfo})
						}
					case 3: // mutate everything a result hands out
						res, err := s.CalcAmountIn(inParams(s, 0, e18(10), nil))
						if err == nil {
							res.TokenAmountIn.Amount.SetInt64(0)
							res.Fee.Amount.SetInt64(-1)
							res.RemainingTokenAmountOut.Amount.SetInt64(5)
							info := res.SwapInfo.(*SwapInfo)
							info.AmountIn.Clear()
							info.FeeCollected.SetAllOne()
						}
					}
				}
			})
		}
		wg.Wait()
		require.Equal(t, base, snapOf(s), "original state unchanged")
		require.Equal(t, wantNil, quoteAllIn(s, noLimit))
	}
}

// Failed quotes are pure too.
func TestCalcAmountOutErrorPathPure(t *testing.T) {
	for _, s := range raceSims(t) {
		base := snapOf(s)
		zero := swaplimit.NewInventory(DexType, map[string]*big.Int{s.limitKeys[0]: big.NewInt(0), s.limitKeys[1]: big.NewInt(0)})
		for in := range 2 {
			for _, a := range []*uint256.Int{e18(1), e18(1_000_000_000), new(uint256.Int).Lsh(uint256.NewInt(1), 250)} {
				_, _ = quoteOut(s, in, a, zero)
				_, _ = quoteOut(s, in, a, nil)
			}
		}
		require.Equal(t, base, snapOf(s))
	}
}

func BenchmarkCalcAmountOut(b *testing.B) {
	for _, ls := range liveShapes {
		for _, dir := range []int{0, 1} {
			sim := newSim(b, poolOpts{program: buildProgram(ls.wrappers, ls.salt)})
			params := pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{Token: sim.Info.Tokens[dir], Amount: e18(10).ToBig()},
				TokenOut:      sim.Info.Tokens[1-dir],
			}
			b.Run(ls.name+map[int]string{0: "/0to1", 1: "/1to0"}[dir], func(b *testing.B) {
				b.ReportAllocs()
				for range b.N {
					if _, err := sim.CalcAmountOut(params); err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}

func BenchmarkCalcAmountOutUpdateBalance(b *testing.B) {
	sim := newSim(b, poolOpts{program: buildProgram("PCF", "trail")})
	params := pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: sim.Info.Tokens[0], Amount: e18(1).ToBig()},
		TokenOut:      sim.Info.Tokens[1],
	}
	b.ReportAllocs()
	for range b.N {
		clone := sim.CloneState()
		res, err := clone.CalcAmountOut(params)
		if err != nil {
			b.Fatal(err)
		}
		clone.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: params.TokenAmountIn, TokenAmountOut: *res.TokenAmountOut, Fee: *res.Fee, SwapInfo: res.SwapInfo})
	}
}

// BenchmarkCalcAmountIn: the exact-out search of every live shape, both directions, next to
// BenchmarkCalcAmountOut (the same target is what 10e18 in pays out).
func BenchmarkCalcAmountIn(b *testing.B) {
	for _, ls := range liveShapes {
		for _, dir := range []int{0, 1} {
			sim := newSim(b, poolOpts{program: buildProgram(ls.wrappers, ls.salt)})
			res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{Token: sim.Info.Tokens[dir], Amount: e18(10).ToBig()}, TokenOut: sim.Info.Tokens[1-dir]})
			if err != nil {
				b.Fatal(err)
			}
			params := pool.CalcAmountInParams{TokenAmountOut: *res.TokenAmountOut, TokenIn: sim.Info.Tokens[dir]}
			b.Run(ls.name+map[int]string{0: "/0to1", 1: "/1to0"}[dir], func(b *testing.B) {
				b.ReportAllocs()
				for range b.N {
					if _, err := sim.CalcAmountIn(params); err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}
