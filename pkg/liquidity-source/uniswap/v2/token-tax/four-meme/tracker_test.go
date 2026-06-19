package fourmeme

import (
	"math/big"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"

	tokentax "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v2/token-tax"
)

func TestTrackerTaxInfo(t *testing.T) {
	t.Parallel()
	poolAddress := "0x9053a8607902b8a3e971f2fae2562c4e2aa64b05"

	t.Run("canonical pair", func(t *testing.T) {
		tracker := tracker{
			poolAddress:  poolAddress,
			tokenAddress: "0xagent",
			pairAddress:  common.HexToAddress(poolAddress),
			buyTaxPct:    big.NewInt(1),
			sellTaxPct:   big.NewInt(10),
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

	t.Run("different pair is unsupported", func(t *testing.T) {
		tracker := tracker{
			poolAddress: poolAddress,
			pairAddress: common.HexToAddress("0xdead"),
		}
		result := resolveTracker(&tracker, []bool{true, true, true})
		assert.Equal(t, tokentax.TaxInfo{Checked: true}, result)
	})

	t.Run("partial tax read keeps successful side", func(t *testing.T) {
		tracker := tracker{
			poolAddress:  poolAddress,
			tokenAddress: "0xagent",
			pairAddress:  common.HexToAddress(poolAddress),
			buyTaxPct:    big.NewInt(1),
		}
		result := resolveTracker(&tracker, []bool{true, true, false})
		assert.Equal(t, tokentax.TaxInfo{
			Protocol:  Protocol,
			Token:     "0xagent",
			BuyTaxBps: uint256.NewInt(100),
			Checked:   true,
		}, result)
	})

	t.Run("both tax methods reverted marks token unsupported", func(t *testing.T) {
		tracker := tracker{
			poolAddress:  poolAddress,
			tokenAddress: "0xagent",
			pairAddress:  common.HexToAddress(poolAddress),
		}
		result := resolveTracker(&tracker, []bool{true, false, false})
		assert.Equal(t, tokentax.TaxInfo{Checked: true}, result)
	})

	t.Run("pair method reverted marks token unsupported", func(t *testing.T) {
		tracker := tracker{
			poolAddress:  poolAddress,
			tokenAddress: "0xagent",
			buyTaxPct:    big.NewInt(1),
		}
		result := resolveTracker(&tracker, []bool{false, true, false})
		assert.Equal(t, tokentax.TaxInfo{Checked: true}, result)
	})

	t.Run("verified pair refreshes both taxes without pair result", func(t *testing.T) {
		tracker := tracker{
			poolAddress:  poolAddress,
			tokenAddress: "0xagent",
			previous: tokentax.TaxInfo{
				Protocol: Protocol, Token: "0xagent",
				BuyTaxBps: uint256.NewInt(100), SellTaxBps: uint256.NewInt(100),
				Checked: true,
			},
			pairVerified: true,
			buyTaxPct:    big.NewInt(2),
			sellTaxPct:   big.NewInt(3),
		}
		result := resolveTracker(&tracker, []bool{true, true})
		assert.Equal(t, tokentax.TaxInfo{
			Protocol:   Protocol,
			Token:      "0xagent",
			BuyTaxBps:  uint256.NewInt(200),
			SellTaxBps: uint256.NewInt(300),
			Checked:    true,
		}, result)
	})
}

func TestTrackerAddCalls(t *testing.T) {
	t.Parallel()

	t.Run("unchecked pair reads pair and taxes", func(t *testing.T) {
		request := new(ethrpc.Client).NewRequest()
		NewTracker(
			"0x0000000000000000000000000000000000000001",
			"0xagent",
			tokentax.TaxInfo{},
		).AddCalls(request)

		assert.Len(t, request.Calls, 3)
		assert.Equal(t, methodPair, request.Calls[0].Method)
	})

	t.Run("verified pair only refreshes taxes", func(t *testing.T) {
		request := new(ethrpc.Client).NewRequest()
		NewTracker(
			"0x0000000000000000000000000000000000000001",
			"0xagent",
			tokentax.TaxInfo{Protocol: Protocol, Token: "0xagent", Checked: true},
		).AddCalls(request)

		assert.Len(t, request.Calls, 2)
		assert.Equal(t, methodBuyTax, request.Calls[0].Method)
		assert.Equal(t, methodSellTax, request.Calls[1].Method)
	})
}

func resolveTracker(tracker *tracker, results []bool) tokentax.TaxInfo {
	request := new(ethrpc.Client).NewRequest()
	tracker.AddCalls(request)
	return tracker.Resolve(&ethrpc.Response{Result: results})
}
