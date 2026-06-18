package uniswapv2

import (
	"context"
	"math/big"
	"strings"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

// TokenTax is the per-pool tax state a tracker detects and the simulator consumes.
// Rates are in basis points. Checked marks that detection already ran.
type TokenTax struct {
	Token   string
	BuyTax  *uint256.Int
	SellTax *uint256.Int
	Checked bool
}

// resolveTokenTax decides, from the pool's factory, tokens and previously stored state, whether and
// how to (re)fetch transfer tax. Pools matching no protocol, or already probed as non-tax, do no RPC.
func resolveTokenTax(
	ctx context.Context, client *ethrpc.Client, p entity.Pool, prev TokenTax, factory string, blockNumber *big.Int,
) (TokenTax, error) {
	if prev.Checked && prev.Token == "" {
		return prev, nil // probed before, not a supported tax pool
	}

	// Virtual: rates mutable and pools can be added/removed, so re-fetch every cycle.
	if _, ok := virtualFactories[factory]; ok {
		if agent := pairedToken(p, virtualBaseTokens); agent != "" {
			return fetchVirtualTax(ctx, client, p, agent, blockNumber)
		}
	}

	// four.meme: rates immutable on a single pair, so probe once then reuse the cache.
	if _, ok := fourMemeFactories[factory]; ok {
		if agent := pairedToken(p, fourMemeBaseTokens); agent != "" {
			if prev.Checked {
				return prev, nil
			}
			return fetchFourMemeTax(ctx, client, p, agent, blockNumber)
		}
	}

	return TokenTax{Checked: true}, nil
}

// pairedToken returns the non-base token (lowercase) when the pool pairs a base token, else "".
func pairedToken(p entity.Pool, baseTokens map[string]struct{}) string {
	if len(p.Tokens) != 2 {
		return ""
	}
	for i, tok := range p.Tokens {
		if _, ok := baseTokens[strings.ToLower(tok.Address)]; ok {
			return strings.ToLower(p.Tokens[1-i].Address)
		}
	}
	return ""
}

// fetchVirtualTax probes a Virtual agent token (isLiquidityPool / totalBuyTaxBasisPoints /
// totalSellTaxBasisPoints). Rates are mutable and pools can be added/removed, so it re-fetches each
// cycle; Token stays set (keeping the pool eligible) while rates are nil when this pool is unregistered.
func fetchVirtualTax(
	ctx context.Context, client *ethrpc.Client, p entity.Pool, agent string, blockNumber *big.Int,
) (TokenTax, error) {
	var (
		isLP            bool
		buyTax, sellTax *big.Int
	)
	req := client.NewRequest().SetContext(ctx)
	if blockNumber != nil {
		req.SetBlockNumber(blockNumber)
	}
	req.AddCall(&ethrpc.Call{ABI: tokenTaxABI, Target: agent, Method: tokenMethodIsLiquidityPool, Params: []any{common.HexToAddress(p.Address)}}, []any{&isLP})
	req.AddCall(&ethrpc.Call{ABI: tokenTaxABI, Target: agent, Method: tokenMethodTotalBuyTax}, []any{&buyTax})
	req.AddCall(&ethrpc.Call{ABI: tokenTaxABI, Target: agent, Method: tokenMethodTotalSellTax}, []any{&sellTax})

	resp, err := req.TryAggregate()
	if err != nil {
		return TokenTax{}, err
	}
	if !resp.Result[1] && !resp.Result[2] {
		return TokenTax{Checked: true}, nil // not a Virtual agent token
	}

	tax := TokenTax{Token: agent, Checked: true}
	if isLP {
		tax.BuyTax = toUint256(resp.Result[1], buyTax)
		tax.SellTax = toUint256(resp.Result[2], sellTax)
	}
	return tax, nil
}

// fetchFourMemeTax probes a four.meme agent token (pair / feeRateBuy / feeRateSell, rates in percent).
// It taxes only its canonical pair; rates are immutable, so this runs once per pool.
//
// Known limitation: a sell also triggers the token's _dispatchFee, which (when tokenAccumulated >=
// minDispatch) swaps accumulated tax tokens into this same pair before the user's swap, shifting
// reserves. That autoswap is not modeled, so sell quotes can be slightly high when dispatch fires.
func fetchFourMemeTax(
	ctx context.Context, client *ethrpc.Client, p entity.Pool, agent string, blockNumber *big.Int,
) (TokenTax, error) {
	var (
		pairAddr        common.Address
		feeBuy, feeSell *big.Int
	)
	req := client.NewRequest().SetContext(ctx)
	if blockNumber != nil {
		req.SetBlockNumber(blockNumber)
	}
	req.AddCall(&ethrpc.Call{ABI: tokenTaxABI, Target: agent, Method: tokenMethodPair}, []any{&pairAddr})
	req.AddCall(&ethrpc.Call{ABI: tokenTaxABI, Target: agent, Method: tokenMethodFeeRateBuy}, []any{&feeBuy})
	req.AddCall(&ethrpc.Call{ABI: tokenTaxABI, Target: agent, Method: tokenMethodFeeRateSell}, []any{&feeSell})

	resp, err := req.TryAggregate()
	if err != nil {
		return TokenTax{}, err
	}
	if !resp.Result[0] || pairAddr != common.HexToAddress(p.Address) {
		return TokenTax{Checked: true}, nil // not the four.meme token of this pool
	}

	return TokenTax{
		Token:   agent,
		BuyTax:  percentToBps(resp.Result[1], feeBuy),
		SellTax: percentToBps(resp.Result[2], feeSell),
		Checked: true,
	}, nil
}

func toUint256(ok bool, v *big.Int) *uint256.Int {
	if !ok || v == nil {
		return nil
	}
	out, _ := uint256.FromBig(v)
	return out
}

func percentToBps(ok bool, v *big.Int) *uint256.Int {
	out := toUint256(ok, v)
	if out == nil {
		return nil
	}
	return out.Mul(out, big256.U100)
}

func tokenAtIndex(p entity.Pool, idx int) string {
	if idx < 0 || idx >= len(p.Tokens) {
		return ""
	}
	return strings.ToLower(p.Tokens[idx].Address)
}

func findTokenIndex(tokens []*entity.PoolToken, token string) int {
	if token == "" {
		return -1
	}
	for i, tok := range tokens {
		if strings.ToLower(tok.Address) == token {
			return i
		}
	}
	return -1
}
