package poe

import (
	"context"
	"math/big"
	"testing"
	"testing/synctest"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/test/ethrpcmock"
)

const (
	poeTokenX = "0x0000000000000000000000000000000000000001"
	poeTokenY = "0x0000000000000000000000000000000000000002"
)

func TestGetNewPoolStatePinsOracleToPoolBlock(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		srv, recorder := ethrpcmock.New(t, poolABI, oracleABI)
		client := ethrpc.New(srv.URL)
		updated, err := NewPoolTracker(&Config{DexId: string(DexType)}, client).GetNewPoolState(
			context.Background(), entity.Pool{
				Address: "0x0000000000000000000000000000000000000003",
				Tokens:  []*entity.PoolToken{{Address: poeTokenX}, {Address: poeTokenY}},
			}, pool.GetNewPoolStateParams{},
		)
		require.NoError(t, err)
		require.Equal(t, ethrpcmock.Block, updated.BlockNumber)
		require.Equal(t, entity.PoolReserves{"1", "1"}, updated.Reserves)
		var extra Extra
		require.NoError(t, json.Unmarshal([]byte(updated.Extra), &extra))
		require.Equal(t, common.Address{}.Hex(), extra.Oracle)
		ethrpcmock.RequireRequestsPinned(t, recorder, 2)
		ethrpcmock.CheckPool(t, "lfj-poe-tracker-pin", updated)
		ethrpcmock.CheckRequests(t, "lfj-poe-tracker-pin", recorder)
	})
}

func newTestPoolSimulator(t *testing.T) *PoolSimulator {
	t.Helper()

	extra := Extra{
		Price:   uint256.NewInt(2_000_000_000_000_000_000),
		FeeHbps: uint256.NewInt(3000),
		Alpha:   uint256.NewInt(10500),
		Expiry:  uint64(time.Now().Unix()) + 3600,
	}
	extraBytes, err := json.Marshal(extra)
	require.NoError(t, err)

	sim, err := NewPoolSimulator(entity.Pool{
		Address:  "0xpool",
		Exchange: string(DexType),
		Type:     string(DexType),
		Reserves: entity.PoolReserves{"10000000000000000000", "20000000000"},
		Tokens: []*entity.PoolToken{
			{Address: poeTokenX, Swappable: true},
			{Address: poeTokenY, Swappable: true},
		},
		Extra: string(extraBytes),
	})
	require.NoError(t, err)
	return sim
}

func TestPoolSimulator_CalcAmountOut_XtoY_NotCapped(t *testing.T) {
	sim := newTestPoolSimulator(t)

	res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: poeTokenX, Amount: mustBig("10000000000")},
		TokenOut:      poeTokenY,
	})
	require.NoError(t, err)

	require.Equal(t, mustBig("18991"), res.TokenAmountOut.Amount)
	require.Equal(t, poeTokenY, res.Fee.Token) // fee always denominated in tokenY
	require.Equal(t, mustBig("58"), res.Fee.Amount)
	require.Zero(t, res.RemainingTokenAmountIn.Amount.Sign()) // fully consumed
}

func TestPoolSimulator_CalcAmountOut_XtoY_Capped_PartialFill(t *testing.T) {
	sim := newTestPoolSimulator(t)

	amountIn := mustBig("100000000000000000")
	res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: poeTokenX, Amount: amountIn},
		TokenOut:      poeTokenY,
	})
	require.NoError(t, err)

	require.Equal(t, mustBig("19940000000"), res.TokenAmountOut.Amount)
	require.Equal(t, poeTokenY, res.Fee.Token)
	require.Equal(t, mustBig("60000000"), res.Fee.Amount)

	// partial fill: some of the requested amountIn must be returned as remaining.
	require.True(t, res.RemainingTokenAmountIn.Amount.Sign() > 0)
	consumed := new(big.Int).Sub(amountIn, res.RemainingTokenAmountIn.Amount)
	require.Equal(t, mustBig("10499475576837990"), consumed)
}

func TestPoolSimulator_CalcAmountOut_YtoX_FeeInTokenY(t *testing.T) {
	sim := newTestPoolSimulator(t)

	res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: poeTokenY, Amount: mustBig("5000000000")},
		TokenOut:      poeTokenX,
	})
	require.NoError(t, err)

	require.Equal(t, mustBig("2492500000000000"), res.TokenAmountOut.Amount)
	require.Equal(t, poeTokenY, res.Fee.Token) // fee always denominated in tokenY, even for Y->X
	require.Equal(t, mustBig("15000000"), res.Fee.Amount)
}

func TestPoolSimulator_UpdateBalance_ConsumesActualIn(t *testing.T) {
	sim := newTestPoolSimulator(t)

	amountIn := mustBig("100000000000000000")
	res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: poeTokenX, Amount: amountIn},
		TokenOut:      poeTokenY,
	})
	require.NoError(t, err)

	consumed := new(big.Int).Sub(amountIn, res.RemainingTokenAmountIn.Amount)

	sim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: poeTokenX, Amount: consumed},
		TokenAmountOut: pool.TokenAmount{Token: poeTokenY, Amount: res.TokenAmountOut.Amount},
		Fee:            *res.Fee,
	})

	require.Equal(t, new(big.Int).Add(mustBig("10000000000000000000"), consumed), sim.Info.Reserves[0])
	require.Equal(t, new(big.Int).Sub(mustBig("20000000000"), res.TokenAmountOut.Amount), sim.Info.Reserves[1])
}

func TestPoolSimulator_CalcAmountIn_MatchesCalcAmountOut(t *testing.T) {
	sim := newTestPoolSimulator(t)

	desiredOut := mustBig("18991")
	inRes, err := sim.CalcAmountIn(pool.CalcAmountInParams{
		TokenAmountOut: pool.TokenAmount{Token: poeTokenY, Amount: desiredOut},
		TokenIn:        poeTokenX,
	})
	require.NoError(t, err)

	outRes, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: poeTokenX, Amount: inRes.TokenAmountIn.Amount},
		TokenOut:      poeTokenY,
	})
	require.NoError(t, err)

	require.True(t, outRes.TokenAmountOut.Amount.Cmp(desiredOut) >= 0)
}

func TestPoolSimulator_ExpiredOracle(t *testing.T) {
	sim := newTestPoolSimulator(t)
	sim.expiry = uint64(time.Now().Unix()) - 1

	_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: poeTokenX, Amount: mustBig("1000000")},
		TokenOut:      poeTokenY,
	})
	require.ErrorIs(t, err, ErrExpiredOracle)
}
