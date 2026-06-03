package bouncetech

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

const (
	testUSDC = "0x0000000000000000000000000000000000000001"
	testLT   = "0x0000000000000000000000000000000000000002"
)

func TestCalcAmountOutRedeemUsesGrossAmountForReserveAndState(t *testing.T) {
	s := newBounceTechTestSimulator(t, "1000000000")
	amountIn := mustBig("100000000000000000000") // 100 LT

	res, err := s.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testLT, Amount: amountIn},
		TokenOut:      testUSDC,
	})
	require.NoError(t, err)
	require.Equal(t, "97000000", res.TokenAmountOut.Amount.String())
	require.Equal(t, "3000000", res.Fee.Amount.String())

	s.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: testLT, Amount: amountIn},
		TokenAmountOut: *res.TokenAmountOut,
		Fee:            *res.Fee,
		SwapInfo:       res.SwapInfo,
	})

	require.Equal(t, "900000000", s.Info.Reserves[0].String())
	require.Equal(t, "900000000000000000000", s.Info.Reserves[1].String())
}

func TestCalcAmountOutRedeemRejectsWhenGrossAmountExceedsBaseBalance(t *testing.T) {
	s := newBounceTechTestSimulator(t, "99000000")

	_, err := s.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testLT, Amount: mustBig("100000000000000000000")},
		TokenOut:      testUSDC,
	})
	require.ErrorIs(t, err, ErrInsufficientBalance)
}

func TestCalcAmountOutMintRejectsBelowMinTransactionSize(t *testing.T) {
	s := newBounceTechTestSimulator(t, "1000000000")

	_, err := s.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testUSDC, Amount: big.NewInt(9_999_999)},
		TokenOut:      testLT,
	})
	require.ErrorIs(t, err, ErrBelowMinAmount)
}

func newBounceTechTestSimulator(t *testing.T, baseBalance string) *PoolSimulator {
	t.Helper()

	extraBytes, err := json.Marshal(Extra{
		ExchangeRate:       uint256.NewInt(1e18),
		RedemptionFee:      uint256.NewInt(1e16), // 1%
		TargetLeverage:     uint256.NewInt(3e18),
		MinTransactionSize: uint256.NewInt(10_000_000),
	})
	require.NoError(t, err)

	s, err := NewPoolSimulator(entity.Pool{
		Address:  testLT,
		Exchange: DexType,
		Type:     DexType,
		Reserves: entity.PoolReserves{
			baseBalance,
			"1000000000000000000000",
		},
		Tokens: []*entity.PoolToken{
			{Address: testUSDC, Swappable: true},
			{Address: testLT, Swappable: true},
		},
		Extra: string(extraBytes),
	})
	require.NoError(t, err)
	return s
}

func mustBig(s string) *big.Int {
	v, ok := new(big.Int).SetString(s, 10)
	if !ok {
		panic("invalid big int")
	}
	return v
}
