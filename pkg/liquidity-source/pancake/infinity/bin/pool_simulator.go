package bin

import (
	"fmt"
	"math/big"
	"slices"
	"strings"

	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake/infinity/shared"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var (
	defaultGas int64 = 125000
)

type PoolSimulator struct {
	pool.Pool
	hook     Hook
	exchange string

	vault, binPoolManager, permit2, hookAddress common.Address
	parameters                                  string

	isNative         [2]bool
	chainID          valueobject.ChainID
	fee, protocolFee uint32
	hookSwapFee      uint32
	swapFee          *uint256.Int

	bins     []Bin
	activeId uint32
	binStep  uint16
}

var _ = pool.RegisterFactory1(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool, chainID valueobject.ChainID) (*PoolSimulator, error) {
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, fmt.Errorf("failed to unmarshal static extra: %w", err)
	}

	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, fmt.Errorf("failed to unmarshal extra: %w", err)
	}

	hook, ok := GetHook(staticExtra.HooksAddress)
	if !ok && staticExtra.HasSwapPermissions {
		return nil, shared.ErrUnsupportedHook
	}
	exchange := entityPool.Exchange
	if exchange == DexType { // temp fix, won't work for omni-bin morphing into omni-bin-dynamic case
		exchange = strings.Replace(hook.GetExchange(), DexType, entityPool.Exchange, 1)
	}

	return &PoolSimulator{
		Info: pool.PoolInfo{
			Address:  entityPool.Address,
			Exchange: entityPool.Exchange,
			Type:     entityPool.Type,
			Tokens: lo.Map(entityPool.Tokens,
				func(item *entity.PoolToken, index int) string { return item.Address }),
			Reserves: lo.Map(entityPool.Reserves,
				func(item string, index int) *big.Int { return bignumber.NewBig(item) }),
			BlockNumber: entityPool.BlockNumber,
		},
		vault:          staticExtra.VaultAddress,
		binPoolManager: staticExtra.PoolManagerAddress,
		permit2:        staticExtra.Permit2Address,
		hookAddress:    staticExtra.HooksAddress,
		parameters:     staticExtra.Parameters,
		fee:            staticExtra.Fee,
		swapFee:        uint256.NewInt(uint64(entityPool.SwapFee)),
		protocolFee:    extra.ProtocolFee,
		hookSwapFee:    extra.HookSwapFee,
		bins:           extra.Bins,
		hook:           hook,
		exchange:       exchange,
		activeId:       extra.ActiveBinID,
		binStep:        staticExtra.BinStep,
		isNative:       staticExtra.IsNative,
		chainID:        chainID,
	}, nil
}

func (p *PoolSimulator) GetExchange() string {
	return p.exchange
}

// swapU is swap on a uint256 amount. It consumes amountsLeft.
func (p *PoolSimulator) swapU(exactIn, swapForY bool, amountsLeft *uint256.Int) (*swapResult, error) {
	protocolFee := getZeroForOneFee(p.protocolFee)
	if !swapForY {
		protocolFee = getOneForZeroFee(p.protocolFee)
	}

	swapFee := p.swapFee

	if p.hookSwapFee > 0 {
		// beforeSwap: fee = floor(amount * hookSwapFee / 1e6) of the specified amount, taken
		// by the hook. Exact-in swaps amount - fee, exact-out swaps amount + fee.
		// amount < 2^128 (int128 on-chain) and fee < 2^20, so the product fits in 256 bits.
		var hookFee uint256.Int
		hookFee.SetUint64(uint64(p.hookSwapFee))
		hookFee.Mul(&hookFee, amountsLeft).Div(&hookFee, _PIPS)
		if exactIn {
			amountsLeft.Sub(amountsLeft, &hookFee)
		} else if _, overflow := amountsLeft.AddOverflow(amountsLeft, &hookFee); overflow {
			return nil, shared.ErrInvalidAmountIn
		}
		if amountsLeft.IsZero() {
			return nil, ErrInsufficientAmountUnSpecified
		}
	}

	id := p.activeId
	var (
		amountsUnspecified                                 uint256.Int
		amountsInWithFees, amountsOutOfBin, totalFee, pFee *uint256.Int
		binsReserveChanges                                 []binReserveChanges
	)

	for {
		bin, err := GetBinById(p.bins, id)
		if err != nil {
			return nil, err
		}

		if !bin.IsEmpty(!swapForY) {
			if exactIn {
				amountsInWithFees, amountsOutOfBin, totalFee, err = bin.GetAmountsOut(swapFee, p.binStep, swapForY, amountsLeft)
				amountsLeft.Sub(amountsLeft, amountsInWithFees)
				amountsUnspecified.Add(&amountsUnspecified, amountsOutOfBin)
			} else {
				amountsInWithFees, amountsOutOfBin, totalFee, err = bin.GetAmountsIn(swapFee, p.binStep, swapForY, amountsLeft)
				amountsLeft.Sub(amountsLeft, amountsOutOfBin)
				amountsUnspecified.Add(&amountsUnspecified, amountsInWithFees)
			}

			if err != nil {
				return nil, err
			}

			if amountsInWithFees.Sign() > 0 {
				pFee = getProtocolFeeAmt(totalFee, protocolFee, swapFee)
				if !pFee.IsZero() {
					amountsInWithFees.Sub(amountsInWithFees, pFee)
				}
			}

			newBin := bin.Clone()
			if swapForY {
				newBin.ReserveX.Sub(newBin.ReserveX, amountsInWithFees)
				newBin.ReserveY.Add(newBin.ReserveY, amountsOutOfBin)
			} else {
				newBin.ReserveX.Add(newBin.ReserveX, amountsOutOfBin)
				newBin.ReserveY.Sub(newBin.ReserveY, amountsInWithFees)
			}

			price, err := getPriceFromID(id, p.binStep)
			if err != nil {
				return nil, err
			}

			liquidity, err := newBin.GetLiquidity(price)
			if err != nil {
				return nil, err
			}

			if liquidity.Gt(_MAX_LIQUIDITY_PER_BIN) {
				return nil, ErrMaxLiquidityPerBin
			}

			binsReserveChanges = append(binsReserveChanges, newBinReserveChanges(
				id, !swapForY, amountsInWithFees, amountsOutOfBin,
			))
		}

		if amountsLeft.IsZero() {
			break
		}

		id, err = GetNextNonEmptyBin(swapForY, p.bins, id)
		if err != nil {
			return nil, err
		}
	}

	if amountsUnspecified.IsZero() {
		return nil, ErrInsufficientAmountUnSpecified
	}

	return &swapResult{
		Amount:             &amountsUnspecified,
		Fee:                pFee,
		NewActiveID:        id,
		BinsReserveChanges: binsReserveChanges,
	}, nil
}

func (p *PoolSimulator) gas() int64 {
	if p.hookSwapFee > 0 {
		return defaultGas + hookSwapFeeGas
	}
	return defaultGas
}

// CalcAmtOut is CalcAmountOut with uint256 amounts.
func (p *PoolSimulator) CalcAmtOut(params pool.CalcAmtOutParams) (*pool.CalcAmtOutResult, error) {
	tokenIn, tokenOut := params.TokenAmountIn.Token, params.TokenOut
	indexIn, indexOut := p.GetTokenIndex(tokenIn), p.GetTokenIndex(tokenOut)
	if indexIn < 0 || indexOut < 0 {
		return nil, shared.ErrInvalidToken
	}

	if p.activeId == 0 {
		return nil, shared.ErrUninitializedPool
	}

	amountIn := params.TokenAmountIn.Amount
	// A native-flagged side trades at native's own decimals on-chain; only Arc's differ (18 vs 6).
	if p.isNative[indexIn] && p.chainID == valueobject.ChainIDArc {
		if _, overflow := amountIn.MulOverflow(&amountIn, arcNativeScale); overflow {
			return nil, shared.ErrInvalidAmountIn
		}
	}

	res, err := p.swapU(true, indexIn == 0, &amountIn)
	if err != nil {
		return nil, err
	}

	amountOut := res.Amount
	if p.isNative[indexOut] && p.chainID == valueobject.ChainIDArc {
		amountOut.Div(amountOut, arcNativeScale)
	}

	out := &pool.CalcAmtOutResult{
		TokenAmountOut:         &pool.TokenAmt{Token: tokenOut, Amount: *amountOut},
		RemainingTokenAmountIn: &pool.TokenAmt{Token: tokenIn},
		Gas:                    p.gas(),
		SwapInfo: SwapInfo{
			NewActiveID:        res.NewActiveID,
			BinsReserveChanges: res.BinsReserveChanges,
		},
	}
	return out, nil
}

// arcNativeScale is the 1e12 decimals gap between Arc's native currency and its wrapped native token.
var arcNativeScale = uint256.NewInt(1e12)

// CalcAmtIn is the exact-out quote. On Arc a native-flagged side trades at native's 18 decimals, so the amount out is
// scaled up and the amount in scaled down (floor, as the legacy CalcAmountIn did).
func (p *PoolSimulator) CalcAmtIn(params pool.CalcAmtInParams) (*pool.CalcAmtInResult, error) {
	tokenIn, tokenOut := params.TokenIn, params.TokenAmountOut.Token
	indexIn, indexOut := p.GetTokenIndex(tokenIn), p.GetTokenIndex(tokenOut)
	if indexIn < 0 || indexOut < 0 {
		return nil, shared.ErrInvalidToken
	}

	if p.activeId == 0 {
		return nil, shared.ErrUninitializedPool
	}

	amountOut := params.TokenAmountOut.Amount
	if p.isNative[indexOut] && p.chainID == valueobject.ChainIDArc {
		if _, overflow := amountOut.MulOverflow(&amountOut, arcNativeScale); overflow {
			return nil, shared.ErrInvalidAmountIn
		}
	}

	res, err := p.swapU(false, indexIn == 0, &amountOut)
	if err != nil {
		return nil, err
	}

	amountIn := res.Amount
	if p.isNative[indexIn] && p.chainID == valueobject.ChainIDArc {
		amountIn.Div(amountIn, arcNativeScale)
	}

	return &pool.CalcAmtInResult{
		TokenAmountIn:           &pool.TokenAmt{Token: tokenIn, Amount: *amountIn},
		RemainingTokenAmountOut: &pool.TokenAmt{Token: tokenOut},
		Gas:                     p.gas(),
		SwapInfo: SwapInfo{
			NewActiveID:        res.NewActiveID,
			BinsReserveChanges: res.BinsReserveChanges,
		},
	}, nil
}

// UpdateState moves the active bin and applies the per-bin reserve changes. The swap only needs SwapInfo, so the
// amounts are ignored. Reserves are replaced, not mutated, because clones share the Int pointers.
func (p *PoolSimulator) UpdateState(params pool.UpdateStateParams) {
	swapInfo, ok := params.SwapInfo.(SwapInfo)
	if !ok {
		logger.WithFields(logger.Fields{
			"address": p.Info.Address,
		}).Warn("invalid swap info")
		return
	}

	p.activeId = swapInfo.NewActiveID
	for _, c := range swapInfo.BinsReserveChanges {
		idx, err := FindBinArrIndex(p.bins, c.BinID)
		if err != nil {
			continue
		}
		b := &p.bins[idx]
		x := new(uint256.Int).Add(b.ReserveX, c.AmountXIn)
		y := new(uint256.Int).Add(b.ReserveY, c.AmountYIn)
		b.ReserveX, b.ReserveY = x.Sub(x, c.AmountXOut), y.Sub(y, c.AmountYOut)
	}
}

func (p *PoolSimulator) CloneState() pool.IPoolSim {
	cloned := *p
	cloned.bins = slices.Clone(p.bins)
	return &cloned
}

// GetReserves returns the reserves as uint256. A nil, negative, or oversized reserve becomes zero.
func (p *PoolSimulator) GetReserves() []uint256.Int {
	out := make([]uint256.Int, len(p.Info.Reserves))
	for i, r := range p.Info.Reserves {
		if r != nil && r.Sign() >= 0 {
			if u, overflow := uint256.FromBig(r); !overflow {
				out[i] = *u
			}
		}
	}
	return out
}

// CalcLimit returns nil: bin pools have no shared inventory limit.
func (p *PoolSimulator) CalcLimit() map[string]*uint256.Int {
	return nil
}

func (p *PoolSimulator) GetMetaInfo(tokenIn string, tokenOut string) any {
	tokenInAddress, tokenOutAddress := valueobject.AddrZero, valueobject.AddrZero
	if !p.isNative[p.GetTokenIndex(tokenIn)] {
		tokenInAddress = common.HexToAddress(tokenIn)
	}
	if !p.isNative[p.GetTokenIndex(tokenOut)] {
		tokenOutAddress = common.HexToAddress(tokenOut)
	}

	return PoolMetaInfo{
		Vault:       p.vault,
		PoolManager: p.binPoolManager,
		Permit2Addr: p.permit2,
		TokenIn:     tokenInAddress,
		TokenOut:    tokenOutAddress,
		Fee:         p.fee,
		SwapFee:     p.swapFee.Uint64(),
		Parameters:  p.parameters,
		HookAddress: p.hookAddress,
		HookData:    []byte{},
		BlockNumber: p.Info.BlockNumber,
	}
}

func (s *PoolSimulator) SwapReceiveNativeIn(tokenIn, tokenOut string, _ valueobject.ChainID) bool {
	meta := s.GetMetaInfo(tokenIn, tokenOut).(PoolMetaInfo)
	return meta.TokenIn == valueobject.AddrZero
}

func (s *PoolSimulator) SwapReturnNativeOut(tokenIn, tokenOut string, _ valueobject.ChainID) bool {
	meta := s.GetMetaInfo(tokenIn, tokenOut).(PoolMetaInfo)
	return meta.TokenOut == valueobject.AddrZero
}
