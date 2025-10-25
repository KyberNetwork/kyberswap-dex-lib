package synthetix

import (
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/swaplimit"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type Gas struct {
	ExchangeAtomically int64
	Exchange           int64
}

type Meta struct {
	SourceCurrencyKey      string `json:"sourceCurrencyKey"`
	DestinationCurrencyKey string `json:"destinationCurrencyKey"`
	UseAtomicExchange      bool   `json:"useAtomicExchange"`
}

type PoolSimulator struct {
	pool.Pool

	poolStateVersion PoolStateVersion
	poolState        *PoolState
	gas              Gas
}

var _ = pool.RegisterFactory1(DexTypeSynthetix, NewPoolSimulator)
var _ = pool.RegisterUseSwapLimit(DexTypeSynthetix)

func NewPoolSimulator(entityPool entity.Pool, chainID valueobject.ChainID) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	tokens := make([]string, 0, len(entityPool.Tokens))
	for _, poolToken := range entityPool.Tokens {
		tokens = append(tokens, poolToken.Address)
	}

	info := pool.PoolInfo{
		Address:  entityPool.Address,
		Exchange: entityPool.Exchange,
		Type:     entityPool.Type,
		Tokens:   tokens,
	}

	return &PoolSimulator{
		Pool: pool.Pool{
			Info: info,
		},
		poolStateVersion: getPoolStateVersion(chainID),
		poolState:        extra.PoolState,
		gas:              DefaultGas,
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(
	param pool.CalcAmountOutParams,
) (*pool.CalcAmountOutResult, error) {
	var (
		tokenAmountIn = param.TokenAmountIn
		tokenOut      = param.TokenOut
		limit         = param.Limit
	)
	if limit == nil {
		return nil, ErrNoSwapLimit
	}
	amountOutAfterFees, feeAmount, err := p.getAmountOut(
		p.getCurrencyKeyFromToken(tokenAmountIn.Token),
		p.getCurrencyKeyFromToken(tokenOut),
		tokenAmountIn.Amount,
	)
	if err != nil {
		return nil, err
	}

	tokenAmountOut := &pool.TokenAmount{
		Token:  tokenOut,
		Amount: amountOutAfterFees,
	}
	tokenAmountFee := &pool.TokenAmount{
		Token:  tokenOut,
		Amount: feeAmount,
	}

	var estimatedGas int64
	if p.poolStateVersion == PoolStateVersionAtomic {
		estimatedGas = p.gas.ExchangeAtomically
	} else {
		estimatedGas = p.gas.Exchange
	}

	synthetixTradeVolume, err := p.GetAtomicVolume(tokenAmountIn, tokenOut)
	if err != nil {
		return &pool.CalcAmountOutResult{
			TokenAmountOut: tokenAmountOut,
			Fee:            tokenAmountFee,
			Gas:            estimatedGas,
		}, err
	}
	if synthetixTradeVolume != nil {
		allowedVol := limit.GetLimit(strconv.FormatUint(p.poolState.BlockTimestamp, 10))

		if allowedVol.Cmp(synthetixTradeVolume) < 0 {
			return nil, ErrSurpassedVolumeLimit
		}
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: tokenAmountOut,
		Fee:            tokenAmountFee,
		Gas:            estimatedGas,
	}, nil
}

// GetAtomicVolume returns the atomic volume of the trade in case of Ethereum
func (p *PoolSimulator) GetAtomicVolume(
	tokenAmountIn pool.TokenAmount,
	tokenOut string,
) (*big.Int, error) {
	// Normal Synthetix pool does not have to validate the total volume
	if !p.useAtomicExchange() {
		return nil, nil
	}

	exchanger := GetExchanger(p.poolStateVersion, p.poolState)

	exchangerWithFeeRecAlternatives, ok := exchanger.(*ExchangerWithFeeRecAlternatives)
	if !ok {
		return nil, errors.New("can not cast to ExchangerWithFeeRecAlternatives")
	}

	return exchangerWithFeeRecAlternatives.getSourceSUSDValue(
		tokenAmountIn.Amount,
		p.getCurrencyKeyFromToken(tokenAmountIn.Token),
		p.getCurrencyKeyFromToken(tokenOut),
	)
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {}

func (p *PoolSimulator) CanSwapFrom(address string) []string { return p.CanSwapTo(address) }

func (p *PoolSimulator) CanSwapTo(address string) []string {
	var tokenIndex = p.GetTokenIndex(address)
	if tokenIndex < 0 {
		return nil
	}

	synths := p.poolState.Synths

	swappableTokens := make([]string, 0, len(synths)-1)
	for _, token := range synths {
		tokenAddress := token

		if strings.EqualFold(address, hexutil.Encode(tokenAddress[:])) {
			continue
		}

		swappableTokens = append(swappableTokens, hexutil.Encode(tokenAddress[:]))
	}

	return swappableTokens
}

func (p *PoolSimulator) GetMetaInfo(tokenIn string, tokenOut string) interface{} {
	sourceCurrencyKey := p.poolState.CurrencyKeyBySynth[common.HexToAddress(tokenIn)]
	destinationCurrencyKey := p.poolState.CurrencyKeyBySynth[common.HexToAddress(tokenOut)]
	useAtomicExchange := p.useAtomicExchange()

	return Meta{
		SourceCurrencyKey:      sourceCurrencyKey,
		DestinationCurrencyKey: destinationCurrencyKey,
		UseAtomicExchange:      useAtomicExchange,
	}
}

// getAmountOut returns amountOutAfterFees, feeAmount and error
func (p *PoolSimulator) getAmountOut(sourceCurrencyKey string, destinationCurrencyKey string, amountIn *big.Int) (*big.Int, *big.Int, error) {
	// Check if amountIn is valid
	if err := p.validateAmountIn(sourceCurrencyKey, amountIn); err != nil {
		return nil, nil, err
	}

	exchanger := GetExchanger(p.poolStateVersion, p.poolState)

	amountReceived, fee, _, err := exchanger.GetAmountsOut(amountIn, sourceCurrencyKey, destinationCurrencyKey)

	return amountReceived, fee, err
}

func (p *PoolSimulator) getCurrencyKeyFromToken(token string) string {
	currencyKeyBySynth := p.poolState.CurrencyKeyBySynth

	return currencyKeyBySynth[common.HexToAddress(token)]
}

func (p *PoolSimulator) useAtomicExchange() bool {
	return p.poolStateVersion == PoolStateVersionAtomic
}

// validateAmountIn Check if amountIn is valid
func (p *PoolSimulator) validateAmountIn(currencyKey string, amount *big.Int) error {
	currencyKeyTotalSupply := p.poolState.SynthsTotalSupply[currencyKey]

	// Check if the amount of synth is bigger than the total supply of synth
	// If true, return error
	if amount.Cmp(currencyKeyTotalSupply) > 0 {
		return ErrAmountExceedsTotalSupply
	}

	return nil
}

func (p *PoolSimulator) GetPoolState() *PoolState {
	return p.poolState
}

func (p *PoolSimulator) GetPoolStateVersion() PoolStateVersion {
	return p.poolStateVersion
}

func (p *PoolSimulator) CalculateLimit() map[string]*big.Int {
	var (
		s      = p.poolState
		maxVol = big.NewInt(0).Set(p.poolState.AtomicMaxVolumePerBlock)
	)
	if s.BlockTimestamp == s.LastAtomicVolume.Time {
		maxVol = maxVol.Sub(p.poolState.AtomicMaxVolumePerBlock, p.poolState.LastAtomicVolume.Volume)
	}
	return map[string]*big.Int{
		fmt.Sprint(s.BlockTimestamp): maxVol,
	}
}

// AtomicLimits is an alias for swaplimit.Inventory
// Deprecated: directly use swaplimit.Inventory.
type AtomicLimits = swaplimit.Inventory

// NewLimits has key: "blockTimeStamp", value: limit and only decrease without increasing.
// Deprecated: directly use swaplimit.NewInventory.
func NewLimits(atomicMaxVolumePerBlocks map[string]*big.Int) pool.SwapLimit {
	return swaplimit.NewInventory(DexTypeSynthetix, atomicMaxVolumePerBlocks)
}
