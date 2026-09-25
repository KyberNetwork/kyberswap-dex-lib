package prop

import (
	"math/big"
	"os"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testdata/snapshot_ethereum.hex is a real KipseliPropLens revert from
// ethereum's pAMM venue (router 0x5cdb…588c, dest = executor) under a live
// Titan override, probing WETH->USDC [1, 1000] WETH and USDC->WETH
// [2500, 10M] USDC.
func loadEthereumSnapshot(t *testing.T) []byte {
	t.Helper()
	raw, err := os.ReadFile("testdata/snapshot_ethereum.hex")
	require.NoError(t, err)
	data, err := hexutil.Decode(strings.TrimSpace(string(raw)))
	require.NoError(t, err)
	return data
}

func TestDecodeSnapshot_LiveEthereum(t *testing.T) {
	s, err := decodeSnapshot(loadEthereumSnapshot(t))
	require.NoError(t, err)

	// the migrated quoter, not the stale lens still configured before
	assert.Equal(t, common.HexToAddress("0xC9A956b5196DFe110DEBE8781857Bc8B97d32091"), s.Quoter)
	assert.Equal(t, common.HexToAddress("0x054F0377e07d2F460151F935Dffc4D880017E63a"), s.SwapImpl)
	assert.Equal(t, common.HexToAddress("0xC841C89609656A39d9365fd0bAB4fD8D59B99155"), s.Wallet)
	assert.Equal(t, common.HexToAddress("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"), s.QuoteToken)
	assert.Len(t, s.ListedTokens, 2)

	require.Len(t, s.Balances, 2)
	require.Len(t, s.AmountsOut, 2)
	assert.Positive(t, s.AmountsOut[0][0].Sign(), "1 WETH must quote under the override")
	assert.Zero(t, s.AmountsOut[0][1].Sign(), "1000 WETH is past the quoter's curve")
	// SwapImpl caps output at the wallet's balance: 10M USDC buys exactly it
	assert.Equal(t, s.Balances[0], s.AmountsOut[1][1])
	assert.Zero(t, s.Caps[1].Cmp(new(big.Int).SetBytes(common.MaxHash[:])), "quote token is uncapped")
}

// Any other revert (a bad router, an out-of-gas, a provider error blob) must
// fail loudly rather than be misread as an empty snapshot.
func TestDecodeSnapshot_RejectsForeignRevert(t *testing.T) {
	data := loadEthereumSnapshot(t)
	data[0] ^= 0xff
	_, err := decodeSnapshot(data)
	assert.ErrorIs(t, err, ErrUnexpectedLensRevert)

	_, err = decodeSnapshot([]byte{0x01})
	assert.ErrorIs(t, err, ErrUnexpectedLensRevert)
}

func TestLensConstructorArgs(t *testing.T) {
	router, dest := common.HexToAddress("0x01"), common.HexToAddress("0x02")

	listing, err := lensABI.Pack("", router, dest, []common.Address{}, [][]*big.Int{})
	require.NoError(t, err)
	assert.Len(t, listing, 6*32, "router, dest, two offsets, two empty lengths")

	_, err = lensABI.Pack("", router, dest, []common.Address{router, dest},
		[][]*big.Int{{big.NewInt(1)}, {big.NewInt(2), big.NewInt(3)}})
	require.NoError(t, err)
}
