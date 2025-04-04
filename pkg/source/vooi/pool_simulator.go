package vooi

import (
	"errors"
	"math/big"

	"github.com/KyberNetwork/blockchain-toolkit/dsmath"
	"github.com/KyberNetwork/blockchain-toolkit/integer"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	utils "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var (
	ErrPoolIsPaused         = errors.New("pool is paused")
	ErrAssetDeactivated     = errors.New("asset was deactivated by owner")
	ErrMaxSupplyExceeded    = errors.New("forbidden: max supply exceeded")
	ErrSameAddress          = errors.New("same address")
	ErrInitialAmountTooHigh = errors.New("initial amount too high")
	ErrInvalidValue         = errors.New("invalid value")
	ErrNotEnoughCash        = errors.New("not enough cash")
	ErrAmountTooLow         = errors.New("amount too low")
	ErrForbidden            = errors.New("forbidden")
)

type (
	PoolSimulator struct {
		pool.Pool

		a      *big.Int
		lpFee  *big.Int
		paused bool

		assetByToken map[string]Asset
		indexByToken map[string]int

		gas Gas
	}

	PoolExtra struct {
		AssetByToken map[string]Asset `json:"assetByToken"`
		IndexByToken map[string]int   `json:"indexByToken"`
		A            *big.Int         `json:"a"`
		LPFee        *big.Int         `json:"lpFee"`
		Paused       bool             `json:"paused"`
	}

	PoolSimulatorMetadata struct {
		FromID int `json:"fromId"`
		ToID   int `json:"toId"`
	}

	Gas struct {
		Swap int64
	}

	vooiSwapInfo struct {
		newFromAssetCash *big.Int
		newToAssetCash   *big.Int
	}
)

var _ = pool.RegisterFactory0(DexTypeVooi, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var poolExtra PoolExtra
	if err := json.Unmarshal([]byte(entityPool.Extra), &poolExtra); err != nil {
		return nil, err
	}

	return &PoolSimulator{
		Pool: pool.Pool{
			Info: pool.PoolInfo{
				Address:  entityPool.Address,
				Exchange: entityPool.Exchange,
				Type:     entityPool.Type,
				Tokens:   lo.Map(entityPool.Tokens, func(item *entity.PoolToken, index int) string { return item.Address }),
				Reserves: lo.Map(entityPool.Reserves, func(item string, index int) *big.Int { return utils.NewBig(item) }),
			},
		},

		a:      poolExtra.A,
		lpFee:  poolExtra.LPFee,
		paused: poolExtra.Paused,

		assetByToken: poolExtra.AssetByToken,
		indexByToken: poolExtra.IndexByToken,

		gas: defaultGas,
	}, nil
}

// CalcAmountOut calculate amount out from amount in, token in and token out
// Reference: https://lineascan.build/address/0xBc7f67fA9C72f9fcCf917cBCEe2a50dEb031462A
func (s *PoolSimulator) CalcAmountOut(
	param pool.CalcAmountOutParams,
) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn := param.TokenAmountIn
	tokenOut := param.TokenOut
	if s.paused {
		return nil, ErrPoolIsPaused
	}

	if tokenAmountIn.Token == tokenOut {
		return nil, ErrSameAddress
	}

	if tokenAmountIn.Amount.Cmp(integer.Zero()) <= 0 {
		return nil, ErrInvalidValue
	}

	fromAsset, toAsset := s.assetByToken[tokenAmountIn.Token], s.assetByToken[tokenOut]

	if !fromAsset.Active {
		return nil, ErrAssetDeactivated
	}

	if !toAsset.Active {
		return nil, ErrAssetDeactivated
	}

	if new(big.Int).Add(fromAsset.Cash, dsmath.ToWAD(tokenAmountIn.Amount, fromAsset.Decimals)).Cmp(fromAsset.MaxSupply) > 0 {
		return nil, ErrMaxSupplyExceeded
	}

	actualToAmount, lpFeeAmount, newFromAssetCash, newToAssetCash, err := s._swap(
		fromAsset,
		toAsset,
		dsmath.ToWAD(tokenAmountIn.Amount, fromAsset.Decimals),
		integer.Zero(),
	)
	if err != nil {
		return nil, err
	}

	actualToAmount = dsmath.FromWAD(actualToAmount, toAsset.Decimals)
	lpFeeAmount = dsmath.FromWAD(lpFeeAmount, toAsset.Decimals)

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: tokenOut, Amount: actualToAmount},
		Fee:            &pool.TokenAmount{Token: tokenOut, Amount: lpFeeAmount},
		Gas:            s.gas.Swap,
		SwapInfo: vooiSwapInfo{
			newFromAssetCash: newFromAssetCash,
			newToAssetCash:   newToAssetCash,
		},
	}, nil
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	swapInfo, ok := params.SwapInfo.(vooiSwapInfo)
	if !ok {
		logger.Warnf("failed to UpdateBalance for Vooi %v %v pool, wrong swapInfo type", s.Info.Address, s.Info.Exchange)
		return
	}
	fromAsset, toAsset := s.assetByToken[params.TokenAmountIn.Token], s.assetByToken[params.TokenAmountOut.Token]

	fromAsset.Cash = new(big.Int).Set(swapInfo.newFromAssetCash)
	toAsset.Cash = new(big.Int).Set(swapInfo.newToAssetCash)

	s.assetByToken[params.TokenAmountIn.Token] = fromAsset
	s.assetByToken[params.TokenAmountOut.Token] = toAsset
}

func (s *PoolSimulator) GetMetaInfo(tokenIn string, tokenOut string) interface{} {
	return PoolSimulatorMetadata{
		FromID: s.indexByToken[tokenIn],
		ToID:   s.indexByToken[tokenOut],
	}
}

// _swap expect fromAmount and minimumToAmount to be in WAD
func (s *PoolSimulator) _swap(
	fromAsset Asset,
	toAsset Asset,
	fromAmount *big.Int,
	minimumToAmount *big.Int,
) (*big.Int, *big.Int, *big.Int, *big.Int, error) {
	if !_isConvertableToInt256(new(big.Int).Add(fromAmount, fromAsset.Cash)) {
		return nil, nil, nil, nil, ErrInitialAmountTooHigh
	}

	actualToAmount, lpFeeAmount, err := s._quoteFrom(fromAsset, toAsset, fromAmount)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	if minimumToAmount.Cmp(actualToAmount) > 0 {
		return nil, nil, nil, nil, ErrAmountTooLow
	}

	newFromAssetCash := new(big.Int).Add(fromAsset.Cash, fromAmount)
	newToAssetCash := new(big.Int).Add(
		new(big.Int).Sub(toAsset.Cash, actualToAmount),
		lpFeeAmount,
	)

	// revert if cov ratio < 1% to avoid precision error
	if dsmath.WDiv(newToAssetCash, toAsset.Liability).Cmp(new(big.Int).Div(dsmath.WAD, big.NewInt(100))) < 0 {
		return nil, nil, nil, nil, ErrForbidden
	}

	return actualToAmount, lpFeeAmount, newFromAssetCash, newToAssetCash, nil
}

// _quoteFrom Quotes the actual amount user would receive in a swap, taking in account slippage and lpFeeAmount
//   - @param fromAsset The initial asset
//   - @param toAsset The asset wanted by user
//   - @param fromAmount The amount to quote
//   - @return actualToAmount The actual amount user would receive
//   - @return lpFeeAmount The lpFeeAmount that will be applied
//     */
func (s *PoolSimulator) _quoteFrom(
	fromAsset Asset,
	toAsset Asset,
	fromAmount *big.Int,
) (*big.Int, *big.Int, error) {
	var (
		idealToAmount  *big.Int
		actualToAmount *big.Int
		toCash         = toAsset.Cash
		toLiability    = toAsset.Liability
		fromCash       = fromAsset.Cash
		fromLiability  = fromAsset.Liability
		ampFactor      = s.a
	)

	if toLiability.Cmp(integer.Zero()) == 0 || fromLiability.Cmp(integer.Zero()) == 0 {
		return nil, nil, ErrInvalidValue
	}

	d := new(big.Int).Sub(
		new(big.Int).Add(fromCash, toCash),
		dsmath.WMul(
			ampFactor,
			new(big.Int).Add(
				new(big.Int).Div(
					new(big.Int).Mul(fromLiability, fromLiability),
					fromCash,
				),
				new(big.Int).Div(
					new(big.Int).Mul(toLiability, toLiability),
					toCash,
				),
			),
		),
	)

	rx := dsmath.WDiv(new(big.Int).Add(fromCash, fromAmount), fromLiability)
	b := new(big.Int).Sub(
		new(big.Int).Div(
			new(big.Int).Mul(
				fromLiability,
				new(big.Int).Sub(
					rx,
					dsmath.WDiv(ampFactor, rx),
				),
			),
			toLiability,
		),
		dsmath.WDiv(d, toLiability),
	)
	ry := _solveQuad(b, ampFactor)
	dy := new(big.Int).Sub(dsmath.WMul(toLiability, ry), toCash)

	if dy.Cmp(integer.Zero()) < 0 {
		idealToAmount = new(big.Int).Mul(dy, big.NewInt(-1))
	} else {
		idealToAmount = new(big.Int).Set(dy)
	}

	if toCash.Cmp(idealToAmount) < 0 {
		return nil, nil, ErrNotEnoughCash
	}

	lpFeeAmount := dsmath.WMul(idealToAmount, s.lpFee)
	if fromAmount.Cmp(integer.Zero()) > 0 {
		actualToAmount = new(big.Int).Sub(idealToAmount, lpFeeAmount)
	} else {
		actualToAmount = new(big.Int).Set(idealToAmount)
	}

	return actualToAmount, lpFeeAmount, nil
}

func _isConvertableToInt256(value *big.Int) bool {
	return value.Cmp(abi.MaxUint256) <= 0
}

// _solveQuad Solve quadratic equation
// (((b * b) + (c * 4 * WAD_I)).sqrt(b) - b) / 2;
//   - @param b quadratic equation b coefficient
//   - @param c quadratic equation c coefficient
func _solveQuad(b *big.Int, c *big.Int) *big.Int {
	return new(big.Int).Div(
		new(big.Int).Sub(
			sqrt(
				new(big.Int).Add(
					new(big.Int).Mul(b, b),
					new(big.Int).Mul(new(big.Int).Mul(c, integer.Four()), dsmath.WAD),
				),
				b,
			),
			b,
		),
		integer.Two(),
	)
}
