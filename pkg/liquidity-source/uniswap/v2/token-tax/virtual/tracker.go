package virtual

import (
	"math/big"
	"strings"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	tokentax "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v2/token-tax"
)

func SupportsFactory(factory string) bool {
	_, ok := factories[strings.ToLower(factory)]
	return ok
}

func FindTaxToken(pool entity.Pool) string {
	return tokentax.FindPairedToken(pool, baseTokens)
}

func NewTracker(poolAddress, tokenAddress, factory string, previous tokentax.TaxInfo) tokentax.Tracker {
	buyTaxMethod, sellTaxMethod := methodBuyTax, methodSellTax
	if _, ok := projectTaxFactories[strings.ToLower(factory)]; ok {
		buyTaxMethod, sellTaxMethod = methodProjectBuyTax, methodProjectSellTax
	}
	return &tracker{
		poolAddress:         poolAddress,
		tokenAddress:        tokenAddress,
		buyTaxMethod:        buyTaxMethod,
		sellTaxMethod:       sellTaxMethod,
		previous:            previous,
		isLiquidityPoolCall: -1,
		buyTaxCall:          -1,
		sellTaxCall:         -1,
	}
}

type tracker struct {
	poolAddress   string
	tokenAddress  string
	buyTaxMethod  string
	sellTaxMethod string
	previous      tokentax.TaxInfo

	isLiquidityPoolCall int
	buyTaxCall          int
	sellTaxCall         int

	isLiquidityPool bool
	buyTaxBps       *big.Int
	sellTaxBps      *big.Int
}

func (t *tracker) AddCalls(request *ethrpc.Request) {
	t.isLiquidityPoolCall = len(request.Calls)
	request.AddCall(&ethrpc.Call{
		ABI: tokenTaxABI, Target: t.tokenAddress, Method: methodIsLiquidityPool,
		Params: []any{common.HexToAddress(t.poolAddress)},
	}, []any{&t.isLiquidityPool})

	t.buyTaxCall = len(request.Calls)
	request.AddCall(&ethrpc.Call{
		ABI: tokenTaxABI, Target: t.tokenAddress, Method: t.buyTaxMethod,
	}, []any{&t.buyTaxBps})

	t.sellTaxCall = len(request.Calls)
	request.AddCall(&ethrpc.Call{
		ABI: tokenTaxABI, Target: t.tokenAddress, Method: t.sellTaxMethod,
	}, []any{&t.sellTaxBps})
}

func (t *tracker) Resolve(response *ethrpc.Response) tokentax.TaxInfo {
	isLiquidityPoolOK := tokentax.CallSucceeded(response, t.isLiquidityPoolCall)
	buyTaxOK := tokentax.CallSucceeded(response, t.buyTaxCall)
	sellTaxOK := tokentax.CallSucceeded(response, t.sellTaxCall)

	// A Virtual agent token supports these methods. Only mark it unsupported when all probes fail;
	// any successful probe is enough to remember the token and refresh it again next cycle.
	if !isLiquidityPoolOK && !buyTaxOK && !sellTaxOK {
		if t.previous.Token != "" {
			return t.previous
		}
		return tokentax.TaxInfo{Checked: true}
	}

	info := tokentax.TaxInfo{
		Protocol: Protocol,
		Token:    t.tokenAddress,
		Checked:  true,
	}
	if !isLiquidityPoolOK {
		info.BuyTaxBps = t.previous.BuyTaxBps
		info.SellTaxBps = t.previous.SellTaxBps
		return info
	}
	if !t.isLiquidityPool {
		return info
	}
	if buyTaxOK {
		info.BuyTaxBps = tokentax.ToUint256(t.buyTaxBps)
	} else {
		info.BuyTaxBps = t.previous.BuyTaxBps
	}
	if sellTaxOK {
		info.SellTaxBps = tokentax.ToUint256(t.sellTaxBps)
	} else {
		info.SellTaxBps = t.previous.SellTaxBps
	}
	return info
}
