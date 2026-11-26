package syntropiaprop

import (
	"math/big"
	"testing"
	"time"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	orderbook "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/order-book"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

const (
	testPool = "0x05ca2d5786703b34fca134bc64ae44392e753713"
	testLens = "0x5c60e442a1592e52bf96646b05c8279a50f2131a"
	testWETH = "0x4200000000000000000000000000000000000006"
	testUSDC = "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913"
)

// The lens's WETH/USDC book on a fork of Base at block 62,603,623, with the vault holding
// 20.0005 WETH and 60,000 USDC and the pair priced at 1,683.11 with a 3 bps spread.
var (
	fixtureBids = []Level{ // spend WETH, receive USDC
		{bi("1114009185377069829"), bi("1874174999")},
		{bi("2228018370754139658"), bi("3747828749")},
		{bi("4456036741508279316"), bi("7493564999")},
		{bi("8912073483016558632"), bi("14978774999")},
		{bi("17824146966033117265"), bi("29924129999")},
		{bi("35648293932066234530"), bi("59766119999")},
	}
	fixtureAsks = []Level{ // spend USDC, receive WETH
		{bi("1051967761"), bi("624778010460600910")},
		{bi("2103935522"), bi("1249457268667197033")},
		{bi("4207871044"), bi("2498524528432501737")},
		{bi("8415742089"), bi("4995489021851022808")},
		{bi("16831484178"), bi("9984727901042772010")},
		{bi("33662968357"), bi("19944455232040924240")},
	}
	fixtureBalanceWETH = bi("20000456510518448683")
	fixtureBalanceUSDC = bi("60000000000")
)

func bi(value string) *big.Int {
	n, ok := new(big.Int).SetString(value, 10)
	if !ok {
		panic(value)
	}
	return n
}

// fixturePool is the entity the tracker would store for the fixture book, read `age` ago.
func fixturePool(t testing.TB, buffer int64, age time.Duration) entity.Pool {
	t.Helper()
	bids, err := levelsToOrderbook(fixtureBids, 18, 6, buffer)
	require.NoError(t, err)
	asks, err := levelsToOrderbook(fixtureAsks, 6, 18, buffer)
	require.NoError(t, err)
	extra, err := json.Marshal(orderbook.Extra{LevelsFrom: [2][]orderbook.Level{bids, asks}})
	require.NoError(t, err)
	return entity.Pool{
		Address:   testPool + "_" + testWETH + "_" + testUSDC,
		Exchange:  DexType,
		Type:      DexType,
		Timestamp: time.Now().Add(-age).Unix(),
		Reserves:  entity.PoolReserves{fixtureBalanceWETH.String(), fixtureBalanceUSDC.String()},
		Tokens: []*entity.PoolToken{
			{Address: testWETH, Decimals: 18, Swappable: true},
			{Address: testUSDC, Decimals: 6, Swappable: true},
		},
		Extra:       string(extra),
		StaticExtra: `{"pool":"` + testPool + `"}`,
		BlockNumber: 62603623,
	}
}

func fixtureSimulator(t testing.TB, buffer int64) *PoolSimulator {
	t.Helper()
	simulator, err := NewPoolSimulator(pool.FactoryParams{EntityPool: fixturePool(t, buffer, 0)})
	require.NoError(t, err)
	return simulator
}
