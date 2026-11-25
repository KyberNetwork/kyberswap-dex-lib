package efficientprop

import (
	"context"
	"math/big"
	"os"
	"strings"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/titan"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// TestForkBNBFillsTheSampledQuote runs on a local fork of BNB Chain holding a rehearsal's deployment of its build with
// a listing of desk 1 (its desk's last update landed, the node an anvil that impersonates). It lists the front's doors
// from the chain, then for listing 1 stands in for a builder: the desk's quote word as the head holds it, stamped with
// the head's next block, is served by a local /state url as that block's update (the listing's bound price contract
// beside it, unchanged), the tracker refreshes, and the same word is written into the desk for the next block, in
// which the taker fills the ladder's first size through the door. The fill pays the simulator's output to the wei.
// The fork's blocks carry no milliTimestamp, so the head is read at the start of its second and the fill's block is
// mined in the second after, the window's late sample.
//
//	EFFICIENT_PROP_BNB_FORK=<anvil url> EFFICIENT_PROP_TAKER=<taker> \
//	  go test ./pkg/liquidity-source/efficient-prop -run TestForkBNB -v
//
// The taker must hold the input token, or EFFICIENT_PROP_BALANCE_SLOTS name the token's balance mapping (below).
// Listing 1's credit figures here are stand-ins: what is checked is the output.
func TestForkBNBFillsTheSampledQuote(t *testing.T) {
	rpcURL, takerHex := os.Getenv("EFFICIENT_PROP_BNB_FORK"), os.Getenv("EFFICIENT_PROP_TAKER")
	if rpcURL == "" || takerHex == "" {
		t.Skip("EFFICIENT_PROP_BNB_FORK and EFFICIENT_PROP_TAKER are not set")
	}
	ctx := context.Background()
	client := ethrpc.New(rpcURL).SetMulticallContract(multicall3)
	rc := client.GetETHClient().Client()
	taker := common.HexToAddress(takerHex)
	front := common.HexToAddress(liveFront)
	bnb := liveChains[1]
	st := newStateServer(t)
	cfg := &Config{ChainID: valueobject.ChainIDBSC, ApprovedDesks: []string{bnb.desk1.Hex()}, Fronts: []Front{{
		Address: liveFront, DoorFactory: liveFactory, Layout: LayoutBuilder, CellCodehash: bnb.cell.Hex(),
		FillMode: FillModeDoor, Implementations: []string{liveImpl.Hex()},
		ApprovedSources: []string{priceContract(front, 1).Hex()},
		Titan:           &titan.Config{URLs: []string{st.URL}},
		CreditListings: []CreditListing{{Listing: 1, CustodyBaseIn: 300_000, CustodyQuoteIn: 300_000,
			RouteBaseIn: 1_200_000, RouteQuoteIn: 1_200_000}},
	}}}

	u, err := NewPoolsListUpdater(cfg, client)
	require.NoError(t, err)
	pools, _, err := u.GetNewPools(ctx, nil)
	require.NoError(t, err)
	var p entity.Pool
	for _, q := range pools {
		var se StaticExtra
		require.NoError(t, json.Unmarshal([]byte(q.StaticExtra), &se))
		if se.Listing == 1 {
			p = q
		}
	}
	require.NotEmpty(t, p.Address, "listing 1's door is listed from the front")
	for _, tok := range p.Tokens {
		var dec uint8
		_, err := client.NewRequest().SetContext(ctx).
			AddCall(&ethrpc.Call{ABI: abi.Erc20ABI, Target: tok.Address, Method: "decimals"}, []any{&dec}).Call()
		require.NoError(t, err)
		tok.Decimals = dec
	}
	door := common.HexToAddress(p.Address)
	send := func(to common.Address, data []byte) common.Hash {
		var hash common.Hash
		require.NoError(t, rc.CallContext(ctx, &hash, "eth_sendTransaction", map[string]any{"from": taker, "to": to,
			"data": hexutil.Bytes(data), "gas": hexutil.Uint64(3_000_000)}))
		return hash
	}
	// the approvals in a block of their own, a second after the fork's head, so the desk's last word stands on
	head0, err := client.GetETHClient().HeaderByNumber(ctx, nil)
	require.NoError(t, err)
	require.NoError(t, rc.CallContext(ctx, nil, "evm_setNextBlockTimestamp", hexutil.Uint64(head0.Time+1)))
	for _, tok := range p.Tokens {
		data := append(crypto.Keccak256([]byte("approve(address,uint256)"))[:4], common.LeftPadBytes(door[:], 32)...)
		data = append(data, common.LeftPadBytes(new(big.Int).Lsh(big.NewInt(1), 255).Bytes(), 32)...)
		send(common.HexToAddress(tok.Address), data)
	}
	require.NoError(t, rc.CallContext(ctx, nil, "evm_mine"))

	head, err := client.GetETHClient().HeaderByNumber(ctx, nil)
	require.NoError(t, err)
	var out doorViewOut
	_, err = client.NewRequest().SetContext(ctx).AddCall(&ethrpc.Call{ABI: frontABI, Target: liveFront,
		Method: "doorView", Params: []any{big.NewInt(1)}}, []any{&out}).Call()
	require.NoError(t, err)
	v := out.View
	cell := deskContractOf(front, bnb.cell, v.Desk)
	require.Equal(t, bnb.desk1, cell)
	quoteAt, firmAt := pairSlot(quoteWordBase, v.WordIndex), pairSlot(firmWordBase, v.WordIndex)
	storage := func(a common.Address, slot common.Hash) common.Hash {
		b, err := client.GetETHClient().StorageAt(ctx, a, slot, head.Number)
		require.NoError(t, err)
		return common.BytesToHash(b)
	}
	quote, firmWord := storage(cell, quoteAt).Big(), storage(cell, firmAt)
	target := head.Number.Uint64() + 1
	quote.AndNot(quote, big.NewInt(0xffffffff)).Or(quote, new(big.Int).SetUint64(target&0xffffffff))
	diff := map[string]string{quoteAt.Hex(): common.BigToHash(quote).Hex()}
	if firmWord != (common.Hash{}) {
		diff[firmAt.Hex()] = firmWord.Hex()
	}
	entry := map[string]any{cell.Hex(): map[string]any{"stateDiff": diff, "balance": "0x0", "nonce": "0x1"}}
	live := storage(front, liveSourceSlot(1)).Big()
	if id := bits(live, 0, 32); id != 0 {
		pc := priceContract(front, uint32(id))
		entry[pc.Hex()] = map[string]any{"stateDiff": map[string]string{
			common.Hash{}.Hex(): storage(pc, common.Hash{}).Hex()}, "balance": "0x0", "nonce": "0x1"}
	}
	st.mu.Lock()
	st.m = map[string]any{"blockNumber": hexutil.EncodeUint64(target), "slot": target,
		"entry": map[string]any{"stateOverride": entry}}
	st.mu.Unlock()

	tr, err := NewPoolTracker(cfg, client)
	require.NoError(t, err)
	got, err := tr.GetNewPoolState(ctx, p, pool.GetNewPoolStateParams{})
	require.NoError(t, err)
	e := extraOf(t, got)
	t.Logf("listing 1 at %d: credit mode %d, idle %v, validUntilMs %d, ladders %v", got.BlockNumber, e.CreditMode,
		e.Idle, e.ValidUntilMs, e.ladders())
	require.Equal(t, head.Time*1000+450, e.ValidUntilMs, "the head's second plus one 450 ms block")
	ladders := e.ladders()
	dir := -1
	for d := range 2 {
		if len(ladders[d]) > 0 {
			dir = d
			break
		}
	}
	require.GreaterOrEqual(t, dir, 0, "a direction quoted")
	size := ladders[dir][0][0].ToBig()
	// a taker short of the input is funded by a write of its balance where EFFICIENT_PROP_BALANCE_SLOTS names the
	// token's balance mapping (token=slot, comma separated): the mapping's key is keccak256(holder . slot)
	var held *big.Int
	_, err = client.NewRequest().SetContext(ctx).AddCall(&ethrpc.Call{ABI: abi.Erc20ABI,
		Target: p.Tokens[dir].Address, Method: "balanceOf", Params: []any{taker}}, []any{&held}).Call()
	require.NoError(t, err)
	if held.Cmp(size) < 0 {
		for _, kv := range strings.Split(os.Getenv("EFFICIENT_PROP_BALANCE_SLOTS"), ",") {
			tok, slot, ok := strings.Cut(strings.TrimSpace(kv), "=")
			if !ok || common.HexToAddress(tok) != common.HexToAddress(p.Tokens[dir].Address) {
				continue
			}
			key := crypto.Keccak256Hash(common.LeftPadBytes(taker[:], 32), common.HexToHash(slot).Bytes())
			require.NoError(t, rc.CallContext(ctx, nil, "anvil_setStorageAt", common.HexToAddress(tok), key,
				common.BigToHash(new(big.Int).Add(held, new(big.Int).Mul(size, big.NewInt(2))))))
		}
	}
	sim, err := NewPoolSimulator(pool.FactoryParams{EntityPool: got})
	require.NoError(t, err)
	res, err := calc(sim, p.Tokens[dir].Address, p.Tokens[1-dir].Address, size, nil)
	require.NoError(t, err)

	// the builder's block: the desk's update, then the fill, a second after the head's
	require.NoError(t, rc.CallContext(ctx, nil, "anvil_setStorageAt", cell, quoteAt, common.BigToHash(quote)))
	require.NoError(t, rc.CallContext(ctx, nil, "evm_setNextBlockTimestamp", hexutil.Uint64(head.Time+1)))
	tokenIn, tokenOut := common.HexToAddress(p.Tokens[dir].Address), common.HexToAddress(p.Tokens[1-dir].Address)
	word := func(x []byte) []byte { return common.LeftPadBytes(x, 32) }
	data := crypto.Keccak256([]byte("swapFrom(address,address,uint256,uint256,address,uint256)"))[:4]
	for _, x := range [][]byte{tokenIn[:], tokenOut[:], size.Bytes(), {1}, taker[:],
		new(big.Int).SetUint64(head.Time + 120).Bytes()} {
		data = append(data, word(x)...)
	}
	balance := func(n *big.Int) *big.Int {
		var b *big.Int
		_, err := client.NewRequest().SetContext(ctx).SetBlockNumber(n).AddCall(&ethrpc.Call{ABI: abi.Erc20ABI,
			Target: tokenOut.Hex(), Method: "balanceOf", Params: []any{taker}}, []any{&b}).Call()
		require.NoError(t, err)
		return b
	}
	hash := send(door, data)
	require.NoError(t, rc.CallContext(ctx, nil, "evm_mine"))
	r, err := client.GetETHClient().TransactionReceipt(ctx, hash)
	require.NoError(t, err)
	require.Equal(t, target, r.BlockNumber.Uint64(), "the fill landed in the target block")
	require.Equal(t, types.ReceiptStatusSuccessful, r.Status, "the fill reverted")
	paid := new(big.Int).Sub(balance(r.BlockNumber), balance(head.Number))
	t.Logf("direction %d, %s in: the fill paid %s, the simulator %s; gas %d (the receipt's %d)", dir, size, paid,
		res.TokenAmountOut.Amount, res.Gas, r.GasUsed)
	require.Equal(t, res.TokenAmountOut.Amount, paid, "the fill pays the simulator's output")
}
