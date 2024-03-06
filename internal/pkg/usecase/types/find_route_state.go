package types

import (
	"context"

	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"

	"github.com/KyberNetwork/router-service/pkg/logger"
)

// FindRouteState enclose the data we need for a findRoute rquest
type FindRouteState struct {
	// map PoolAddress - IPoolSimulator implementation
	Pools map[string]poolpkg.IPoolSimulator
	// map LimitType-SwapLimit
	SwapLimit map[string]poolpkg.SwapLimit
	// IS PMM staled is set to true if the reserve of PMM doesnt change for config.PoolManager.StallingPMMThreshold sec
	IsPMMStalled bool
}

type AdjacentList struct {
	Data map[string]*AddressList
}

// AddressList will store the list of Pool to use for Adjacent map.
// It is managed by mempool and once it's out of scope, call memPool.ReturnAddressList() to return it.
// Note that the Arr underneath will expand to max N pool with N equal to number of best pools selected for each request.
// TrueLen denotes the real len of the AddressList, it is set to 0 when mempool need to zero-out the memory before re-using.
type AddressList struct {
	Arr     []string
	TrueLen int
}

// AddAddress is a bit hacky, we use the underlying arr to store the address
func (a *AddressList) AddAddress(ctx context.Context, address string) {
	if a.TrueLen == len(a.Arr) {
		// have to do append here
		a.Arr = append(a.Arr, address)
	} else if a.TrueLen < len(a.Arr) {
		a.Arr[a.TrueLen] = address
	} else {
		logger.Errorf(ctx, "AddressList TrueLen %d is greater than underlying a.Arr's len %d", a.TrueLen, len(a.Arr))
	}

	a.TrueLen++

}
