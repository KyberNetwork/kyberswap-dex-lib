//go:generate go run github.com/tinylib/msgp -unexported -tests=false -v
//msgp:tuple PoolSimulator Gas AtomicLimits
//msgp:ignore Meta
//msgp:shim PoolStateVersion as:uint using:uint/PoolStateVersion
//msgp:shim *big.Int as:[]byte using:msgpencode.EncodeInt/msgpencode.DecodeInt

package synthetix

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
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

	_initializeOnce sync.Once `msg:"-"`
}

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

	p := &PoolSimulator{
		Pool: pool.Pool{
			Info: info,
		},
		poolStateVersion: getPoolStateVersion(chainID),
		poolState:        extra.PoolState,
		gas:              DefaultGas,
	}
	if err := p.initializeOnce(); err != nil {
		return nil, err
	}
	return p, nil
}

func (p *PoolSimulator) initializeOnce() error {
	var err error
	p._initializeOnce.Do(func() {
		err = p.poolState.initialize()
	})
	return err
}

func (p *PoolSimulator) CalcAmountOut(
	param pool.CalcAmountOutParams,
) (*pool.CalcAmountOutResult, error) {
	if err := p.initializeOnce(); err != nil {
		return nil, err
	}

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
		return &pool.CalcAmountOutResult{}, err
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

		if strings.EqualFold(address, tokenAddress.String()) {
			continue
		}

		swappableTokens = append(swappableTokens, strings.ToLower(tokenAddress.String()))
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

// AtomicLimits implement pool.SwapLimit for synthetic
// key is blockTimestamp, and the limit is its balance
// The balance is stored WITHOUT decimals
// DONOT directly modify it, use UpdateLimit if needed
type AtomicLimits struct {
	lock   *sync.RWMutex `msg:"-"`
	Limits map[string]*big.Int
}

func NewLimits(atomicMaxVolumePerBlocks map[string]*big.Int) pool.SwapLimit {
	return &AtomicLimits{
		lock:   &sync.RWMutex{},
		Limits: atomicMaxVolumePerBlocks,
	}
}

// GetLimit returns a copy of balance for the token in Inventory
func (i *AtomicLimits) GetLimit(blockTimeStamp string) *big.Int {
	i.lock.RLock()
	defer i.lock.RUnlock()
	balance, avail := i.Limits[blockTimeStamp]
	if !avail {
		return big.NewInt(0)
	}
	return big.NewInt(0).Set(balance)
}

// UpdateLimit will reduce the limit to reflect the change in inventory
// note this delta is amount without Decimal
func (i *AtomicLimits) UpdateLimit(blockTimeStamp, _ string, decreaseDelta, _ *big.Int) (*big.Int, *big.Int, error) {
	i.lock.Lock()
	defer i.lock.Unlock()
	volLimit, avail := i.Limits[blockTimeStamp]
	if !avail {
		return big.NewInt(0), big.NewInt(0), pool.ErrTokenNotAvailable
	}
	if volLimit.Cmp(decreaseDelta) < 0 {
		return big.NewInt(0), big.NewInt(0), pool.ErrNotEnoughInventory
	}
	i.Limits[blockTimeStamp] = volLimit.Sub(volLimit, decreaseDelta)

	return big.NewInt(0).Set(i.Limits[blockTimeStamp]), big.NewInt(0), nil
}
