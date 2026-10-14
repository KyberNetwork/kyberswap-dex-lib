package two_test

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/msgpack"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/curve/two"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// Pool simulators are registered for msgpack (de)serialization; a decoded simulator must quote exactly like the
// original.
func TestMsgpackRoundTrip(t *testing.T) {
	t.Parallel()
	p, err := two.NewPoolSimulator(entity.Pool{
		Address:  "0xb576491f1e6e5e62f1d8f26062ee822b40b0e0d4",
		Reserves: entity.PoolReserves{"1213926003558609983857", "1582278078919869531746642"},
		Tokens:   []*entity.PoolToken{{Address: "weth"}, {Address: "cvx"}},
		Extra: `{"A":"400000","D":"2473891443596100529676","gamma":"145000000000000","priceScale":"796372805131279",` +
			`"lastPrices":"779859076866219","priceOracle":"779859124398978","feeGamma":"230000000000000",` +
			`"midFee":"26000000","outFee":"45000000","futureAGammaTime":0,` +
			`"futureAGamma":"136112946768375385385349842972852284582400000","initialAGammaTime":0,` +
			`"initialAGamma":"136112946768375385385349842972852284582400000","lastPricesTimestamp":1791952679,` +
			`"lpSupply":"39266915138093884385106","xcpProfit":"1232511904047188462","virtualPrice":"1116260734821589802",` +
			`"allowedExtraProfit":"2000000000000","adjustmentStep":"146000000000000","maHalfTime":"600"}`,
		StaticExtra: `{"lpToken":"lp","precisionMultipliers":["1","1"]}`,
	})
	require.NoError(t, err)

	encoded, err := msgpack.EncodePoolSimulatorsMap(map[string]pool.IPoolSimulator{p.GetAddress(): p})
	require.NoError(t, err)
	decoded, err := msgpack.DecodePoolSimulatorsMap(encoded)
	require.NoError(t, err)
	q := decoded[p.GetAddress()]
	require.IsType(t, &two.PoolSimulator{}, q)
	require.Equal(t, p, q.(*two.PoolSimulator))

	params := pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: "weth", Amount: big.NewInt(1e18)}, TokenOut: "cvx"}
	want, err := p.CalcAmountOut(params)
	require.NoError(t, err)
	got, err := q.CalcAmountOut(params)
	require.NoError(t, err)
	require.Equal(t, want.TokenAmountOut.Amount.String(), got.TokenAmountOut.Amount.String())
	require.Equal(t, want.Fee.Amount.String(), got.Fee.Amount.String())
}
