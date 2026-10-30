package v2

import (
	"errors"
	"fmt"
	"math/big"
	"slices"
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
