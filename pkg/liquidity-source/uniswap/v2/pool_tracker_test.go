package uniswapv2

import (
	"context"
	"math/big"
	"testing"
	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type staticLogDecoder struct {
	reserveData ReserveData
	blockNumber *big.Int
}

func (d staticLogDecoder) Decode([]types.Log, map[uint64]entity.BlockHeader) (ReserveData, *big.Int, error) {
	return d.reserveData, d.blockNumber, nil
}

func TestPoolTracker_RegularPoolWithLegacyExtra(t *testing.T) {
	t.Parallel()

	const (
		token0 = "0x1111111111111111111111111111111111111111"
		token1 = "0x2222222222222222222222222222222222222222"
	)

	tracker := &PoolTracker{
		// ChainID intentionally left unset (unsupported by the tax detector) so this test stays a
		// pure unit test: with a supported chain, newTokenTaxTracker now tracks every 2-token pool
		// (no base-token allowlist gate), which would require real RPC access. See the *_RealChain
		// tests in pool_tracker_integration_test.go for that path.
		config: &Config{
			FactoryAddress: "0x5c69bee701ef814a2b6a3edd4b1652cb9cc5aa6f",
			Fee:            30,
			FeePrecision:   10000,
		},
		ethrpcClient: ethrpc.New("http://127.0.0.1:0"),
		logDecoder: staticLogDecoder{
			reserveData: ReserveData{
				Reserve0:           big.NewInt(1000),
				Reserve1:           big.NewInt(2000),
				BlockTimestampLast: 200,
			},
			blockNumber: big.NewInt(11),
		},
	}

	legacyPool := entity.Pool{
		Address:     "0x0000000000000000000000000000000000000001",
		Reserves:    entity.PoolReserves{"900", "1900"},
		Tokens:      []*entity.PoolToken{{Address: token0}, {Address: token1}},
		Extra:       `{"fee":25,"feePrecision":10000}`,
		BlockNumber: 10,
		Timestamp:   100,
	}

	updated, err := tracker.GetNewPoolState(context.Background(), legacyPool, pool.GetNewPoolStateParams{
		Logs: []types.Log{{}},
	})
	require.NoError(t, err)
	assert.Equal(t, entity.PoolReserves{"1000", "2000"}, updated.Reserves)
	assert.Equal(t, uint64(11), updated.BlockNumber)
	assert.Equal(t, int64(200), updated.Timestamp)

	var extra Extra
	require.NoError(t, json.Unmarshal([]byte(updated.Extra), &extra))
	assert.Equal(t, uint64(30), extra.Fee)
	assert.Equal(t, uint64(10000), extra.FeePrecision)
	// ChainID is unsupported here (see comment above), so tax detection never runs.
	assert.Nil(t, extra.TaxInfo)
}
