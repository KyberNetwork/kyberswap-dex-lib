package types

import (
	"context"
	"math/big"
	"testing"

	"github.com/KyberNetwork/aggregator-encoding/pkg/types"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/stretchr/testify/assert"
)

func TestEncodeCollectAmountUseExecutorBalance(t *testing.T) {
	builder := NewEncodingDataBuilder(context.Background(), nil, false)
	builder.data.TokenIn = "in"

	route := [][]types.EncodingSwap{
		{{Pool: "p1", TokenIn: "in", TokenOut: "mid1", SwapAmount: big.NewInt(1000)}},
		{{Pool: "p2", TokenIn: "in", TokenOut: "mid2", SwapAmount: big.NewInt(2000)}},
		{{Pool: "p3", TokenIn: "mid1", TokenOut: "mid2", SwapAmount: big.NewInt(3000)}},
		{{Pool: "p4", TokenIn: "mid1", TokenOut: "mid2", SwapAmount: big.NewInt(4000)}},
		{{Pool: "p5", TokenIn: "mid2", TokenOut: "out", SwapAmount: big.NewInt(5000)}},
	}

	route = builder.updateSwapRecipientAndCollectAmount(route, "normal", "")
	assert.Equal(t, big.NewInt(2000), route[1][0].CollectAmount)
	assert.Equal(t, big.NewInt(3000), route[2][0].CollectAmount)
	assert.Equal(t, bignumber.MAX_UINT_128, route[3][0].CollectAmount)
	assert.Equal(t, bignumber.MAX_UINT_128, route[4][0].CollectAmount)
}
