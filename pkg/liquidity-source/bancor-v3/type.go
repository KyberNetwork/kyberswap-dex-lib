package bancorv3

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type SwapInfo struct {
	IsSourceNative bool                       `json:"isSourceNative"`
	IsTargetNative bool                       `json:"isTargetNative"`
	TradeInfo      []*poolCollectionTradeInfo `json:"-"`
}

type StaticExtra struct {
	BNT     string              `json:"bnt"`
	ChainID valueobject.ChainID `json:"chainId"`
}

type Extra struct {
	NativeIdx        int                        `json:"nativeIdx"`
	CollectionByPool map[string]string          `json:"collectionByPool"`
	PoolCollections  map[string]*poolCollection `json:"poolCollections"`
}

type (
	poolCollectionResp struct {
		PoolData      map[string]*poolDataResp
		NetworkFeePMM uint32
	}

	poolDataResp struct {
		PoolToken         common.Address
		TradingFeePPM     uint32
		TradingEnabled    bool
		DepositingEnabled bool
		AverageRates      averageRatesResp
		PoolLiquidity     poolLiquidityResp
	}

	averageRatesResp struct {
		BlockNumber uint32
		Rate        fraction112Resp
		InvRate     fraction112Resp
	}

	fraction112Resp struct {
		N *big.Int
		D *big.Int
	}

	poolLiquidityResp struct {
		BntTradingLiquidity       *big.Int
		BaseTokenTradingLiquidity *big.Int
		StakedBalance             *big.Int
	}
)
