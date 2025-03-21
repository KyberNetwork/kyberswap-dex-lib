package swapbasedperp

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type VaultPriceFeed struct {
	BNB                        string   `json:"bnb"`
	BTC                        string   `json:"btc"`
	ETH                        string   `json:"eth"`
	FavorPrimaryPrice          bool     `json:"favorPrimaryPrice"`
	IsAmmEnabled               bool     `json:"isAmmEnabled"`
	IsSecondaryPriceEnabled    bool     `json:"isSecondaryPriceEnabled"`
	MaxStrictPriceDeviation    *big.Int `json:"maxStrictPriceDeviation"`
	PriceSampleSpace           *big.Int `json:"priceSampleSpace"`
	SpreadThresholdBasisPoints *big.Int `json:"spreadThresholdBasisPoints"`
	UseV2Pricing               bool     `json:"useV2Pricing"`

	PriceDecimals         map[string]*big.Int `json:"priceDecimals"`
	SpreadBasisPoints     map[string]*big.Int `json:"spreadBasisPoints"`
	AdjustmentBasisPoints map[string]*big.Int `json:"adjustmentBasisPoints"`
	StrictStableTokens    map[string]bool     `json:"strictStableTokens"`
	IsAdjustmentAdditive  map[string]bool     `json:"isAdjustmentAdditive"`

	BNBBUSDAddress common.Address `json:"-"`
	BNBBUSD        *PancakePair   `json:"bnbBusd,omitempty"`

	BTCBNBAddress common.Address `json:"-"`
	BTCBNB        *PancakePair   `json:"btcBnb,omitempty"`

	ETHBNBAddress common.Address `json:"-"`
	ETHBNB        *PancakePair   `json:"ethBnb,omitempty"`

	ChainlinkFlagsAddress common.Address  `json:"-"`
	ChainlinkFlags        *ChainlinkFlags `json:"chainlinkFlags,omitempty"`

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
	vaultPriceFeedMethodFavorPrimaryPrice          = "favorPrimaryPrice"
	vaultPriceFeedMethodIsSecondaryPriceEnabled    = "isSecondaryPriceEnabled"
	vaultPriceFeedMethodMaxStrictPriceDeviation    = "maxStrictPriceDeviation"
	vaultPriceFeedMethodPriceSampleSpace           = "priceSampleSpace"
	vaultPriceFeedMethodSecondaryPriceFeed         = "secondaryPriceFeed"
	vaultPriceFeedMethodSpreadThresholdBasisPoints = "spreadThresholdBasisPoints"

	vaultPriceFeedMethodPriceFeeds            = "priceFeeds"
	vaultPriceFeedMethodPriceDecimals         = "priceDecimals"
	vaultPriceFeedMethodSpreadBasisPoints     = "spreadBasisPoints"
	vaultPriceFeedMethodAdjustmentBasisPoints = "adjustmentBasisPoints"
	vaultPriceFeedMethodStrictStableTokens    = "strictStableTokens"
	vaultPriceFeedMethodIsAdjustmentAdditive  = "isAdjustmentAdditive"
)

func (pf *VaultPriceFeed) UnmarshalJSON(bytes []byte) error {
	var priceFeed struct {
		BNB                        string                `json:"bnb"`
		BTC                        string                `json:"btc"`
		ETH                        string                `json:"eth"`
		FavorPrimaryPrice          bool                  `json:"favorPrimaryPrice"`
		IsAmmEnabled               bool                  `json:"isAmmEnabled"`
		IsSecondaryPriceEnabled    bool                  `json:"isSecondaryPriceEnabled"`
		MaxStrictPriceDeviation    *big.Int              `json:"maxStrictPriceDeviation"`
		PriceSampleSpace           *big.Int              `json:"priceSampleSpace"`
		SpreadThresholdBasisPoints *big.Int              `json:"spreadThresholdBasisPoints"`
		UseV2Pricing               bool                  `json:"useV2Pricing"`
		PriceDecimals              map[string]*big.Int   `json:"priceDecimals"`
		SpreadBasisPoints          map[string]*big.Int   `json:"spreadBasisPoints"`
		AdjustmentBasisPoints      map[string]*big.Int   `json:"adjustmentBasisPoints"`
		StrictStableTokens         map[string]bool       `json:"strictStableTokens"`
		IsAdjustmentAdditive       map[string]bool       `json:"isAdjustmentAdditive"`
		BNBBUSD                    *PancakePair          `json:"bnbBusd,omitempty"`
		BTCBNB                     *PancakePair          `json:"btcBnb,omitempty"`
		ETHBNB                     *PancakePair          `json:"ethBnb,omitempty"`
		ChainlinkFlags             *ChainlinkFlags       `json:"chainlinkFlags,omitempty"`
		SecondaryPriceFeedVersion  int                   `json:"secondaryPriceFeedVersion"`
		PriceFeeds                 map[string]*PriceFeed `json:"priceFeeds"`
	}

	if err := json.Unmarshal(bytes, &priceFeed); err != nil {
		return err
	}

	pf.BNB = priceFeed.BNB
	pf.BTC = priceFeed.BTC
	pf.ETH = priceFeed.ETH
	pf.FavorPrimaryPrice = priceFeed.FavorPrimaryPrice
	pf.IsAmmEnabled = priceFeed.IsAmmEnabled
	pf.IsSecondaryPriceEnabled = priceFeed.IsSecondaryPriceEnabled
	pf.MaxStrictPriceDeviation = priceFeed.MaxStrictPriceDeviation
	pf.PriceSampleSpace = priceFeed.PriceSampleSpace
	pf.SpreadThresholdBasisPoints = priceFeed.SpreadThresholdBasisPoints
	pf.UseV2Pricing = priceFeed.UseV2Pricing
	pf.PriceDecimals = priceFeed.PriceDecimals
	pf.SpreadBasisPoints = priceFeed.SpreadBasisPoints
	pf.AdjustmentBasisPoints = priceFeed.AdjustmentBasisPoints
	pf.StrictStableTokens = priceFeed.StrictStableTokens
	pf.IsAdjustmentAdditive = priceFeed.IsAdjustmentAdditive
	pf.BNBBUSD = priceFeed.BNBBUSD
	pf.BTCBNB = priceFeed.BTCBNB
	pf.ETHBNB = priceFeed.ETHBNB
	pf.ChainlinkFlags = priceFeed.ChainlinkFlags
	pf.SecondaryPriceFeedVersion = priceFeed.SecondaryPriceFeedVersion
	pf.PriceFeeds = priceFeed.PriceFeeds

	if err := pf.UnmarshalJSONSecondaryPriceFeed(bytes); err != nil {
		return err
	}

	return nil
}

func (pf *VaultPriceFeed) UnmarshalJSONSecondaryPriceFeed(bytes []byte) error {
	switch pf.SecondaryPriceFeedVersion {
	case 1:
		var priceFeed struct {
			SecondaryPriceFeed *FastPriceFeedV1 `json:"secondaryPriceFeed"`
		}

		if err := json.Unmarshal(bytes, &priceFeed); err != nil {
			return nil
		}

		pf.SecondaryPriceFeed = priceFeed.SecondaryPriceFeed
	case 2:
		var priceFeed struct {
			SecondaryPriceFeed *FastPriceFeedV2 `json:"secondaryPriceFeed"`
		}

		if err := json.Unmarshal(bytes, &priceFeed); err != nil {
			return nil
		}

		pf.SecondaryPriceFeed = priceFeed.SecondaryPriceFeed
	default:
		return ErrInvalidSecondaryPriceFeedVersion
	}

	return nil
}

func (pf *VaultPriceFeed) GetPrice(token string, maximise bool, includeAmmPrice bool, _ bool) (*big.Int, error) {
	var price *big.Int
	var err error

	if pf.UseV2Pricing {
		price, err = pf.getPriceV2(token, maximise, includeAmmPrice)
		if err != nil {
			return nil, err
		}
	} else {
		price, err = pf.getPriceV1(token, maximise, includeAmmPrice)
		if err != nil {
			return nil, err
		}
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

func (pf *VaultPriceFeed) getPriceV1(token string, maximise bool, includeAmmPrice bool) (*big.Int, error) {
	price, err := pf.getPrimaryPrice(token, maximise)
	if err != nil {
		return nil, err
	}

	if includeAmmPrice && pf.IsAmmEnabled {
		ammPrice := pf.getAmmPrice(token)
		if ammPrice.Cmp(bignumber.ZeroBI) > 0 {
			if maximise && ammPrice.Cmp(price) > 0 {
				price = ammPrice
			}
			if !maximise && ammPrice.Cmp(price) < 0 {
				price = ammPrice
			}
		}
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

func (pf *VaultPriceFeed) getPriceV2(token string, maximise bool, includeAmmPrice bool) (*big.Int, error) {
	price, err := pf.getPrimaryPrice(token, maximise)
	if err != nil {
		return nil, err
	}

	if includeAmmPrice && pf.IsAmmEnabled {
		price = pf.getAmmPriceV2(token, maximise, price)
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

	if pf.ChainlinkFlags != nil {
		isRaised := pf.ChainlinkFlags.GetFlag(flagArbitrumSeqOffline)

		if isRaised {
			return nil, ErrVaultPriceFeedChainlinkFeedsNotUpdated
		}
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

func (pf *VaultPriceFeed) getAmmPrice(token string) *big.Int {
	if token == pf.BNB {
		return pf.getPairPrice(pf.BNBBUSD, true)
	}

	if token == pf.ETH {
		price0 := pf.getPairPrice(pf.BNBBUSD, true)
		price1 := pf.getPairPrice(pf.ETHBNB, true)

		return new(big.Int).Div(new(big.Int).Mul(price0, price1), PricePrecision)
	}

	if token == pf.BTC {
		price0 := pf.getPairPrice(pf.BNBBUSD, true)
		price1 := pf.getPairPrice(pf.BTCBNB, true)

		return new(big.Int).Div(new(big.Int).Mul(price0, price1), PricePrecision)
	}

	return bignumber.ZeroBI
}

func (pf *VaultPriceFeed) getAmmPriceV2(token string, maximise bool, primaryPrice *big.Int) *big.Int {
	ammPrice := pf.getAmmPrice(token)
	if ammPrice.Cmp(bignumber.ZeroBI) == 0 {
		return primaryPrice
	}

	var diff *big.Int
	if ammPrice.Cmp(primaryPrice) > 0 {
		diff = new(big.Int).Sub(ammPrice, primaryPrice)
	} else {
		diff = new(big.Int).Sub(primaryPrice, ammPrice)
	}

	if new(big.Int).Mul(diff, BasisPointsDivisor).Cmp(new(big.Int).Mul(primaryPrice, pf.SpreadThresholdBasisPoints)) < 0 {
		if pf.FavorPrimaryPrice {
			return primaryPrice
		}
		return ammPrice
	}

	if maximise && ammPrice.Cmp(primaryPrice) > 0 {
		return ammPrice
	}

	if !maximise && ammPrice.Cmp(primaryPrice) < 0 {
		return ammPrice
	}

	return primaryPrice
}

func (pf *VaultPriceFeed) getPairPrice(pair *PancakePair, divByReserve0 bool) *big.Int {
	reserve0, reserve1, _ := pair.GetReserves()

	if divByReserve0 {
		if reserve0.Cmp(bignumber.ZeroBI) == 0 {
			return bignumber.ZeroBI
		}

		return new(big.Int).Div(new(big.Int).Mul(reserve1, PricePrecision), reserve0)
	}

	if reserve1.Cmp(bignumber.ZeroBI) == 0 {
		return bignumber.ZeroBI
	}

	return new(big.Int).Div(new(big.Int).Mul(reserve0, PricePrecision), reserve1)
}
