package pools

import (
	"encoding/binary"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	ekubomath "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/v3/math"
)

const onePercentVe33Fee = uint64((uint64(1) << 63) / 50)

func newVe33FullRangePool(t *testing.T, swapFee uint64) *Ve33Pool {
	t.Helper()

	key := NewPoolKey(
		common.HexToAddress("0x1"),
		common.HexToAddress("0x2"),
		NewPoolConfig(
			common.HexToAddress("0xd100000000000000000000000000000000000000"),
			0,
			NewFullRangePoolTypeConfig(),
		),
	)
	underlying := NewFullRangePool(
		key,
		NewFullRangePoolState(
			NewFullRangePoolSwapState(new(uint256.Int).Lsh(uint256.NewInt(1), 128)),
			uint256.NewInt(1_000_000),
		),
	)
	return NewVe33Pool(underlying, swapFee)
}

func TestVe33PoolExactInputTakesFeeFromOutput(t *testing.T) {
	t.Parallel()

	pool := newVe33FullRangePool(t, onePercentVe33Fee)
	amount := uint256.NewInt(1_000)
	quote, err := pool.Quote(amount, false)
	require.NoError(t, err)

	require.Equal(t, "989", quote.CalculatedAmount.String())
	require.Equal(t, pool.GetKey().Extension(), *quote.SwapInfo.Forward)
	firstForward := quote.SwapInfo.Forward
	quote, err = pool.Quote(amount, false)
	require.NoError(t, err)
	require.Same(t, firstForward, quote.SwapInfo.Forward)
}

func TestVe33PoolExactOutputGrossesUpInput(t *testing.T) {
	t.Parallel()

	withoutFee := newVe33FullRangePool(t, 0)
	underlyingQuote, err := withoutFee.Quote(new(uint256.Int).Neg(uint256.NewInt(500)), false)
	require.NoError(t, err)

	withFee := newVe33FullRangePool(t, onePercentVe33Fee)
	quote, err := withFee.Quote(new(uint256.Int).Neg(uint256.NewInt(500)), false)
	require.NoError(t, err)

	expected, err := ekubomath.AmountBeforeFee(underlyingQuote.CalculatedAmount, onePercentVe33Fee)
	require.NoError(t, err)
	require.Equal(t, expected, quote.CalculatedAmount)
}

func TestVe33PoolAppliesVoteWeightEvent(t *testing.T) {
	t.Parallel()

	pool := newVe33FullRangePool(t, 0)
	poolID, err := pool.GetKey().NumId()
	require.NoError(t, err)

	data := make([]byte, voteWeightAppliedEncodedDataLength)
	copy(data[voteWeightAppliedPoolIDOffset:voteWeightAppliedPoolIDOffset+abiWordSize], poolID)
	binary.BigEndian.PutUint64(
		data[voteWeightAppliedSwapFeeOffset:voteWeightAppliedSwapFeeOffset+8],
		onePercentVe33Fee,
	)
	require.NoError(t, pool.ApplyEvent(EventVoteWeightApplied, data, 0))

	state := pool.GetState().(*Ve33PoolState[any])
	require.Equal(t, onePercentVe33Fee, state.SwapFee)
}
