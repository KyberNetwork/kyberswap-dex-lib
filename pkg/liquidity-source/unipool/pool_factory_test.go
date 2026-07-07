package unipool

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

const (
	testFactoryAddress = "0x1234567890123456789012345678901234567890"
	testDexID          = "unipool"
	testToken0Address  = "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	testToken1Address  = "0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	testPairAddress    = "0xcccccccccccccccccccccccccccccccccccccccc"
	testTotalPairs     = uint64(7)
	testBlockNumber    = uint64(1_234_567)
)

func newTestFactory() *PoolFactory {
	return NewPoolFactory(&Config{
		DexID:          testDexID,
		FactoryAddress: testFactoryAddress,
	})
}

// pairCreatedLog builds a synthetic PairCreated log:
//
//	event PairCreated(address indexed token0, address indexed token1, address pair, uint256 totalPairs)
func pairCreatedLog(t *testing.T, factory, token0, token1, pair string, totalPairs uint64) types.Log {
	t.Helper()
	data := make([]byte, 0, 64)
	data = append(data, leftPad32(common.HexToAddress(pair).Bytes())...)
	data = append(data, leftPad32(new(big.Int).SetUint64(totalPairs).Bytes())...)
	return types.Log{
		Address: common.HexToAddress(factory),
		Topics: []common.Hash{
			uniPoolFactoryABI.Events[factoryEventPairCreated].ID,
			common.BytesToHash(common.HexToAddress(token0).Bytes()),
			common.BytesToHash(common.HexToAddress(token1).Bytes()),
		},
		Data:        data,
		BlockNumber: testBlockNumber,
	}
}

func leftPad32(b []byte) []byte {
	if len(b) >= 32 {
		return b
	}
	out := make([]byte, 32)
	copy(out[32-len(b):], b)
	return out
}

func TestPoolFactory_IsEventSupported(t *testing.T) {
	t.Parallel()
	f := newTestFactory()

	assert.True(t,
		f.IsEventSupported(uniPoolFactoryABI.Events[factoryEventPairCreated].ID),
		"PairCreated event should be supported")

	unknown := common.HexToHash("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef")
	assert.False(t, f.IsEventSupported(unknown), "random hash must not be supported")
}

func TestPoolFactory_DecodePoolCreated_Valid(t *testing.T) {
	t.Parallel()
	f := newTestFactory()
	log := pairCreatedLog(t, testFactoryAddress, testToken0Address, testToken1Address, testPairAddress, testTotalPairs)

	p, err := f.DecodePoolCreated(log)
	require.NoError(t, err)
	require.NotNil(t, p)

	assert.Equal(t, strings.ToLower(testPairAddress), strings.ToLower(p.Address))
	assert.Equal(t, testDexID, p.Exchange)
	assert.Equal(t, DexType, p.Type)
	assert.Equal(t, testBlockNumber, p.BlockNumber)
	assert.Equal(t, entity.PoolReserves{"0", "0"}, p.Reserves)
	require.Len(t, p.Tokens, 2)
	assert.Equal(t, strings.ToLower(testToken0Address), strings.ToLower(p.Tokens[0].Address))
	assert.Equal(t, strings.ToLower(testToken1Address), strings.ToLower(p.Tokens[1].Address))
	assert.True(t, p.Tokens[0].Swappable)
	assert.True(t, p.Tokens[1].Swappable)

	// StaticExtra should round-trip the configured factory address.
	assert.Contains(t, p.StaticExtra, strings.ToLower(testFactoryAddress))
}

func TestPoolFactory_DecodePoolCreated_WrongFactory(t *testing.T) {
	t.Parallel()
	f := newTestFactory()
	otherFactory := "0x0000000000000000000000000000000000000099"
	log := pairCreatedLog(t, otherFactory, testToken0Address, testToken1Address, testPairAddress, testTotalPairs)

	_, err := f.DecodePoolCreated(log)
	require.Error(t, err, "log from a different factory must be rejected")
}

func TestPoolFactory_DecodePoolCreated_WrongEventHash(t *testing.T) {
	t.Parallel()
	f := newTestFactory()
	log := pairCreatedLog(t, testFactoryAddress, testToken0Address, testToken1Address, testPairAddress, testTotalPairs)
	// Replace event topic with a random one.
	log.Topics[0] = common.HexToHash("0xaaaabbbbccccddddeeeeffff0000111122223333444455556666777788889999")

	_, err := f.DecodePoolCreated(log)
	require.Error(t, err)
}

func TestPoolFactory_DecodePoolCreated_TooFewTopics(t *testing.T) {
	t.Parallel()
	f := newTestFactory()
	log := pairCreatedLog(t, testFactoryAddress, testToken0Address, testToken1Address, testPairAddress, testTotalPairs)
	log.Topics = log.Topics[:2] // drop the second indexed token

	_, err := f.DecodePoolCreated(log)
	require.Error(t, err)
}

func TestPoolFactory_DecodePoolCreated_ZeroFactoryAddress(t *testing.T) {
	t.Parallel()
	f := newTestFactory()
	log := pairCreatedLog(t, testFactoryAddress, testToken0Address, testToken1Address, testPairAddress, testTotalPairs)
	log.Address = common.Address{} // zero address

	_, err := f.DecodePoolCreated(log)
	require.Error(t, err)
}

func TestPoolFactory_DecodePoolAddressesFromFactoryLog_Valid(t *testing.T) {
	t.Parallel()
	f := newTestFactory()
	log := pairCreatedLog(t, testFactoryAddress, testToken0Address, testToken1Address, testPairAddress, testTotalPairs)

	addrs, err := f.DecodePoolAddressesFromFactoryLog(context.Background(), log)
	require.NoError(t, err)
	require.Len(t, addrs, 1)
	assert.Equal(t, strings.ToLower(testPairAddress), strings.ToLower(addrs[0]))
}

func TestPoolFactory_DecodePoolAddressesFromFactoryLog_Invalid(t *testing.T) {
	t.Parallel()
	f := newTestFactory()
	log := pairCreatedLog(t, "0x0000000000000000000000000000000000000099", testToken0Address, testToken1Address, testPairAddress, testTotalPairs)

	_, err := f.DecodePoolAddressesFromFactoryLog(context.Background(), log)
	require.Error(t, err)
}

func TestPoolFactory_FactoryAddressCaseInsensitive(t *testing.T) {
	t.Parallel()
	// Config has lowercased address; log has checksummed/uppercased.
	checksummed := strings.ToUpper(testFactoryAddress)
	f := NewPoolFactory(&Config{
		DexID:          testDexID,
		FactoryAddress: checksummed, // mixed case
	})
	log := pairCreatedLog(t, testFactoryAddress, testToken0Address, testToken1Address, testPairAddress, 1)

	_, err := f.DecodePoolCreated(log)
	require.NoError(t, err, "factory address comparison must be case-insensitive")
}

// Sanity: the ABI must expose the PairCreated event; otherwise nothing works.
func TestPoolFactory_ABISanity(t *testing.T) {
	t.Parallel()
	ev, ok := uniPoolFactoryABI.Events[factoryEventPairCreated]
	require.True(t, ok, "PairCreated must be in the embedded factory ABI")
	require.Len(t, ev.Inputs, 4, "PairCreated must have 4 inputs (token0, token1, pair, totalPairs)")

	// Document the canonical event signature so the hash binding is auditable.
	sig := fmt.Sprintf("%s(%s,%s,%s,%s)",
		factoryEventPairCreated,
		ev.Inputs[0].Type.String(),
		ev.Inputs[1].Type.String(),
		ev.Inputs[2].Type.String(),
		ev.Inputs[3].Type.String(),
	)
	assert.Equal(t, "PairCreated(address,address,address,uint256)", sig)

	// Topics layout: ID + 2 indexed args (token0, token1).
	indexed := 0
	for _, input := range ev.Inputs {
		if input.Indexed {
			indexed++
		}
	}
	assert.Equal(t, 2, indexed, "exactly 2 args should be indexed")
}
