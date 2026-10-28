package oneswap

import (
	"context"
	"testing"
	"testing/synctest"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/test/ethrpcmock"
)

func TestGetNewPoolStatePinsTotalSupply(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		srv, recorder := ethrpcmock.New(t, oneSwapABI, erc20ABI)
		client := ethrpc.New(srv.URL).SetMulticallContract(common.HexToAddress("0x4444444444444444444444444444444444444444"))
		tracker := NewPoolTracker(client)
		p := entity.Pool{Address: "0x7777777777777777777777777777777777777777", Tokens: []*entity.PoolToken{{Address: "0x8888888888888888888888888888888888888888"}, {Address: "0x9999999999999999999999999999999999999999"}}}
		got, err := tracker.GetNewPoolState(context.Background(), p, pool.GetNewPoolStateParams{})
		require.NoError(t, err)
		require.Equal(t, uint64(ethrpcmock.Block), got.BlockNumber)
		require.Len(t, got.Reserves, 3)
		ethrpcmock.CheckPool(t, "oneswap_TestGetNewPoolStatePinsTotalSupply", got)
		ethrpcmock.CheckRequests(t, "oneswap_TestGetNewPoolStatePinsTotalSupply", recorder)
		ethrpcmock.RequireRequestsPinned(t, recorder, 2)
	})
}
