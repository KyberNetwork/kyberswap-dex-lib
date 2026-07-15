package virtual

import (
	"math/big"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"

	tokentax "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v2/token-tax"
)

func TestTrackerTaxInfo(t *testing.T) {
	t.Parallel()

	t.Run("registered pool", func(t *testing.T) {
		tracker := tracker{
			tokenAddress:    "0xagent",
			isLiquidityPool: true,
			buyTaxBps:       big.NewInt(100), sellTaxBps: big.NewInt(1000),
		}
		result := resolveTracker(&tracker, []bool{true, true, true})
		assert.Equal(t, tokentax.TaxInfo{
			Protocol:   Protocol,
			Token:      "0xagent",
			BuyTaxBps:  uint256.NewInt(100),
			SellTaxBps: uint256.NewInt(1000),
			Checked:    true,
		}, result)
	})

	t.Run("unregistered pool keeps protocol and token", func(t *testing.T) {
		tracker := tracker{
			tokenAddress: "0xagent",
			buyTaxBps:    big.NewInt(100), sellTaxBps: big.NewInt(1000),
		}
		result := resolveTracker(&tracker, []bool{true, true, true})
		assert.Equal(t, tokentax.TaxInfo{
			Protocol: Protocol,
			Token:    "0xagent",
			Checked:  true,
		}, result)
	})

	t.Run("partial tax read keeps successful side", func(t *testing.T) {
		tracker := tracker{
			tokenAddress:    "0xagent",
			isLiquidityPool: true,
			buyTaxBps:       big.NewInt(100),
		}
		result := resolveTracker(&tracker, []bool{true, true, false})
		assert.Equal(t, tokentax.TaxInfo{
			Protocol:  Protocol,
			Token:     "0xagent",
			BuyTaxBps: uint256.NewInt(100),
			Checked:   true,
		}, result)
	})

	t.Run("reverted tax methods mark token unsupported", func(t *testing.T) {
		result := resolveTracker(&tracker{}, []bool{false, false, false})
		assert.Equal(t, tokentax.TaxInfo{Checked: true}, result)
	})

	t.Run("is liquidity pool alone identifies virtual token", func(t *testing.T) {
		tracker := tracker{
			tokenAddress:    "0xagent",
			isLiquidityPool: true,
		}
		result := resolveTracker(&tracker, []bool{true, false, false})
		assert.Equal(t, tokentax.TaxInfo{
			Protocol: Protocol,
			Token:    "0xagent",
			Checked:  true,
		}, result)
	})

	t.Run("reverted refresh preserves known virtual token", func(t *testing.T) {
		previous := tokentax.TaxInfo{
			Protocol: Protocol, Token: "0xagent", BuyTaxBps: uint256.NewInt(100), Checked: true,
		}
		result := resolveTracker(&tracker{previous: previous}, []bool{false, false, false})
		assert.Equal(t, previous, result)
	})
}

func TestTrackerAddCalls(t *testing.T) {
	t.Parallel()

	request := new(ethrpc.Client).NewRequest()
	NewTracker(
		"0x0000000000000000000000000000000000000001", "0xagent", "0xfactory", tokentax.TaxInfo{},
	).AddCalls(request)

	assert.Len(t, request.Calls, 3)
	assert.Equal(t, "0xagent", request.Calls[0].Target)
	assert.Equal(t, methodBuyTax, request.Calls[1].Method)
	assert.Equal(t, methodSellTax, request.Calls[2].Method)
}

func TestTrackerAddCallsProjectTaxFactory(t *testing.T) {
	t.Parallel()

	request := new(ethrpc.Client).NewRequest()
	NewTracker(
		"0x0000000000000000000000000000000000000001", "0xagent",
		"0x8bcEaA40B9AcdfAedF85AdF4FF01F5Ad6517937f", tokentax.TaxInfo{},
	).AddCalls(request)

	assert.Len(t, request.Calls, 3)
	assert.Equal(t, methodProjectBuyTax, request.Calls[1].Method)
	assert.Equal(t, methodProjectSellTax, request.Calls[2].Method)
}

func resolveTracker(tracker *tracker, results []bool) tokentax.TaxInfo {
	request := new(ethrpc.Client).NewRequest()
	tracker.AddCalls(request)
	return tracker.Resolve(&ethrpc.Response{Result: results})
}
