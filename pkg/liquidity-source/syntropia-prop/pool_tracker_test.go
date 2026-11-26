package syntropiaprop

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	orderbook "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/order-book"
)

func TestLensABIDecode(t *testing.T) {
	t.Parallel()
	book := Book{
		Base:                common.HexToAddress(testWETH),
		Quote:               common.HexToAddress(testUSDC),
		AskCumulativeLevels: fixtureAsks,
		BidCumulativeLevels: fixtureBids,
		BalanceBase:         fixtureBalanceWETH,
		BalanceQuote:        fixtureBalanceUSDC,
	}
	encoded, err := lensABI.Methods["getOrderbook"].Outputs.Pack(book, big.NewInt(62603623), big.NewInt(1795711106))
	require.NoError(t, err)
	var decoded getOrderbookResult
	require.NoError(t, lensABI.UnpackIntoInterface(&decoded, "getOrderbook", encoded))
	require.Equal(t, book, decoded.Book)
	require.Equal(t, uint64(62603623), decoded.BlockNumber.Uint64())
	require.Equal(t, int64(1795711106), decoded.BlockTimestamp.Int64())
}

func TestLevelsToOrderbook(t *testing.T) {
	t.Parallel()
	probes := []Level{
		{big.NewInt(10), big.NewInt(100)},
		{big.NewInt(10), big.NewInt(150)}, // no more input than the level before: dropped
		{big.NewInt(30), big.NewInt(200)},
	}
	levels, err := levelsToOrderbook(probes, 0, 0, 9_900)
	require.NoError(t, err)
	require.Equal(t, []orderbook.Level{{0, 0}, {10, 9.9}, {20, 4.95}}, levels)
	require.Equal(t, int64(100), probes[0].AmountOut.Int64(), "the lens's data must not be modified")

	// An inactive pair has no levels on either side, and must not quote.
	levels, err = levelsToOrderbook(nil, 18, 6, 9_990)
	require.NoError(t, err)
	require.Nil(t, levels)

	_, err = levelsToOrderbook([]Level{{AmountIn: big.NewInt(1)}}, 0, 0, 10_000)
	require.ErrorIs(t, err, ErrInvalidBook)
}

func TestConfigValidate(t *testing.T) {
	t.Parallel()
	cfg := Config{Pool: "0x05ca2d5786703B34fCa134bc64ae44392E753713", Lens: "0x5C60e442A1592e52bF96646B05c8279A50f2131a", Buffer: 9_990}
	require.NoError(t, cfg.validate())
	require.Equal(t, testPool, cfg.Pool)
	require.Equal(t, testLens, cfg.Lens)

	for _, bad := range []Config{
		{Lens: testLens, Buffer: 9_990},
		{Pool: testPool, Lens: "0x0000000000000000000000000000000000000000", Buffer: 9_990},
		{Pool: testPool, Lens: testLens},
		{Pool: testPool, Lens: testLens, Buffer: 10_001},
	} {
		require.ErrorIs(t, bad.validate(), ErrInvalidConfig)
	}
}
