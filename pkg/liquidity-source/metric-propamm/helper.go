package metricpropamm

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"

// ConvertBins maps API bid/ask bins into simulator Bins.
func ConvertBins(apiBins []PairBin) []Bin {
	bins := make([]Bin, len(apiBins))
	for i, bin := range apiBins {
		bins[i] = Bin{
			BinIdx:           bin.BinIdx,
			Price:            bignumber.NewBig(bin.Price),
			CumulativeVolume: bignumber.NewBig(bin.CummlativeVolume),
		}
	}
	return bins
}
