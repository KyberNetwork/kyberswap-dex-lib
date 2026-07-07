package axima

import (
	"context"
	"fmt"

	"github.com/go-resty/resty/v2"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

func fetchPoolState(
	ctx context.Context,
	client *resty.Client,
	config *Config,
	pairIdentifier string,
) (Extra, []string, error) {
	var pairData PairData
	res, err := client.R().
		SetContext(ctx).
		SetResult(&pairData).
		Get(fmt.Sprintf("/%s/%s/bid_ask", config.Chain, pairIdentifier))

	if err != nil {
		return Extra{}, nil, err
	} else if res.IsError() {
		return Extra{}, nil, fmt.Errorf("API error: %s", res.String())
	}

	return convertAximaPoolState(pairData, config)
}

func convertAximaPoolState(pairData PairData, config *Config) (Extra, []string, error) {
	reserves := []string{pairData.TotalToken0Available, pairData.TotalToken1Available}

	var extra Extra

	extra.InitBid = bignumber.NewBig(pairData.Bid)
	extra.InitAsk = bignumber.NewBig(pairData.Ask)
	extra.QuoteAvailable = pairData.QuoteAvailable
	extra.MaxAge = config.MaxAge
	extra.IsV2 = config.IsV2

	extra.Bids = ConvertBins(pairData.Depth.Bids)
	extra.Asks = ConvertBins(pairData.Depth.Asks)

	return extra, reserves, nil
}

// ConvertBins maps API bid/ask bins into simulator Bins. Shared by the axima and
// metric-propamm paths.
func ConvertBins(aximaBins []AximaBin) []Bin {
	bins := make([]Bin, len(aximaBins))
	for i, bin := range aximaBins {
		bins[i] = Bin{
			BinIdx:           bin.BinIdx,
			Price:            bignumber.NewBig(bin.Price),
			CumulativeVolume: bignumber.NewBig(bin.CummlativeVolume),
		}
	}
	return bins
}
