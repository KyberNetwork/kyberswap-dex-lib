//go:generate go run github.com/tinylib/msgp -unexported -tests=false -v
//msgp:tuple VaultPriceFeed
//msgp:shim *big.Int as:[]byte using:msgpencode.EncodeInt/msgpencode.DecodeInt
//msgp:shim common.Address as:[]byte using:(common.Address).Bytes/common.BytesToAddress
//msgp:shim IFastPriceFeed as:[]byte using:encodePriceFeed/decodePriceFeed
//msgp:ignore IFastPriceFeed

package quickperps

import (
	"encoding/json"
	"math/big"
	"time"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/ethereum/go-ethereum/common"
)

type VaultPriceFeed struct {
	FavorPrimaryPrice          bool     `json:"favorPrimaryPrice"`
	IsSecondaryPriceEnabled    bool     `json:"isSecondaryPriceEnabled"`
	MaxStrictPriceDeviation    *big.Int `json:"maxStrictPriceDeviation"`
	PriceSampleSpace           *big.Int `json:"priceSampleSpace"`
	SpreadThresholdBasisPoints *big.Int `json:"spreadThresholdBasisPoints"`
	ExpireTimeForPriceFeed     *big.Int `json:"expireTimeForPriceFeed"`

	PriceDecimals         map[string]*big.Int `json:"priceDecimals"`
	SpreadBasisPoints     map[string]*big.Int `json:"spreadBasisPoints"`
	AdjustmentBasisPoints map[string]*big.Int `json:"adjustmentBasisPoints"`
	StrictStableTokens    map[string]bool     `json:"strictStableTokens"`
	IsAdjustmentAdditive  map[string]bool     `json:"isAdjustmentAdditive"`

	SecondaryPriceFeedAddress common.Address `json:"-"`
	SecondaryPriceFeed        IFastPriceFeed `json:"secondaryPriceFeed"`
	SecondaryPriceFeedVersion int            `json:"secondaryPriceFeedVersion"`

	PriceFeedsAddresses map[string]common.Address `json:"-"`
	PriceFeedProxies    map[string]*PriceFeed     `json:"priceFeeds"`
}

func NewVaultPriceFeed() *VaultPriceFeed {
	return &VaultPriceFeed{
		PriceDecimals:         make(map[string]*big.Int),
		SpreadBasisPoints:     make(map[string]*big.Int),
		AdjustmentBasisPoints: make(map[string]*big.Int),
		StrictStableTokens:    make(map[string]bool),
		IsAdjustmentAdditive:  make(map[string]bool),
		PriceFeedsAddresses:   make(map[string]common.Address),
		PriceFeedProxies:      make(map[string]*PriceFeed),
	}
}

const (
	vaultPriceFeedMethodFavorPrimaryPrice          = "favorPrimaryPrice"
	vaultPriceFeedMethodIsSecondaryPriceEnabled    = "isSecondaryPriceEnabled"
	vaultPriceFeedMethodMaxStrictPriceDeviation    = "maxStrictPriceDeviation"
	vaultPriceFeedMethodSecondaryPriceFeed         = "secondaryPriceFeed"
	vaultPriceFeedMethodSpreadThresholdBasisPoints = "spreadThresholdBasisPoints"
	vaultPriceFeedMethodExpireTimeForPriceFeed     = "expireTimeForPriceFeed"

	vaultPriceFeedMethodPriceFeedProxies      = "priceFeedProxies"
	vaultPriceFeedMethodPriceDecimals         = "priceDecimals"
	vaultPriceFeedMethodSpreadBasisPoints     = "spreadBasisPoints"
	vaultPriceFeedMethodAdjustmentBasisPoints = "adjustmentBasisPoints"
	vaultPriceFeedMethodStrictStableTokens    = "strictStableTokens"
	vaultPriceFeedMethodIsAdjustmentAdditive  = "isAdjustmentAdditive"
)

func (pf *VaultPriceFeed) UnmarshalJSON(bytes []byte) error {
	var priceFeed struct {
		FavorPrimaryPrice          bool                  `json:"favorPrimaryPrice"`
		IsSecondaryPriceEnabled    bool                  `json:"isSecondaryPriceEnabled"`
		MaxStrictPriceDeviation    *big.Int              `json:"maxStrictPriceDeviation"`
		SpreadThresholdBasisPoints *big.Int              `json:"spreadThresholdBasisPoints"`
		ExpireTimeForPriceFeed     *big.Int              `json:"expireTimeForPriceFeed"`
		PriceDecimals              map[string]*big.Int   `json:"priceDecimals"`
		SpreadBasisPoints          map[string]*big.Int   `json:"spreadBasisPoints"`
		AdjustmentBasisPoints      map[string]*big.Int   `json:"adjustmentBasisPoints"`
		StrictStableTokens         map[string]bool       `json:"strictStableTokens"`
		IsAdjustmentAdditive       map[string]bool       `json:"isAdjustmentAdditive"`
		SecondaryPriceFeedVersion  int                   `json:"secondaryPriceFeedVersion"`
		PriceFeedProxies           map[string]*PriceFeed `json:"priceFeeds"`
	}

	if err := json.Unmarshal(bytes, &priceFeed); err != nil {
		return err
	}

	pf.FavorPrimaryPrice = priceFeed.FavorPrimaryPrice
	pf.IsSecondaryPriceEnabled = priceFeed.IsSecondaryPriceEnabled
	pf.MaxStrictPriceDeviation = priceFeed.MaxStrictPriceDeviation
	pf.SpreadThresholdBasisPoints = priceFeed.SpreadThresholdBasisPoints
	pf.ExpireTimeForPriceFeed = priceFeed.ExpireTimeForPriceFeed
	pf.PriceDecimals = priceFeed.PriceDecimals
	pf.SpreadBasisPoints = priceFeed.SpreadBasisPoints
	pf.AdjustmentBasisPoints = priceFeed.AdjustmentBasisPoints
	pf.StrictStableTokens = priceFeed.StrictStableTokens
	pf.IsAdjustmentAdditive = priceFeed.IsAdjustmentAdditive
	pf.SecondaryPriceFeedVersion = priceFeed.SecondaryPriceFeedVersion
	pf.PriceFeedProxies = priceFeed.PriceFeedProxies

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

func (pf *VaultPriceFeed) GetPrice(token string, maximise bool, _ bool, _ bool) (*big.Int, error) {
	var price *big.Int
	var err error

	price, err = pf.getPriceV1(token, maximise)
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
	price, err := pf.getPrimaryPrice(token)
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

func (pf *VaultPriceFeed) getPrimaryPrice(token string) (*big.Int, error) {
	priceFeed, ok := pf.PriceFeedProxies[token]
	if !ok {
		return nil, ErrVaultPriceFeedInvalidPriceFeed
	}
	price := priceFeed.Price

	if price.Cmp(bignumber.ZeroBI) <= 0 {
		return nil, ErrVaultPriceFeedInvalidPrice
	}
	timestamp := big.NewInt(int64(priceFeed.Timestamp))
	if new(big.Int).Add(timestamp, pf.ExpireTimeForPriceFeed).Cmp(big.NewInt(time.Now().Unix())) <= 0 {
		return nil, ErrVaultPriceFeedExpired
	}

	priceDecimals := new(big.Int).Set(pf.PriceDecimals[token])
	price = new(big.Int).Div(
		new(big.Int).Mul(price, PricePrecision),
		new(big.Int).Exp(big.NewInt(10), priceDecimals, nil),
	)

	return price, nil
}

func (pf *VaultPriceFeed) getSecondaryPrice(token string, referencePrice *big.Int, maximise bool) *big.Int {
	if pf.SecondaryPriceFeed == nil {
		return referencePrice
	}

	return pf.SecondaryPriceFeed.GetPrice(token, referencePrice, maximise)
}
