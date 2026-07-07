package metavault

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type VaultPriceFeed struct {
	IsSecondaryPriceEnabled bool     `json:"isSecondaryPriceEnabled"`
	MaxStrictPriceDeviation *big.Int `json:"maxStrictPriceDeviation"`
	PriceSampleSpace        *big.Int `json:"priceSampleSpace"`

	PriceDecimals         map[string]*big.Int `json:"priceDecimals"`
	SpreadBasisPoints     map[string]*big.Int `json:"spreadBasisPoints"`
	AdjustmentBasisPoints map[string]*big.Int `json:"adjustmentBasisPoints"`
	StrictStableTokens    map[string]bool     `json:"strictStableTokens"`
	IsAdjustmentAdditive  map[string]bool     `json:"isAdjustmentAdditive"`

	SecondaryPriceFeedAddress common.Address `json:"-"`
	SecondaryPriceFeed        IFastPriceFeed `json:"secondaryPriceFeed"`
	SecondaryPriceFeedVersion int            `json:"secondaryPriceFeedVersion"`

	PriceFeedsAddresses map[string]common.Address `json:"-"`
	PriceFeeds          map[string]*PriceFeed     `json:"priceFeeds"`
}

func NewVaultPriceFeed() *VaultPriceFeed {
	return &VaultPriceFeed{
		PriceDecimals:         make(map[string]*big.Int),
		SpreadBasisPoints:     make(map[string]*big.Int),
		AdjustmentBasisPoints: make(map[string]*big.Int),
		StrictStableTokens:    make(map[string]bool),
		IsAdjustmentAdditive:  make(map[string]bool),
		PriceFeedsAddresses:   make(map[string]common.Address),
		PriceFeeds:            make(map[string]*PriceFeed),
	}
}

const (
	VaultPriceFeedMethodIsSecondaryPriceEnabled = "isSecondaryPriceEnabled"
	VaultPriceFeedMethodMaxStrictPriceDeviation = "maxStrictPriceDeviation"
	VaultPriceFeedMethodPriceSampleSpace        = "priceSampleSpace"
	VaultPriceFeedMethodSecondaryPriceFeed      = "secondaryPriceFeed"

	VaultPriceFeedMethodPriceFeeds            = "priceFeeds"
	VaultPriceFeedMethodPriceDecimals         = "priceDecimals"
	VaultPriceFeedMethodSpreadBasisPoints     = "spreadBasisPoints"
	VaultPriceFeedMethodAdjustmentBasisPoints = "adjustmentBasisPoints"
	VaultPriceFeedMethodStrictStableTokens    = "strictStableTokens"
	VaultPriceFeedMethodIsAdjustmentAdditive  = "isAdjustmentAdditive"
)

func (pf *VaultPriceFeed) UnmarshalJSON(bytes []byte) error {
	var priceFeed struct {
		IsSecondaryPriceEnabled   bool                  `json:"isSecondaryPriceEnabled"`
		MaxStrictPriceDeviation   *big.Int              `json:"maxStrictPriceDeviation"`
		PriceSampleSpace          *big.Int              `json:"priceSampleSpace"`
		PriceDecimals             map[string]*big.Int   `json:"priceDecimals"`
		SpreadBasisPoints         map[string]*big.Int   `json:"spreadBasisPoints"`
		AdjustmentBasisPoints     map[string]*big.Int   `json:"adjustmentBasisPoints"`
		StrictStableTokens        map[string]bool       `json:"strictStableTokens"`
		IsAdjustmentAdditive      map[string]bool       `json:"isAdjustmentAdditive"`
		SecondaryPriceFeedVersion int                   `json:"secondaryPriceFeedVersion"`
		PriceFeeds                map[string]*PriceFeed `json:"priceFeeds"`
		SecondaryPriceFeed        json.RawMessage       `json:"secondaryPriceFeed"`
	}

	if err := json.Unmarshal(bytes, &priceFeed); err != nil {
		return err
	}

	pf.IsSecondaryPriceEnabled = priceFeed.IsSecondaryPriceEnabled
	pf.MaxStrictPriceDeviation = priceFeed.MaxStrictPriceDeviation
	pf.PriceSampleSpace = priceFeed.PriceSampleSpace
	pf.PriceDecimals = priceFeed.PriceDecimals
	pf.SpreadBasisPoints = priceFeed.SpreadBasisPoints
	pf.AdjustmentBasisPoints = priceFeed.AdjustmentBasisPoints
	pf.StrictStableTokens = priceFeed.StrictStableTokens
	pf.IsAdjustmentAdditive = priceFeed.IsAdjustmentAdditive
	pf.SecondaryPriceFeedVersion = priceFeed.SecondaryPriceFeedVersion
	pf.PriceFeeds = priceFeed.PriceFeeds

	if err := pf.UnmarshalJSONSecondaryPriceFeed(priceFeed.SecondaryPriceFeed); err != nil {
		return err
	}

	return nil
}

func (pf *VaultPriceFeed) UnmarshalJSONSecondaryPriceFeed(data json.RawMessage) error {
	switch pf.SecondaryPriceFeedVersion {
	case 1:
		var secondaryPriceFeed FastPriceFeedV1

		if err := json.Unmarshal(data, &secondaryPriceFeed); err != nil {
			return err
		}

		pf.SecondaryPriceFeed = &secondaryPriceFeed
	case 2:
		var secondaryPriceFeed FastPriceFeedV2

		if err := json.Unmarshal(data, &secondaryPriceFeed); err != nil {
			return err
		}

		pf.SecondaryPriceFeed = &secondaryPriceFeed
	default:
		return ErrInvalidSecondaryPriceFeedVersion
	}

	return nil
}

func (pf *VaultPriceFeed) GetPrice(token string, maximise bool, _ bool, _ bool) (*big.Int, error) {
	price, err := pf.getPriceV1(token, maximise)
	if err != nil {
		return nil, err
	}

	adjustmentBps := pf.AdjustmentBasisPoints[token]

	if adjustmentBps.Cmp(bignumber.ZeroBI) > 0 {
		isAdditive := pf.IsAdjustmentAdditive[token]

		if isAdditive {
			price = new(big.Int).Div(
				new(big.Int).Mul(
					price,
					new(big.Int).Add(BasisPointsDivisor, adjustmentBps),
				),
				BasisPointsDivisor,
			)
		} else {
			price = new(big.Int).Div(
				new(big.Int).Mul(
					price,
					new(big.Int).Sub(BasisPointsDivisor, adjustmentBps),
				),
				BasisPointsDivisor,
			)
		}
	}

	return price, nil
}

func (pf *VaultPriceFeed) getPriceV1(token string, maximise bool) (*big.Int, error) {
	price, err := pf.getPrimaryPrice(token, maximise)
	if err != nil {
		return nil, err
	}

	if pf.IsSecondaryPriceEnabled {
		price = pf.getSecondaryPrice(token, price, maximise)
	}

	if pf.StrictStableTokens[token] {
		var delta *big.Int
		if price.Cmp(OneUSD) > 0 {
			delta = new(big.Int).Sub(price, OneUSD)
		} else {
			delta = new(big.Int).Sub(OneUSD, price)
		}

		if delta.Cmp(pf.MaxStrictPriceDeviation) <= 0 {
			return OneUSD, nil
		}

		if maximise && price.Cmp(OneUSD) > 0 {
			return price, nil
		}

		if !maximise && price.Cmp(OneUSD) < 0 {
			return price, nil
		}

		return OneUSD, nil
	}

	spreadBasisPoint := pf.SpreadBasisPoints[token]

	if maximise {
		return new(big.Int).Div(
			new(big.Int).Mul(
				price,
				new(big.Int).Add(BasisPointsDivisor, spreadBasisPoint),
			),
			BasisPointsDivisor,
		), nil
	}

	return new(big.Int).Div(
		new(big.Int).Mul(
			price,
			new(big.Int).Sub(BasisPointsDivisor, spreadBasisPoint),
		),
		BasisPointsDivisor,
	), nil
}

func (pf *VaultPriceFeed) getPrimaryPrice(token string, maximise bool) (*big.Int, error) {
	priceFeed, ok := pf.PriceFeeds[token]
	if !ok {
		return nil, ErrVaultPriceFeedInvalidPriceFeed
	}

	price := bignumber.ZeroBI
	roundID := priceFeed.LatestRound()

	for i := big.NewInt(0); i.Cmp(pf.PriceSampleSpace) < 0; i = new(big.Int).Add(i, big.NewInt(1)) {
		if roundID.Cmp(i) <= 0 {
			break
		}

		var p *big.Int
		if i.Cmp(bignumber.ZeroBI) == 0 {
			p = priceFeed.LatestAnswer()

			if p.Cmp(bignumber.ZeroBI) <= 0 {
				return nil, ErrVaultPriceFeedInvalidPrice
			}
		} else {
			_, p, _, _, _ = priceFeed.GetRoundData(new(big.Int).Sub(roundID, bignumber.One))

			if p.Cmp(bignumber.ZeroBI) <= 0 {
				return nil, ErrVaultPriceFeedInvalidPrice
			}
		}

		if price.Cmp(bignumber.ZeroBI) == 0 {
			price = p
			continue
		}

		if !maximise && p.Cmp(price) < 0 {
			price = p
		}
	}

	if price.Cmp(bignumber.ZeroBI) <= 0 {
		return nil, ErrVaultPriceFeedCouldNotFetchPrice
	}

	priceDecimal := pf.PriceDecimals[token]

	return new(big.Int).Div(
		new(big.Int).Mul(price, PricePrecision),
		new(big.Int).Exp(big.NewInt(10), priceDecimal, nil),
	), nil
}

func (pf *VaultPriceFeed) getSecondaryPrice(token string, referencePrice *big.Int, maximise bool) *big.Int {
	if pf.SecondaryPriceFeed == nil {
		return referencePrice
	}

	return pf.SecondaryPriceFeed.GetPrice(token, referencePrice, maximise)
}
