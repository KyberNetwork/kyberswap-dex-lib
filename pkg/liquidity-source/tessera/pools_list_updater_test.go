package tessera

import (
	"context"
	"os"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

func TestPoolsListUpdater(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip()
	}

	cfg := &Config{
		DexId:          "tessera",
		TesseraIndexer: "0x505352DA2918C6a06f12F3d59FFb79905d43439f",
		TesseraEngine:  "0x31E99E05fEE3DCe580aF777c3fd63Ee1b3b40c17",
		TesseraSwap:    "0x55555522005BcAE1c2424D474BfD5ed477749E3e",
	}
	client := ethrpc.New("https://base.kyberengineering.io").SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))
	updater := NewPoolsListUpdater(cfg, client)

	// 1. Initial Call
	pools, metadata, err := updater.GetNewPools(context.Background(), nil)
	assert.NoError(t, err)
	assert.NotNil(t, pools)
	assert.NotNil(t, metadata)

	// 2. Subsequent Call with metadata - should return nothing more if all were fetched
	pools2, metadata2, err := updater.GetNewPools(context.Background(), metadata)
	assert.NoError(t, err)
	if pools2 != nil {
		assert.NotEmpty(t, pools2)
	}
	assert.NotNil(t, metadata2)

	assert.NotNil(t, updater)
	assert.Equal(t, cfg, updater.cfg)
}
