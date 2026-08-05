package tokentax

import (
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

func TestSupportsChain(t *testing.T) {
	t.Parallel()
	assert.True(t, SupportsChain(valueobject.ChainIDEthereum))
	assert.True(t, SupportsChain(valueobject.ChainIDBSC))
	assert.False(t, SupportsChain(valueobject.ChainID(999999)))
}

func TestBuildProbes(t *testing.T) {
	t.Parallel()

	// No base-token knowledge needed at all: the self-pair probes are built purely from the
	// pool's own two tokens, regardless of what they are.
	probes := buildProbes("0xaaaa", "0xbbbb")
	assert.Equal(t, []probe{
		{candidate: "0xaaaa", base: "0xbbbb"},
		{candidate: "0xbbbb", base: "0xaaaa"},
	}, probes)
}

func TestNewTrackerAddCalls(t *testing.T) {
	t.Parallel()

	request := new(ethrpc.Client).NewRequest()
	tr := NewTracker(valueobject.ChainIDEthereum, "0xtoken0", "0xtoken1", TaxInfo{})
	tr.AddCalls(request)

	expected := 2 * len(detectorsFor(valueobject.ChainIDEthereum)) // 2 self-pair probes per instance
	require.Len(t, request.Calls, expected)
	for _, call := range request.Calls {
		assert.Equal(t, methodValidate, call.Method)
	}
}

func TestTrackerResolve(t *testing.T) {
	t.Parallel()

	t.Run("first successful candidate wins", func(t *testing.T) {
		tr := &Tracker{
			instances: []detectorInstance{{address: "0xuniswap"}},
			probes: []probe{
				{candidate: "0xagent", base: "0xweth"},
				{candidate: "0xagent", base: "0xusdt"},
				{candidate: "0xweth", base: "0xagent"},
			},
			calls: []int{0, 1, 2},
			results: []validateResult{
				{}, // PairLookupFailed, marked failed below
				{Fees: tokenFees{BuyFeeBps: big.NewInt(100), SellFeeBps: big.NewInt(200)}},
				{Fees: tokenFees{BuyFeeBps: big.NewInt(9999), SellFeeBps: big.NewInt(9999)}},
			},
		}
		result := tr.Resolve(&ethrpc.Response{Result: []bool{false, true, true}})
		assert.Equal(t, TaxInfo{
			Token:      "0xagent",
			BuyTaxBps:  uint256.NewInt(100),
			SellTaxBps: uint256.NewInt(200),
			Checked:    true,
		}, result)
	})

	t.Run("no candidate pair found marks pool unsupported", func(t *testing.T) {
		tr := &Tracker{
			instances: []detectorInstance{{address: "0xuniswap"}},
			probes: []probe{
				{candidate: "0xagent", base: "0xweth"},
				{candidate: "0xweth", base: "0xagent"},
			},
			calls:   []int{0, 1},
			results: []validateResult{{}, {}},
		}
		result := tr.Resolve(&ethrpc.Response{Result: []bool{false, false}})
		assert.Equal(t, TaxInfo{Checked: true}, result)
	})

	t.Run("total probe failure keeps previously known tax instead of wiping it", func(t *testing.T) {
		previous := TaxInfo{
			Token: "0xagent", BuyTaxBps: uint256.NewInt(100), SellTaxBps: uint256.NewInt(100), Checked: true,
		}
		tr := &Tracker{
			instances: []detectorInstance{{address: "0xuniswap"}},
			probes:    []probe{{candidate: "0xagent", base: "0xweth"}},
			calls:     []int{0},
			results:   []validateResult{{}},
			previous:  previous,
		}
		// The call fails this cycle (e.g. transient RPC error), not "no pair exists": must not
		// report Token == "" here, or the caller would mark this token permanently unsupported.
		result := tr.Resolve(&ethrpc.Response{Result: []bool{false}})
		assert.Equal(t, previous, result)
	})

	t.Run("zero fee result is still a definitive check", func(t *testing.T) {
		tr := &Tracker{
			instances: []detectorInstance{{address: "0xuniswap"}},
			probes:    []probe{{candidate: "0xagent", base: "0xweth"}},
			calls:     []int{0},
			results: []validateResult{
				{Fees: tokenFees{BuyFeeBps: big.NewInt(0), SellFeeBps: big.NewInt(0)}},
			},
		}
		result := tr.Resolve(&ethrpc.Response{Result: []bool{true}})
		assert.Equal(t, TaxInfo{
			Token:      "0xagent",
			BuyTaxBps:  uint256.NewInt(0),
			SellTaxBps: uint256.NewInt(0),
			Checked:    true,
		}, result)
	})

	t.Run("nonzero result wins even if a zero result came first", func(t *testing.T) {
		// Some tokens only charge tax through their own recognized pair; an earlier-tried probe
		// can land on some other, unrecognized pair and report a false 0%.
		tr := &Tracker{
			instances: []detectorInstance{{address: "0xuniswap"}},
			probes: []probe{
				{candidate: "0xagent", base: "0xweth"},
				{candidate: "0xagent", base: "0xvirtual"},
			},
			calls: []int{0, 1},
			results: []validateResult{
				{Fees: tokenFees{BuyFeeBps: big.NewInt(0), SellFeeBps: big.NewInt(0)}},
				{Fees: tokenFees{BuyFeeBps: big.NewInt(100), SellFeeBps: big.NewInt(100)}},
			},
		}
		result := tr.Resolve(&ethrpc.Response{Result: []bool{true, true}})
		assert.Equal(t, TaxInfo{
			Token:      "0xagent",
			BuyTaxBps:  uint256.NewInt(100),
			SellTaxBps: uint256.NewInt(100),
			Checked:    true,
		}, result)
	})

	t.Run("basic (PancakeSwap-shaped) instance decodes correctly", func(t *testing.T) {
		tr := &Tracker{
			instances: []detectorInstance{{address: "0xpancake", basic: true}},
			probes:    []probe{{candidate: "0xagent", base: "0xwbnb"}},
			calls:     []int{0},
			basicResults: []validateBasicResult{
				{Fees: tokenFeesBasic{BuyFeeBps: big.NewInt(100), SellFeeBps: big.NewInt(1000)}},
			},
		}
		result := tr.Resolve(&ethrpc.Response{Result: []bool{true}})
		assert.Equal(t, TaxInfo{
			Token:      "0xagent",
			BuyTaxBps:  uint256.NewInt(100),
			SellTaxBps: uint256.NewInt(1000),
			Checked:    true,
		}, result)
	})
}

func TestValidateABIDecode_RealOnChainResponse(t *testing.T) {
	t.Parallel()

	// Captured via eth_call on Ethereum mainnet against USDT/WETH: buyFeeBps=0, sellFeeBps=0.
	noTax, err := hex.DecodeString(
		"0000000000000000000000000000000000000000000000000000000000000000" +
			"0000000000000000000000000000000000000000000000000000000000000000" +
			"0000000000000000000000000000000000000000000000000000000000000000" +
			"0000000000000000000000000000000000000000000000000000000000000000" +
			"0000000000000000000000000000000000000000000000000000000000000000")
	require.NoError(t, err)
	var noTaxResult validateResult
	require.NoError(t, detectorABI.UnpackIntoInterface(&noTaxResult, methodValidate, noTax))
	assert.Equal(t, 0, noTaxResult.Fees.BuyFeeBps.Cmp(big.NewInt(0)))
	assert.Equal(t, 0, noTaxResult.Fees.SellFeeBps.Cmp(big.NewInt(0)))

	// Captured via eth_call on Base against REPPO/VIRTUAL: buyFeeBps=100, sellFeeBps=100 (1% each).
	withTax, err := hex.DecodeString(
		"0000000000000000000000000000000000000000000000000000000000000064" +
			"0000000000000000000000000000000000000000000000000000000000000064" +
			"0000000000000000000000000000000000000000000000000000000000000000" +
			"0000000000000000000000000000000000000000000000000000000000000000" +
			"0000000000000000000000000000000000000000000000000000000000000000")
	require.NoError(t, err)
	var withTaxResult validateResult
	require.NoError(t, detectorABI.UnpackIntoInterface(&withTaxResult, methodValidate, withTax))
	assert.Equal(t, 0, withTaxResult.Fees.BuyFeeBps.Cmp(big.NewInt(100)))
	assert.Equal(t, 0, withTaxResult.Fees.SellFeeBps.Cmp(big.NewInt(100)))
	assert.False(t, withTaxResult.Fees.SellReverted)

	// Captured via eth_call on BSC against PancakeSwap's detector for a known four.meme token
	// paired with WBNB: buyFeeBps=100 (1%), sellFeeBps=1000 (10%). PancakeSwap's TokenFees struct
	// only has these 2 fields, so this must decode with detectorBasicABI, not detectorABI.
	basicWithTax, err := hex.DecodeString(
		"0000000000000000000000000000000000000000000000000000000000000064" +
			"00000000000000000000000000000000000000000000000000000000000003e8")
	require.NoError(t, err)
	var basicResult validateBasicResult
	require.NoError(t, detectorBasicABI.UnpackIntoInterface(&basicResult, methodValidate, basicWithTax))
	assert.Equal(t, 0, basicResult.Fees.BuyFeeBps.Cmp(big.NewInt(100)))
	assert.Equal(t, 0, basicResult.Fees.SellFeeBps.Cmp(big.NewInt(1000)))
}
