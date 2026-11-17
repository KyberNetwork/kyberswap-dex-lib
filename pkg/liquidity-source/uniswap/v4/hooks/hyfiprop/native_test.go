package hyfiprop

import (
	"math/big"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

// nativeFixture is an anvil fork of Base at block 58612723 (testdata/native_swaps.json). The pair
// 0x4e68... is ETH/USDC: currency0 is native ETH. The updater pushed a book for it, and a taker then
// swapped through swapExactInDirect. calls holds what Track's multicall read on the fork after the
// push, and ethBalance the contract's balance from eth_getBalance there.
type nativeFixture struct {
	Hook          string            `json:"hook"`
	Pool          string            `json:"pool"`
	Currency1     string            `json:"currency1"`
	Multicall3    string            `json:"multicall3"`
	Block         uint64            `json:"block"`
	Timestamp     uint64            `json:"timestamp"`
	BookTimestamp uint64            `json:"bookTimestamp"`
	EthBalance    string            `json:"ethBalance"`
	Calls         map[string]string `json:"calls"`
	Steps         []struct {
		ZeroForOne bool   `json:"zeroForOne"`
		In         string `json:"in"`
		OK         bool   `json:"ok"`
		Out        string `json:"out"`
		Now        uint64 `json:"now"`
	} `json:"steps"`
}

// A pair with a native currency is tracked, its native inventory being the contract's ETH balance:
// the contract pays ETH out of it, not out of the wrapped native the pool lists. The swaps then
// match the chain: a swap paying out more ETH than the contract holds reverts there, and is refused
// here; the others pay what the chain paid.
func TestTrack_NativeCurrencyPaysFromEthBalance(t *testing.T) {
	raw, err := os.ReadFile("testdata/native_swaps.json")
	require.NoError(t, err)
	var f nativeFixture
	require.NoError(t, json.Unmarshal(raw, &f))

	staticExtra, err := json.Marshal(uniswapv4.StaticExtra{IsNative: [2]bool{true, false},
		HooksAddress: common.HexToAddress(f.Hook)})
	require.NoError(t, err)
	stub := testutil.NewMulticallStub(t, testutil.RecordedCalls(t, f.Calls)).At(f.Block, f.Timestamp)
	hookAddress := common.HexToAddress(f.Hook)
	hook, ok := uniswapv4.GetHook(hookAddress, nil)
	require.True(t, ok)
	hookExtra, err := hook.Track(t.Context(), &uniswapv4.HookParam{
		Cfg:       &uniswapv4.Config{Multicall3Address: f.Multicall3},
		RpcClient: stub.Client(),
		Pool: &entity.Pool{Address: f.Pool, StaticExtra: string(staticExtra), Tokens: []*entity.PoolToken{
			{Address: "0x4200000000000000000000000000000000000006"}, {Address: f.Currency1}}},
		HookAddress: hookAddress,
	})
	require.NoError(t, err)

	var h Hook
	require.NoError(t, json.Unmarshal(hookExtra, &h))
	assert.Equal(t, f.EthBalance, h.Inventory[0].Dec(), "the ETH balance of the contract")

	for i, s := range f.Steps {
		setNow(t, &h, max(s.Now, f.BookTimestamp))
		in := u(s.In)
		res, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: s.ZeroForOne,
			AmountSpecified: in.ToBig()})
		if !s.OK {
			require.ErrorIs(t, err, ErrInsufficientInv, "step %d: the contract cannot pay the ETH", i)
			continue
		}
		require.NoError(t, err, "step %d", i)
		assert.Equal(t, s.Out, new(big.Int).Neg(res.DeltaUnspecified).String(), "step %d out", i)
		h.UpdateBalance(res.SwapInfo)
	}
}
