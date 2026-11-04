package dexv2

import (
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/swaplimit"
)

func testLiquiditySimulator() *PoolSimulator {
	p := &PoolSimulator{staticExtra: StaticExtra{TrackLiquidityLimits: true}, extra: Extra{SupplyOnlySettlement: true}, liquidityKeys: [2]string{"vault:token0:", "vault:token1:"}}
	for i := range 2 {
		s := &p.liquidityLimits[i]
		s.Withdraw.SetUint64(1000)
		s.Borrow.SetUint64(1000)
		s.Debt.SetUint64(1000)
		s.Cash.SetUint64(1000)
		s.Growth.SetUint64(10000)
		s.Allowed = [4]bool{true, true, true, true}
		s.DepositSafe = true
	}
	return p
}

func TestFloatAggregateEndpointLimits(t *testing.T) {
	for _, input := range []bool{true, false} {
		p := testLiquiditySimulator()
		for i := range 2 {
			p.liquidityLimits[i].Float.SetUint64(100)
			p.liquidityLimits[i].Withdraw.SetUint64(100)
		}
		if input {
			p.liquidityLimits[0].Allowed[0] = false
		}
		limit := swaplimit.NewInventory(DexType, p.CalculateLimit())
		usage, err := p.quoteLiquidity(limit, 0, 1, D3_MODULE, uint256.NewInt(75), uint256.NewInt(75), uint256.NewInt(0))
		require.NoError(t, err)
		p.applyLiquidity(usage, limit)
		_, err = p.quoteLiquidity(limit, 0, 1, D3_MODULE, uint256.NewInt(75), uint256.NewInt(75), uint256.NewInt(0))
		require.ErrorIs(t, err, ErrUnsupportedLiquidityState)
	}
}

func TestFloatAggregateAcrossPools(t *testing.T) {
	p := testLiquiditySimulator()
	p.liquidityLimits[1].Float.SetUint64(100)
	p.liquidityLimits[1].Withdraw.SetUint64(100)
	other := *p
	limit := swaplimit.NewInventory(DexType, p.CalculateLimit())
	usage, err := p.quoteLiquidity(limit, 0, 1, D3_MODULE, uint256.NewInt(75), uint256.NewInt(75), uint256.NewInt(0))
	require.NoError(t, err)
	p.applyLiquidity(usage, limit)
	_, err = other.quoteLiquidity(limit, 0, 1, D3_MODULE, uint256.NewInt(75), uint256.NewInt(75), uint256.NewInt(0))
	require.ErrorIs(t, err, ErrUnsupportedLiquidityState)
}

func TestFloatSkipsPausedSupplyAndWithdraw(t *testing.T) {
	p := testLiquiditySimulator()
	p.liquidityLimits[0].Float.SetUint64(500)
	p.liquidityLimits[0].Allowed[0] = false
	p.liquidityLimits[1].Float.SetUint64(500)
	p.liquidityLimits[1].Allowed[1] = false
	before := p.liquidityLimits
	_, err := p.quoteLiquidity(nil, 0, 1, D3_MODULE, uint256.NewInt(75), uint256.NewInt(75), uint256.NewInt(0))
	require.NoError(t, err)
	require.Equal(t, before, p.liquidityLimits)
}

func TestSupplyOnlyD4ChecksConversionEvenWithFloat(t *testing.T) {
	p := testLiquiditySimulator()
	for i := range 2 {
		p.liquidityLimits[i].Float.SetUint64(1000)
	}
	_, err := p.quoteLiquidity(nil, 0, 1, 4, uint256.NewInt(75), uint256.NewInt(75), uint256.NewInt(1))
	require.NoError(t, err)
	p.liquidityLimits[0].Allowed[1] = false
	_, err = p.quoteLiquidity(nil, 0, 1, 4, uint256.NewInt(75), uint256.NewInt(75), uint256.NewInt(1))
	require.ErrorIs(t, err, ErrUnsupportedLiquidityState)
	p.liquidityLimits[0].Allowed[1] = true
	p.liquidityLimits[0].Withdraw.SetUint64(50)
	_, err = p.quoteLiquidity(nil, 0, 1, 4, uint256.NewInt(75), uint256.NewInt(75), uint256.NewInt(1))
	require.ErrorIs(t, err, ErrUnsupportedLiquidityState)
}

func TestWithdrawConsumesBorrowHeadroom(t *testing.T) {
	p := testLiquiditySimulator()
	p.liquidityLimits[1].Borrow.SetUint64(300)
	usage, err := p.quoteLiquidity(nil, 0, 1, D3_MODULE, uint256.NewInt(350), uint256.NewInt(350), uint256.NewInt(0))
	require.NoError(t, err)
	p.applyLiquidity(usage, nil)
	require.True(t, p.liquidityLimits[1].Borrow.IsZero())
	_, err = p.quoteLiquidity(nil, 0, 1, 4, uint256.NewInt(150), uint256.NewInt(150), uint256.NewInt(0))
	require.ErrorIs(t, err, ErrUnsupportedLiquidityState)
}

func TestConversionRechecksPreviousOutputReservation(t *testing.T) {
	p := testLiquiditySimulator()
	p.liquidityLimits[0].Withdraw.SetUint64(100)
	usage, err := p.quoteLiquidity(nil, 1, 0, D3_MODULE, uint256.NewInt(75), uint256.NewInt(75), uint256.NewInt(0))
	require.NoError(t, err)
	p.applyLiquidity(usage, nil)
	_, err = p.quoteLiquidity(nil, 0, 1, 4, uint256.NewInt(75), uint256.NewInt(75), uint256.NewInt(0))
	require.ErrorIs(t, err, ErrUnsupportedLiquidityState)
}

func TestLiquidityGrowthAndRoundingBound(t *testing.T) {
	p := testLiquiditySimulator()
	p.liquidityLimits[0].Growth.SetUint64(100)
	p.liquidityLimits[0].RoundingBuffer.SetUint64(10)
	limit := swaplimit.NewInventory(DexType, p.CalculateLimit())
	_, err := p.quoteLiquidity(limit, 0, 1, D3_MODULE, uint256.NewInt(10), uint256.NewInt(20), uint256.NewInt(0))
	require.ErrorIs(t, err, ErrUnsupportedLiquidityState)
	usage, err := p.quoteLiquidity(limit, 0, 1, D3_MODULE, uint256.NewInt(60), uint256.NewInt(20), uint256.NewInt(0))
	require.NoError(t, err)
	p.applyLiquidity(usage, limit)
	other := testLiquiditySimulator()
	other.liquidityLimits[0].Growth.SetUint64(100)
	other.liquidityLimits[0].RoundingBuffer.SetUint64(10)
	_, err = other.quoteLiquidity(limit, 0, 1, D3_MODULE, uint256.NewInt(41), uint256.NewInt(20), uint256.NewInt(0))
	require.ErrorIs(t, err, ErrUnsupportedLiquidityState)
	var tooLarge uint256.Int
	tooLarge.SetOne().Lsh(&tooLarge, 80)
	_, err = p.quoteLiquidity(nil, 0, 1, D3_MODULE, &tooLarge, uint256.NewInt(20), uint256.NewInt(0))
	require.ErrorIs(t, err, ErrUnsupportedLiquidityState)
}

func TestUnsupportedSettlementAndAggregateLegBound(t *testing.T) {
	p := testLiquiditySimulator()
	p.extra.SupplyOnlySettlement = false
	_, err := p.quoteLiquidity(nil, 0, 1, 4, uint256.NewInt(75), uint256.NewInt(75), uint256.NewInt(1))
	require.ErrorIs(t, err, ErrUnsupportedLiquidityState)
	p = testLiquiditySimulator()
	for i := range 2 {
		s := &p.liquidityLimits[i]
		s.Float.SetAllOne()
		s.Withdraw.SetAllOne()
		s.Cash.SetAllOne()
		s.Borrow.SetAllOne()
		s.Growth.SetAllOne()
	}
	var amount uint256.Int
	amount.SetOne().Lsh(&amount, 78)
	for range 2 {
		usage, e := p.quoteLiquidity(nil, 0, 1, D3_MODULE, &amount, &amount, uint256.NewInt(0))
		require.NoError(t, e)
		p.applyLiquidity(usage, nil)
	}
	_, err = p.quoteLiquidity(nil, 0, 1, D3_MODULE, &amount, &amount, uint256.NewInt(0))
	require.ErrorIs(t, err, ErrUnsupportedLiquidityState)
}

func TestSubBaseDepositsShareHeadroom(t *testing.T) {
	p := testLiquiditySimulator()
	p.liquidityLimits[0].DepositSafe = false
	p.liquidityLimits[0].DepositHeadroom.SetUint64(100)
	p.liquidityLimits[0].Withdraw.SetUint64(100)
	limit := swaplimit.NewInventory(DexType, p.CalculateLimit())
	usage, err := p.quoteLiquidity(limit, 0, 1, D3_MODULE, uint256.NewInt(75), uint256.NewInt(75), uint256.NewInt(0))
	require.NoError(t, err)
	p.applyLiquidity(usage, limit)
	require.Equal(t, uint64(25), p.liquidityLimits[0].DepositHeadroom.Uint64())
	other := testLiquiditySimulator()
	other.liquidityLimits[0].DepositSafe = false
	other.liquidityLimits[0].DepositHeadroom.SetUint64(100)
	usage, err = other.quoteLiquidity(limit, 0, 1, D3_MODULE, uint256.NewInt(75), uint256.NewInt(75), uint256.NewInt(0))
	require.NoError(t, err)
	other.applyLiquidity(usage, limit)
	require.Zero(t, limit.GetLimit("vault:token0:withdraw").Sign())
	require.Zero(t, limit.GetLimit("vault:token0:depositHeadroom").Sign())
	_, err = p.quoteLiquidity(limit, 1, 0, D3_MODULE, uint256.NewInt(1), uint256.NewInt(1), uint256.NewInt(0))
	require.ErrorIs(t, err, ErrUnsupportedLiquidityState)
}
