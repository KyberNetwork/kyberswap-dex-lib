package mantisswap

import "math/big"

type PoolState struct {
	Paused      bool           `json:"Paused"`
	SwapAllowed bool           `json:"SwapAllowed"`
	BaseFee     *big.Int       `json:"BaseFee"`
	LpRatio     *big.Int       `json:"LpRatio"`
	SlippageA   *big.Int       `json:"SlippageA"`
	SlippageN   *big.Int       `json:"SlippageN"`
	SlippageK   *big.Int       `json:"SlippageK"`
	LPs         map[string]*LP `json:"LPs"`
}

type Extra struct {
	Paused      bool           `json:"Paused"`
	SwapAllowed bool           `json:"SwapAllowed"`
	BaseFee     *big.Int       `json:"BaseFee"`
	LpRatio     *big.Int       `json:"LpRatio"`
	SlippageA   *big.Int       `json:"SlippageA"`
	SlippageN   *big.Int       `json:"SlippageN"`
	SlippageK   *big.Int       `json:"SlippageK"`
	LPs         map[string]*LP `json:"LPs"`
}

type swapInfo struct {
	lps map[string]*LP
}

type LP struct {
	Address        string   `json:"address"`
	Decimals       uint8    `json:"decimals"`
	Asset          *big.Int `json:"asset"`
	Liability      *big.Int `json:"liability"`
	LiabilityLimit *big.Int `json:"liabilityLimit"`
}

type Gas struct {
	Swap int64
}
