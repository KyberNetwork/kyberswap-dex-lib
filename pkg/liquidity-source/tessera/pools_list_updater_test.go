package tessera

import (
	"context"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

func TestPoolsListUpdater(t *testing.T) {
	cfg := &Config{
		DexId:       "tessera",
		IndexerAddr: "0x505352DA2918C6a06f12F3d59FFb79905d43439f",
		EngineAddr:  "0x31E99E05fEE3DCe580aF777c3fd63Ee1b3b40c17",
	}
	client := ethrpc.New("https://base.kyberengineering.io").SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))
	updater := NewPoolsListUpdater(cfg, client)

	pools, _, err := updater.GetNewPools(context.Background(), nil)
	assert.NoError(t, err)
	assert.NotNil(t, pools)

	assert.NotNil(t, updater)
	assert.Equal(t, cfg, updater.cfg)
}
