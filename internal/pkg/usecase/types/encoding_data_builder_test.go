package types

import (
	"context"
	"math/big"
	"testing"

	"github.com/KyberNetwork/aggregator-encoding/pkg/types"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

func TestEncodeCollectAmountUseExecutorBalance(t *testing.T) {
	builder := NewEncodingDataBuilder(context.Background(), valueobject.ChainIDEthereum, nil, valueobject.FeatureFlags{IsMergeDuplicateSwapEnabled: true})
	builder.data.TokenIn = "in"

	// Check normal case
	route := [][]types.EncodingSwap{
		{{Pool: "p1", TokenIn: "in", TokenOut: "mid1", SwapAmount: big.NewInt(1000)}},
		{{Pool: "p2", TokenIn: "in", TokenOut: "mid2", SwapAmount: big.NewInt(2000)}},
		{{Pool: "p3", TokenIn: "mid1", TokenOut: "mid2", SwapAmount: big.NewInt(3000)}},
		{{Pool: "p4", TokenIn: "mid1", TokenOut: "mid2", SwapAmount: big.NewInt(4000)}},
		{{Pool: "p5", TokenIn: "mid2", TokenOut: "out", SwapAmount: big.NewInt(5000)}},
	}

	updatedRoute := builder.updateSwapRecipientAndCollectAmount(route, "normal", "")
	assert.Equal(t, big.NewInt(2000), updatedRoute[1][0].CollectAmount)
	assert.Equal(t, big.NewInt(3000), updatedRoute[2][0].CollectAmount)
	assert.Equal(t, bignumber.MAX_UINT_128, updatedRoute[3][0].CollectAmount)
	assert.Equal(t, bignumber.MAX_UINT_128, updatedRoute[4][0].CollectAmount)

	// Should not set collect amount when disable merge duplicate swap feature flags
	builder.featureFlags.IsMergeDuplicateSwapEnabled = false
	updatedRoute = builder.updateSwapRecipientAndCollectAmount(route, "normal", "")
	assert.Equal(t, big.NewInt(2000), updatedRoute[1][0].CollectAmount)
	assert.Equal(t, big.NewInt(3000), updatedRoute[2][0].CollectAmount)
	assert.Equal(t, big.NewInt(4000), updatedRoute[3][0].CollectAmount)
	assert.Equal(t, big.NewInt(5000), updatedRoute[4][0].CollectAmount)

	// Should not set collect amount in case of wrap token
	builder.data.TokenIn = "0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"
	route = [][]types.EncodingSwap{
		{{Pool: "p1", TokenIn: "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", TokenOut: "0xdac17f958d2ee523a2206206994597c13d831ec7", SwapAmount: big.NewInt(1000)}},
	}
	updatedRoute = builder.updateSwapRecipientAndCollectAmount(route, "normal", "")
	assert.Equal(t, big.NewInt(1000), updatedRoute[0][0].CollectAmount)

}
