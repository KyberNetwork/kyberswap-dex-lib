package oneinchaqua

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/test"
)

const shipABIJSON = `[{"type":"function","name":"ship","stateMutability":"nonpayable","inputs":[{"name":"app","type":"address"},{"name":"strategy","type":"bytes"},{"name":"tokens","type":"address[]"},{"name":"amounts","type":"uint256[]"}],"outputs":[{"name":"strategyHash","type":"bytes32"}]}]`

// oracleFile is the recorded router answer for every shippable cell of the matrix, in cell order: "o:<amountOut>"
// when the router quotes (amountIn equals the amount), "r:<selector>" when it reverts with a custom error,
// "p:<code>" for a Panic, "-" when the cell cannot be shipped.
type oracleFile struct {
	Block   uint64   `json:"block"`
	Digest  string   `json:"digest"`
	Results []string `json:"results"`
}

func (m *matrix) digest() string {
	h := sha256.New()
	for _, s := range m.Strategies {
		_, _ = fmt.Fprintf(h, "S|%s|%s|%x|%v|%s|%s;", s.Group, s.Name, s.Program, s.Launch, s.Raw[0].Dec(), s.Raw[1].Dec())
	}
	for _, c := range m.Cells {
		_, _ = fmt.Fprintf(h, "C|%s|%d|%d|%s;", c.Id, c.Strategy, c.In, c.Amount.Dec())
	}
	return hex.EncodeToString(h.Sum(nil))[:16]
}

func loadOracle(t testing.TB) *oracleFile {
	t.Helper()
	b, err := os.ReadFile("testdata/matrix_oracle.json")
	require.NoError(t, err)
	var f oracleFile
	require.NoError(t, json.Unmarshal(b, &f))
	return &f
}

// forkClient connects to an anvil fork of Ethereum mainnet (ONEINCH_AQUA_ANVIL_URL).
func forkClient(t *testing.T) (*rpc.Client, context.Context) {
	t.Helper()
	url := os.Getenv("ONEINCH_AQUA_ANVIL_URL")
	if url == "" {
		t.Skip("set ONEINCH_AQUA_ANVIL_URL to an anvil fork of Ethereum")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	t.Cleanup(cancel)
	cl, err := rpc.DialContext(ctx, url)
	require.NoError(t, err)
	t.Cleanup(cl.Close)
	return cl, ctx
}

// shipOnFork ships strategy i of the matrix from its impersonated maker on the fork.
func (m *matrix) shipOnFork(t *testing.T, ctx context.Context, cl *rpc.Client, shipABI abi.ABI, i int) {
	t.Helper()
	s := m.Strategies[i]
	maker := m.maker(i)
	if i == 0 {
		var ignored any
		require.NoError(t, cl.CallContext(ctx, &ignored, "anvil_setBalance", maker, "0x56bc75e2d63100000"))
		require.NoError(t, cl.CallContext(ctx, &ignored, "anvil_impersonateAccount", maker))
	}
	data, err := shipABI.Pack("ship", m.app(i), m.order(i).Encode(), []common.Address{matrixToken0, matrixToken1}, []*big.Int{s.Raw[0].ToBig(), s.Raw[1].ToBig()})
	require.NoError(t, err)
	var txHash common.Hash
	require.NoError(t, cl.CallContext(ctx, &txHash, "eth_sendTransaction", map[string]any{"from": maker, "to": testAquaHex, "data": hexutil.Bytes(data), "gas": "0x7a1200"}), s.Name)
	require.Equal(t, "0x1", waitReceiptStatus(t, ctx, cl, txHash), "ship of %s", s.Name)
}

// revertSelector extracts "r:<selector>" or "p:<code>" from the error of an eth_call.
func revertResult(err error) string {
	if de, ok := err.(rpc.DataError); ok {
		if s, ok := de.ErrorData().(string); ok {
			b, derr := hexutil.Decode(s)
			if derr == nil && len(b) >= 4 {
				if hexutil.Encode(b[:4]) == "0x4e487b71" && len(b) >= 36 {
					return "p:" + new(big.Int).SetBytes(b[4:36]).String()
				}
				return "r:" + hexutil.Encode(b[:4])
			}
		}
	}
	return "r:" + strings.TrimSpace(err.Error())
}

var matrixTakerTraits = hexutil.MustDecode(parityTakerTraits)

func (m *matrix) quoteElem(i int, c matrixCell) (rpc.BatchElem, *hexutil.Bytes) {
	tokens := [2]common.Address{matrixToken0, matrixToken1}
	o := m.order(c.Strategy)
	data, err := routerABI.Pack("quote", struct {
		Maker  common.Address
		Traits *big.Int
		Data   []byte
	}{o.Maker, o.Traits.ToBig(), o.Data}, tokens[c.In], tokens[1-c.In], c.Amount.ToBig(), matrixTakerTraits)
	if err != nil {
		panic(err)
	}
	var out hexutil.Bytes
	return rpc.BatchElem{Method: "eth_call", Args: []any{map[string]any{
		"from": "0x00000000000000000000000000000000000dEaDb", "to": m.app(c.Strategy), "data": hexutil.Bytes(data)}, "latest"}, Result: &out}, &out
}

// TestMatrixOracle regenerates testdata/matrix_oracle.json: it ships every shippable strategy of the matrix
// on an anvil fork of Ethereum from an impersonated maker, on the current or the launch router as the
// strategy says, and records the router quote of each cell.
//
// Start the fork with: anvil --fork-url <archive node> --fork-block-number <block> --port 18545
// and set ONEINCH_AQUA_ANVIL_URL=http://127.0.0.1:18545 and ONEINCH_AQUA_ORACLE_WRITE=1 to write the file.
func TestMatrixOracle(t *testing.T) {
	test.SkipCI(t)
	cl, ctx := forkClient(t)
	shipABI, err := abi.JSON(strings.NewReader(shipABIJSON))
	require.NoError(t, err)
	m := buildMatrix()

	var block hexutil.Uint64
	require.NoError(t, cl.CallContext(ctx, &block, "eth_blockNumber"))
	shippable := map[int]bool{}
	for i, s := range m.Strategies {
		if s.Unshippable {
			continue
		}
		m.shipOnFork(t, ctx, cl, shipABI, i)
		shippable[i] = true
	}

	results := make([]string, len(m.Cells))
	const batch = 100
	for lo := 0; lo < len(m.Cells); lo += batch {
		hi := min(lo+batch, len(m.Cells))
		elems := make([]rpc.BatchElem, 0, hi-lo)
		outs := make([]*hexutil.Bytes, 0, hi-lo)
		idx := make([]int, 0, hi-lo)
		for i := lo; i < hi; i++ {
			if !shippable[m.Cells[i].Strategy] {
				results[i] = "-"
				continue
			}
			e, out := m.quoteElem(i, m.Cells[i])
			elems, outs, idx = append(elems, e), append(outs, out), append(idx, i)
		}
		if len(elems) == 0 {
			continue
		}
		require.NoError(t, cl.BatchCallContext(ctx, elems))
		for k, e := range elems {
			i := idx[k]
			if e.Error != nil {
				results[i] = revertResult(e.Error)
				continue
			}
			res := *outs[k]
			require.Len(t, res, 96, m.Cells[i].Id)
			gotIn, gotOut := new(uint256.Int).SetBytes32(res[:32]), new(uint256.Int).SetBytes32(res[32:64])
			require.True(t, gotIn.Eq(m.Cells[i].Amount), "%s: amountIn %s", m.Cells[i].Id, gotIn.Dec())
			results[i] = "o:" + gotOut.Dec()
		}
	}
	f := oracleFile{Block: uint64(block), Digest: m.digest(), Results: results}
	b, err := json.Marshal(f)
	require.NoError(t, err)
	t.Logf("matrix: %d strategies, %d cells, oracle %d bytes at block %d", len(m.Strategies), len(m.Cells), len(b), uint64(block))
	if os.Getenv("ONEINCH_AQUA_ORACLE_WRITE") == "1" {
		require.NoError(t, os.WriteFile("testdata/matrix_oracle.json", b, 0o644))
	}
}

// waitReceiptStatus polls for the receipt of a mined transaction and returns its status.
func waitReceiptStatus(t testing.TB, ctx context.Context, cl *rpc.Client, hash common.Hash) string {
	t.Helper()
	for range 200 {
		var receipt *struct {
			Status string `json:"status"`
		}
		require.NoError(t, cl.CallContext(ctx, &receipt, "eth_getTransactionReceipt", hash))
		if receipt != nil {
			return receipt.Status
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("no receipt for %s", hash)
	return ""
}

// TestMatrixExactOutOracle settles the exact-out vectors on an anvil fork: the router `quote` of the
// simulator's amountIn pays at least the target, and the quote of one wei less pays less or reverts with a
// zero output. It needs an archive node, like TestMatrixOracle (same fork and
// variables): ONEINCH_AQUA_ANVIL_URL.
func TestMatrixExactOutOracle(t *testing.T) {
	test.SkipCI(t)
	cl, ctx := forkClient(t)
	shipABI, err := abi.JSON(strings.NewReader(shipABIJSON))
	require.NoError(t, err)
	m := buildMatrix()
	vectors := m.exactOutVectors(t)
	shipped := map[int]bool{}
	for _, v := range vectors {
		if !shipped[v.Strategy] {
			m.shipOnFork(t, ctx, cl, shipABI, v.Strategy)
			shipped[v.Strategy] = true
		}
	}
	quote := func(v exactOutVector, amount *big.Int) (out *uint256.Int, reverted string) {
		cell := matrixCell{Strategy: v.Strategy, In: v.In, Amount: uint256.MustFromBig(amount)}
		elem, res := m.quoteElem(0, cell)
		batch := []rpc.BatchElem{elem} // the batch holds a copy: the error is read from it, not from elem
		require.NoError(t, cl.BatchCallContext(ctx, batch))
		if batch[0].Error != nil {
			return nil, revertResult(batch[0].Error)
		}
		b := *res
		require.Len(t, b, 96, "%s: quote of amountIn %s on strategy %d returned %x", v.Id, amount, v.Strategy, b)
		return new(uint256.Int).SetBytes32(b[32:64]), ""
	}
	var checked int
	for _, v := range vectors {
		target, _ := new(big.Int).SetString(v.Target, 10)
		a, _ := new(big.Int).SetString(v.AmountIn, 10)
		out, rev := quote(v, a)
		require.Empty(t, rev, "%s: the router reverts at the amountIn %s", v.Id, v.AmountIn)
		require.GreaterOrEqual(t, out.ToBig().Cmp(target), 0, "%s: the router pays %s for %s, less than %s", v.Id, out.Dec(), v.AmountIn, v.Target)
		if a.Cmp(big.NewInt(1)) == 0 {
			continue
		}
		less, rev := quote(v, new(big.Int).Sub(a, big.NewInt(1)))
		if rev != "" {
			require.Equal(t, "zero_out", routerErrors[strings.TrimPrefix(rev, "r:")], "%s: amountIn-1 reverts with %s", v.Id, rev)
		} else {
			require.Negative(t, less.ToBig().Cmp(target), "%s: the router pays %s for amountIn-1", v.Id, less.Dec())
		}
		checked++
	}
	t.Logf("router replay of %d exact-out vectors, %d with an amountIn-1 check", len(vectors), checked)
}

// settlementResult is what the router did with a settled swap.
type settlementResult struct {
	Id        string    `json:"id"`
	Reverted  bool      `json:"reverted"`
	AmountOut string    `json:"amountOut"`
	FeePaid   string    `json:"feePaid"`
	RawAfter  [2]string `json:"rawAfter"`
}

type settlementFile struct {
	Block   uint64             `json:"block"`
	Results []settlementResult `json:"results"`
}

var (
	settlementMaker = common.HexToAddress("0x00000000000000000000000000000000B0B00001")
	settlementTaker = common.HexToAddress("0x00000000000000000000000000000000B0B00002")
	// takerTraits: exact in, transferFrom plus Aqua push, a 20-byte recipient, no threshold or deadline.
	settlementTraitsPrefix = "0x" + strings.Repeat("0014", 9) + "0000" + "0041"
)

const approveABIJSON = `[{"type":"function","name":"approve","stateMutability":"nonpayable","inputs":[{"name":"spender","type":"address"},{"name":"amount","type":"uint256"}],"outputs":[{"name":"","type":"bool"}]}]`

const swapABIJSON = `[{"type":"function","name":"swap","stateMutability":"nonpayable","inputs":[{"name":"order","type":"tuple","components":[{"name":"maker","type":"address"},{"name":"traits","type":"uint256"},{"name":"data","type":"bytes"}]},{"name":"tokenIn","type":"address"},{"name":"tokenOut","type":"address"},{"name":"amount","type":"uint256"},{"name":"takerTraitsAndData","type":"bytes"}],"outputs":[{"name":"amountIn","type":"uint256"},{"name":"amountOut","type":"uint256"},{"name":"orderHash","type":"bytes32"}]}]`

func settlementOrder(i int, c settlementCase) *Order {
	traits := aquaTraits()
	traits[3] |= uint64(0x10000+i) << 32
	return &Order{Maker: settlementMaker, Traits: traits, Data: c.Program}
}

type fork struct {
	t   *testing.T
	ctx context.Context
	cl  *rpc.Client
}

func (f *fork) call(result any, method string, args ...any) {
	f.t.Helper()
	require.NoError(f.t, f.cl.CallContext(f.ctx, result, method, args...), method)
}

func (f *fork) balance(token, holder common.Address) *big.Int {
	f.t.Helper()
	data, err := erc20ABI.Pack("balanceOf", holder)
	require.NoError(f.t, err)
	var out hexutil.Bytes
	f.call(&out, "eth_call", map[string]any{"to": token, "data": hexutil.Bytes(data)}, "latest")
	return new(big.Int).SetBytes(out)
}

// setBalance writes an ERC-20 balance by trying the mapping slots the tokens use.
func (f *fork) setBalance(token, holder common.Address, value *uint256.Int) {
	f.t.Helper()
	for _, slot := range []int64{0, 1, 2, 3} {
		key := crypto.Keccak256Hash(common.LeftPadBytes(holder.Bytes(), 32), common.LeftPadBytes(big.NewInt(slot).Bytes(), 32))
		var ignored any
		f.call(&ignored, "anvil_setStorageAt", token, key, common.BigToHash(value.ToBig()))
		if f.balance(token, holder).Cmp(value.ToBig()) == 0 {
			return
		}
	}
	f.t.Fatalf("no balance slot found for %s", token)
}

// send mines a transaction from an impersonated account and reports whether it succeeded.
func (f *fork) send(from, to common.Address, data []byte) bool {
	f.t.Helper()
	var hash common.Hash
	f.call(&hash, "eth_sendTransaction", map[string]any{"from": from, "to": to, "data": hexutil.Bytes(data), "gas": "0x2dc6c0"})
	return waitReceiptStatus(f.t, f.ctx, f.cl, hash) == "0x1"
}

func (f *fork) rawBalance(app common.Address, hash common.Hash, token common.Address) string {
	f.t.Helper()
	data, err := aquaABI.Pack("rawBalances", settlementMaker, app, [32]byte(hash), token)
	require.NoError(f.t, err)
	var out hexutil.Bytes
	f.call(&out, "eth_call", map[string]any{"to": testAqua, "data": hexutil.Bytes(data)}, "latest")
	return new(big.Int).SetBytes(out[:32]).String()
}

// TestSettlementOracle regenerates testdata/settlement_oracle.json on an anvil fork (see TestMatrixOracle):
// each case is shipped by an impersonated maker with its wallet set up, then settled by `swap` from an
// impersonated taker through the router the case names, and the outcome recorded.
func TestSettlementOracle(t *testing.T) {
	test.SkipCI(t)
	cl, ctx := forkClient(t)
	f := &fork{t: t, ctx: ctx, cl: cl}
	shipABI, err := abi.JSON(strings.NewReader(shipABIJSON))
	require.NoError(t, err)
	swapABI, err := abi.JSON(strings.NewReader(swapABIJSON))
	require.NoError(t, err)
	approveABI, err := abi.JSON(strings.NewReader(approveABIJSON))
	require.NoError(t, err)

	var ignored any
	for _, a := range []common.Address{settlementMaker, settlementTaker} {
		f.call(&ignored, "anvil_setBalance", a, "0x56bc75e2d63100000")
		f.call(&ignored, "anvil_impersonateAccount", a)
	}
	tokens := [2]common.Address{matrixToken0, matrixToken1}
	recipient := testRecipient
	var block hexutil.Uint64
	f.call(&block, "eth_blockNumber")

	var out settlementFile
	out.Block = uint64(block)
	for i, c := range settlementCases() {
		app := testRouter
		if c.Launch {
			app = testLaunch
		}
		order := settlementOrder(i, c)
		hash := order.Hash()
		shipData, err := shipABI.Pack("ship", app, order.Encode(), tokens[:], []*big.Int{c.Raw[0].ToBig(), c.Raw[1].ToBig()})
		require.NoError(t, err)
		require.True(t, f.send(settlementMaker, testAqua, shipData), c.Id)

		for k, token := range tokens {
			f.setBalance(token, settlementMaker, c.Wallet[k])
			approve, err := approveABI.Pack("approve", testAqua, c.Allowance[k].ToBig())
			require.NoError(t, err)
			require.True(t, f.send(settlementMaker, token, approve), c.Id)
		}
		tokenIn, tokenOut := tokens[c.In], tokens[1-c.In]
		f.setBalance(tokenIn, settlementTaker, new(uint256.Int).Mul(c.Amount, uint256.NewInt(2)))
		approve, err := approveABI.Pack("approve", app, new(big.Int).Lsh(big.NewInt(1), 255))
		require.NoError(t, err)
		require.True(t, f.send(settlementTaker, tokenIn, approve), c.Id)

		feeBefore := f.balance(tokenIn, recipient)
		outBefore := f.balance(tokenOut, settlementTaker)
		takerTraits := append(hexutil.MustDecode(settlementTraitsPrefix), settlementTaker.Bytes()...)
		swapData, err := swapABI.Pack("swap", struct {
			Maker  common.Address
			Traits *big.Int
			Data   []byte
		}{order.Maker, order.Traits.ToBig(), order.Data}, tokenIn, tokenOut, c.Amount.ToBig(), takerTraits)
		require.NoError(t, err)
		ok := f.send(settlementTaker, app, swapData)

		res := settlementResult{Id: c.Id, Reverted: !ok}
		res.RawAfter = [2]string{f.rawBalance(app, hash, tokens[0]), f.rawBalance(app, hash, tokens[1])}
		if ok {
			res.AmountOut = new(big.Int).Sub(f.balance(tokenOut, settlementTaker), outBefore).String()
			res.FeePaid = new(big.Int).Sub(f.balance(tokenIn, recipient), feeBefore).String()
		}
		out.Results = append(out.Results, res)
		t.Logf("%s: %+v", c.Id, res)
	}
	b, err := json.MarshalIndent(out, "", " ")
	require.NoError(t, err)
	if os.Getenv("ONEINCH_AQUA_ORACLE_WRITE") == "1" {
		require.NoError(t, os.WriteFile("testdata/settlement_oracle.json", b, 0o644))
	}
}

func loadSettlement(t testing.TB) map[string]settlementResult {
	t.Helper()
	b, err := os.ReadFile("testdata/settlement_oracle.json")
	require.NoError(t, err)
	var f settlementFile
	require.NoError(t, json.Unmarshal(b, &f))
	out := map[string]settlementResult{}
	for _, r := range f.Results {
		out[r.Id] = r
	}
	return out
}

// TestSettlementVectors replays every settled case through the simulator: the amounts and the fee the
// router paid equal the simulator's quote, a revert is a refusal, and UpdateBalance lands on the raw
// balances the registry holds afterwards.
func TestSettlementVectors(t *testing.T) {
	t.Parallel()
	recorded := loadSettlement(t)
	cases := settlementCases()
	require.Len(t, recorded, len(cases))
	tokens := []string{addrHex(matrixToken0), addrHex(matrixToken1)}
	for i, c := range cases {
		t.Run(c.Id, func(t *testing.T) {
			r, ok := recorded[c.Id]
			require.True(t, ok)
			require.Equal(t, c.WantRevert != "", r.Reverted, "the router outcome matches the case's expectation")
			app := testRouter
			if c.Launch {
				app = testLaunch
			}
			e := newEntity(t, poolOpts{program: c.Program, app: app, maker: settlementMaker, raw: c.Raw, wallet: c.Wallet,
				allowance: c.Allowance, tokens: [2]string{tokens[0], tokens[1]}})
			_ = i
			sim, err := NewPoolSimulator(e)
			if err != nil {
				require.True(t, r.Reverted)
				requireReason(t, err, DiscardReason(c.WantRevert), c.Id)
				return
			}
			res, err := quoteOut(sim, c.In, c.Amount, nil)
			if r.Reverted {
				switch c.WantRevert {
				case "fee_shortfall":
					require.ErrorIs(t, err, ErrFeeShortfall)
				case "overflow":
					require.ErrorIs(t, err, ErrOverflow)
				default:
					require.Error(t, err)
				}
				return
			}
			require.NoError(t, err)
			require.Equal(t, r.AmountOut, res.TokenAmountOut.Amount.String(), "amountOut")
			require.Equal(t, r.FeePaid, res.Fee.Amount.String(), "fee collected")
			sim.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: pool.TokenAmount{Token: sim.Info.Tokens[c.In], Amount: c.Amount.ToBig()},
				TokenAmountOut: *res.TokenAmountOut, Fee: *res.Fee, SwapInfo: res.SwapInfo})
			require.Equal(t, r.RawAfter[0], sim.State.Raw[0].Dec(), "raw balance of token0 after the swap")
			require.Equal(t, r.RawAfter[1], sim.State.Raw[1].Dec(), "raw balance of token1 after the swap")
		})
	}
}

// settlementCase is a swap settled on a fork of the chain: the strategy, the maker wallet it draws on and
// the taker's amount. The recorded outcome is the router's, replayed by TestSettlementVectors.
type settlementCase struct {
	Id        string
	Program   []byte
	Launch    bool
	Raw       [2]*uint256.Int
	Wallet    [2]*uint256.Int // maker balance of token0, token1
	Allowance [2]*uint256.Int // maker allowance to Aqua
	In        int
	Amount    *uint256.Int
	// WantRevert is the error the simulator must refuse the case with when the router reverts the swap:
	// "" (settles), "fee_shortfall", "overflow" or the discard reason of the program.
	WantRevert string
}

func settlementCases() []settlementCase {
	rich := [2]*uint256.Int{e18(1_000_000), e18(1_000_000)}
	unl := [2]*uint256.Int{unlimited, unlimited}
	fee1 := prog(iProtocol(10_000_000, testRecipient), iSwap())
	std := [2]*uint256.Int{e18(1000), e18(2000)}
	amount := e18(100)
	mk := func(id string, program []byte, launch bool, raw, wallet, allowance [2]*uint256.Int, amt *uint256.Int) settlementCase {
		c := settlementCase{Id: id, Program: program, Launch: launch, Raw: raw, Wallet: wallet, Allowance: allowance, In: 0, Amount: amt}
		switch {
		case strings.Contains(id, "launch-fee-short"), strings.Contains(id, "launch-stacked"):
			c.WantRevert = "fee_shortfall"
		case strings.Contains(id, "launch-zero-recipient"):
			c.WantRevert = string(ReasonLaunchFeeRecipent)
		case id == "current-zero-recipient-bps1":
			c.WantRevert = string(ReasonFeeRecipient)
		case strings.Contains(id, "overflows"):
			c.WantRevert = "overflow"
		}
		return c
	}
	lowIn := [2]*uint256.Int{u("500000000000000000"), e18(2000)}
	shortWallet := [2]*uint256.Int{u("500000000000000000"), e18(1_000_000)}
	noAllowance := [2]*uint256.Int{new(uint256.Int), unlimited}
	rawOut := new(uint256.Int).Lsh(uint256.NewInt(1), 180)
	huge := u("1000000000000000000000") // 1e21
	zeroRecip := func(bps uint32) []byte { return prog(iProtocol(bps, common.Address{}), iSwap()) }
	stacked := prog(iSalt(1), iProtocol(100_000_000, testRecipient), iProtocol(200_000_000, testRecipient), iFlat(50_000_000), iSwap())
	return []settlementCase{
		mk("current-fee-collected", fee1, false, std, rich, unl, amount),
		mk("current-fee-skipped-strategy-short", fee1, false, lowIn, rich, unl, amount),
		mk("current-fee-skipped-wallet-short", fee1, false, std, shortWallet, unl, amount),
		mk("current-fee-skipped-no-allowance", fee1, false, std, rich, noAllowance, amount),
		mk("current-fee-rounds-to-zero", fee1, false, std, rich, unl, uint256.NewInt(99)),
		mk("current-fee-at-exact-strategy-balance", fee1, false, [2]*uint256.Int{e18(1), e18(2000)}, rich, unl, amount),
		mk("current-zero-recipient-bps0", zeroRecip(0), false, std, rich, unl, amount),
		mk("current-zero-recipient-bps1", zeroRecip(1), false, std, rich, unl, amount),
		mk("current-stacked-first-collected-second-short", stacked, false, [2]*uint256.Int{u("250000000000000000"), e18(2000)}, rich, unl, e18(1)),
		mk("current-stacked-both-collected", stacked, false, [2]*uint256.Int{e18(1), e18(2000)}, rich, unl, e18(1)),
		mk("launch-fee-collected", fee1, true, std, rich, unl, amount),
		mk("launch-fee-short-strategy", fee1, true, lowIn, rich, unl, amount),
		mk("launch-fee-short-wallet", fee1, true, std, shortWallet, unl, amount),
		mk("launch-fee-rounds-to-zero", fee1, true, std, rich, unl, uint256.NewInt(99)),
		mk("launch-zero-recipient-bps0", zeroRecip(0), true, std, rich, unl, amount),
		mk("launch-zero-recipient-bps1", zeroRecip(1), true, std, rich, unl, amount),
		mk("launch-stacked-first-collected-second-short", stacked, true, [2]*uint256.Int{u("250000000000000000"), e18(2000)}, rich, unl, e18(1)),
		mk("current-push-fits-uint248", prog(iSwap()), false, [2]*uint256.Int{new(uint256.Int).Sub(max248, huge), rawOut}, rich, unl, huge),
		mk("current-push-overflows-uint248", prog(iSwap()), false, [2]*uint256.Int{new(uint256.Int).Sub(max248, new(uint256.Int).SubUint64(huge, 1)), rawOut}, rich, unl, huge),
	}
}
