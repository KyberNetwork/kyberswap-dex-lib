package v3

import (
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	tickspkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3/ticks"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
	graphqlpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

const sbSeedExtra = `{"unlocked":true,"liquidity":1000,"sqrtPriceX96":79228162514264337593543950336,"tick":0,` +
	`"ticks":[{"index":-60,"liquidityGross":1000,"liquidityNet":1000},{"index":60,"liquidityGross":1000,"liquidityNet":-1000}],` +
	`"vaults":[` + sbVault + `,` + sbVault + `]}`

const sbVault = `{"DepositPaused":false,"RedeemPaused":false,"MinDeposit":"0","ExchangeRate":"0","MinRedeemInterval":"0","RedeemCoolDownExempt":false}`

type stateBlockFixture struct {
	t       *testing.T
	stub    *testutil.MulticallStub
	state   *testutil.MethodResponder
	tracker *PoolTracker
}

// newStateBlockFixture is a tracker whose chain answers its multicalls at the block of At.
func newStateBlockFixture(t *testing.T) *stateBlockFixture {
	state := testutil.NewMethodResponder(t, poolABI, lpTokenABI, abi.Erc20ABI)
	stub := testutil.NewMulticallStub(t, state.Respond).At(500, 1)
	gql := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":{"pool":{"ticks":[]}}}`)) // the subgraph has no ticks: they do not matter here
	}))
	t.Cleanup(gql.Close)
	f := &stateBlockFixture{t: t, stub: stub, state: state,
		tracker: NewPoolTracker(&Config{DexID: DexType}, stub.Client(), graphqlpkg.NewClient(gql.URL))}
	f.setPool(1000)
	return f
}

// setPool is the state of the pool on chain: its liquidity, with a price and vaults that never change.
func (f *stateBlockFixture) setPool(liquidity int64) {
	f.state.Set("liquidity", big.NewInt(liquidity)).
		Set("slot0", new(big.Int).Lsh(big.NewInt(1), 96), big.NewInt(0), uint16(0), uint16(0), uint16(0), uint32(0), true).
		Set("tickSpacing", big.NewInt(60)).
		Set("balanceOf", big.NewInt(1e18)).
		Set("underlying", common.Address{}).
		Set("minDeposit", big.NewInt(0)).
		Set("depositPaused", false).
		Set("redeemPaused", false).
		Set("exchangeRate", big.NewInt(0)).
		Set("minRedeemInterval", big.NewInt(0)).
		Set("redeemCooldownExempt", false)
}

func stateBlockPool(blockNumber uint64, extra string) entity.Pool {
	return entity.Pool{
		Address:  "0x1111111111111111111111111111111111111111",
		Exchange: DexType, Type: DexType,
		BlockNumber: blockNumber,
		Tokens: []*entity.PoolToken{
			{Address: "0x2222222222222222222222222222222222222222", Decimals: 18, Swappable: true},
			{Address: "0x3333333333333333333333333333333333333333", Decimals: 18, Swappable: true},
		},
		Extra:       extra,
		StaticExtra: `{"tickSpacing":60,"scan":true}`,
	}
}

// update is one event-driven write of p: with no logs the ticks stay, the scalar state is read.
func (f *stateBlockFixture) update(p entity.Pool) entity.Pool {
	ticksBasedPool, err := tickspkg.NewTicksBasedPool(p)
	require.NoError(f.t, err)
	out, err := f.tracker.updateState(f.t.Context(), p, ticksBasedPool, nil, nil)
	require.NoError(f.t, err)
	return out
}
