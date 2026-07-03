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
	_, ok := factories[strings.ToLower(factoryAddress)]
	return ok
}

func FindTaxToken(pool entity.Pool) string {
	return tokentax.FindPairedToken(pool, baseTokens)
}

func NewTracker(poolAddress, tokenAddress string, previous tokentax.TaxInfo) tokentax.Tracker {
	return &tracker{
		poolAddress:  poolAddress,
		tokenAddress: tokenAddress,
		previous:     previous,
		pairVerified: previous.Protocol == Protocol && previous.Token == tokenAddress,
		pairCall:     -1,
		feeRateCall:  -1,
		buyTaxCall:   -1,
		sellTaxCall:  -1,
	}
}

type tracker struct {
	poolAddress  string
	tokenAddress string
	previous     tokentax.TaxInfo
	pairVerified bool

	pairCall    int
	feeRateCall int
	buyTaxCall  int
	sellTaxCall int

	pairAddress common.Address
	feeRatePct  *big.Int
	buyTaxPct   *big.Int
	sellTaxPct  *big.Int
}

func (t *tracker) AddCalls(request *ethrpc.Request) {
	if !t.pairVerified {
		t.pairCall = len(request.Calls)
		request.AddCall(&ethrpc.Call{
			ABI: tokenTaxABI, Target: t.tokenAddress, Method: methodPair,
		}, []any{&t.pairAddress})
	}

	t.feeRateCall = len(request.Calls)
	request.AddCall(&ethrpc.Call{
		ABI: tokenTaxABI, Target: t.tokenAddress, Method: methodFeeRate,
	}, []any{&t.feeRatePct})

	t.buyTaxCall = len(request.Calls)
	request.AddCall(&ethrpc.Call{
		ABI: tokenTaxABI, Target: t.tokenAddress, Method: methodBuyTax,
	}, []any{&t.buyTaxPct})

	t.sellTaxCall = len(request.Calls)
	request.AddCall(&ethrpc.Call{
		ABI: tokenTaxABI, Target: t.tokenAddress, Method: methodSellTax,
	}, []any{&t.sellTaxPct})
}

func (t *tracker) Resolve(response *ethrpc.Response) tokentax.TaxInfo {
	feeRateOK := tokentax.CallSucceeded(response, t.feeRateCall)
	buyTaxOK := tokentax.CallSucceeded(response, t.buyTaxCall)
	sellTaxOK := tokentax.CallSucceeded(response, t.sellTaxCall)

	// Fee methods identify four.meme tokens. If none responds, this token is unsupported and should
	// not be probed again. The immutable pair read only verifies the canonical pool on first run.
	if !feeRateOK && !buyTaxOK && !sellTaxOK {
		return tokentax.TaxInfo{Checked: true}
	}

	if !t.pairVerified {
		if !tokentax.CallSucceeded(response, t.pairCall) {
			return tokentax.TaxInfo{Checked: true}
		}
		if t.pairAddress != common.HexToAddress(t.poolAddress) {
			return tokentax.TaxInfo{Checked: true}
		}
	}

	info := tokentax.TaxInfo{
		Protocol: Protocol,
		Token:    t.tokenAddress,
		Checked:  true,
	}

	// Token8: feeRateBuy/feeRateSell are expressed in percent
	if buyTaxOK || sellTaxOK {
		if buyTaxOK {
			info.BuyTaxBps = percentToBps(t.buyTaxPct)
		} else {
			info.BuyTaxBps = t.previous.BuyTaxBps
		}
		if sellTaxOK {
			info.SellTaxBps = percentToBps(t.sellTaxPct)
		} else {
			info.SellTaxBps = t.previous.SellTaxBps
		}
		return info
	}

	// Token5: fee = amount * feeRate / 10000, charged symmetrically on buy and sell, so feeRate is
	// already the basis-point rate for both directions
	if feeRateOK {
		info.BuyTaxBps = tokentax.ToUint256(t.feeRatePct)
		info.SellTaxBps = tokentax.ToUint256(t.feeRatePct)
	}

	return info
}

func percentToBps(value *big.Int) *uint256.Int {
	result := tokentax.ToUint256(value)
	if result == nil {
		return nil
	}
	return result.Mul(result, big256.U100)
}
