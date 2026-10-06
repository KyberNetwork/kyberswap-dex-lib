package msgpack

import (
	"reflect"
	"testing"
	"time"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	orderbook "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/order-book"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

func TestOrderbookExpirySerialization(t *testing.T) {
	extra := orderbook.Extra{ValidUntil: time.Now().Unix() + 60, LevelsFrom: [2][]orderbook.Level{{{0.005, 2}, {1, 2}}, nil}}
	raw, err := json.Marshal(extra)
	require.NoError(t, err)
	ep := entity.Pool{Address: "expiry_fixture", Type: orderbook.DexType, Timestamp: time.Now().Unix(), Extra: string(raw), Reserves: []string{"1000000000000000000", "2000000"},
		Tokens: []*entity.PoolToken{{Address: "a", Decimals: 18}, {Address: "b", Decimals: 6}}}
	before, err := orderbook.NewPoolSimulator(pool.FactoryParams{EntityPool: ep})
	require.NoError(t, err)
	packed, err := EncodePoolSimulatorsMap(map[string]pool.IPoolSimulator{ep.Address: before})
	require.NoError(t, err)
	after, err := DecodePoolSimulatorsMap(packed)
	require.NoError(t, err)
	require.True(t, reflect.DeepEqual(before, after[ep.Address]), "expiry and raw minimum must survive serialization")
}
