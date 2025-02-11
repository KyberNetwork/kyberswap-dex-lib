package bancorv3

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type Gas struct {
	Swap int64
}

type SwapInfo struct {
	IsSourceNative bool                       `json:"isSourceNative"`
	IsTargetNative bool                       `json:"isTargetNative"`
	TradeInfo      []*poolCollectionTradeInfo `json:"-"`
}

type PoolMetaInfo struct{}

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

	tradeTokens struct {
		SourceToken string
		TargetToken string
	}

	tradeParams struct {
		Amount         *uint256.Int
		Limit          *uint256.Int
		BySourceAmount bool
		IgnoreFees     bool
	}

	tradeResult struct {
		SourceAmount     *uint256.Int
		TargetAmount     *uint256.Int
		TradingFeeAmount *uint256.Int
		NetworkFeeAmount *uint256.Int

		PoolCollectionTradeInfo *poolCollectionTradeInfo
	}
)
