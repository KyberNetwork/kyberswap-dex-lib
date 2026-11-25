package efficientprop

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// A pool that takes a fee in the token it pays out leaves a later fill less room than the door's samples show: the
// spread fee and the notional part are owed in the pair's quote and come off the quote's room (FLAMMRoute._open), so
// the fill that pays the quote (the base token in, the bid) finds less of it after each fill. The fill with the quote
// in pays the base, and the fee it owes is in the token it takes, so it cuts no room it pays from. The fee is read
// from the chain, in the head multicall: the pool's own rates, which its Factory holds (FLAMMFactory.
// spreadFeeRatesOf, one word: the spread rate, and the notional rates of the two kinds of pair), and the operator's
// share, which the pool holds (FLAMMFloatLib.operatorShare).
var (
	// poolFeeABI is the two views read on the pool: its Factory, and its operator's share.
	poolFeeABI = mustABI([]byte(`[{"type":"function","name":"factory","stateMutability":"view","inputs":[],` +
		`"outputs":[{"name":"","type":"address"}]},{"type":"function","name":"operatorShare","stateMutability":"view",` +
		`"inputs":[],"outputs":[{"name":"shareBp","type":"uint16"},{"name":"pendingBp","type":"uint16"},` +
		`{"name":"executableAt","type":"uint48"}]}]`))
	// feeFactoryABI is the view read on the Factory: the pool's rates word.
	feeFactoryABI = mustABI([]byte(`[{"type":"function","name":"spreadFeeRatesOf","stateMutability":"view",` +
		`"inputs":[{"name":"pool","type":"address"}],"outputs":[{"name":"","type":"uint48"}]}]`))
)

// shareOut is operatorShare's answer; ExecutableAt is nil where the answer was short.
type shareOut struct {
	ShareBp      uint16
	PendingBp    uint16
	ExecutableAt *big.Int
}

// feeCalls are the two head calls that read a pool's fee: spreadFeeRatesOf(pool) on its Factory, and operatorShare()
// on the pool. The pool is the listing's custody.
func feeCalls(factory, pool common.Address) []call {
	return []call{
		{abi: &feeFactoryABI, target: factory, method: "spreadFeeRatesOf", params: []any{pool}},
		{abi: &poolFeeABI, target: pool, method: "operatorShare"},
	}
}

// takesFee is whether the pool is read as taking a fee: its rates word is nonzero, or its operator's share is, or
// either read failed or decoded short (rates and share nil). A pool whose fee cannot be read is priced as one that
// takes it.
func takesFee(rates *big.Int, share *shareOut) bool {
	if rates == nil || share == nil || share.ExecutableAt == nil {
		return true
	}
	return rates.Sign() != 0 || share.ShareBp != 0
}

// feeOf is takesFee from the two calls' return data, nil where a call reverted.
func feeOf(ratesRet, shareRet []byte) bool {
	rates := unpackUint(&feeFactoryABI, "spreadFeeRatesOf", ratesRet)
	var share *shareOut
	if shareRet != nil {
		var s shareOut
		if err := poolFeeABI.UnpackIntoInterface(&s, "operatorShare", shareRet); err == nil {
			share = &s
		}
	}
	return takesFee(rates, share)
}
