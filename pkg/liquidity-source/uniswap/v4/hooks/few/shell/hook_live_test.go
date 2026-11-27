package fewshell

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// TestLive_QuoteMatchesChain builds the simulator from a pool the router API serves, with hX from Track, all read at one
// block, and compares CalcAmountOut with the hook's own quote(bool,int256,bytes32) eth_call (the V4Quoter running the
// real beforeSwap) at that block. Read-only. Run it with
//
//	FEWSHELL_LIVE_RPC=https://ethereum-rpc.publicnode.com go test ./pkg/liquidity-source/uniswap/v4/hooks/few/shell -run Live -v
//
// FEWSHELL_LIVE_API overrides the router API pools endpoint.
func TestLive_QuoteMatchesChain(t *testing.T) {
	rpcURL := os.Getenv("FEWSHELL_LIVE_RPC")
	if rpcURL == "" {
		t.Skip("set FEWSHELL_LIVE_RPC for the read-only live check")
	}
	api := os.Getenv("FEWSHELL_LIVE_API")
	if api == "" {
		api = "https://router-api.kyberengineering.io/ethereum/api/v1/pools"
	}
	ctx := context.Background()
	client, err := ethclient.Dial(rpcURL)
	require.NoError(t, err)
	defer client.Close()
	head, err := client.BlockNumber(ctx)
	require.NoError(t, err)
	block := new(big.Int).SetUint64(head - 1) // a block every node still serves state for
	rpc := ethrpc.New(rpcURL).SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))

	quoteABI, err := abi.JSON(strings.NewReader(`[{"type":"function","name":"quote","stateMutability":"nonpayable",
		"inputs":[{"name":"zeroForOne","type":"bool"},{"name":"amountSpecified","type":"int256"},{"name":"poolId","type":"bytes32"}],
		"outputs":[{"name":"amountUnspecified","type":"uint256"}]}]`))
	require.NoError(t, err)
	hookAddress := HookAddresses[0]

	cases := []struct {
		poolID string
		// amounts of token0 (zeroForOne) and token1 to swap in
		in0, in1 []string
		// amounts the chain reverts: past the PoolManager's input balance, and past the output wrapper's backing
		revert0, revert1 []string
	}{
		{"0xff2441552bc87bf82d0647f28f51e8a218218ff6a0fd7bee865359276fea3ef2", // WETH/USDT
			[]string{"1000000000000", "10000000000000", "100000000000000", "1000000000000000", "1000000000000000000"},
			[]string{"1000000", "100000000", "1000000000"}, nil, nil},
		{"0x47e76bcd20b63685e7230640057a14f6d86bb7f4cb64abb6eb7eb1010478d382", // USDC/WETH
			[]string{"1000000", "10000000", "100000000", "1000000000", "10000000000"},
			[]string{"1000000000000", "10000000000000", "100000000000000", "1000000000000000", "1000000000000000000"},
			nil, []string{"500000000000000000000"}},
		{"0xc626966e214fd2a9a169048cff86666815f768d9ad524a082123649925ddbefb", // ARB/WETH: a third pool of the hook
			[]string{"1000000000000000000", "100000000000000000000"},
			[]string{"1000000000000000", "100000000000000000"}, nil, nil},
	}
	for _, c := range cases {
		resp, err := http.Get(api + "?ids=" + c.poolID)
		require.NoError(t, err)
		body, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		require.NoError(t, err)
		var got struct {
			Data struct {
				Pools []entity.Pool `json:"pools"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(body, &got))
		require.Len(t, got.Data.Pools, 1, c.poolID)
		p := got.Data.Pools[0]

		// hX at the block, from Track, as the pool tracker would store it.
		param := &uniswapv4.HookParam{Cfg: &uniswapv4.Config{ChainID: valueobject.ChainIDEthereum}, RpcClient: rpc,
			Pool: &p, HookAddress: hookAddress, BlockNumber: block}
		h, ok := uniswapv4.GetHook(hookAddress, param)
		require.True(t, ok)
		require.IsType(t, &Hook{}, h)
		hX, err := h.Track(ctx, param)
		require.NoError(t, err)
		var extra map[string]json.RawMessage
		require.NoError(t, json.Unmarshal([]byte(p.Extra), &extra))
		extra["hX"] = json.RawMessage(hX)
		raw, err := json.Marshal(extra)
		require.NoError(t, err)
		p.Extra = string(raw)

		// pF of the API's pool against slot0 at the block.
		var slot0 [32]byte
		slot := crypto.Keccak256Hash(common.HexToHash(c.poolID).Bytes(), common.BigToHash(big.NewInt(6)).Bytes())
		pmABI, err := abi.JSON(strings.NewReader(`[{"type":"function","name":"extsload","stateMutability":"view",
			"inputs":[{"name":"slot","type":"bytes32"}],"outputs":[{"name":"","type":"bytes32"}]}]`))
		require.NoError(t, err)
		_, err = rpc.NewRequest().SetBlockNumber(block).AddCall(&ethrpc.Call{ABI: pmABI,
			Target: uniswapv4.PoolManagerOf(&p, valueobject.ChainIDEthereum).Hex(), Method: "extsload",
			Params: []any{slot}}, []any{&slot0}).Aggregate()
		require.NoError(t, err)
		onchainProtocolFee := new(big.Int).SetBytes(slot0[:]).Rsh(new(big.Int).SetBytes(slot0[:]), 160+24).Uint64() & 0xffffff
		var pF struct {
			Fees uint64 `json:"pF"`
		}
		require.NoError(t, json.Unmarshal([]byte(p.Extra), &pF))
		assert.EqualValues(t, onchainProtocolFee, pF.Fees&0xffffff, "pF in the API pool is slot0's protocol fee at the block")

		sim, err := uniswapv4.NewPoolSimulator(p, valueobject.ChainIDEthereum)
		require.NoError(t, err)
		tokens := sim.GetTokens()
		t.Logf("pool %s block %s hX %s", c.poolID, block, hX)
		for zeroForOne, amounts := range map[bool][]string{true: c.in0, false: c.in1} {
			tokenIn, tokenOut := tokens[0], tokens[1]
			if !zeroForOne {
				tokenIn, tokenOut = tokens[1], tokens[0]
			}
			for _, amount := range amounts {
				amountIn, _ := new(big.Int).SetString(amount, 10)
				res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: amountIn}, TokenOut: tokenOut})
				require.NoError(t, err)

				data, err := quoteABI.Pack("quote", zeroForOne, new(big.Int).Neg(amountIn), common.HexToHash(c.poolID))
				require.NoError(t, err)
				ret, err := client.CallContract(ctx, ethereum.CallMsg{To: &hookAddress, Data: data}, block)
				require.NoError(t, err)
				out, err := quoteABI.Unpack("quote", ret)
				require.NoError(t, err)
				chainOut := out[0].(*big.Int)

				t.Logf("LIVE %s zeroForOne=%v amountIn=%s sim=%s chain=%s block=%s", c.poolID[:8], zeroForOne, amount,
					res.TokenAmountOut.Amount, chainOut, block)
				assert.Equal(t, chainOut.String(), res.TokenAmountOut.Amount.String(),
					fmt.Sprintf("pool %s zeroForOne=%v amountIn=%s", c.poolID, zeroForOne, amount))

				// CalcAmountIn of that output is the least exact-in amount the chain also gives it for.
				chainOutOf := func(in *big.Int) *big.Int {
					data, err := quoteABI.Pack("quote", zeroForOne, new(big.Int).Neg(in), common.HexToHash(c.poolID))
					require.NoError(t, err)
					ret, err := client.CallContract(ctx, ethereum.CallMsg{To: &hookAddress, Data: data}, block)
					require.NoError(t, err)
					out, err := quoteABI.Unpack("quote", ret)
					require.NoError(t, err)
					return out[0].(*big.Int)
				}
				resIn, err := sim.CalcAmountIn(pool.CalcAmountInParams{
					TokenAmountOut: pool.TokenAmount{Token: tokenOut, Amount: chainOut}, TokenIn: tokenIn})
				require.NoError(t, err)
				inMin := resIn.TokenAmountIn.Amount
				t.Logf("LIVE CALCIN %s zeroForOne=%v amountOut=%s calcIn=%s (exact-in amount was %s)", c.poolID[:8],
					zeroForOne, chainOut, inMin, amount)
				assert.GreaterOrEqual(t, chainOutOf(inMin).Cmp(chainOut), 0, "chain exact-in of CalcAmountIn reaches the output")
				assert.Less(t, chainOutOf(new(big.Int).Sub(inMin, big.NewInt(1))).Cmp(chainOut), 0, "one less does not")
			}
		}

		// Reverts: the same amounts must fail in the simulator.
		hk := h.(*Hook)
		revert0 := append([]string{new(big.Int).Add(hk.Inventory[0].ToBig(), big.NewInt(1)).String()}, c.revert0...)
		revert1 := append([]string{new(big.Int).Add(hk.Inventory[1].ToBig(), big.NewInt(1)).String()}, c.revert1...)
		for zeroForOne, amounts := range map[bool][]string{true: revert0, false: revert1} {
			tokenIn, tokenOut := tokens[0], tokens[1]
			if !zeroForOne {
				tokenIn, tokenOut = tokens[1], tokens[0]
			}
			for _, amount := range amounts {
				amountIn, _ := new(big.Int).SetString(amount, 10)
				_, simErr := sim.CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: amountIn}, TokenOut: tokenOut})
				data, err := quoteABI.Pack("quote", zeroForOne, new(big.Int).Neg(amountIn), common.HexToHash(c.poolID))
				require.NoError(t, err)
				_, chainErr := client.CallContract(ctx, ethereum.CallMsg{To: &hookAddress, Data: data}, block)
				t.Logf("LIVE REVERT %s zeroForOne=%v amountIn=%s sim=%v chain=%v", c.poolID[:8], zeroForOne, amount, simErr, chainErr)
				assert.Error(t, simErr)
				assert.Error(t, chainErr)
			}
		}
	}
}
