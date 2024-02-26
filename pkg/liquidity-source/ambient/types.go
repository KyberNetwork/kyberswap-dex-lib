package ambient

import (
	"math/big"
)

const (
	DexType    = "ambient"
	fetchLimit = 1000
)

type FetchPoolsResponse struct {
	Pools []Pool `json:"pools"`
}

type Pool struct {
	ID          string `json:"id"`
	BlockCreate string `json:"blockCreate"`
	TimeCreate  uint64 `json:"timeCreate,string"`
	Base        string `json:"base"`
	Quote       string `json:"quote"`
	PoolIdx     string `json:"poolIdx"`
}

type PoolListUpdaterMetadata struct {
	LastCreateTime uint64 `json:"lastCreateTime"`
}

type StaticExtra struct {
	Base    string `json:"base"`
	Quote   string `json:"quote"`
	PoolIdx string `json:"pool_idx"`
}

type PoolData struct {
}

type swapPool struct {
	hash         string
	feeRate      uint16
	protocolTake uint8
}

/* @notice Represents the accumulated flow between user and pool within a transaction.
*
* @param baseFlow_ Represents the cumulative base side token flow. Negative for
*   flow going to the user, positive for flow going to the pool.
* @param quoteFlow_ The cumulative quote side token flow.
* @param baseProto_ The total amount of base side tokens being collected as protocol
*   fees. The above baseFlow_ value is inclusive of this quantity.
* @param quoteProto_ The total amount of quote tokens being collected as protocol
*   fees. The above quoteFlow_ value is inclusive of this quantity. */
type pairFlow struct {
	baseFlow   *big.Int
	quoteFlow  *big.Int
	baseProto  *big.Int
	quoteProto *big.Int
}

/* @notice Defines a single requested swap on a pre-specified pool.
*
* @dev A directive indicating no swap action must set *both* qty and limitPrice to
*      zero. qty=0 alone will indicate the use of a flexible back-filled rolling
*      quantity.
*
* @param isBuy_ If true, swap converts base-side token to quote-side token.
*               Vice-versa if false.
* @param inBaseQty_ If true, swap quantity is denominated in base-side token.
*                   If false in quote side token.
* @param rollType_  The flavor of rolling gap fill that should be applied (if any)
*                   to this leg of the directive. See Chaining.sol for list of
*                   rolling type codes.
* @param qty_ The total amount to be swapped. (Or rolling target if rollType_ is
*             enabled)
* @param limitPrice_ The maximum (minimum) *price to pay, if a buy (sell) swap
*           *at the margin*. I.e. the swap will keep exeucting until the curve
*           reaches this price (or exhausts the specified quantity.) Represented
*           as the square root of the pool's price ratio in Q64.64 fixed-point. */
type swapDirective struct {
	isBuy      bool
	inBaseQty  bool
	rollType   uint8
	qty        *big.Int
	limitPrice *big.Int
}
