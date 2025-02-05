package gmx

import (
	"math/big"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type VaultPriceFeed struct {
	Address string `json:"-"`

	BNB                        string        `json:"bnb,omitempty"`
	BTC                        string        `json:"btc,omitempty"`
	ETH                        string        `json:"eth,omitempty"`
	FavorPrimaryPrice          bool          `json:"favorPrimaryPrice,omitempty"`
	IsAmmEnabled               bool          `json:"isAmmEnabled,omitempty"`
	IsSecondaryPriceEnabled    bool          `json:"isSecondaryPriceEnabled,omitempty"`
	MaxStrictPriceDeviation    *big.Int      `json:"maxStrictPriceDeviation,omitempty"`
	PriceSampleSpace           *big.Int      `json:"priceSampleSpace,omitempty"`
	SpreadThresholdBasisPoints *big.Int      `json:"spreadThresholdBasisPoints,omitempty"`
	UseV2Pricing               bool          `json:"useV2Pricing,omitempty"`
	PriceFeedType              PriceFeedType `json:"priceFeedType,omitempty"`

	PriceDecimals         map[string]*big.Int `json:"priceDecimals,omitempty"`
	SpreadBasisPoints     map[string]*big.Int `json:"spreadBasisPoints,omitempty"`
	AdjustmentBasisPoints map[string]*big.Int `json:"adjustmentBasisPoints,omitempty"`
	StrictStableTokens    map[string]bool     `json:"strictStableTokens,omitempty"`
	IsAdjustmentAdditive  map[string]bool     `json:"isAdjustmentAdditive,omitempty"`

	BNBBUSDAddress common.Address `json:"-"`
	BNBBUSD        *PancakePair   `json:"bnbBusd,omitempty"`

	BTCBNBAddress common.Address `json:"-"`
	BTCBNB        *PancakePair   `json:"btcBnb,omitempty"`

	ETHBNBAddress common.Address `json:"-"`
	ETHBNB        *PancakePair   `json:"ethBnb,omitempty"`

	ChainlinkFlagsAddress common.Address  `json:"-"`
	ChainlinkFlags        *ChainlinkFlags `json:"chainlinkFlags,omitempty"`

	SecondaryPriceFeedAddress common.Address `json:"-"`
	SecondaryPriceFeed        IFastPriceFeed `json:"secondaryPriceFeed,omitempty"`
	SecondaryPriceFeedVersion int            `json:"secondaryPriceFeedVersion,omitempty"`

	PriceFeedsAddresses map[string]common.Address `json:"-"`
	PriceFeeds          map[string]*PriceFeed     `json:"priceFeeds,omitempty"`
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
	vaultPriceFeedMethodBNB                        = "bnb"
	vaultPriceFeedMethodBNBBUSD                    = "bnbBusd"
	vaultPriceFeedMethodBTC                        = "btc"
	vaultPriceFeedMethodBTCBNB                     = "btcBnb"
	vaultPriceFeedMethodChainlinkFlags             = "chainlinkFlags"
	vaultPriceFeedMethodETH                        = "eth"
	vaultPriceFeedMethodETHBNB                     = "ethBnb"
	vaultPriceFeedMethodFavorPrimaryPrice          = "favorPrimaryPrice"
	vaultPriceFeedMethodIsAmmEnabled               = "isAmmEnabled"
	vaultPriceFeedMethodIsSecondaryPriceEnabled    = "isSecondaryPriceEnabled"
	vaultPriceFeedMethodMaxStrictPriceDeviation    = "maxStrictPriceDeviation"
	vaultPriceFeedMethodPriceSampleSpace           = "priceSampleSpace"
	vaultPriceFeedMethodSecondaryPriceFeed         = "secondaryPriceFeed"
	vaultPriceFeedMethodSpreadThresholdBasisPoints = "spreadThresholdBasisPoints"
	vaultPriceFeedMethodUseV2Pricing               = "useV2Pricing"

	vaultPriceFeedMethodPriceFeeds            = "priceFeeds"
	vaultPriceFeedMethodPriceDecimals         = "priceDecimals"
	vaultPriceFeedMethodSpreadBasisPoints     = "spreadBasisPoints"
	vaultPriceFeedMethodAdjustmentBasisPoints = "adjustmentBasisPoints"
	vaultPriceFeedMethodStrictStableTokens    = "strictStableTokens"
	vaultPriceFeedMethodIsAdjustmentAdditive  = "isAdjustmentAdditive"
	vaultPriceFeedMethodGetPrimaryPrice       = "getPrimaryPrice"
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
		PriceFeedType              PriceFeedType         `json:"priceFeedType"`
		PriceDecimals              map[string]*big.Int   `json:"priceDecimals"`
		SpreadBasisPoints          map[string]*big.Int   `json:"spreadBasisPoints"`
		AdjustmentBasisPoints      map[string]*big.Int   `json:"adjustmentBasisPoints"`
		StrictStableTokens         map[string]bool       `json:"strictStableTokens"`
		IsAdjustmentAdditive       map[string]bool       `json:"isAdjustmentAdditive"`
		BNBBUSD                    *PancakePair          `json:"bnbBusd"`
		BTCBNB                     *PancakePair          `json:"btcBnb"`
		ETHBNB                     *PancakePair          `json:"ethBnb"`
		ChainlinkFlags             *ChainlinkFlags       `json:"chainlinkFlags"`
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
	pf.PriceFeedType = priceFeed.PriceFeedType
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

	if adjustmentBps.Sign() > 0 {
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
		if ammPrice.Sign() > 0 {
			if maximise == (ammPrice.Cmp(price) > 0) {
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
		delta := new(big.Int).Sub(price, OneUSD)
		if delta.Abs(delta).Cmp(pf.MaxStrictPriceDeviation) <= 0 {
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
		price = new(big.Int).Mul(
			price,
			new(big.Int).Add(BasisPointsDivisor, spreadBasisPoint),
		)
	} else {
		price = new(big.Int).Mul(
			price,
			new(big.Int).Sub(BasisPointsDivisor, spreadBasisPoint),
		)
	}
	return price.Div(price, BasisPointsDivisor), nil
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
		delta := new(big.Int).Sub(price, OneUSD)
		if delta.Abs(delta).Cmp(pf.MaxStrictPriceDeviation) <= 0 {
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
		price = new(big.Int).Mul(
			price,
			new(big.Int).Add(BasisPointsDivisor, spreadBasisPoint),
		)
	} else {
		price = new(big.Int).Mul(
			price,
			new(big.Int).Sub(BasisPointsDivisor, spreadBasisPoint),
		)
	}
	return price.Div(price, BasisPointsDivisor), nil
}

func (pf *VaultPriceFeed) getPrimaryPrice(token string, maximise bool) (*big.Int, error) {
	priceFeed, ok := pf.PriceFeeds[token]
	if !ok {
		return nil, ErrVaultPriceFeedInvalidPriceFeed
	}

	if pf.ChainlinkFlags != nil {
		isRaised := pf.ChainlinkFlags.GetFlag(FlagArbitrumSeqOffline)

		if isRaised {
			return nil, ErrVaultPriceFeedChainlinkFeedsNotUpdated
		}
	}

	if pf.PriceFeedType == PriceFeedTypeDirect {
		return lo.CoalesceOrEmpty(priceFeed.Answers[strconv.FormatBool(maximise)], bignumber.ZeroBI), nil
	} else if pf.PriceSampleSpace == nil {
		return priceFeed.LatestAnswer(), nil
	}

	price := bignumber.ZeroBI
	roundID := priceFeed.LatestRound()

	for i := new(big.Int); i.Cmp(pf.PriceSampleSpace) < 0; i.Add(i, bignumber.One) {
		if roundID.Cmp(i) <= 0 {
			break
		}

		var p *big.Int
		if i.Sign() == 0 {
			p = priceFeed.LatestAnswer()

			if p.Sign() <= 0 {
				return nil, ErrVaultPriceFeedInvalidPrice
			}
		} else {
			_, p, _, _, _ = priceFeed.GetRoundData(new(big.Int).Sub(roundID, i))

			if p.Sign() <= 0 {
				return nil, ErrVaultPriceFeedInvalidPrice
			}
		}

		if price.Sign() == 0 {
			price = p
			continue
		}

		if maximise && p.Cmp(price) > 0 {
			price = p
			continue
		}

		if !maximise && p.Cmp(price) < 0 {
			price = p
		}
	}

	if price.Sign() <= 0 {
		return nil, ErrVaultPriceFeedCouldNotFetchPrice
	}

	priceDecimal := pf.PriceDecimals[token]
	price = new(big.Int).Mul(price, PricePrecision)
	return price.Div(price, bignumber.TenPowInt(priceDecimal.Int64())), nil
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

		price := new(big.Int).Mul(price0, price1)
		return price.Div(price, PricePrecision)
	}

	if token == pf.BTC {
		price0 := pf.getPairPrice(pf.BNBBUSD, true)
		price1 := pf.getPairPrice(pf.BTCBNB, true)

		price := new(big.Int).Mul(price0, price1)
		return price.Div(price, PricePrecision)
	}

	return bignumber.ZeroBI
}

func (pf *VaultPriceFeed) getAmmPriceV2(token string, maximise bool, primaryPrice *big.Int) *big.Int {
	ammPrice := pf.getAmmPrice(token)
	if ammPrice.Sign() == 0 {
		return primaryPrice
	}

	diff := new(big.Int).Sub(ammPrice, primaryPrice)
	if diff.Mul(diff.Abs(diff), BasisPointsDivisor).Cmp(
		new(big.Int).Mul(primaryPrice, pf.SpreadThresholdBasisPoints)) < 0 {
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
		if reserve0.Sign() == 0 {
			return bignumber.ZeroBI
		}

		price := new(big.Int).Mul(reserve1, PricePrecision)
		return price.Div(price, reserve0)
	}

	if reserve1.Sign() == 0 {
		return bignumber.ZeroBI
	}

	price := new(big.Int).Mul(reserve0, PricePrecision)
	return price.Div(price, reserve1)
}
