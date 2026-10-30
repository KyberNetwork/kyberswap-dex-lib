package testutil

import (
	"math/big"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

// EventLog is a log of ev emitted by address at block, removed by a reorg when removed. Its
// arguments are args by name, every other one zero.
func EventLog(t testing.TB, ev abi.Event, address common.Address, block uint64, removed bool,
	args map[string]any) ethtypes.Log {
	t.Helper()
	topics := []common.Hash{ev.ID}
	var data []any
	for _, in := range ev.Inputs {
		v, ok := args[in.Name]
		if !ok {
			zero := reflect.New(in.Type.GetType()).Elem().Interface()
			if _, isInt := zero.(*big.Int); isInt {
				zero = new(big.Int)
			}
			v = zero
		}
		if !in.Indexed {
			data = append(data, v)
			continue
		}
		topic, err := abi.MakeTopics([]any{v})
		require.NoError(t, err)
		topics = append(topics, topic[0][0])
	}
	packed, err := ev.Inputs.NonIndexed().Pack(data...)
	require.NoError(t, err)
	return ethtypes.Log{Address: address, BlockNumber: block, Topics: topics, Data: packed, Removed: removed}
}
