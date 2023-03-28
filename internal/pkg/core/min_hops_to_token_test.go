package core

import (
	"testing"

	"github.com/stretchr/testify/assert"

	poolPkg "github.com/KyberNetwork/router-service/internal/pkg/core/pool"
	"github.com/KyberNetwork/router-service/internal/pkg/core/uni"
	"github.com/KyberNetwork/router-service/internal/pkg/entity"
)

func Test_minHopsToToken(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		poolByAddress  map[string]poolPkg.IPool
		tokenByAddress map[string]entity.Token
		token          string
		minHopsByToken map[string]uint32
	}{
		{
			name: "it should return empty minHopsByToken when token doesn't included in tokenByAddress",
			tokenByAddress: map[string]entity.Token{
				"token2": {Address: "token2"},
				"token3": {Address: "token3"},
				"token5": {Address: "token5"},
				"token6": {Address: "token6"},
			},
			token:          "token1",
			minHopsByToken: map[string]uint32{},
		},
		{
			name: "it should return correct minHopsByToken",
			poolByAddress: map[string]poolPkg.IPool{
				"pool1": &uni.Pool{
					Pool: poolPkg.Pool{
						Info: poolPkg.PoolInfo{
							Tokens: []string{"token1", "token2"},
						},
					},
				},
				"pool2": &uni.Pool{
					Pool: poolPkg.Pool{
						Info: poolPkg.PoolInfo{
							Tokens: []string{"token2", "token3"},
						},
					},
				},
				"pool3": &uni.Pool{
					Pool: poolPkg.Pool{
						Info: poolPkg.PoolInfo{
							Tokens: []string{"token2", "token4"},
						},
					},
				},
				"pool4": &uni.Pool{
					Pool: poolPkg.Pool{
						Info: poolPkg.PoolInfo{
							Tokens: []string{"token3", "token4"},
						},
					},
				},
				"pool5": &uni.Pool{
					Pool: poolPkg.Pool{
						Info: poolPkg.PoolInfo{
							Tokens: []string{"token4", "token5"},
						},
					},
				},
				"pool6": &uni.Pool{
					Pool: poolPkg.Pool{
						Info: poolPkg.PoolInfo{
							Tokens: []string{"token3", "token6"},
						},
					},
				},
			},
			tokenByAddress: map[string]entity.Token{
				"token1": {Address: "token1"},
				"token2": {Address: "token2"},
				"token3": {Address: "token3"},
				"token5": {Address: "token5"},
				"token6": {Address: "token6"},
			},
			token: "token1",
			minHopsByToken: map[string]uint32{
				"token1": 0,
				"token2": 1,
				"token3": 2,
				"token6": 3,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			minHopsByToken := minHopsToToken(tc.poolByAddress, tc.tokenByAddress, tc.token)

			assert.EqualValues(t, tc.minHopsByToken, minHopsByToken)
		})
	}
}
