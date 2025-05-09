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

func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenIn, amountIn := params.TokenAmountIn.Token, uint256.MustFromBig(params.TokenAmountIn.Amount)

	asset, err := s.beforeDeposit(tokenIn)
	if err != nil {
		return nil, err
	}

	shares, err := s.erc20Deposit(tokenIn, amountIn, utils.ZeroBI, asset)
	if err != nil {
		return nil, err
	}

	if err = s.afterPublicDeposit(); err != nil {
		return nil, err
	}

	return &poolpkg.CalcAmountOutResult{
		TokenAmountOut: &poolpkg.TokenAmount{Token: params.TokenOut, Amount: shares.ToBig()},
		Fee:            &poolpkg.TokenAmount{Token: tokenIn, Amount: bignumber.ZeroBI},
		Gas:            s.gas.Deposit,
	}, nil
}

func (s *PoolSimulator) beforeDeposit(tokenIn string) (*Asset, error) {
	if s.isTellerPaused {
		return nil, ErrTellerPaused
	}

	asset, ok := s.assets[tokenIn]
	if !ok || !asset.AllowDeposits {
		return nil, ErrTellerAssetNotSupported
	}

	return &asset, nil
}

func (s *PoolSimulator) erc20Deposit(
	depositAsset string,
	depositAmount, minimumMint *uint256.Int,
	asset *Asset,
) (*uint256.Int, error) {
	if depositAmount.IsZero() {
		return nil, ErrTellerZeroAssets
	}

	rateInQuote, err := s.getRateInQuoteSafe(depositAsset)
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

func (s *PoolSimulator) getRateInQuoteSafe(quote string) (rateInQuote *uint256.Int, err error) {
	if s.accountantState.IsPaused {
		return nil, ErrAccountantPaused
	}

	if strings.EqualFold(quote, s.base) {
		return uint256.MustFromBig(s.accountantState.ExchangeRate), nil
	} else {
		data := s.rateProviders[quote]
		quoteDecimals := s.tokenDecimals[quote]
		exchangeRateInQuoteDecimals := s.changeDecimals(uint256.MustFromBig(s.accountantState.ExchangeRate), s.decimals, quoteDecimals)
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

func (s *PoolSimulator) changeDecimals(amount *uint256.Int, fromDecimals, toDecimals uint8) *uint256.Int {
	if fromDecimals == toDecimals {
		return amount
	} else if fromDecimals < toDecimals {
		return amount.Mul(amount, utils.TenPowInt(toDecimals-fromDecimals))
	} else {
		return amount.Div(amount, utils.TenPowInt(fromDecimals-toDecimals))
	}
}

func (s *PoolSimulator) afterPublicDeposit() error {
	// If the share lock period is greater than 0, then users will not be able to mint and transfer in the same tx
	if s.shareLockPeriod > 0 {
		return ErrTellerSharesAreLocked
	}
	return nil
}

func (s *PoolSimulator) UpdateBalance(_ pool.UpdateBalanceParams) {}

func (s *PoolSimulator) GetMetaInfo(_, tokenOut string) interface{} {
	return PoolMeta{
		BlockNumber:     s.Pool.Info.BlockNumber,
		ApprovalAddress: strings.ToLower(tokenOut), // tokenOut is vault
	}
}

func (s *PoolSimulator) CanSwapTo(token string) []string {
	if token != s.Pool.Info.Tokens[0] {
		return []string{}
	}
	tokens := make([]string, 0, len(s.Pool.Info.Tokens)-1)
	for _, t := range s.Pool.Info.Tokens {
		if !strings.EqualFold(t, token) && s.assets[t].AllowDeposits {
			tokens = append(tokens, t)
		}
	}
	return tokens
}

func (s *PoolSimulator) CanSwapFrom(token string) []string {
	if asset, ok := s.assets[token]; ok && asset.AllowDeposits {
		return []string{s.Pool.Info.Tokens[0]}
	}
	return []string{}
}
