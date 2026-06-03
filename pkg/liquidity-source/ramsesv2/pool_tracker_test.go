package ramsesv2

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_PoolTracker_getTickIndexesFromLogs(t *testing.T) {
	tests := []struct {
		name string
		logs []types.Log
		want []int
	}{
		{
			name: "Mint V3",
			logs: []types.Log{{
				Address: common.HexToAddress("0xee02e3a3034e9ef3bd569b140bc9911fcf1ba067"),
				Topics: []common.Hash{
					common.HexToHash("0xd78218c0d304e8893cb3200abe394bbc8d5b7804d9c51f236df9fdcf481d02d3"),
					common.HexToHash("0x000000000000000000000000b3f77c5134d643483253d22e0ca24627ae42ed51"),
					common.HexToHash("0xfffffffffffffffffffffffffffffffffffffffffffffffffffffffffffc57e8"),
					common.HexToHash("0xfffffffffffffffffffffffffffffffffffffffffffffffffffffffffffc57f2"),
				},
				Data: common.FromHex("0x000000000000000000000000b3f77c5134d643483253d22e0ca24627ae42ed51000000000000000000000000000000000000000000000000000000000000247d000000000000000000000000000000000000000000000000006195d80fdba6de0000000000000000000000000000000000000000000000001e709140f362c0ad0000000000000000000000000000000000000000000000000000000000000000"),
			}},
			want: []int{-239640, -239630},
		},
		{
			name: "Burn V3",
			logs: []types.Log{{
				Address: common.HexToAddress("0xee02e3a3034e9ef3bd569b140bc9911fcf1ba067"),
				Topics: []common.Hash{
					common.HexToHash("0x0c396cd989a39f4459b5fa1aed6a9a8dcdbc45908acfd67e028cd568da98982c"),
					common.HexToHash("0x000000000000000000000000b3f77c5134d643483253d22e0ca24627ae42ed51"),
					common.HexToHash("0xfffffffffffffffffffffffffffffffffffffffffffffffffffffffffffc57f2"),
					common.HexToHash("0xfffffffffffffffffffffffffffffffffffffffffffffffffffffffffffc57fc"),
				},
				Data: common.FromHex("0x0000000000000000000000000000000000000000000000000061a2965d1d517e0000000000000000000000000000000000000000000000001e70a533f0846dce0000000000000000000000000000000000000000000000000000000000000000"),
			}},
			want: []int{-239630, -239620},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			poolTracker := PoolTracker{}
			got, err := poolTracker.getTickIndexesFromLogs(tt.logs)
			require.NoError(t, err)
			assert.ElementsMatch(t, got, tt.want)
		})
	}
}
