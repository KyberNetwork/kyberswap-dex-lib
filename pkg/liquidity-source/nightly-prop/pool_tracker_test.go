package nightlyprop

import (
	"math/big"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// swappableState is a router PoolStateV1 of a routed, unpaused pool with a price and two levels.
func swappableState() rpcPoolStateV1 {
	return rpcPoolStateV1{
		BlockNumber:      big.NewInt(525_043_339),
		BlockTimestamp:   big.NewInt(1_794_921_438),
		Base:             common.HexToAddress(arbitrumWeth),
		Quote:            common.HexToAddress(arbitrumUsdc),
		BaseDecimals:     18,
		QuoteDecimals:    6,
		Routed:           true,
		PriceAvailable:   true,
		MidPrice:         182_107_498_496,
		BidSpread:        283,
		AskSpread:        284,
		PriceTimestampMs: big.NewInt(1_794_921_430_000),
		MaxPriceAgeMs:    30_000,
		PositionLevelID:  0,
		PositionFraction: big.NewInt(7),
		LowestLevelID:    -1,
		Levels: []rpcPoolStateV1PriceLevel{
			{LowerOffset: -100, UpperOffset: 0, BaseAmount: big.NewInt(0), QuoteAmount: big.NewInt(10)},
			{LowerOffset: 0, UpperOffset: 100, BidExtraSpread: 1, AskExtraSpread: 2, BaseAmount: big.NewInt(3),
				QuoteAmount: big.NewInt(4)},
		},
	}
}

func TestNewExtra(t *testing.T) {
	extra := newExtra(swappableState())

	require.Equal(t, Extra{
		Routed:         true,
		PriceAvailable: true,
		BlockTimestamp: 1_794_921_438,
		BaseDecimals:   18,
		QuoteDecimals:  6,
		Price:          Price{MidPrice: 182_107_498_496, BidSpread: 283, AskSpread: 284, TimestampMs: 1_794_921_430_000},
		MaxPriceAgeMs:  30_000,
		Position:       Position{LevelID: 0, Fraction: uint256.NewInt(7)},
		LowestLevelID:  -1,
		Levels: []Level{
			{LowerOffset: -100, UpperOffset: 0, BaseAmount: uint256.NewInt(0), QuoteAmount: uint256.NewInt(10)},
			{LowerOffset: 0, UpperOffset: 100, BidExtraSpread: 1, AskExtraSpread: 2, BaseAmount: uint256.NewInt(3),
				QuoteAmount: uint256.NewInt(4)},
		},
	}, extra)
	require.Equal(t, entity.PoolReserves{"3", "14"}, extra.reserves())
}

// TestExtraReserves_NotSwappable: a pool that the router cannot swap now has zero reserves.
func TestExtraReserves_NotSwappable(t *testing.T) {
	for name, change := range map[string]func(*rpcPoolStateV1){
		"not routed":    func(s *rpcPoolStateV1) { s.Routed = false },
		"router paused": func(s *rpcPoolStateV1) { s.RouterPaused = true },
		"pool paused":   func(s *rpcPoolStateV1) { s.PoolPaused = true },
		"no price":      func(s *rpcPoolStateV1) { s.PriceAvailable = false },
	} {
		t.Run(name, func(t *testing.T) {
			state := swappableState()
			change(&state)
			extra := newExtra(state)
			require.False(t, extra.swappable())
			require.Equal(t, entity.PoolReserves{"0", "0"}, extra.reserves())
		})
	}
}

func TestPoolTracker_InvalidPoolRecord(t *testing.T) {
	tracker := NewPoolTracker(&Config{}, nil)
	p := entity.Pool{
		Address:     arbitrumWethUsdcPool,
		StaticExtra: `{"routerAddress":"0xede7b21238ab8d2bc7c5e62d7e6d141caba5cba2"}`,
		Tokens:      []*entity.PoolToken{{Address: arbitrumWeth}},
	}
	got, err := tracker.GetNewPoolState(t.Context(), p, pool.GetNewPoolStateParams{})
	require.ErrorIs(t, err, ErrInvalidPoolRecord)
	require.Equal(t, p, got)
}

// TestLivePoolTracker: on Arbitrum, each pool that the router lists refreshes from one
// getPoolStateV1 call. Each pool has its position inside its levels. A swappable pool has
// reserves that its token balances at the record's block cover, and any other pool has zero
// reserves. It needs router revision 4: until the Arbitrum router has it, point the URL at a fork
// whose router has it. Run with:
//
//	NIGHTLY_PROP_ARBITRUM_RPC_URL=https://arb1.arbitrum.io/rpc \
//	  go test ./pkg/liquidity-source/nightly-prop/ -run LivePoolTracker -v
func TestLivePoolTracker(t *testing.T) {
	client := testutil.DialLive(t, testutil.LiveURL(t, "NIGHTLY_PROP_ARBITRUM_RPC_URL"))
	cfg := &Config{DexID: DexType, ChainID: valueobject.ChainIDArbitrumOne, RouterAddress: arbitrumRouter}
	u, err := NewPoolsListUpdater(cfg, client.Ethrpc)
	require.NoError(t, err)
	pools, _, err := u.GetNewPools(t.Context(), nil)
	require.NoError(t, err)
	tracker := NewPoolTracker(cfg, client.Ethrpc)

	swappable := 0
	for _, p := range pools {
		got, err := tracker.GetNewPoolState(t.Context(), p, pool.GetNewPoolStateParams{})
		require.NoError(t, err, p.Address)
		var extra Extra
		require.NoError(t, json.Unmarshal([]byte(got.Extra), &extra))
		require.True(t, extra.Routed, p.Address)
		require.NotZero(t, got.BlockNumber, p.Address)
		position := int(extra.Position.LevelID) - int(extra.LowestLevelID)
		require.True(t, position >= 0 && position < len(extra.Levels),
			"%s: position %d is outside the levels", p.Address, extra.Position.LevelID)
		t.Logf("pool=%s block=%d swappable=%v levels=%d position=%d reserves=%v price=%+v",
			got.Address, got.BlockNumber, extra.swappable(), len(extra.Levels), extra.Position.LevelID,
			got.Reserves, extra.Price)
		if !extra.swappable() {
			require.Equal(t, entity.PoolReserves{"0", "0"}, got.Reserves, p.Address)
			continue
		}
		swappable++
		balances := balancesAt(t, client.Ethrpc, got)
		for i, reserve := range got.Reserves {
			amount, ok := new(big.Int).SetString(reserve, 10)
			require.True(t, ok, reserve)
			require.LessOrEqual(t, amount.Cmp(balances[i]), 0,
				"%s: reserve %s is above the balance %s", p.Address, reserve, balances[i])
		}
	}
	require.NotZero(t, swappable, "no listed pool is swappable")
}

func balancesAt(t *testing.T, client *ethrpc.Client, p entity.Pool) [2]*big.Int {
	t.Helper()
	var balances [2]*big.Int
	req := client.NewRequest().SetContext(t.Context()).SetBlockNumber(new(big.Int).SetUint64(p.BlockNumber))
	for i, token := range p.Tokens {
		req.AddCall(&ethrpc.Call{ABI: abi.Erc20ABI, Target: token.Address, Method: abi.Erc20BalanceOfMethod,
			Params: []any{common.HexToAddress(p.Address)}}, []any{&balances[i]})
	}
	_, err := req.Aggregate()
	require.NoError(t, err)
	return balances
}
