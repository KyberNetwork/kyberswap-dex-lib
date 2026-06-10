package cmeth

import (
	"math/big"
	"strings"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	big256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool
	tokenDecimals map[string]uint8

	isTellerPaused  bool
	assets          map[string]Asset
	accountantState AccountantState
	rateProviders   map[string]RateProviderData

	base     string
	decimals uint8

	gas Gas
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	tokenDecimals := make(map[string]uint8)
	tokens := make([]string, 0, len(entityPool.Tokens))
	reserves := make([]*big.Int, 0, len(entityPool.Reserves))
	for i, token := range entityPool.Tokens {
		tokenDecimals[token.Address] = token.Decimals
		tokens = append(tokens, token.Address)
		reserves = append(reserves, bignumber.NewBig(entityPool.Reserves[i]))
	}

	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:     entityPool.Address,
			Exchange:    entityPool.Exchange,
			Type:        entityPool.Type,
			Tokens:      tokens,
			Reserves:    reserves,
			BlockNumber: entityPool.BlockNumber,
		}},
		tokenDecimals: tokenDecimals,

		isTellerPaused:  extra.IsTellerPaused,
		assets:          extra.Assets,
		accountantState: extra.AccountantState,
		rateProviders:   extra.RateProviders,

		base:     staticExtra.Base,
		decimals: staticExtra.Decimals,

		gas: defaultGas,
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenIn, amountIn := params.TokenAmountIn.Token, uint256.MustFromBig(params.TokenAmountIn.Amount)

	asset, err := p.beforeDeposit(tokenIn)
	if err != nil {
		return nil, err
	}

	shares, err := p.erc20Deposit(tokenIn, amountIn, big256.U0, asset)
	if err != nil {
		return nil, err
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: params.TokenOut, Amount: shares.ToBig()},
		Fee:            &pool.TokenAmount{Token: tokenIn, Amount: bignumber.ZeroBI},
		Gas:            p.gas.Deposit,
	}, nil
}

func (p *PoolSimulator) beforeDeposit(tokenIn string) (*Asset, error) {
	if p.isTellerPaused {
		return nil, ErrTellerPaused
	}

	asset, ok := p.assets[tokenIn]
	if !ok || !asset.AllowDeposits {
		return nil, ErrTellerAssetNotSupported
	}

	return &asset, nil
}

func (p *PoolSimulator) erc20Deposit(
	depositAsset string,
	depositAmount, minimumMint *uint256.Int,
	_ *Asset,
) (*uint256.Int, error) {
	if depositAmount.IsZero() {
		return nil, ErrTellerZeroAssets
	}

	rateInQuote, err := p.getRateInQuoteSafe(depositAsset)
	if err != nil {
		return nil, err
	}
	shares, overflow := depositAmount.MulDivOverflow(oneShare, depositAmount, rateInQuote)
	if overflow {
		return nil, ErrMulDivOverflow
	}

	if shares.Lt(minimumMint) {
		return nil, ErrTellerMinimumMintNotMet
	}

	return shares, nil
}

func (p *PoolSimulator) getRateInQuoteSafe(quote string) (rateInQuote *uint256.Int, err error) {
	if p.accountantState.IsPaused {
		return nil, ErrAccountantPaused
	}

	if strings.EqualFold(quote, p.base) {
		return uint256.MustFromBig(p.accountantState.ExchangeRate), nil
	}

	data := p.rateProviders[quote]
	quoteDecimals := p.tokenDecimals[quote]
	exchangeRateInQuoteDecimals := p.changeDecimals(uint256.MustFromBig(p.accountantState.ExchangeRate), p.decimals, quoteDecimals)
	if !data.IsPeggedToBase {
		return nil, ErrTellerAssetNotSupported
	}
	rateInQuote = exchangeRateInQuoteDecimals

	return rateInQuote, nil
}

func (p *PoolSimulator) changeDecimals(amount *uint256.Int, fromDecimals, toDecimals uint8) *uint256.Int {
	if fromDecimals == toDecimals {
		return amount
	} else if fromDecimals < toDecimals {
		return amount.Mul(amount, big256.TenPow(toDecimals-fromDecimals))
	}
	return amount.Div(amount, big256.TenPow(fromDecimals-toDecimals))
}

func (p *PoolSimulator) UpdateBalance(_ pool.UpdateBalanceParams) {}

func (p *PoolSimulator) GetMetaInfo(tokenIn, tokenOut string) any {
	return PoolMeta{
		BlockNumber:     p.Info.BlockNumber,
		ApprovalAddress: p.GetApprovalAddress(tokenIn, tokenOut),
	}
}

func (p *PoolSimulator) GetApprovalAddress(_, tokenOut string) string {
	return tokenOut
}

func (p *PoolSimulator) CanSwapTo(token string) []string {
	if token != p.Info.Tokens[0] {
		return []string{}
	}
	tokens := make([]string, 0, len(p.Info.Tokens)-1)
	for _, t := range p.Info.Tokens {
		if !strings.EqualFold(t, token) && p.assets[t].AllowDeposits {
			tokens = append(tokens, t)
		}
	}
	return tokens
}

func (p *PoolSimulator) CanSwapFrom(token string) []string {
	if asset, ok := p.assets[token]; ok && asset.AllowDeposits {
		return []string{p.Info.Tokens[0]}
	}
	return []string{}
}
