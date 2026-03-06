package axima

import (
	"context"
	"fmt"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/go-resty/resty/v2"
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
		Get(fmt.Sprintf("/%s/%s/bid_ask", config.ChainID.String(), pairIdentifier))

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

	if bids, err := convertAximaBins(pairData.Depth.Bids); err != nil {
		return Extra{}, nil, err
	} else {
		extra.Bids = bids
	}

	if asks, err := convertAximaBins(pairData.Depth.Asks); err != nil {
		return Extra{}, nil, err
	} else {
		extra.Asks = asks
	}

	return extra, reserves, nil
}

func convertAximaBins(aximaBins []AximaBin) ([]Bin, error) {
	bins := make([]Bin, len(aximaBins))
	for i, bin := range aximaBins {
		bins[i] = Bin{
			BinIdx:           bin.BinIdx,
			Price:            bignumber.NewBig(bin.Price),
			CumulativeVolume: bignumber.NewBig(bin.CummlativeVolume),
		}
	}

	return bins, nil
}
