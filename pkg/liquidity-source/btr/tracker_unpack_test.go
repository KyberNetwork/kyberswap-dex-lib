package btr

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

// TestSwapQuoteUnpack pins the getSwapQuote decode path the tracker relies on:
// the single unnamed tuple output must land in rpcSwapQuoteResult.Quote, else
// every ladder probe fails to unpack and the pool never quotes.
func TestSwapQuoteUnpack(t *testing.T) {
	in := rpcSwapQuote{
		AmountOut: big.NewInt(24004), AmountIn: big.NewInt(1e18), SpreadPbps: 2666,
		ProtoFee: big.NewInt(3), LpFee: big.NewInt(30), MarkPrice: big.NewInt(1), MidPrice: big.NewInt(1),
		CovToll:    big.NewInt(16),
		RouteHops:  []common.Address{common.HexToAddress(WMON), common.HexToAddress(USDC)},
		HopAmounts: []*big.Int{big.NewInt(1e18), big.NewInt(24004)},
		HopPrices:  []*big.Int{big.NewInt(1)},
	}
	raw, err := poolABI.Methods["getSwapQuote"].Outputs.Pack(in)
	if err != nil {
		t.Fatalf("pack: %v", err)
	}
	var out rpcSwapQuoteResult
	if err := poolABI.UnpackIntoInterface(&out, "getSwapQuote", raw); err != nil {
		t.Fatalf("unpack: %v", err)
	}
	if out.Quote.AmountOut == nil || out.Quote.AmountOut.Cmp(in.AmountOut) != 0 {
		t.Fatalf("amountOut = %v, want %v", out.Quote.AmountOut, in.AmountOut)
	}
	if out.Quote.SpreadPbps != in.SpreadPbps || len(out.Quote.RouteHops) != 2 {
		t.Fatalf("tuple fields not decoded: %+v", out.Quote)
	}
}
