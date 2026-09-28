package prmfun

import (
	"errors"

	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

var (
	ErrInvalidAmount = errors.New("prm-fun: quote resolves to a zero amount (below curve precision)")
)

var (
	// uSaleSupply is MemeCurve.SALE_SUPPLY (800_000_000 ether) parsed once.
	uSaleSupply = uint256.MustFromDecimal(saleSupply)

	uBps    = uint256.NewInt(bps)
	uFeeBps = uint256.NewInt(feeBpsConst)
	// uBpsMinusFee = BPS - FEE_BPS, the denominator MemeCurve.sol uses to gross up a
	// graduation-capped buy's actual deskUsed from its net (post-fee) amount.
	uBpsMinusFee = uint256.NewInt(bps - feeBpsConst)
)

// QuoteBuyResult mirrors MemeCurve.quoteBuy's (memeOut, deskUsed, fee) return.
type QuoteBuyResult struct {
	MemeOut  *uint256.Int
	DeskUsed *uint256.Int
	Fee      *uint256.Int
}

// QuoteBuy replicates MemeCurve.sol's quoteBuy(uint256 deskIn), verbatim down to the
// rounding direction:
//
//	feeOnAll = ceil(deskIn * FEE_BPS / BPS)
//	net      = deskIn - feeOnAll
//	room     = graduationDesk - deskRaised
//	if net >= room (this buy crosses graduation):
//	    net      = room
//	    deskUsed = ceil(net * BPS / (BPS - FEE_BPS))
//	    fee      = deskUsed - net
//	    memeOut  = saleSupply - memeSold                      // all remaining supply
//	else:
//	    deskUsed = deskIn
//	    fee      = feeOnAll
//	    nextVirtualMeme = ceil(virtualMeme * virtualDesk / (virtualDesk + net))
//	    memeOut  = min(virtualMeme - nextVirtualMeme, saleSupply - memeSold)
//
// Caller (CalcAmountOut) is responsible for the phase/deskIn==0/deskRaised>=graduationDesk
// gating - this function assumes a Trading-phase pool with deskRaised < graduationDesk,
// exactly like the Solidity's own quoteBuy does after MemeCurve.buy's earlier checks.
func QuoteBuy(deskIn, virtualMeme, virtualDesk, memeSold, deskRaised, graduationDesk, saleSupply *uint256.Int) *QuoteBuyResult {
	feeOnAll := big256.MulDivUp(new(uint256.Int), deskIn, uFeeBps, uBps)
	net := new(uint256.Int).Sub(deskIn, feeOnAll)
	room := new(uint256.Int).Sub(graduationDesk, deskRaised)

	remaining := new(uint256.Int).Sub(saleSupply, memeSold)

	if net.Cmp(room) >= 0 {
		deskUsed := big256.MulDivUp(new(uint256.Int), room, uBps, uBpsMinusFee)
		fee := new(uint256.Int).Sub(deskUsed, room)
		return &QuoteBuyResult{MemeOut: remaining, DeskUsed: deskUsed, Fee: fee}
	}

	denom := new(uint256.Int).Add(virtualDesk, net)
	nextVirtualMeme := big256.MulDivUp(new(uint256.Int), virtualMeme, virtualDesk, denom)
	memeOut := big256.Min(new(uint256.Int).Sub(virtualMeme, nextVirtualMeme), remaining)

	return &QuoteBuyResult{MemeOut: memeOut, DeskUsed: new(uint256.Int).Set(deskIn), Fee: feeOnAll}
}

// QuoteSellResult mirrors MemeCurve.quoteSell's (deskOut, fee) return.
type QuoteSellResult struct {
	DeskOut *uint256.Int
	Fee     *uint256.Int
}

// QuoteSell replicates MemeCurve.sol's quoteSell(uint256 memeIn) verbatim:
//
//	nextVirtualDesk = ceil(virtualMeme * virtualDesk / (virtualMeme + memeIn))
//	gross           = virtualDesk - nextVirtualDesk
//	(caller must treat gross > deskRaised as insufficient liquidity - the Solidity
//	 returns (0, 0) here; this package returns ErrInsufficientLiquidity instead, see
//	 CalcAmountOut)
//	fee     = ceil(gross * FEE_BPS / BPS)
//	deskOut = gross - fee
//
// Caller is responsible for the phase/memeIn==0/memeIn>memeSold gating, exactly like
// Solidity's MemeCurve.sell does before calling quoteSell.
func QuoteSell(memeIn, virtualMeme, virtualDesk *uint256.Int) (gross *uint256.Int, result *QuoteSellResult) {
	denom := new(uint256.Int).Add(virtualMeme, memeIn)
	nextVirtualDesk := big256.MulDivUp(new(uint256.Int), virtualMeme, virtualDesk, denom)
	gross = new(uint256.Int).Sub(virtualDesk, nextVirtualDesk)

	fee := big256.MulDivUp(new(uint256.Int), gross, uFeeBps, uBps)
	deskOut := new(uint256.Int).Sub(gross, fee)
	return gross, &QuoteSellResult{DeskOut: deskOut, Fee: fee}
}
