package ghost

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

func TestGetNewPoolState_InvalidStaticExtra(t *testing.T) {
	t.Parallel()

	tracker := NewPoolTracker(&Config{DexID: DexType}, nil)

	_, err := tracker.GetNewPoolState(context.Background(), entity.Pool{
		Address:     "ghost_0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48_0xdac17f958d2ee523a2206206994597c13d831ec7",
		StaticExtra: "not-json",
	}, pool.GetNewPoolStateParams{})

	assert.Error(t, err)
}
