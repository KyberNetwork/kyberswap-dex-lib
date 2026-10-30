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

// send runs a transaction from an impersonated account and returns its gas used and status.
func (c forkChain) send(from, to common.Address, data []byte) (uint64, bool) {
	var hash common.Hash
	c.call(&hash, "eth_sendTransaction", map[string]any{"from": from, "to": to, "data": hexutil.Bytes(data),
		"gas": hexutil.Uint64(8_000_000)})
	var receipt struct {
		GasUsed hexutil.Uint64 `json:"gasUsed"`
		Status  hexutil.Uint64 `json:"status"`
	}
	for range 50 {
		if err := c.rpc.CallContext(context.Background(), &receipt, "eth_getTransactionReceipt", hash); err == nil &&
			receipt.GasUsed > 0 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	return uint64(receipt.GasUsed), receipt.Status == 1
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

// TestForkExecution checks the simulator against the engine itself on an anvil fork (set
// RATE_OB_FORK to its URL): for each pair and size it quotes, sends limitSell / limitBuy exactly
// as the executor would (price = SwapInfo.LimitPrice, n = SwapInfo.N, amount = the used input,
// isMaker false), and requires the received amount to equal the quote to the wei. A second swap
// after UpdateBalance is checked the same way, so the post-swap state is exercised too.
func TestForkExecution(t *testing.T) {
	url := os.Getenv("RATE_OB_FORK")
	if url == "" || os.Getenv("CI") != "" {
		t.Skip("set RATE_OB_FORK to an anvil fork of a chain with Rate deployed")
	}
	client, err := rpc.Dial(url)
	require.NoError(t, err)
	chain := forkChain{t: t, rpc: client}
	taker := common.HexToAddress("0x00000000000000000000000000000000000beef1")
	engine := envOr("RATE_OB_ENGINE", "0x611eec25268ac33061a88fc38c11466ad881e575")
	cfg := &Config{DexID: DexType, ChainID: 46630, Engine: engine, Executor: taker.Hex()}
	eth := ethrpc.New(url).SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))
	ctx := context.Background()

	chain.call(nil, "anvil_impersonateAccount", taker)
	chain.call(nil, "anvil_setBalance", taker, hexutil.EncodeBig(big.NewInt(1e18)))
	chain.call(nil, "anvil_setAutomine", true)

	pools, _, err := NewPoolListUpdater(cfg, eth).GetNewPools(ctx, nil)
	require.NoError(t, err)
	tracker := NewPoolTracker(cfg, eth)
	funded := map[common.Address]bool{}
	big1e30, _ := new(big.Int).SetString("1000000000000000000000000000000", 10)

	type size struct {
		idxIn  int
		amount string
	}
	sizes := []size{{0, "1000000000000000000"}, {0, "250000000000000000000"}, {0, "5000000000000000000000"},
		{0, "1000000000000000000000000"}, {1, "10000"}, {1, "1000000"}, {1, "150000000"}, {1, "100000000000"}}

	checked := 0
	only, onlyIn := os.Getenv("RATE_OB_PAIR"), os.Getenv("RATE_OB_IN")
	for _, p := range pools {
		if only != "" && !strings.EqualFold(only, p.Address) {
			continue
		}
		base, quote := common.HexToAddress(p.Tokens[0].Address), common.HexToAddress(p.Tokens[1].Address)
		for _, tok := range []common.Address{base, quote} {
			if !funded[tok] {
				chain.fund(tok, taker, big1e30)
				data, _ := forkABI.Pack("approve", common.HexToAddress(engine), new(big.Int).Lsh(big.NewInt(1), 255))
				_, ok := chain.send(taker, tok, data)
				require.True(t, ok)
				funded[tok] = true
			}
		}
		for _, sz := range sizes {
			if onlyIn != "" && onlyIn != fmt.Sprint(sz.idxIn) {
				continue
			}
			var snap hexutil.Big
			chain.call(&snap, "evm_snapshot")

			state, err := tracker.GetNewPoolState(ctx, p, pool.GetNewPoolStateParams{})
			require.NoError(t, err)
			sim, err := NewPoolSimulator(pool.FactoryParams{EntityPool: state})
			require.NoError(t, err)

			first := chain.swapAndCompare(t, sim, state, sz.idxIn, sz.amount, engine, taker)
			if first {
				checked++
				// Same direction again, quoted from the UpdateBalance'd simulator.
				if chain.swapAndCompare(t, sim, state, sz.idxIn, sz.amount, engine, taker) {
					checked++
				}
			}
			var ok bool
			chain.call(&ok, "evm_revert", &snap)
			require.True(t, ok)
		}
	}
	require.NotZero(t, checked, "no case filled anything")
	t.Logf("%d swaps matched the simulator to the wei", checked)
}

// swapAndCompare quotes, executes and compares one swap; false when the simulator declined it.
func (c forkChain) swapAndCompare(t *testing.T, sim *PoolSimulator, p entity.Pool, idxIn int, amount string,
	engine string, taker common.Address) bool {
	t.Helper()
	tokenIn, tokenOut := p.Tokens[idxIn].Address, p.Tokens[1-idxIn].Address
	in, _ := new(big.Int).SetString(amount, 10)
	res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: in}, TokenOut: tokenOut})
	if err != nil {
		fmt.Printf("RESULT pair=%s in=%d amount=%s sim=%v\n", p.Address, idxIn, amount, err)
		return false
	}
	si := res.SwapInfo.(SwapInfo)
	used := new(big.Int).Sub(in, res.RemainingTokenAmountIn.Amount)
	price, _ := new(big.Int).SetString(si.LimitPrice, 10)
	method := "limitSell"
	if si.IsBuy {
		method = "limitBuy"
	}
	data, err := forkABI.Pack(method, limitOrderInput{Base: common.HexToAddress(p.Tokens[0].Address),
		Quote: common.HexToAddress(p.Tokens[1].Address), Price: price, Amount: used, N: si.N, Recipient: taker})
	require.NoError(t, err)

	inAddr, outAddr := common.HexToAddress(tokenIn), common.HexToAddress(tokenOut)
	inBefore, outBefore := c.balance(inAddr, taker), c.balance(outAddr, taker)
	gas, ok := c.send(taker, common.HexToAddress(engine), data)
	require.True(t, ok, "%s reverted", method)
	spent := new(big.Int).Sub(inBefore, c.balance(inAddr, taker))
	got := new(big.Int).Sub(c.balance(outAddr, taker), outBefore)

	fmt.Printf("RESULT pair=%s in=%d amount=%s used=%s limit=%s matches=%d quoted=%s actual=%s spent=%s gasQuoted=%d gasUsed=%d match=%v\n",
		p.Address, idxIn, amount, used, si.LimitPrice, si.Matches, res.TokenAmountOut.Amount, got, spent, res.Gas,
		gas, got.Cmp(res.TokenAmountOut.Amount) == 0 && spent.Cmp(used) == 0)
	require.Equal(t, res.TokenAmountOut.Amount.String(), got.String(), "amount out")
	require.Equal(t, used.String(), spent.String(), "amount in")

	sim.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: used},
		TokenAmountOut: *res.TokenAmountOut, Fee: *res.Fee, SwapInfo: res.SwapInfo})
	return true
}
