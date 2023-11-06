package pool

import (
	"errors"
	"math/big"
	"sync"

	"github.com/KyberNetwork/logger"
)

var (
	ErrCalcAmountOutPanic = errors.New("calcAmountOut was panic")
)

type Pool struct {
	Info PoolInfo
}

func (t *Pool) GetInfo() PoolInfo {
	return t.Info
}

func (t *Pool) GetTokens() []string {
	return t.Info.Tokens
}

func (t *Pool) GetReserves() []*big.Int {
	return t.Info.Reserves
}

// CanSwapTo is the base method to get all swappable tokens from a pool by a given token address
// Pools with custom logic should override this method
func (t *Pool) CanSwapTo(address string) []string {
	result := make([]string, 0, len(t.Info.Tokens))
	var tokenIndex = t.GetTokenIndex(address)
	if tokenIndex < 0 {
		return result
	}

	for i := 0; i < len(t.Info.Tokens); i += 1 {
		if i != tokenIndex {
			result = append(result, t.Info.Tokens[i])
		}
	}

	return result
}

// by default pool is bi-directional so just call CanSwapTo here
// Pools with custom logic should override this method
func (t *Pool) CanSwapFrom(address string) []string {
	return t.CanSwapTo(address)
}

func (t *Pool) GetAddress() string {
	return t.Info.Address
}

func (t *Pool) GetExchange() string {
	return t.Info.Exchange
}

func (t *Pool) Equals(other IPoolSimulator) bool {
	return t.GetAddress() == other.GetAddress()
}

func (t *Pool) GetTokenIndex(address string) int {
	return t.Info.GetTokenIndex(address)
}

func (t *Pool) GetType() string {
	return t.Info.Type
}

type CalcAmountOutResult struct {
	TokenAmountOut *TokenAmount
	Fee            *TokenAmount
	Gas            int64
	SwapInfo       interface{}
}

func (r *CalcAmountOutResult) IsValid() bool {
	return r.TokenAmountOut != nil && r.TokenAmountOut.Amount != nil && r.TokenAmountOut.Amount.Cmp(ZeroBI) > 0
}

type UpdateBalanceParams struct {
	TokenAmountIn  TokenAmount
	TokenAmountOut TokenAmount
	Fee            TokenAmount
	SwapInfo       interface{}

	//Inventory is a reference to a per-request inventory balances.
	// key is tokenAddress, balance is big.Float
	// Must use reference (not copy)
	Inventory *Inventory
}

type PoolToken struct {
	Token               string
	Balance             *big.Int
	Weight              uint
	PrecisionMultiplier *big.Int
	VReserve            *big.Int
}

type PoolInfo struct {
	Address    string
	ReserveUsd float64
	SwapFee    *big.Int
	Exchange   string
	Type       string
	Tokens     []string
	Reserves   []*big.Int
	Checked    bool
}

func (t *PoolInfo) GetTokenIndex(address string) int {
	for i, poolToken := range t.Tokens {
		if poolToken == address {
			return i
		}
	}
	return -1
}

// wrap around pool.CalcAmountOut and catch panic
func CalcAmountOut(pool IPoolSimulator, tokenAmountIn TokenAmount, tokenOut string) (res *CalcAmountOutResult, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = ErrCalcAmountOutPanic
			logger.WithFields(
				logger.Fields{
					"recover":     r,
					"poolAddress": pool.GetAddress(),
				}).Warn(err.Error())
		}
	}()

	return pool.CalcAmountOut(tokenAmountIn, tokenOut)
}

// Inventory is a map of tokenAddress- balance.
// The balance is stored WITHOUT decimals
// DONOT directly modify it
type Inventory struct {
	lock    *sync.RWMutex
	Balance map[string]*big.Int
}

func NewInventory(balance map[string]*big.Int) *Inventory {
	return &Inventory{
		lock:    &sync.RWMutex{},
		Balance: balance,
	}
}

// GetBalance returns a copy of balance for the Inventory
func (i *Inventory) GetBalance(tokenAddress string) *big.Int {
	i.lock.RLock()
	defer i.lock.RUnlock()
	balance, avail := i.Balance[tokenAddress]
	if !avail {
		return big.NewInt(0)
	}
	return big.NewInt(0).Set(balance)
}

// UpdateBalance will reduce the Balance to reflect the change in inventory
// note this delta is amount with Decimal
func (i *Inventory) UpdateBalance(decreaseTokenAddress, increaseTokenAddress string, decreaseDelta, increaseDelta *big.Int) (*big.Int, *big.Int, error) {
	i.lock.Lock()
	defer i.lock.Unlock()
	decreasedTokenBalance, avail := i.Balance[decreaseTokenAddress]
	if !avail {
		return big.NewInt(0), big.NewInt(0), ErrTokenNotAvailable
	}
	if decreasedTokenBalance.Cmp(decreaseDelta) < 0 {
		return big.NewInt(0), big.NewInt(0), ErrNotEnoughInventory
	}
	i.Balance[decreaseTokenAddress] = decreasedTokenBalance.Sub(decreasedTokenBalance, decreaseDelta)

	increasedTokenBalance, avail := i.Balance[increaseTokenAddress]
	if !avail {
		return big.NewInt(0), big.NewInt(0), ErrTokenNotAvailable
	}
	i.Balance[increaseTokenAddress] = increasedTokenBalance.Add(decreasedTokenBalance, increaseDelta)
	return big.NewInt(0).Set(i.Balance[decreaseTokenAddress]), big.NewInt(0).Set(i.Balance[increaseTokenAddress]), nil
}
