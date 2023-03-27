package swapdata

import (
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/valueobject"
)

type Fee struct {
	Fee       uint32
	Precision uint32
}

var (
	DefaultFee = Fee{Fee: 30, Precision: 10000}

	FeeByExchange = map[valueobject.Exchange]Fee{
		valueobject.ExchangePancake:         {Fee: 25, Precision: 10000},
		valueobject.ExchangePancakeLegacy:   {Fee: 25, Precision: 10000},
		valueobject.ExchangeApeSwap:         {Fee: 20, Precision: 10000},
		valueobject.ExchangeWault:           {Fee: 20, Precision: 10000},
		valueobject.ExchangeBiSwap:          {Fee: 1, Precision: 1000},
		valueobject.ExchangePolyDex:         {Fee: 10, Precision: 10000},
		valueobject.ExchangeJetSwap:         {Fee: 10, Precision: 10000},
		valueobject.ExchangePolycat:         {Fee: 25, Precision: 10000},
		valueobject.ExchangeSpookySwap:      {Fee: 20, Precision: 10000},
		valueobject.ExchangeAxial:           {Fee: 20, Precision: 10000},
		valueobject.ExchangeCronaSwap:       {Fee: 25, Precision: 10000},
		valueobject.ExchangeGravity:         {Fee: 25, Precision: 10000},
		valueobject.ExchangeKyberSwap:       {Fee: 30, Precision: 10000},
		valueobject.ExchangeKyberSwapStatic: {Fee: 30, Precision: 10000},
		valueobject.ExchangeMMF:             {Fee: 17, Precision: 10000},
		valueobject.ExchangeKrptoDex:        {Fee: 20, Precision: 10000},
		valueobject.ExchangeCometh:          {Fee: 50, Precision: 10000},
		valueobject.ExchangeDinoSwap:        {Fee: 18, Precision: 10000},
		valueobject.ExchangeSafeSwap:        {Fee: 25, Precision: 10000},
		valueobject.ExchangePantherSwap:     {Fee: 20, Precision: 10000},
		valueobject.ExchangeMorpheus:        {Fee: 15, Precision: 10000},
		valueobject.ExchangeSwapr:           {Fee: 25, Precision: 10000},
		valueobject.ExchangeWagyuSwap:       {Fee: 20, Precision: 10000},
		valueobject.ExchangeAstroSwap:       {Fee: 20, Precision: 10000},
		valueobject.ExchangeDystopia:        {Fee: 5, Precision: 10000},
	}

	FeeByChain = map[valueobject.ChainID]map[valueobject.Exchange]Fee{
		valueobject.ChainIDBSC: {
			valueobject.ExchangeJetSwap: {Fee: 30, Precision: 10000},
		},
	}
)

func GetFee(chainID valueobject.ChainID, exchange valueobject.Exchange) Fee {
	fee, ok := getFeeByChain(chainID, exchange)
	if ok {
		return fee
	}

	fee, ok = FeeByExchange[exchange]
	if ok {
		return fee
	}

	return DefaultFee
}

func getFeeByChain(chainID valueobject.ChainID, exchange valueobject.Exchange) (Fee, bool) {
	feeByExchange, ok := FeeByChain[chainID]
	if !ok {
		return Fee{}, false
	}

	fee, ok := feeByExchange[exchange]
	if !ok {
		return Fee{}, false
	}

	return fee, true
}
