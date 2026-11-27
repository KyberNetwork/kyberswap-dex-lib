package rateob

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

var forkABI = func() abi.ABI {
	a, err := abi.JSON(strings.NewReader(`[
{"type":"function","name":"limitSell","inputs":[{"name":"input","type":"tuple","components":[{"name":"base","type":"address"},{"name":"quote","type":"address"},{"name":"price","type":"uint256"},{"name":"amount","type":"uint256"},{"name":"isMaker","type":"bool"},{"name":"n","type":"uint32"},{"name":"recipient","type":"address"}]}],"outputs":[]},
{"type":"function","name":"limitBuy","inputs":[{"name":"input","type":"tuple","components":[{"name":"base","type":"address"},{"name":"quote","type":"address"},{"name":"price","type":"uint256"},{"name":"amount","type":"uint256"},{"name":"isMaker","type":"bool"},{"name":"n","type":"uint32"},{"name":"recipient","type":"address"}]}],"outputs":[]},
{"type":"function","name":"mint","inputs":[{"name":"p","type":"tuple","components":[{"name":"pool","type":"address"},{"name":"bands","type":"uint8[]"},{"name":"baseAmounts","type":"uint256[]"},{"name":"quoteAmounts","type":"uint256[]"},{"name":"minShares","type":"uint128[]"},{"name":"recipient","type":"address"},{"name":"deadline","type":"uint256"}]}],"outputs":[]},
{"type":"function","name":"approve","inputs":[{"name":"spender","type":"address"},{"name":"amount","type":"uint256"}],"outputs":[{"name":"","type":"bool"}]},
{"type":"function","name":"balanceOf","inputs":[{"name":"a","type":"address"}],"outputs":[{"name":"","type":"uint256"}]}]`))
	if err != nil {
		panic(err)
	}
	return a
}()

type limitOrderInput struct {
	Base      common.Address
	Quote     common.Address
	Price     *big.Int
	Amount    *big.Int
	IsMaker   bool
	N         uint32
	Recipient common.Address
}

type mintParams struct {
	Pool         common.Address
	Bands        []uint8
	BaseAmounts  []*big.Int
	QuoteAmounts []*big.Int
	MinShares    []*big.Int
	Recipient    common.Address
	Deadline     *big.Int
}

type forkChain struct {
	t   *testing.T
	rpc *rpc.Client
}

func (c forkChain) call(result any, method string, args ...any) {
	c.t.Helper()
	require.NoError(c.t, c.rpc.CallContext(context.Background(), result, method, args...), method)
}

func (c forkChain) balance(token, account common.Address) *big.Int {
	data, _ := forkABI.Pack("balanceOf", account)
	var out hexutil.Bytes
	c.call(&out, "eth_call", map[string]any{"to": token, "data": hexutil.Bytes(data)}, "latest")
	return new(big.Int).SetBytes(out)
}

// nativeBalance is the account's balance of the chain's gas coin.
func (c forkChain) nativeBalance(account common.Address) *big.Int {
	var out hexutil.Big
	c.call(&out, "eth_getBalance", account, "latest")
	return out.ToInt()
}

// submit sends a transaction from an impersonated account without waiting for it to be mined.
func (c forkChain) submit(from, to common.Address, data []byte) common.Hash {
	var hash common.Hash
	c.call(&hash, "eth_sendTransaction", map[string]any{"from": from, "to": to, "data": hexutil.Bytes(data),
		"gas": hexutil.Uint64(8_000_000)})
	return hash
}

type receipt struct {
	GasUsed           hexutil.Uint64 `json:"gasUsed"`
	EffectiveGasPrice hexutil.Big    `json:"effectiveGasPrice"`
	L1Fee             *hexutil.Big   `json:"l1Fee"`
	Status            hexutil.Uint64 `json:"status"`
	BlockNumber       hexutil.Uint64 `json:"blockNumber"`
}

// receipt waits for a transaction and returns its gas used, its fee in the gas coin (OP Stack:
// the L1 data fee on top), its status and its block.
func (c forkChain) receipt(hash common.Hash) (uint64, *big.Int, bool, uint64) {
	var r receipt
	for range 50 {
		if err := c.rpc.CallContext(context.Background(), &r, "eth_getTransactionReceipt", hash); err == nil &&
			r.GasUsed > 0 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	fee := new(big.Int).Mul(new(big.Int).SetUint64(uint64(r.GasUsed)), r.EffectiveGasPrice.ToInt())
	if r.L1Fee != nil {
		fee.Add(fee, r.L1Fee.ToInt())
	}
	return uint64(r.GasUsed), fee, r.Status == 1, uint64(r.BlockNumber)
}

func (c forkChain) send(from, to common.Address, data []byte) (uint64, *big.Int, bool) {
	gas, fee, ok, _ := c.receipt(c.submit(from, to, data))
	return gas, fee, ok
}

func (c forkChain) mustSend(from, to common.Address, method string, args ...any) {
	c.t.Helper()
	data, err := forkABI.Pack(method, args...)
	require.NoError(c.t, err)
	_, _, ok := c.send(from, to, data)
	require.True(c.t, ok, "%s reverted", method)
}

// fund writes a balance into the token's balance mapping, probing the usual slots.
func (c forkChain) fund(token, account common.Address, amount *big.Int) {
	for slot := range 12 {
		key := crypto.Keccak256Hash(common.LeftPadBytes(account[:], 32), common.LeftPadBytes(big.NewInt(int64(slot)).Bytes(), 32))
		c.call(nil, "anvil_setStorageAt", token, key, common.BigToHash(amount))
		if c.balance(token, account).Cmp(amount) == 0 {
			return
		}
		c.call(nil, "anvil_setStorageAt", token, key, common.Hash{})
	}
	c.t.Fatalf("no balance slot found for %s", token)
}

// account impersonates addr, gives it gas, funds it with both tokens and approves spender.
func (c forkChain) account(addr common.Address, tokens []common.Address, spenders ...common.Address) {
	c.call(nil, "anvil_impersonateAccount", addr)
	c.call(nil, "anvil_setBalance", addr, hexutil.EncodeBig(big.NewInt(1e18)))
	amount, _ := new(big.Int).SetString("1000000000000000000000000000000", 10)
	for _, tok := range tokens {
		c.fund(tok, addr, amount)
		for _, spender := range spenders {
			c.mustSend(addr, tok, "approve", spender, new(big.Int).Lsh(big.NewInt(1), 255))
		}
	}
}

func e(s string) *big.Int {
	v, ok := new(big.Int).SetString(s, 10)
	if !ok {
		panic(s)
	}
	return v
}

// seedForkBook gives the fork's ETH/USDC book and band pool something to quote, all through the
// deployed contracts from impersonated accounts (no key is involved):
//
//  1. A bid at 2263 and a sell into it, so the book's last price is the market's rather than
//     the 2306.705 listing price (a pool anchored there would be arbitraged at once).
//  2. Fifteen asks on ten levels from 2264 and twenty-one bids on eleven levels from 2263,
//     with one more order beyond the engine's reach on each side.
//  3. Liquidity in all three bands through BandPositionManager.mint (band 0 already holds a
//     tiny real position, so that deposit follows its ratio).
//  4. Seven minutes pass, so the 300 s TWAP the pool prices off is exactly lmp.
func seedForkBook(t *testing.T, c forkChain, engine, positionManager, bandPool, base, quote common.Address) {
	maker := common.HexToAddress("0x00000000000000000000000000000000000beef2")
	maker2 := common.HexToAddress("0x00000000000000000000000000000000000beef3")
	lp := common.HexToAddress("0x00000000000000000000000000000000000beef4")
	tokens := []common.Address{base, quote}
	c.account(maker, tokens, engine)
	c.account(maker2, tokens, engine)
	c.account(lp, tokens, positionManager)

	order := func(from common.Address, method string, price int64, amount string, isMaker bool) {
		c.mustSend(from, engine, method, limitOrderInput{Base: base, Quote: quote,
			Price: new(big.Int).Mul(big.NewInt(price), big.NewInt(1e6)), Amount: e(amount), IsMaker: isMaker,
			N: 20, Recipient: from})
	}
	// Prices below are in cents: x 1e6 puts them on the 1e8 grid.
	order(maker, "limitBuy", 226300, "1000000000", true)            // bid 1000 USDC at 2263
	order(maker2, "limitSell", 226300, "100000000000000000", false) // sell 0.1 ETH into it: lmp 2263
	for i := range int64(10) {
		order(maker, "limitSell", 226400+100*i, "300000000000000000", true) // 0.3 ETH asks, 2264..2273
		order(maker, "limitBuy", 226200-100*i, "600000000", true)           // 600 USDC bids, 2262..2253
		order(maker2, "limitBuy", 226200-100*i, "300000000", true)          // and 300 behind each
		if i < 5 {
			order(maker2, "limitSell", 226400+100*i, "200000000000000000", true) // 0.2 ETH behind 2264..2268
		}
	}
	// Beyond the reach (2263 +3% / -3%): a big buy stops at the rail in front of 2400, and 21
	// bids within reach make a big sell stop at maxMatches (20) instead.
	order(maker, "limitSell", 240000, "1000000000000000000", true)
	order(maker, "limitBuy", 215000, "2000000000", true)

	var ts hexutil.Uint64
	var blk struct {
		Timestamp hexutil.Uint64 `json:"timestamp"`
	}
	c.call(&blk, "eth_getBlockByNumber", "latest", false)
	ts = blk.Timestamp
	c.mustSend(lp, positionManager, "mint", mintParams{Pool: bandPool, Bands: []uint8{0, 1, 2},
		BaseAmounts:  []*big.Int{e("1000000000000000000"), e("2000000000000000000"), e("3000000000000000000")},
		QuoteAmounts: []*big.Int{e("1000000000"), e("4000000000"), e("6000000000")},
		MinShares:    []*big.Int{big.NewInt(0), big.NewInt(0), big.NewInt(0)},
		Recipient:    lp, Deadline: new(big.Int).SetUint64(uint64(ts) + 3600)})

	c.call(nil, "evm_increaseTime", hexutil.Uint64(420))
	c.call(nil, "evm_mine")
}

type forkSize struct {
	idxIn  int
	amount string
}

// TestForkExecution checks the simulator against the engine itself on an anvil fork of Base
// mainnet (set RATE_OB_FORK to its URL, e.g. `anvil --fork-url <base rpc> --fork-block-number
// 53485404`). It seeds the book and the band pool (seedForkBook), then for each size quotes,
// sends limitSell / limitBuy exactly as the executor would (price = SwapInfo.LimitPrice, n =
// SwapInfo.N, amount = the used input, isMaker false), and requires the received amount to equal
// the quote to the wei and the spent amount to equal the used input. A second swap, quoted from
// the UpdateBalance'd simulator, goes in the SAME block from a second account, which is the state
// UpdateBalance describes (a later swap in the same route).
func TestForkExecution(t *testing.T) {
	url := os.Getenv("RATE_OB_FORK")
	if url == "" || os.Getenv("CI") != "" {
		t.Skip("set RATE_OB_FORK to an anvil fork of a chain with Rate deployed")
	}
	client, err := rpc.Dial(url)
	require.NoError(t, err)
	chain := forkChain{t: t, rpc: client}
	takers := [2]common.Address{common.HexToAddress("0x00000000000000000000000000000000000beef1"),
		common.HexToAddress("0x00000000000000000000000000000000000beef5")}
	engine := envOr("RATE_OB_ENGINE", "0xc0316fa6d3926d40c77e85d291e1e2456ba1df84") // Base mainnet
	cfg := &Config{DexID: DexType, ChainID: 8453, Engine: engine, Executor: takers[0].Hex()}
	eth := ethrpc.New(url).SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))
	ctx := context.Background()
	chain.call(nil, "anvil_setAutomine", true)

	pools, _, err := NewPoolListUpdater(cfg, eth).GetNewPools(ctx, nil)
	require.NoError(t, err)
	tracker := NewPoolTracker(cfg, eth)

	sizes := []forkSize{
		{1, "100000000"}, {1, "2500000000"}, {1, "12000000000"}, {1, "20000000000"}, {1, "40000000000"},
		{0, "50000000000000000"}, {0, "2000000000000000000"}, {0, "6000000000000000000"},
		{0, "9000000000000000000"}, {0, "20000000000000000000"},
	}
	if only := os.Getenv("RATE_OB_SIZE"); only != "" {
		sizes = []forkSize{{int(only[0] - '0'), only[2:]}}
	}

	checked, withPool := 0, 0
	for _, p := range pools {
		if only := os.Getenv("RATE_OB_PAIR"); only != "" && !strings.EqualFold(only, p.Address) {
			continue
		}
		base, quote := common.HexToAddress(p.Tokens[0].Address), common.HexToAddress(p.Tokens[1].Address)
		for _, taker := range takers {
			chain.account(taker, []common.Address{base, quote}, common.HexToAddress(engine))
		}
		if os.Getenv("RATE_OB_NO_SEED") == "" {
			var bandPool common.Address
			var out hexutil.Bytes
			data, _ := rateABI.Pack("getPool")
			chain.call(&out, "eth_call", map[string]any{"to": p.Address, "data": hexutil.Bytes(data)}, "latest")
			bandPool = common.BytesToAddress(out)
			require.NotEqual(t, common.Address{}, bandPool, "the pair has no band pool")
			seedForkBook(t, chain, common.HexToAddress(engine),
				common.HexToAddress(envOr("RATE_OB_POSITION_MANAGER", "0x438d64536e5daE5357a2CA1522A60246fc43f6F3")),
				bandPool, base, quote)
		}
		for _, sz := range sizes {
			var snap hexutil.Big
			chain.call(&snap, "evm_snapshot")

			state, err := tracker.GetNewPoolState(ctx, p, pool.GetNewPoolStateParams{})
			require.NoError(t, err)
			sim, err := NewPoolSimulator(pool.FactoryParams{EntityPool: state})
			require.NoError(t, err)
			if os.Getenv("RATE_OB_NO_SEED") == "" {
				require.NotNil(t, sim.BandPool, "the seeded pool's anchor should be quotable")
			}

			n, pooled := chain.swapPairAndCompare(t, sim, state, sz.idxIn, sz.amount, engine, takers)
			checked += n
			withPool += pooled

			var ok bool
			chain.call(&ok, "evm_revert", &snap)
			require.True(t, ok)
		}
	}
	require.NotZero(t, checked, "no case filled anything")
	t.Logf("%d swaps matched the simulator to the wei, %d of them with a pool leg", checked, withPool)
}

type forkSwap struct {
	res      *pool.CalcAmountOutResult
	used     *big.Int
	hash     common.Hash
	in       common.Address
	out      func(common.Address) *big.Int
	native   bool
	inBefore *big.Int
	outStart *big.Int
}

// swapPairAndCompare quotes and sends a swap from takers[0], then quotes the same swap again from
// the UpdateBalance'd simulator and sends it from takers[1] in the same block, and compares both
// with what the chain did. It returns how many swaps it compared and how many had a pool leg.
func (c forkChain) swapPairAndCompare(t *testing.T, sim *PoolSimulator, p entity.Pool, idxIn int, amount string,
	engine string, takers [2]common.Address) (int, int) {
	t.Helper()
	c.call(nil, "anvil_setAutomine", false)
	defer c.call(nil, "anvil_setAutomine", true)

	var swaps []*forkSwap
	for i, taker := range takers {
		s := c.quoteAndSubmit(t, sim, p, idxIn, amount, engine, taker)
		if s == nil {
			if i == 0 {
				return 0, 0
			}
			break
		}
		swaps = append(swaps, s)
	}
	c.call(nil, "evm_mine")

	pooled := 0
	var block uint64
	for i, s := range swaps {
		si := s.res.SwapInfo.(SwapInfo)
		gas, fee, ok, blk := c.receipt(s.hash)
		require.True(t, ok, "swap %d reverted", i)
		if i == 0 {
			block = blk
		}
		require.Equal(t, block, blk, "both swaps must land in one block")
		spent := new(big.Int).Sub(s.inBefore, c.balance(s.in, takers[i]))
		got := new(big.Int).Sub(s.out(takers[i]), s.outStart)
		if s.native {
			got.Add(got, fee)
		}
		poolIn := e(orDefault(si.PoolAmountIn, "0"))
		if poolIn.Sign() > 0 {
			pooled++
		}
		match := got.Cmp(s.res.TokenAmountOut.Amount) == 0 && spent.Cmp(s.used) == 0
		fmt.Printf("RESULT swap=%d in=%d amount=%s book=%s pool=%s bands=%d matches=%d limit=%s quoted=%s "+
			"actual=%s spent=%s native=%v gasQuoted=%d gasUsed=%d match=%v\n",
			i, idxIn, amount, new(big.Int).Sub(s.used, poolIn), poolIn, len(si.Fills), si.Matches, si.LimitPrice,
			s.res.TokenAmountOut.Amount, got, spent, s.native, s.res.Gas, gas, match)
		require.Equal(t, s.res.TokenAmountOut.Amount.String(), got.String(), "amount out, swap %d", i)
		require.Equal(t, s.used.String(), spent.String(), "amount in, swap %d", i)
		// The gas model is fitted without the 21k intrinsic cost and must cover every run.
		require.GreaterOrEqual(t, s.res.Gas+21000, int64(gas), "gas, swap %d", i)
	}
	return len(swaps), pooled
}

func orDefault(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

// quoteAndSubmit quotes one swap, submits it from taker exactly as the executor would, and
// applies UpdateBalance; nil when the simulator declines.
func (c forkChain) quoteAndSubmit(t *testing.T, sim *PoolSimulator, p entity.Pool, idxIn int, amount string,
	engine string, taker common.Address) *forkSwap {
	t.Helper()
	tokenIn, tokenOut := p.Tokens[idxIn].Address, p.Tokens[1-idxIn].Address
	in := e(amount)
	res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: in}, TokenOut: tokenOut})
	if err != nil {
		fmt.Printf("RESULT in=%d amount=%s sim=%v\n", idxIn, amount, err)
		return nil
	}
	si := res.SwapInfo.(SwapInfo)
	used := new(big.Int).Sub(in, res.RemainingTokenAmountIn.Amount)
	method := "limitSell"
	if si.IsBuy {
		method = "limitBuy"
	}
	data, err := forkABI.Pack(method, limitOrderInput{Base: common.HexToAddress(p.Tokens[0].Address),
		Quote: common.HexToAddress(p.Tokens[1].Address), Price: e(si.LimitPrice), Amount: used, N: si.N,
		Recipient: taker})
	require.NoError(t, err)

	// A WETH leg is paid out as the gas coin (SwapReturnNativeOut), so measure that balance
	// instead, adding back what the transaction itself paid for gas.
	s := &forkSwap{res: res, used: used, in: common.HexToAddress(tokenIn),
		native: sim.SwapReturnNativeOut(tokenIn, tokenOut, 0)}
	s.out = func(a common.Address) *big.Int {
		if s.native {
			return c.nativeBalance(a)
		}
		return c.balance(common.HexToAddress(tokenOut), a)
	}
	s.inBefore, s.outStart = c.balance(s.in, taker), s.out(taker)
	s.hash = c.submit(taker, common.HexToAddress(engine), data)

	sim.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: used},
		TokenAmountOut: *res.TokenAmountOut, Fee: *res.Fee, SwapInfo: res.SwapInfo})
	return s
}
