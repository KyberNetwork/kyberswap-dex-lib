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
	tokentax "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v2/token-tax"
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
		token0 = "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"
		token1 = "0xdac17f958d2ee523a2206206994597c13d831ec7"
	)

	tracker := &PoolTracker{
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
	require.NotNil(t, extra.TaxInfo)
	assert.Equal(t, tokentax.TaxInfo{Checked: true}, *extra.TaxInfo)
}
