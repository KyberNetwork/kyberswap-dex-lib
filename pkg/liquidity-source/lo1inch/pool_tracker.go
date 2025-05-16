package lo1inch

import (
	"context"
	"strings"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	lo1inchRouter "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/lo1inch/router"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
)

type PoolTracker struct {
}

func NewPoolTracker() *PoolTracker {
	return &PoolTracker{}
}

func (t *PoolTracker) GetNewPoolState(ctx context.Context, p entity.Pool, params pool.GetNewPoolStateParams) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, params, nil)
}

func (t *PoolTracker) getNewPoolState(
	_ context.Context, p entity.Pool,
	params pool.GetNewPoolStateParams,
	_ map[common.Address]gethclient.OverrideAccount,
) (entity.Pool, error) {
	// get all orders from the extra
	var extra Extra
	if err := json.Unmarshal([]byte(p.Extra), &extra); err != nil {
		return p, err
	}

	orderTracking := make(map[string]struct {
		block uint64
		index uint
	})
	// get info from OrderFilled and OrderCancelled events
	for _, log := range params.Logs {
		if log.Removed {
			continue
		}
		switch log.Topics[0] {
		case lo1inchRouter.RouterABI.Events["OrderFilled"].ID:
			// TODO: Convert hex topic to uint256.Int
			// Extract orderHash and remainingAmount from log.Data
			orderHash := common.BytesToHash(log.Data[:32]).Hex()
			remainingMakerAmount := new(uint256.Int).SetBytes(log.Data[32:])

			if _, ok := orderTracking[orderHash]; ok && (orderTracking[orderHash].block > log.BlockNumber || (orderTracking[orderHash].block == log.BlockNumber && orderTracking[orderHash].index >= log.Index)) {
				continue
			}
			tr := orderTracking[orderHash]
			tr.block = log.BlockNumber
			tr.index = log.Index
			orderTracking[orderHash] = tr
			for _, order := range extra.TakeToken0Orders {
				if strings.EqualFold(order.OrderHash, orderHash) {
					order.RemainingMakerAmount = remainingMakerAmount
					break
				}
			}
			for _, order := range extra.TakeToken1Orders {
				if strings.EqualFold(order.OrderHash, orderHash) {
					order.RemainingMakerAmount = remainingMakerAmount
					break
				}
			}
		case lo1inchRouter.RouterABI.Events["OrderCancelled"].ID:
		}
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}
	p.Extra = string(extraBytes)
	return p, nil
}
