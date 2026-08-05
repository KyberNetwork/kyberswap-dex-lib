package tokentax

import (
	"math/big"
	"strings"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

func SupportsChain(chainID valueobject.ChainID) bool {
	_, hasUniswap := uniswapContracts[chainID]
	_, hasPancake := pancakeContracts[chainID]
	return hasUniswap || hasPancake
}

// probe is one (candidate, base) pair to run through validate(candidate, base, amount).
type probe struct {
	candidate string
	base      string
}

// buildProbes returns the two self-pair probes for a 2-token pool: validate(token0, token1) and
// validate(token1, token0). No base-token knowledge is needed - this only resolves for pools that
// live on the same factory a detector instance is bound to (Uniswap-v2's or PancakeSwap-v2's own).
func buildProbes(token0, token1 string) []probe {
	return []probe{
		{candidate: token0, base: token1},
		{candidate: token1, base: token0},
	}
}

// Tracker probes both tokens of a 2-token pool for transfer tax. Build one with NewTracker.
type Tracker struct {
	instances []detectorInstance
	probes    []probe
	previous  TaxInfo

	calls        []int
	results      []validateResult
	basicResults []validateBasicResult
}

func NewTracker(chainID valueobject.ChainID, token0, token1 string, previous TaxInfo) *Tracker {
	return &Tracker{
		instances: detectorsFor(chainID),
		probes:    buildProbes(strings.ToLower(token0), strings.ToLower(token1)),
		previous:  previous,
	}
}

// tokenFees mirrors Uniswap's 5-field TokenFees struct.
type tokenFees struct {
	BuyFeeBps              *big.Int
	SellFeeBps             *big.Int
	FeeTakenOnTransfer     bool
	ExternalTransferFailed bool
	SellReverted           bool
}

// validateResult wraps tokenFees so the ABI decoder fills the inner struct field-by-field instead
// of misinterpreting validate()'s single tuple output as the tuple itself.
type validateResult struct {
	Fees tokenFees
}

// tokenFeesBasic mirrors PancakeSwap's 2-field TokenFees struct.
type tokenFeesBasic struct {
	BuyFeeBps  *big.Int
	SellFeeBps *big.Int
}

type validateBasicResult struct {
	Fees tokenFeesBasic
}

func (t *Tracker) AddCalls(request *ethrpc.Request) {
	n := len(t.instances) * len(t.probes)
	t.calls = make([]int, 0, n)
	t.results = make([]validateResult, n)
	t.basicResults = make([]validateBasicResult, n)
	for _, instance := range t.instances {
		for _, p := range t.probes {
			idx := len(t.calls)
			t.calls = append(t.calls, len(request.Calls))
			params := []any{
				common.HexToAddress(p.candidate),
				common.HexToAddress(p.base),
				uint256.NewInt(amountToBorrowRaw).ToBig(),
			}
			if instance.basic {
				request.AddCall(&ethrpc.Call{
					ABI: detectorBasicABI, Target: instance.address, Method: methodValidate,
					Params: params,
				}, []any{&t.basicResults[idx]})
			} else {
				request.AddCall(&ethrpc.Call{
					ABI: detectorABI, Target: instance.address, Method: methodValidate,
					Params: params,
				}, []any{&t.results[idx]})
			}
		}
	}
}

// Resolve picks a tax result from every (instance, probe) combination that succeeded. A nonzero
// result always wins over a zero one, since a pair unrecognized by the token can legitimately
// report 0% while the pair we actually care about charges real tax.
func (t *Tracker) Resolve(response *ethrpc.Response) TaxInfo {
	var best *TaxInfo
	idx := 0
	for _, instance := range t.instances {
		for _, p := range t.probes {
			call := t.calls[idx]
			if callSucceeded(response, call) {
				var buyFeeBps, sellFeeBps *big.Int
				if instance.basic {
					buyFeeBps, sellFeeBps = t.basicResults[idx].Fees.BuyFeeBps, t.basicResults[idx].Fees.SellFeeBps
				} else {
					buyFeeBps, sellFeeBps = t.results[idx].Fees.BuyFeeBps, t.results[idx].Fees.SellFeeBps
				}
				info := TaxInfo{
					Token:      p.candidate,
					BuyTaxBps:  big256.FromBig(buyFeeBps),
					SellTaxBps: big256.FromBig(sellFeeBps),
					Checked:    true,
				}
				if isNonzeroTax(info) {
					return info
				}
				if best == nil {
					best = &info
				}
			}
			idx++
		}
	}
	if best != nil {
		return *best
	}

	// Every probe failed this cycle. Keep the last known answer instead of reporting Token == "",
	// which would mark this token permanently unsupported.
	if t.previous.Token != "" {
		return t.previous
	}
	return TaxInfo{Checked: true}
}

func isNonzeroTax(info TaxInfo) bool {
	return info.BuyTaxBps != nil && !info.BuyTaxBps.IsZero() ||
		info.SellTaxBps != nil && !info.SellTaxBps.IsZero()
}

func callSucceeded(response *ethrpc.Response, index int) bool {
	return response != nil &&
		index >= 0 &&
		index < len(response.Result) &&
		response.Result[index]
}
