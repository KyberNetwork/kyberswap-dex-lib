package axima

import (
	"context"
	"fmt"
	"strconv"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/go-resty/resty/v2"
	"github.com/samber/lo"
)

func fetchPoolState(
	ctx context.Context,
	client *resty.Client,
	config *Config,
	pair string,
) (Extra, []string, error) {
	var pairData PairData
	_, err := client.R().
		SetContext(ctx).
		SetResult(&pairData).
		Get(fmt.Sprintf("/%s/%s/bid_ask", config.ChainID.String(), pair))

	if err != nil {
		return Extra{}, nil, err
	}

	reserves := []string{pairData.TotalToken0Available, pairData.TotalToken1Available}

	var extra Extra
	bidF, err := strconv.ParseFloat(pairData.Bid, 64)
	if err != nil {
		return Extra{}, nil, err
	}
	extra.ZeroToOneRate = bidF / Q64

	askF, err := strconv.ParseFloat(pairData.Ask, 64)
	if err != nil {
		return Extra{}, nil, err
	}
	extra.OneToZeroRate = Q64 / askF

	extra.QuoteAvailable = pairData.QuoteAvailable
	extra.MaxAge = config.MaxAge

	if bids, err := convertAximaBins(pairData.Depth.Bids, true); err != nil {
		return Extra{}, nil, err
	} else {
		extra.Bids = bids
	}

	if asks, err := convertAximaBins(pairData.Depth.Asks, false); err != nil {
		return Extra{}, nil, err
	} else {
		extra.Asks = asks
	}

	return extra, reserves, nil
}

func convertAximaBins(aximaBins []AximaBin, isBid bool) ([]Bin, error) {
	bins := make([]Bin, len(aximaBins))
	for i, bin := range aximaBins {
		priceF, err := strconv.ParseFloat(bin.Price, 64)
		if err != nil {
			return nil, err
		}

		rate := lo.Ternary(isBid, priceF/Q64, Q64/priceF)

		pie6, _ := strconv.ParseInt(bin.PriceImpactE6, 10, 64)
		bins[i] = Bin{
			BinIdx:           bin.BinIdx,
			Rate:             rate,
			CumulativeVolume: bignumber.NewBig(bin.Price),
			PriceImpactE6:    int(pie6),
		}
	}

	return bins, nil
}
