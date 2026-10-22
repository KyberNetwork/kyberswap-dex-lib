package liquid

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var (
	ErrPaused            = errors.New("teller is paused")
	ErrShareLocked       = errors.New("share lock period is set")
	ErrInvalidRate       = errors.New("invalid accountant rate")
	ErrInvalidToken      = errors.New("invalid token for swap")
	ErrAssetNotSupported = errors.New("asset not supported")
	ErrMinimumMintNotMet = errors.New("minimum mint not met")
)

type PoolSimulator struct {
	pool.Pool
	extra       Extra
	staticExtra StaticExtra
	ONE_SHARE   *big.Int
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:     entityPool.Address,
			Exchange:    entityPool.Exchange,
			Type:        entityPool.Type,
			Tokens:      lo.Map(entityPool.Tokens, func(item *entity.PoolToken, index int) string { return item.Address }),
			Reserves:    lo.Map(entityPool.Reserves, func(item string, index int) *big.Int { return bignumber.NewBig(item) }),
			BlockNumber: entityPool.BlockNumber,
		}},
		extra:       extra,
		staticExtra: staticExtra,
		ONE_SHARE:   bignumber.TenPowInt(entityPool.Tokens[0].Decimals),
	}, nil
}

func (s *PoolSimulator) CanSwapTo(token string) []string {
	if token == s.Info.Tokens[0] {
		res := make([]string, 0, len(s.Info.Tokens)-1)
		for i, t := range s.Info.Tokens[1:] {
			if s.extra.AssetData[i].AllowDeposits {
				res = append(res, t)
			}
		}

		return res
	}

	return nil
}

func (s *PoolSimulator) CanSwapFrom(token string) []string {
	for i, t := range s.Info.Tokens[1:] {
		if t == token && s.extra.AssetData[i].AllowDeposits {
			return []string{s.Info.Tokens[0]}
		}
	}

	return nil
}

func (s *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	if s.extra.IsTellerPaused {
		return nil, ErrPaused
	} else if s.extra.ShareLockPeriod > 0 {
		// Locked shares can't be transferred out of the executor.
		return nil, ErrShareLocked
	}

	indexIn := s.GetTokenIndex(param.TokenAmountIn.Token)
	if indexIn == -1 || indexIn-1 >= len(s.extra.AssetData) || indexIn-1 >= len(s.extra.RateInQuote) {
		return nil, ErrInvalidToken
	}

	asset := s.extra.AssetData[indexIn-1]
	if !asset.AllowDeposits {
		return nil, ErrAssetNotSupported
	}

	rate := s.extra.RateInQuote[indexIn-1]
	if rate == nil || rate.Sign() <= 0 { // getRateInQuoteSafe reverts while the accountant is paused
		return nil, ErrInvalidRate
	}

	var shares, tmp big.Int
	bignumber.MulDivDown(&shares, param.TokenAmountIn.Amount, s.ONE_SHARE, rate)

	if asset.SharePremium > 0 {
		bignumber.MulDivDown(
			&shares,
			&shares,
			tmp.Sub(bignumber.BasisPoint, big.NewInt(int64(asset.SharePremium))),
			bignumber.BasisPoint,
		)
	}

	if shares.Sign() < 0 {
		return nil, ErrMinimumMintNotMet
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: param.TokenOut, Amount: &shares},
		Fee:            &pool.TokenAmount{Token: s.Info.Tokens[0], Amount: bignumber.ZeroBI},
		Gas:            lo.Ternary(s.staticExtra.SimpleTeller, tellerDirectGas, defaultGas),
	}, nil
}

func (s *PoolSimulator) UpdateBalance(param pool.UpdateBalanceParams) {}

func (s *PoolSimulator) GetMetaInfo(_ string, _ string) any {
	approval := s.staticExtra.LiquidRefer
	if s.staticExtra.SimpleTeller {
		approval = common.HexToAddress(s.Info.Address) // the vault pulls tokens in enter()
	}
	return MetaInfo{
		LiquidRefer:     s.staticExtra.LiquidRefer,
		Teller:          s.staticExtra.Teller,
		ApprovalAddress: approval,
		BlockNumber:     s.Info.BlockNumber,
	}
}
