package virtual

import (
	"math/big"
	"strings"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	tokentax "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v2/token-tax"
)

func SupportsFactory(factory string) bool {
	switch strings.ToLower(factory) {
	case factoryBase, factoryEthereum:
		return true
	default:
		return false
	}
}

func NewTracker(pool entity.Pool) tokentax.Tracker {
	tokenAddress := tokentax.FindPairedToken(pool, baseTokens)
	if tokenAddress == "" {
		return tokentax.NewStaticTracker(tokentax.Result{Checked: true})
	}
	return &tracker{poolAddress: pool.Address, tokenAddress: tokenAddress}
}

type tracker struct {
	poolAddress  string
	tokenAddress string

	isLiquidityPool bool
	buyTaxBps       *big.Int
	sellTaxBps      *big.Int
}

func (t *tracker) AddTaxCalls(request *ethrpc.Request) bool {
	request.AddCall(&ethrpc.Call{
		ABI: tokenTaxABI, Target: t.tokenAddress, Method: methodIsLiquidityPool,
		Params: []any{common.HexToAddress(t.poolAddress)},
	}, []any{&t.isLiquidityPool})
	request.AddCall(&ethrpc.Call{
		ABI: tokenTaxABI, Target: t.tokenAddress, Method: methodBuyTax,
	}, []any{&t.buyTaxBps})
	request.AddCall(&ethrpc.Call{
		ABI: tokenTaxABI, Target: t.tokenAddress, Method: methodSellTax,
	}, []any{&t.sellTaxBps})
	return true
}

func (t *tracker) TaxResult() tokentax.Result {
	if t.buyTaxBps == nil && t.sellTaxBps == nil {
		return tokentax.Result{Checked: true}
	}

	result := tokentax.Result{
		Protocol:     Protocol,
		TokenAddress: t.tokenAddress,
		Checked:      true,
	}
	if t.isLiquidityPool {
		result.BuyTaxBps = toUint256(t.buyTaxBps)
		result.SellTaxBps = toUint256(t.sellTaxBps)
	}
	return result
}

func toUint256(value *big.Int) *uint256.Int {
	if value == nil {
		return nil
	}
	result, _ := uint256.FromBig(value)
	return result
}
