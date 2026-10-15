package uscoreprop_test

import (
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ladder"
	uscoreprop "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uscore-prop"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/msgpack"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

func TestFactoryAndSerialization(t *testing.T) {
	require.Equal(t, "uscore-prop", uscoreprop.DexType)
	factory := pool.Factory(uscoreprop.DexType)
	require.NotNil(t, factory)
	p := entity.Pool{
		Address:  "0x1111111111111111111111111111111111111111",
		Exchange: uscoreprop.DexType, Type: uscoreprop.DexType,
		Timestamp: time.Now().Unix(), BlockNumber: 123,
		Tokens: []*entity.PoolToken{
			{Address: "0x2222222222222222222222222222222222222222", Swappable: true},
			{Address: "0x3333333333333333333333333333333333333333", Swappable: true},
		},
		Reserves: entity.PoolReserves{"1000", "1000"},
		Extra:    `{"l":[[[100,200],[200,390]],[[200,100],[400,195]]]}`,
	}
	sim, err := factory(pool.FactoryParams{EntityPool: p, Opts: pool.FactoryOpts{StaleCheck: true}})
	require.NoError(t, err)
	q := pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: p.Tokens[0].Address, Amount: big.NewInt(25)},
		TokenOut:      p.Tokens[1].Address,
	}
	first, err := sim.CalcAmountOut(q)
	require.NoError(t, err)
	sim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn: q.TokenAmountIn, TokenAmountOut: *first.TokenAmountOut, Fee: *first.Fee,
	})
	expected, err := sim.CalcAmountOut(q)
	require.NoError(t, err)
	encoded, err := msgpack.EncodePoolSimulatorsMap(map[string]pool.IPoolSimulator{p.Address: sim})
	require.NoError(t, err)
	restored, err := msgpack.DecodePoolSimulatorsMap(encoded)
	require.NoError(t, err)
	decoded := restored[p.Address]
	require.IsType(t, &uscoreprop.PoolSimulator{}, decoded)
	actual, err := decoded.CalcAmountOut(q)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
	require.Equal(t, sim.GetReserves(), decoded.GetReserves())
	require.Equal(t, ladder.PoolMeta{BlockNumber: 123}, decoded.GetMetaInfo(q.TokenAmountIn.Token, q.TokenOut))
	p.Timestamp = time.Now().Add(-uscoreprop.MaxAge - time.Second).Unix()
	_, err = factory(pool.FactoryParams{EntityPool: p, Opts: pool.FactoryOpts{StaleCheck: true}})
	require.ErrorIs(t, err, ladder.ErrStale)
}
