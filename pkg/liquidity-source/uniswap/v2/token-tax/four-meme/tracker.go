package fourmeme

import (
	"math/big"
	"strings"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	tokentax "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v2/token-tax"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

func SupportsFactory(factoryAddress string) bool {
	return strings.EqualFold(factoryAddress, factory)
}

func NewTracker(pool entity.Pool, previous tokentax.Result) tokentax.Tracker {
	tokenAddress := tokentax.FindPairedToken(pool, baseTokens)
	if tokenAddress == "" {
		return tokentax.NewStaticTracker(tokentax.Result{Checked: true})
	}
	if previous.Checked {
		previous.Protocol = Protocol
		return tokentax.NewStaticTracker(previous)
	}
	return &tracker{poolAddress: pool.Address, tokenAddress: tokenAddress}
}

type tracker struct {
	poolAddress  string
	tokenAddress string

	pairAddress common.Address
	buyTaxPct   *big.Int
	sellTaxPct  *big.Int
}

func (t *tracker) AddTaxCalls(request *ethrpc.Request) bool {
	request.AddCall(&ethrpc.Call{
		ABI: tokenTaxABI, Target: t.tokenAddress, Method: methodPair,
	}, []any{&t.pairAddress})
	request.AddCall(&ethrpc.Call{
		ABI: tokenTaxABI, Target: t.tokenAddress, Method: methodBuyTax,
	}, []any{&t.buyTaxPct})
	request.AddCall(&ethrpc.Call{
		ABI: tokenTaxABI, Target: t.tokenAddress, Method: methodSellTax,
	}, []any{&t.sellTaxPct})
	return true
}

func (t *tracker) TaxResult() tokentax.Result {
	if t.pairAddress != common.HexToAddress(t.poolAddress) {
		return tokentax.Result{Checked: true}
	}
	return tokentax.Result{
		Protocol:     Protocol,
		TokenAddress: t.tokenAddress,
		BuyTaxBps:    percentToBps(t.buyTaxPct),
		SellTaxBps:   percentToBps(t.sellTaxPct),
		Checked:      true,
	}
}

func percentToBps(value *big.Int) *uint256.Int {
	if value == nil {
		return nil
	}
	result, _ := uint256.FromBig(value)
	return result.Mul(result, big256.U100)
}
