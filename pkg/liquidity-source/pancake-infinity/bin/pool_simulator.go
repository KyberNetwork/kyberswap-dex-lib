package bin

import (
	"fmt"
	"math/big"
	"slices"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake-infinity/shared"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/eth"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
	"github.com/KyberNetwork/logger"
)

var (
	defaultGas int64 = 125000
)

type PoolSimulator struct {
	pool.Pool
	hook Hook

	vault, binPoolManager, permit2, hookAddress common.Address
	parameters                                  string

	isNative           [2]bool
	lpFee, protocolFee *uint256.Int

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

	if staticExtra.HasSwapPermissions {
		return nil, shared.ErrUnsupportedHook
	}

	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:  entityPool.Address,
			Exchange: entityPool.Exchange,
			Type:     entityPool.Type,
			Tokens: lo.Map(entityPool.Tokens,
				func(item *entity.PoolToken, index int) string { return item.Address }),
			Reserves: lo.Map(entityPool.Reserves,
				func(item string, index int) *big.Int { return bignumber.NewBig(item) }),
		}},
		vault:          staticExtra.VaultAddress,
		binPoolManager: staticExtra.PoolManagerAddress,
		permit2:        staticExtra.Permit2Address,
		hookAddress:    staticExtra.HooksAddress,
		parameters:     staticExtra.Parameters,
		lpFee:          uint256.NewInt(uint64(entityPool.SwapFee)),
		protocolFee:    extra.ProtocolFee,
		bins:           extra.Bins,
		hook:           GetHook(staticExtra.HooksAddress),
		activeId:       extra.ActiveBinID,
		binStep:        staticExtra.BinStep,
		isNative:       staticExtra.IsNative,
	}, nil
}

func (p *PoolSimulator) GetExchange() string {
	return p.hook.GetExchange()
}

func (p *PoolSimulator) swap(exactIn, swapForY bool, amountIn *big.Int) (*swapResult, error) {
	protocolFee := getZeroForOneFee(p.protocolFee)
	if !swapForY {
		protocolFee = getOneForZeroFee(p.protocolFee)
	}

	swapFee := p.lpFee
	if !protocolFee.IsZero() {
		swapFee = calculateSwapFee(protocolFee, p.lpFee)
	}

	amountsLeft, overflow := uint256.FromBig(amountIn)
	if overflow {
		return nil, shared.ErrInvalidAmountIn
	}

	id := p.activeId
	var (
		amountsUnspecified                                 uint256.Int
		amountsInWithFees, amountsOutOfBin, totalFee, pFee *uint256.Int
		binsReserveChanges                                 []binReserveChanges
	)

	for !amountsLeft.IsZero() {
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

func (p *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenIn, tokenOut := params.TokenAmountIn, params.TokenOut
	indexIn, indexOut := p.GetTokenIndex(tokenIn.Token), p.GetTokenIndex(tokenOut)
	if indexIn < 0 || indexOut < 0 {
		return nil, shared.ErrInvalidToken
	}

	if p.activeId == 0 {
		return nil, shared.ErrUninitializedPool
	}

	res, err := p.swap(true, indexIn == 0, tokenIn.Amount)
	if err != nil {
		return nil, err
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: res.Amount.ToBig(),
		},
		RemainingTokenAmountIn: &pool.TokenAmount{
			Token:  tokenIn.Token,
			Amount: bignumber.ZeroBI,
		},
		Fee: &pool.TokenAmount{
			Token:  tokenIn.Token,
			Amount: res.Fee.ToBig(),
		},
		Gas: defaultGas,
		SwapInfo: SwapInfo{
			NewActiveID:        res.NewActiveID,
			BinsReserveChanges: res.BinsReserveChanges,
		},
	}, nil
}

func (p *PoolSimulator) CalcAmountIn(params pool.CalcAmountInParams) (*pool.CalcAmountInResult, error) {
	tokenIn, tokenOut := params.TokenIn, params.TokenAmountOut
	indexIn, indexOut := p.GetTokenIndex(tokenIn), p.GetTokenIndex(tokenOut.Token)
	if indexIn < 0 || indexOut < 0 {
		return nil, shared.ErrInvalidToken
	}

	if p.activeId == 0 {
		return nil, shared.ErrUninitializedPool
	}

	res, err := p.swap(false, indexIn == 0, tokenOut.Amount)
	if err != nil {
		return nil, err
	}

	return &pool.CalcAmountInResult{
		TokenAmountIn: &pool.TokenAmount{
			Token:  tokenIn,
			Amount: res.Amount.ToBig(),
		},
		RemainingTokenAmountOut: &pool.TokenAmount{
			Token:  tokenOut.Token,
			Amount: bignumber.ZeroBI,
		},
		Fee: &pool.TokenAmount{
			Token:  tokenIn,
			Amount: res.Fee.ToBig(),
		},
		Gas: defaultGas,
		SwapInfo: SwapInfo{
			NewActiveID:        res.NewActiveID,
			BinsReserveChanges: res.BinsReserveChanges,
		},
	}, nil
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	swapInfo, ok := params.SwapInfo.(SwapInfo)
	if !ok {
		logger.WithFields(logger.Fields{
			"address": p.Info.Address,
		}).Warn("invalid swap info")
		return
	}

	// active bin ID
	p.activeId = swapInfo.NewActiveID

	// update reserves of bins
	totalBinReserveChanges := make(map[uint32][2]*uint256.Int)
	for _, b := range swapInfo.BinsReserveChanges {
		changes, ok := totalBinReserveChanges[b.BinID]
		if !ok {
			totalBinReserveChanges[b.BinID] = [2]*uint256.Int{
				new(uint256.Int).Sub(b.AmountXIn, b.AmountXOut),
				new(uint256.Int).Sub(b.AmountYIn, b.AmountYOut),
			}
			continue
		}

		changes[0].Add(changes[0], b.AmountXIn).Sub(changes[0], b.AmountXOut)
		changes[1].Add(changes[1], b.AmountYIn).Sub(changes[1], b.AmountYOut)
	}
	newBins := p.bins[:0]
	for _, b := range p.bins {
		if changes, ok := totalBinReserveChanges[b.ID]; ok {
			b.ReserveX = changes[0].Add(b.ReserveX, changes[0])
			b.ReserveY = changes[1].Add(b.ReserveY, changes[1])
		}

		newBins = append(newBins, b)
	}
	p.bins = newBins
}

func (p *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *p
	cloned.bins = slices.Clone(p.bins)
	return &cloned
}

func (p *PoolSimulator) GetMetaInfo(tokenIn string, tokenOut string) any {
	tokenInAddress, tokenOutAddress := eth.AddressZero, eth.AddressZero
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
		Fee:         uint32(p.lpFee.Uint64()),
		Parameters:  p.parameters,
		HookAddress: p.hookAddress,
		HookData:    []byte{},
	}
}
