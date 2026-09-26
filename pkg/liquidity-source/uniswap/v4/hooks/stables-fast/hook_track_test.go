package stablesfast

import (
	"context"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
)

// The live tests next door skip whenever CI is set, so without this file the only exercise of
// Track's RPC path -- the multicall, the uint24 and address decoding, the registration sentinel
// and the ceiling check -- happens on a developer's machine against mainnet. CI could then pass
// with the ABI wiring or the rate-tracking logic broken, which is the one failure that would
// silently mis-price every pool this package is responsible for.
//
// These tests stub the JSON-RPC endpoint instead, so they are deterministic and run everywhere.

const (
	// The hook and a pool id from Robinhood mainnet. Neither is dialled; they only have to be
	// well formed, because the stub answers whatever is asked.
	testHookAddress = "0xc9932584c5154e4F58313a2e5423522E74e540Cc"
	testPoolID      = "0xf71c2e4fd2dee46e714a146f63235b4246e1cef46e40de59eec4dadedef95e61"

	// The live rate on every Stables pool: 5,000 pips, 0.50%.
	testFeePips = 5_000
)

// aggregateReturnType is Multicall3.aggregate's return, (uint256 blockNumber, bytes[]
// returnData). ethrpc's Aggregate packs the two calls into that method, so a stub has to answer
// in its shape rather than with two bare words.
func aggregateReturnType(t *testing.T) abi.Arguments {
	t.Helper()

	uint256Ty, err := abi.NewType("uint256", "", nil)
	require.NoError(t, err)
	bytesSliceTy, err := abi.NewType("bytes[]", "", nil)
	require.NoError(t, err)

	return abi.Arguments{{Type: uint256Ty}, {Type: bytesSliceTy}}
}

// stubRPC answers every eth_call with one encoded Multicall3.aggregate result carrying
// feePipsFor and feeRecipientOf, in the order Track adds them.
func stubRPC(t *testing.T, feePips *big.Int, recipient common.Address) *ethrpc.Client {
	t.Helper()

	// Both hook returns are a single word: uint24 is right-aligned like any integer, and an
	// address is right-aligned in 32 bytes.
	returnData := [][]byte{
		common.BigToHash(feePips).Bytes(),
		common.BytesToHash(recipient.Bytes()).Bytes(),
	}
	encoded, err := aggregateReturnType(t).Pack(big.NewInt(1), returnData)
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID any `json:"id"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0",
			"id":      req.ID,
			"result":  hexutil.Encode(encoded),
		})
	}))
	t.Cleanup(server.Close)

	return ethrpc.New(server.URL)
}

func trackParam(client *ethrpc.Client) *uniswapv4.HookParam {
	return &uniswapv4.HookParam{
		RpcClient:   client,
		HookAddress: common.HexToAddress(testHookAddress),
		Pool:        &entity.Pool{Address: testPoolID},
	}
}

// The ordinary case: a registered pool at the live rate. This is the test that fails if the
// embedded ABI, the method names or the uint24 decoding ever drift from the contract.
func TestTrack_DecodesTheRateAndRegistration(t *testing.T) {
	recipient := common.HexToAddress("0x28569c1716EF81f307d666A1EC08bDAE92AC0373")
	hook := &Hook{Hook: &uniswapv4.BaseHook{}}

	raw, err := hook.Track(context.Background(), trackParam(stubRPC(t, big.NewInt(testFeePips), recipient)))
	require.NoError(t, err)

	var got Hook
	require.NoError(t, json.Unmarshal(raw, &got))
	assert.Equal(t, int64(testFeePips), got.FeePips, "feePipsFor was not decoded")
	assert.True(t, got.Registered, "a non-zero feeRecipientOf means the pool is registered")
	assert.True(t, got.Tracked, "Track must mark the pool tracked or every quote refuses")
}

// A zero recipient is the hook's own sentinel for "this pool is not mine". It is not an error
// and it must not be confused with a registered pool sitting at a zero rate: the unregistered
// pool gets no skim and no oracle write, so BeforeSwap must not charge gas for one.
func TestTrack_UnregisteredPoolIsTrackedAndFree(t *testing.T) {
	hook := &Hook{Hook: &uniswapv4.BaseHook{}}

	raw, err := hook.Track(context.Background(), trackParam(stubRPC(t, big.NewInt(0), common.Address{})))
	require.NoError(t, err)

	var got Hook
	require.NoError(t, json.Unmarshal(raw, &got))
	assert.False(t, got.Registered, "a zero feeRecipientOf must read as unregistered")
	assert.True(t, got.Tracked, "an unregistered pool is still tracked, it just never charges")

	after, err := got.AfterSwap(&uniswapv4.AfterSwapParams{
		BeforeSwapParams: &uniswapv4.BeforeSwapParams{CalcOut: true},
		AmountOut:        big.NewInt(1e18),
	})
	require.NoError(t, err)
	assert.Zero(t, after.HookFee.Sign(), "an unregistered pool must not be charged")

	before, err := got.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: true})
	require.NoError(t, err)
	assert.Zero(t, before.Gas, "an unregistered pool gets no oracle write, so no observation gas")
}

// A rate above the contract's own MAX_FEE_PIPS means the address is not the hook this package
// models. Quoting it anyway would under-deliver by the difference, so Track refuses.
func TestTrack_RejectsARateAboveTheCeiling(t *testing.T) {
	recipient := common.HexToAddress("0x28569c1716EF81f307d666A1EC08bDAE92AC0373")
	above := new(big.Int).Add(maxFeePips, big.NewInt(1))
	hook := &Hook{Hook: &uniswapv4.BaseHook{}}

	_, err := hook.Track(context.Background(), trackParam(stubRPC(t, above, recipient)))
	assert.ErrorIs(t, err, ErrFeeAboveMax)
}

// The ceiling itself is allowed. An off-by-one here would drop a pool the owner had legitimately
// moved to the maximum.
func TestTrack_AcceptsExactlyTheCeiling(t *testing.T) {
	recipient := common.HexToAddress("0x28569c1716EF81f307d666A1EC08bDAE92AC0373")
	hook := &Hook{Hook: &uniswapv4.BaseHook{}}

	raw, err := hook.Track(context.Background(), trackParam(stubRPC(t, maxFeePips, recipient)))
	require.NoError(t, err)

	var got Hook
	require.NoError(t, json.Unmarshal(raw, &got))
	assert.Equal(t, maxFeePips.Int64(), got.FeePips)
}
