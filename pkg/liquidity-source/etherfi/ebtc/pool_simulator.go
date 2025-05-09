package etherfiebtc

import (
	"math/big"
	"strings"

	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	utils "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool
	tokenDecimals map[string]uint8

	isTellerPaused  bool
	shareLockPeriod uint64
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
		Pool: poolpkg.Pool{Info: poolpkg.PoolInfo{
			Address:     entityPool.Address,
			ReserveUsd:  entityPool.ReserveUsd,
			Exchange:    entityPool.Exchange,
			Type:        entityPool.Type,
			Tokens:      tokens,
			Reserves:    reserves,
			BlockNumber: entityPool.BlockNumber,
		}},
		tokenDecimals: tokenDecimals,

		isTellerPaused:  extra.IsTellerPaused,
		shareLockPeriod: extra.ShareLockPeriod,
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

	shares, err := p.erc20Deposit(tokenIn, amountIn, utils.ZeroBI, asset)
	if err != nil {
		return nil, err
	}

	if err = p.afterPublicDeposit(); err != nil {
		return nil, err
	}

	return &poolpkg.CalcAmountOutResult{
		TokenAmountOut: &poolpkg.TokenAmount{Token: params.TokenOut, Amount: shares.ToBig()},
		Fee:            &poolpkg.TokenAmount{Token: tokenIn, Amount: bignumber.ZeroBI},
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
	asset *Asset,
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
	if asset.SharePremium > 0 {
		shares, overflow = shares.MulDivOverflow(
			shares, uint256.NewInt(utils.BasisPointUint256.Uint64()-uint64(asset.SharePremium)),
			utils.BasisPointUint256,
		)
		if overflow {
			return nil, ErrMulDivOverflow
		}
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
	} else {
		data := p.rateProviders[quote]
		quoteDecimals := p.tokenDecimals[quote]
		exchangeRateInQuoteDecimals := p.changeDecimals(uint256.MustFromBig(p.accountantState.ExchangeRate), p.decimals, quoteDecimals)
		if !data.IsPeggedToBase {
			logger.WithFields(logger.Fields{
				"quote":         quote,
				"rate provider": data.RateProvider,
			}).Warn("rate provider is not pegged to base")
		}
		rateInQuote = exchangeRateInQuoteDecimals
	}

	return rateInQuote, nil
}

func (p *PoolSimulator) changeDecimals(amount *uint256.Int, fromDecimals, toDecimals uint8) *uint256.Int {
	if fromDecimals == toDecimals {
		return amount
	} else if fromDecimals < toDecimals {
		return amount.Mul(amount, utils.TenPowInt(toDecimals-fromDecimals))
	} else {
		return amount.Div(amount, utils.TenPowInt(fromDecimals-toDecimals))
	}
}

func (p *PoolSimulator) afterPublicDeposit() error {
	// If the share lock period is greater than 0, then users will not be able to mint and transfer in the same tx
	if p.shareLockPeriod > 0 {
		return ErrTellerSharesAreLocked
	}
	return nil
}

func (p *PoolSimulator) UpdateBalance(_ pool.UpdateBalanceParams) {}

func (p *PoolSimulator) GetMetaInfo(_ string, tokenOut string) interface{} {
	return PoolMeta{
		BlockNumber:     p.Pool.Info.BlockNumber,
		ApprovalAddress: strings.ToLower(tokenOut), // tokenOut is vault
	}
}

func (p *PoolSimulator) CanSwapTo(token string) []string {
	if token != p.Pool.Info.Tokens[0] {
		return []string{}
	}
	tokens := make([]string, 0, len(p.Pool.Info.Tokens)-1)
	for _, t := range p.Pool.Info.Tokens {
		if !strings.EqualFold(t, token) && p.assets[t].AllowDeposits {
			tokens = append(tokens, t)
		}
	}
	return tokens
}

func (p *PoolSimulator) CanSwapFrom(token string) []string {
	if asset, ok := p.assets[token]; ok && asset.AllowDeposits {
		return []string{p.Pool.Info.Tokens[0]}
	}
	return []string{}
}
