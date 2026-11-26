package v2

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/euler-swap/shared"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/ephemeral"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

// TestLensTooLarge: only running out of gas splits a batch; the lens's other error and a node
// failure do not, since a smaller batch cannot fix them.
func TestLensTooLarge(t *testing.T) {
	assert.True(t, lensTooLarge(&ephemeral.RevertError{Err: &ephemeral.ContractError{Name: "GasExhausted"}}))
	assert.True(t, lensTooLarge(fmt.Errorf("%w: out of gas", ephemeral.ErrNoRevertData)))
	assert.False(t, lensTooLarge(&ephemeral.RevertError{Err: &ephemeral.ContractError{Name: "MismatchedVaultInput"}}))
	assert.False(t, lensTooLarge(errors.New("429 too many requests")))
}

// lensChain is a node whose state lens reads every pool with empty rounds, but reverts with nothing
// on a call that holds bad, as a pool whose read always runs the lens out of gas makes it. It
// counts the lens calls and the multicalls it is sent; it answers no multicall.
type lensChain struct {
	*testutil.NodeStub
	lensCalls, multicalls atomic.Int64
}

func newLensChain(t *testing.T, bad common.Address) *lensChain {
	c := &lensChain{}
	c.NodeStub = testutil.NewNodeStub(t, 1, 100, func(call testutil.NodeCall) ([]byte, error) {
		if call.To != nil {
			c.multicalls.Add(1)
			return nil, errors.New("no multicall here")
		}
		c.lensCalls.Add(1)
		part := lensPools(t, call.Data)
		if slices.ContainsFunc(part, func(p poolInput) bool { return p.Pool == bad }) {
			return nil, testutil.Revert(nil)
		}
		out, err := lens.PackResult(big.NewInt(100), make([]poolReads, len(part)))
		require.NoError(t, err)
		return nil, testutil.Revert(out)
	})
	return c
}

// lensPools is the pools a lens call reads.
func lensPools(t *testing.T, call []byte) []poolInput {
	args, err := lens.UnpackArgs(call)
	require.NoError(t, err)
	return *abi.ConvertType(args[0], new([]poolInput)).(*[]poolInput)
}

// TestLensReadsAroundAPoolItCannotRead: a pool the lens cannot read even alone gets its error, and
// the others are read through the lens all the same.
func TestLensReadsAroundAPoolItCannotRead(t *testing.T) {
	bad := common.HexToAddress("0xbad")
	chain := newLensChain(t, bad)
	tr, err := NewPoolTracker(&shared.Config{UseLens: true}, chain.Client())
	require.NoError(t, err)

	inputs := []poolInput{{Pool: common.HexToAddress("0x1")}, {Pool: bad}, {Pool: common.HexToAddress("0x3")}}
	block, reads, err := tr.readLens(t.Context(), inputs, nil, 0)
	require.NoError(t, err)
	assert.EqualValues(t, 100, block)
	require.Len(t, reads, 3)
	assert.NoError(t, reads[0].err)
	assert.ErrorIs(t, reads[1].err, ephemeral.ErrNoRevertData)
	assert.NoError(t, reads[2].err)
}

// TestBatchWithoutLensReadsPoolsAlone: without UseLens a batch reads each pool the usual way and
// never calls the lens.
func TestBatchWithoutLensReadsPoolsAlone(t *testing.T) {
	chain := newLensChain(t, common.Address{})
	tr, err := NewPoolTracker(&shared.Config{}, chain.Client())
	require.NoError(t, err)

	p := entity.Pool{
		Address:     "0x1",
		Tokens:      []*entity.PoolToken{{Address: "0xa", Decimals: 6}, {Address: "0xb", Decimals: 18}},
		StaticExtra: `{"sv0":"0x10","sv1":"0x11","ea":"0x12","evc":"0x13"}`,
	}
	_, errs := tr.GetNewPoolStates(t.Context(), []entity.Pool{p, p})
	assert.Error(t, errs[0])
	assert.Error(t, errs[1])
	assert.Zero(t, chain.lensCalls.Load(), "the lens was called")
	assert.EqualValues(t, 2, chain.multicalls.Load())
}

// maxInitCode is EIP-3860's limit, which geth's state transition applies to a to-less eth_call
// (core/vm.CheckMaxInitCodeSize) before running it: a plain error, no revert, no data.
const maxInitCode = 49152

// initCodeNode answers a lens call like a Shanghai node: an oversize one fails outright, a smaller
// one reads every pool with empty rounds.
func initCodeNode(t *testing.T, lensCalls, oversize *atomic.Int64) *testutil.NodeStub {
	return testutil.NewNodeStub(t, 1, 100, func(c testutil.NodeCall) ([]byte, error) {
		if c.To != nil {
			return nil, errors.New("no multicall here")
		}
		lensCalls.Add(1)
		if len(c.Data) > maxInitCode {
			oversize.Add(1)
			return nil, fmt.Errorf("max initcode size exceeded: code size %d limit %d", len(c.Data), maxInitCode)
		}
		out, err := lens.PackResult(big.NewInt(100), make([]poolReads, len(lensPools(t, c.Data))))
		require.NoError(t, err)
		return nil, testutil.Revert(out)
	})
}

func manyPools(n int) []entity.Pool {
	pools := make([]entity.Pool, n)
	for i := range pools {
		pools[i] = entity.Pool{
			Address: common.BigToAddress(big.NewInt(int64(0x1000 + i))).Hex(),
			Tokens:  []*entity.PoolToken{{Address: "0x00000000000000000000000000000000000000aa", Decimals: 6}, {Address: "0x00000000000000000000000000000000000000bb", Decimals: 18}},
			StaticExtra: fmt.Sprintf(`{"sv0":"%s","sv1":"%s","ea":"%s","evc":"0x0000000000000000000000000000000000000c0c"}`,
				common.BigToAddress(big.NewInt(int64(0x2000+i))).Hex(), common.BigToAddress(big.NewInt(int64(0x3000+i))).Hex(),
				common.BigToAddress(big.NewInt(int64(0x4000+i))).Hex()),
		}
	}
	return pools
}

// TestLensBatchOverInitCodeLimitIsSplit: GetNewPoolStates hands readLens every pool of a batch in
// one call. Past ~110 pools of two vaults the call data passes EIP-3860's 49152 bytes, which a node
// refuses before running it; the batch is then read in smaller parts, not given up for multicalls.
func TestLensBatchOverInitCodeLimitIsSplit(t *testing.T) {
	var lensCalls, oversize atomic.Int64
	tr, err := NewPoolTracker(&shared.Config{UseLens: true}, initCodeNode(t, &lensCalls, &oversize).Client())
	require.NoError(t, err)

	pools := manyPools(130)
	var inputs []poolInput
	for _, p := range pools {
		se, vaults, err := poolVaults(p)
		require.NoError(t, err)
		inputs = append(inputs, lensInput(p.Address, se, vaults))
	}
	full, err := lens.PackCall(inputs)
	require.NoError(t, err)
	half, err := lens.PackCall(inputs[:50])
	require.NoError(t, err)
	require.Greater(t, len(full), maxInitCode)
	require.Less(t, len(half), maxInitCode)

	_, reads, err := tr.readLens(t.Context(), inputs, nil, 0)
	assert.NoError(t, err, "a batch over the init code limit should be read in smaller parts, not given up")
	assert.Len(t, reads, len(inputs))
}

// TestBatchIssuesNoDecimalsCall: the quote amount is a constant, so a batch reads nothing but the lens before
// it has a result: it makes no multicall, and decimals() is not asked of any vault. The pool's own reads come
// after the lens's empty rounds, and fail as a whole here.
func TestBatchIssuesNoDecimalsCall(t *testing.T) {
	decimalsID := vaultABI.Methods["decimals"].ID
	var (
		mu                  sync.Mutex
		lensCalls, decimals int
		order               []string
	)
	node := testutil.NewNodeStub(t, 1, 100, func(call testutil.NodeCall) ([]byte, error) {
		mu.Lock()
		defer mu.Unlock()
		if call.To != nil {
			_, _, err := testutil.Multicall(*call.To, call.Data, 100, 0,
				func(_ common.Address, data []byte) (bool, []byte) {
					if bytes.HasPrefix(data, decimalsID) {
						decimals++
					}
					return false, nil
				})
			order = append(order, "multicall")
			return nil, errors.Join(err, errors.New("no multicall here"))
		}
		lensCalls++
		order = append(order, "lens")
		out, err := lens.PackResult(big.NewInt(100), make([]poolReads, len(lensPools(t, call.Data))))
		require.NoError(t, err)
		return nil, testutil.Revert(out)
	})
	tr, err := NewPoolTracker(&shared.Config{UseLens: true}, node.Client())
	require.NoError(t, err)

	p := entity.Pool{
		Address:     "0x1",
		Tokens:      []*entity.PoolToken{{Address: "0xa"}, {Address: "0xb"}},
		StaticExtra: `{"sv0":"0x10","sv1":"0x11","bv0":"0x14","ea":"0x12","evc":"0x13"}`,
	}
	tr.GetNewPoolStates(t.Context(), []entity.Pool{p, p})

	mu.Lock()
	defer mu.Unlock()
	assert.Zero(t, decimals, "decimals() was asked of a vault")
	assert.Equal(t, 1, lensCalls, "one lens call for the batch")
	require.NotEmpty(t, order)
	assert.Equal(t, "lens", order[0], "the lens is the first call of the batch")
}

// TestLensBatchDoesNotDependOnVaultReads: whatever the vaults answer, the batch is handed to the lens as it is;
// a vault whose decimals() reverts, or an RPC that cannot answer one, no longer fails the pools of the batch
// before the lens runs.
func TestLensBatchDoesNotDependOnVaultReads(t *testing.T) {
	chain := newLensChain(t, common.Address{})
	tr, err := NewPoolTracker(&shared.Config{UseLens: true}, chain.Client())
	require.NoError(t, err)

	p := entity.Pool{
		Address:     "0x1",
		Tokens:      []*entity.PoolToken{{Address: "0xa"}, {Address: "0xb"}},
		StaticExtra: `{"sv0":"0x10","sv1":"0x11","ea":"0x12","evc":"0x13"}`,
	}
	_, errs := tr.GetNewPoolStates(t.Context(), []entity.Pool{p, p})
	require.Len(t, errs, 2)
	assert.EqualValues(t, 1, chain.lensCalls.Load(), "the lens was called")
	// The lens's empty rounds send each pool to its own read, a multicall each that this node refuses.
	assert.EqualValues(t, 2, chain.multicalls.Load(), "one read per pool, no decimals read for the batch")
}

// TestLensQuoteAmountMatchesTracker: the lens quotes its controller and collateral with a constant of its own,
// where the tracker's replay divides every quote by shared.QuoteAmount, so the two must be the same number. The
// lens's bytecode is built from this source (see util/ephemeral/contracts/sources.json), so the constant in the
// source is the one that runs.
func TestLensQuoteAmountMatchesTracker(t *testing.T) {
	src, err := os.ReadFile("../../../util/ephemeral/contracts/euler-swap/v2/EulerSwapV2Lens.sol")
	require.NoError(t, err)
	m := regexp.MustCompile(`uint256 internal constant QUOTE_AMOUNT = ([0-9e_]+);`).FindSubmatch(src)
	require.NotNil(t, m, "the lens has no QUOTE_AMOUNT constant")
	lit := strings.ReplaceAll(string(m[1]), "_", "")
	mantissa, exp, found := strings.Cut(lit, "e")
	require.True(t, found, "QUOTE_AMOUNT %s is not written as <n>e<k>", lit)
	n, ok := new(big.Int).SetString(mantissa, 10)
	require.True(t, ok)
	k, err := strconv.Atoi(exp)
	require.NoError(t, err)
	got := n.Mul(n, new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(k)), nil))
	assert.Equal(t, shared.QuoteAmount.String(), got.String())
	assert.NotContains(t, string(src), "IEVault.decimals", "the lens reads no decimals")
}
