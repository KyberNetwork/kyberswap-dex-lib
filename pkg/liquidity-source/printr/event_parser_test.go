package printr

import (
	"context"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
)

func TestEventParser_Decode(t *testing.T) {
	t.Parallel()

	printrAddr := common.HexToAddress("0xb77726291b125515d0a7affeea2b04f2ff243172")
	token := common.HexToAddress("0xc37f74311b2C7A3bFb0Ea98a2158131fACB8b6e7")
	poolAddr := strings.ToLower(token.Hex())

	ep := NewPoolFactory(&Config{PrintrAddr: printrAddr.Hex()})

	topicAddr := func(a common.Address) common.Hash {
		return common.BytesToHash(common.LeftPadBytes(a.Bytes(), common.HashLength))
	}

	tokenTradeID := printrABI.Events["TokenTrade"].ID
	tokenGraduatedID := printrABI.Events["TokenGraduated"].ID

	tests := []struct {
		name   string
		logs   []types.Log
		expect map[string]int
	}{
		{
			name: "TokenTrade maps to token pool",
			logs: []types.Log{
				{
					Address: printrAddr,
					Topics:  []common.Hash{tokenTradeID, topicAddr(token)},
				},
			},
			expect: map[string]int{poolAddr: 1},
		},
		{
			name: "TokenGraduated maps to token pool",
			logs: []types.Log{
				{
					Address: printrAddr,
					Topics:  []common.Hash{tokenGraduatedID, topicAddr(token)},
				},
			},
			expect: map[string]int{poolAddr: 1},
		},
		{
			name: "Ignore wrong contract address",
			logs: []types.Log{
				{
					Address: common.HexToAddress("0x0000000000000000000000000000000000000001"),
					Topics:  []common.Hash{tokenTradeID, topicAddr(token)},
				},
			},
			expect: map[string]int{},
		},
		{
			name: "Ignore unsupported event topic",
			logs: []types.Log{
				{
					Address: printrAddr,
					Topics:  []common.Hash{common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111"), topicAddr(token)},
				},
			},
			expect: map[string]int{},
		},
		{
			name: "Ignore missing topics",
			logs: []types.Log{
				{Address: printrAddr, Topics: []common.Hash{}},
				{Address: printrAddr, Topics: []common.Hash{tokenTradeID}},
			},
			expect: map[string]int{},
		},
		{
			name: "Mixed logs only returns relevant pools",
			logs: []types.Log{
				{Address: printrAddr, Topics: []common.Hash{tokenTradeID, topicAddr(token)}},
				{Address: printrAddr, Topics: []common.Hash{tokenGraduatedID, topicAddr(token)}},
				{Address: printrAddr, Topics: []common.Hash{tokenTradeID}}, // invalid
				{Address: common.HexToAddress("0x0000000000000000000000000000000000000001"), Topics: []common.Hash{tokenTradeID, topicAddr(token)}},
			},
			expect: map[string]int{poolAddr: 2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ep.Decode(context.Background(), tt.logs)
			assert.NoError(t, err)

			assert.Equal(t, len(tt.expect), len(got))
			for addr, cnt := range tt.expect {
				assert.Contains(t, got, addr)
				assert.Len(t, got[addr], cnt)
			}
		})
	}
}
