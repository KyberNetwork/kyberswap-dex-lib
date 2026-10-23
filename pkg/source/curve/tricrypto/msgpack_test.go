package tricrypto_test

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/msgpack"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/curve/tricrypto"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// Pool simulators are registered for msgpack (de)serialization; a decoded simulator must quote and update exactly
// like the original.
func TestMsgpackRoundTrip(t *testing.T) {
	t.Parallel()
	p, err := tricrypto.NewPoolSimulator(entity.Pool{
		Address:  "0xd51a44d3fae010294c616388b506acda1bfaae46",
		Reserves: entity.PoolReserves{"3947114941094", "4687873211", "1554787014549222229616"},
		Tokens:   []*entity.PoolToken{{Address: "usdt"}, {Address: "wbtc"}, {Address: "weth"}},
		Extra: `{"A":"1707629","D":"11722711763398766437016767","gamma":"11809167828997",` +
			`"priceScale":["82724804211703587475975","2506887709019354722973",""],` +
			`"lastPrices":["83557713878122097375341","2526812438494472454161",""],` +
			`"priceOracle":["83374526269551916167919","2520068483911547726169",""],` +
			`"feeGamma":"500000000000000","midFee":"3000000","outFee":"30000000","futureAGammaTime":0,` +
			`"futureAGamma":"581076037942835227425498917514114728328226821","initialAGammaTime":1633548703,` +
			`"initialAGamma":"183752478137306770270222288013175834186240000","lastPricesTimestamp":1791868775,` +
			`"lpSupply":"6153038484735248206702","xcpProfit":"1145801814313530617","virtualPrice":"1072902665505676632",` +
			`"allowedExtraProfit":"2000000000000","adjustmentStep":"490000000000000","maHalfTime":"600"}`,
		StaticExtra: `{"lpToken":"lp","precisionMultipliers":["1000000000000","10000000000","1"]}`,
	})
	require.NoError(t, err)

	encoded, err := msgpack.EncodePoolSimulatorsMap(map[string]pool.IPoolSimulator{p.GetAddress(): p})
	require.NoError(t, err)
	decoded, err := msgpack.DecodePoolSimulatorsMap(encoded)
	require.NoError(t, err)
	q := decoded[p.GetAddress()]
	require.IsType(t, &tricrypto.PoolSimulator{}, q)
	require.Equal(t, p, q.(*tricrypto.PoolSimulator))

	params := pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: "weth", Amount: big.NewInt(4e18)}, TokenOut: "usdt"}
	want, err := p.CalcAmountOut(params)
	require.NoError(t, err)
	got, err := q.CalcAmountOut(params)
	require.NoError(t, err)
	require.Equal(t, want.TokenAmountOut.Amount.String(), got.TokenAmountOut.Amount.String())
	require.Equal(t, want.Fee.Amount.String(), got.Fee.Amount.String())
}
