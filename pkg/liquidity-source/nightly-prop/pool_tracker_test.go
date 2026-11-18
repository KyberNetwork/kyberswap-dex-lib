package nightlyprop

import (
	"math/big"
	"os"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/nightly-prop/math"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// swappableState is a router PoolStateV1 of a routed, unpaused pool with every role, an enabled
// partner, a price and two levels.
func swappableState() rpcPoolStateV1 {
	return rpcPoolStateV1{
		BlockNumber:            big.NewInt(525_043_339),
		BlockTimestamp:         big.NewInt(1_794_921_438),
		Base:                   common.HexToAddress(arbitrumWeth),
		Quote:                  common.HexToAddress(arbitrumUsdc),
		BaseDecimals:           18,
		QuoteDecimals:          6,
		Routed:                 true,
		RouterSwapper:          true,
		PoolPriceReader:        true,
		PartnerRole:            true,
		PartnerFeeRecipientSet: true,
		PriceAvailable:         true,
		MidPrice:               182_107_498_496,
		BidSpread:              283,
		AskSpread:              284,
		PriceTimestampMs:       big.NewInt(1_794_921_430_000),
		MaxPriceAgeMs:          30_000,
		PositionLevelID:        0,
		PositionFraction:       big.NewInt(7),
		LowestLevelID:          -1,
		Levels: []rpcPoolStateV1PriceLevel{
			{LowerOffset: -100, UpperOffset: 0, BaseAmount: big.NewInt(0), QuoteAmount: big.NewInt(10)},
			{LowerOffset: 0, UpperOffset: 100, BidExtraSpread: 1, AskExtraSpread: 2, BaseAmount: big.NewInt(3),
				QuoteAmount: big.NewInt(4)},
		},
	}
}

func TestNewExtra(t *testing.T) {
	extra, err := newExtra(swappableState())
	require.NoError(t, err)

	position, err := math.FractionFromRaw(uint256.NewInt(7))
	require.NoError(t, err)
	require.Equal(t, Extra{
		Routed:                 true,
		RouterSwapper:          true,
		PoolPriceReader:        true,
		PartnerRole:            true,
		PartnerFeeRecipientSet: true,
		PriceAvailable:         true,
		BlockTimestamp:         1_794_921_438,
		Price:                  math.OraclePrice{MidPrice: 182_107_498_496, BidSpread: 283, AskSpread: 284, TimestampMs: 1_794_921_430_000},
		Pool: math.Pool{
			BaseDecimals:  18,
			QuoteDecimals: 6,
			MaxPriceAgeMs: 30_000,
			LowestLevelID: -1,
			Levels: []math.Level{
				{
					Config:    math.PriceLevelConfig{LowerOffset: -100, UpperOffset: 0},
					Liquidity: math.Liquidity{QuoteAmount: *uint256.NewInt(10)},
				},
				{
					Config:    math.PriceLevelConfig{LowerOffset: 0, UpperOffset: 100, BidExtraSpread: 1, AskExtraSpread: 2},
					Liquidity: math.Liquidity{BaseAmount: *uint256.NewInt(3), QuoteAmount: *uint256.NewInt(4)},
				},
			},
			Position: math.Position{LevelID: 0, Fraction: position},
		},
	}, extra)
	require.Equal(t, entity.PoolReserves{"3", "14"}, extra.reserves())
}

// TestNewExtra_JSON: Extra round-trips through JSON, which the record stores.
func TestNewExtra_JSON(t *testing.T) {
	extra, err := newExtra(swappableState())
	require.NoError(t, err)
	encoded, err := json.Marshal(&extra)
	require.NoError(t, err)
	var decoded Extra
	require.NoError(t, json.Unmarshal(encoded, &decoded))
	require.Equal(t, extra, decoded)
}

// TestNewExtra_FractionAboveUint104: the contract's position fraction is a uint104.
func TestNewExtra_FractionAboveUint104(t *testing.T) {
	state := swappableState()
	state.PositionFraction = new(big.Int).Lsh(big.NewInt(1), 104)
	_, err := newExtra(state)
	require.ErrorIs(t, err, math.ErrOverflow)
}

// TestExtraReserves_NotSwappable: a pool whose partner swaps revert now has zero reserves, and
// swapError tells why.
func TestExtraReserves_NotSwappable(t *testing.T) {
	for name, c := range map[string]struct {
		change func(*rpcPoolStateV1)
		want   error
	}{
		"not routed":                {func(s *rpcPoolStateV1) { s.Routed = false }, ErrNoRoute},
		"router paused":             {func(s *rpcPoolStateV1) { s.RouterPaused = true }, ErrPaused},
		"pool paused":               {func(s *rpcPoolStateV1) { s.PoolPaused = true }, ErrPaused},
		"router not swapper":        {func(s *rpcPoolStateV1) { s.RouterSwapper = false }, ErrMissingRole},
		"pool not price reader":     {func(s *rpcPoolStateV1) { s.PoolPriceReader = false }, ErrMissingRole},
		"partner without role":      {func(s *rpcPoolStateV1) { s.PartnerRole = false }, ErrPartnerNotEnabled},
		"partner without recipient": {func(s *rpcPoolStateV1) { s.PartnerFeeRecipientSet = false }, ErrPartnerNotEnabled},
		"no price":                  {func(s *rpcPoolStateV1) { s.PriceAvailable = false }, ErrPriceUnavailable},
	} {
		t.Run(name, func(t *testing.T) {
			state := swappableState()
			c.change(&state)
			extra, err := newExtra(state)
			require.NoError(t, err)
			require.ErrorIs(t, extra.swapError(), c.want)
			require.Equal(t, entity.PoolReserves{"0", "0"}, extra.reserves())
		})
	}
}

// TestExtraReserves_StalePrice: a price older than the pool's maximum age keeps the price and the
// reserves. The simulator checks the age when it quotes.
func TestExtraReserves_StalePrice(t *testing.T) {
	state := swappableState()
	state.PriceTimestampMs = big.NewInt(1_000)
	extra, err := newExtra(state)
	require.NoError(t, err)
	require.NoError(t, extra.swapError())
	require.Equal(t, uint64(1_000), extra.Price.TimestampMs)
	require.Equal(t, entity.PoolReserves{"3", "14"}, extra.reserves())
}

func TestPoolTracker_InvalidPoolRecord(t *testing.T) {
	tracker := NewPoolTracker(&Config{}, nil)
	for name, p := range map[string]entity.Pool{
		"one token": {
			Address:     arbitrumWethUsdcPool,
			StaticExtra: `{"routerAddress":"` + simRouter + `","partnerAddress":"` + simPartner + `"}`,
			Tokens:      []*entity.PoolToken{{Address: arbitrumWeth}},
		},
		"no partner": {
			Address:     arbitrumWethUsdcPool,
			StaticExtra: `{"routerAddress":"` + simRouter + `"}`,
			Tokens:      []*entity.PoolToken{{Address: arbitrumWeth}, {Address: arbitrumUsdc}},
		},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := tracker.GetNewPoolState(t.Context(), p, pool.GetNewPoolStateParams{})
			require.ErrorIs(t, err, ErrInvalidPoolRecord)
			require.Equal(t, p, got)
		})
	}
}

// TestLivePoolTracker: on Arbitrum, each pool that the router lists refreshes from one
// getPoolStateV1 call for the partner. Each pool has its position inside its levels. A swappable
// pool has reserves that its token balances at the record's block cover, and any other pool has
// zero reserves. It needs router revision 5: until the Arbitrum router has it, point the URL at a
// fork whose router has it. The partner must hold the router's PARTNER_ROLE and have a fee
// recipient, or no pool is swappable. Run with:
//
//	NIGHTLY_PROP_ARBITRUM_RPC_URL=https://arb1.arbitrum.io/rpc NIGHTLY_PROP_ARBITRUM_PARTNER=<partner> \
//	  go test ./pkg/liquidity-source/nightly-prop/ -run LivePoolTracker -v
func TestLivePoolTracker(t *testing.T) {
	client := testutil.DialLive(t, testutil.LiveURL(t, "NIGHTLY_PROP_ARBITRUM_RPC_URL"))
	partner := os.Getenv("NIGHTLY_PROP_ARBITRUM_PARTNER")
	if partner == "" {
		t.Skip("set NIGHTLY_PROP_ARBITRUM_PARTNER to run this")
	}
	cfg := &Config{DexID: DexType, ChainID: valueobject.ChainIDArbitrumOne, RouterAddress: arbitrumRouter,
		PartnerAddress: partner}
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
		state := extra.Pool
		position := int(state.Position.LevelID) - int(state.LowestLevelID)
		require.True(t, position >= 0 && position < len(state.Levels),
			"%s: position %d is outside the levels", p.Address, state.Position.LevelID)
		t.Logf("pool=%s block=%d swapError=%v levels=%d position=%d reserves=%v price=%+v",
			got.Address, got.BlockNumber, extra.swapError(), len(state.Levels), state.Position.LevelID,
			got.Reserves, extra.Price)
		if extra.swapError() != nil {
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
